package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
)

type principalKey struct{}

// WithAccess configures the server before Handler is used; it is immutable afterwards.
func (s *Server) WithAccess(verifier *access.Verifier, logger *slog.Logger) *Server {
	s.access = verifier
	s.auditLogger = logger
	return s
}

func requestActor(r *http.Request) string {
	if principal, ok := r.Context().Value(principalKey{}).(access.Principal); ok {
		return principal.Subject
	}
	return strings.TrimSpace(r.Header.Get("X-Sentinel-Actor"))
}

func (s *Server) register(pattern string, handler http.HandlerFunc, roles ...string) {
	s.policies[pattern] = roles
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) authorise(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && (r.URL.Path == "/health" || r.URL.Path == "/ready") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/auth/login" {
			s.handleLogin(w, r)
			return
		}
		if s.access == nil {
			// Forwarded headers never make a remote client a trusted local developer.
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
				writeAPIError(w, http.StatusForbidden, "development_local_only", "development API access requires a loopback client")
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		_, pattern := s.mux.Handler(r)
		headers := r.Header.Values("Authorization")
		var principal access.Principal
		var err error
		cookies := r.CookiesNamed(s.sessionCookie)
		if len(headers) == 1 && len(cookies) == 0 {
			principal, err = s.access.Authenticate(headers[0], time.Now().UTC())
		} else if len(headers) == 0 && len(cookies) == 1 {
			principal, r, err = s.cookiePrincipal(r)
		} else {
			err = access.ErrUnauthenticated
		}

		if err != nil {
			if errors.Is(err, errCSRF) {
				s.logAccess(r, pattern, access.Principal{}, "csrf_rejected", 403)
				writeAPIError(w, 403, "csrf_rejected", "session mutation requires a valid origin and CSRF token")
				return
			}
			if !errors.Is(err, access.ErrUnauthenticated) && !errors.Is(err, access.ErrSessionNotFound) {
				writeAPIError(w, 503, "session_unavailable", "session verification is unavailable")
				return
			}
			s.logAccess(r, pattern, access.Principal{}, "unauthenticated", http.StatusUnauthorized)
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeAPIError(w, http.StatusUnauthorized, "unauthenticated", "valid bearer credential required")
			return
		}
		permitted := false
		for _, role := range s.policies[pattern] {
			if principal.Role == role {
				permitted = true
			}
		}
		if !permitted {
			s.logAccess(r, pattern, principal, "forbidden", http.StatusForbidden)
			writeAPIError(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		if s.operations != nil {
			rate, burst := float64(30), float64(60)
			if principal.Role == access.Collector {
				rate = 100
				burst = 200
			}
			if !s.limits.allow("subject:"+principal.Subject, rate, burst, time.Now()) {
				s.operations.RateDenied.Add(1)
				w.Header().Set("Retry-After", "1")
				s.logAccess(r, pattern, principal, "rate_limited", 429)
				writeAPIError(w, 429, "rate_limited", "principal request limit reached")
				return
			}
			if principal.Role == access.Collector && r.URL.Path == "/api/v1/telemetry" {
				var code string
				var status int
				r, code, status = s.requireCollector(r, principal)
				if status != 0 {
					s.logAccess(r, pattern, principal, code, status)
					writeAPIError(w, status, code, "collector request could not be verified")
					return
				}
			}
			if !s.persistAccess(r, pattern, principal, "allowed", 0) {
				writeAPIError(w, 503, "audit_unavailable", "security audit storage is unavailable")
				return
			}
		}
		r = r.WithContext(context.WithValue(r.Context(), principalKey{}, principal))
		response := &auditResponse{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(response, r)
		s.logAccess(r, pattern, principal, "completed", response.status)
	})
}

type auditResponse struct {
	http.ResponseWriter
	status int
}

func (w *auditResponse) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (s *Server) logAccess(r *http.Request, pattern string, principal access.Principal, outcome string, status int) {
	s.persistAccess(r, pattern, principal, outcome, status)
	if s.auditLogger != nil {
		s.auditLogger.Info("API access decision", "method", r.Method, "route", pattern, "actor_id", principal.Subject,
			"role", principal.Role, "outcome", outcome, "status", status)
	}
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	principal, ok := r.Context().Value(principalKey{}).(access.Principal)
	if !ok {
		writeAPIError(w, http.StatusUnauthorized, "authentication_disabled", "development mode has no authenticated session")
		return
	}
	if session, ok := r.Context().Value(sessionKey{}).(sessionContext); ok {
		writeJSON(w, 200, map[string]interface{}{"subject": principal.Subject, "role": principal.Role, "expires_at": session.ExpiresAt, "csrf_token": access.CSRFToken(session.ID)})
		return
	}
	writeJSON(w, http.StatusOK, principal)
}
