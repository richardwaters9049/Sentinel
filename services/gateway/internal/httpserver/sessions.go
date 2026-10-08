package httpserver

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
)

type sessionKey struct{}
type sessionContext struct {
	ID        string
	ExpiresAt time.Time
}

func (s *Server) WithSessions(store access.SessionStore, origin string) *Server {
	s.sessions = store
	s.consoleOrigin = origin
	s.sessionCookie = "sentinel_session"
	if len(origin) >= 8 && origin[:8] == "https://" {
		s.sessionCookie = "__Host-sentinel_session"
	}
	return s
}
func (s *Server) originAllowed(r *http.Request) bool {
	origins := r.Header.Values("Origin")
	return len(origins) == 1 && origins[0] == s.consoleOrigin
}
func (s *Server) sessionID(r *http.Request) (string, bool) {
	cookies := r.CookiesNamed(s.sessionCookie)
	returnValue := ""
	if len(cookies) == 1 {
		returnValue = cookies[0].Value
	}
	return returnValue, access.ValidSessionID(returnValue)
}
func (s *Server) setSessionCookie(w http.ResponseWriter, id string, expires time.Time) {
	cookie := &http.Cookie{Name: s.sessionCookie, Value: id, Path: "/", HttpOnly: true, Secure: s.sessionCookie == "__Host-sentinel_session", SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: int(time.Until(expires).Seconds())}
	if id == "" {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, cookie)
}
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.access == nil || s.sessions == nil {
		writeAPIError(w, 503, "sign_in_unavailable", "interactive sign-in is not enabled")
		return
	}
	if !s.originAllowed(r) {
		s.logAccess(r, "POST /api/v1/auth/login", access.Principal{}, "invalid_origin", 403)
		writeAPIError(w, 403, "invalid_origin", "sign-in requires the configured console origin")
		return
	}
	if len(r.Header.Values("Authorization")) != 0 {
		writeAPIError(w, 400, "ambiguous_credentials", "sign-in requires one credential in the request body")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	var input struct {
		Token string `json:"token"`
	}
	if err := decodeStrictJSON(r, &input); err != nil {
		writeAPIError(w, 400, "invalid_json", "invalid sign-in request")
		return
	}
	now := time.Now().UTC()
	principal, err := s.access.Authenticate("Bearer "+input.Token, now)
	if err != nil {
		s.logAccess(r, "POST /api/v1/auth/login", access.Principal{}, "unauthenticated", 401)
		writeAPIError(w, 401, "unauthenticated", "credential is invalid or expired")
		return
	}
	if principal.Role == access.Collector {
		s.logAccess(r, "POST /api/v1/auth/login", principal, "forbidden", 403)
		writeAPIError(w, 403, "forbidden", "collector credentials cannot sign in to the console")
		return
	}
	id, err := access.RandomSessionID()
	if err != nil {
		writeAPIError(w, 503, "session_unavailable", "session could not be created")
		return
	}
	expires := now.Add(access.SessionLifetime)
	if principal.ExpiresAt.Before(expires) {
		expires = principal.ExpiresAt
	}
	session := access.Session{Hash: access.Digest(id), CredentialHash: access.Digest(input.Token), Principal: principal, CreatedAt: now, LastSeenAt: now, ExpiresAt: expires}
	old, _ := s.sessionID(r)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err = s.sessions.CreateSession(ctx, session, access.Digest(old)); err != nil {
		if errors.Is(err, access.ErrSessionCapacity) {
			writeAPIError(w, 429, "session_capacity", "active session limit reached")
			return
		}
		writeAPIError(w, 503, "session_unavailable", "session could not be created")
		return
	}
	s.setSessionCookie(w, id, expires)
	s.logAccess(r, "POST /api/v1/auth/login", principal, "signed_in", 201)
	writeJSON(w, 201, map[string]interface{}{"subject": principal.Subject, "role": principal.Role, "expires_at": expires, "csrf_token": access.CSRFToken(id)})
}
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(sessionKey{}).(sessionContext)
	if !ok {
		writeAPIError(w, 400, "session_required", "logout requires a console session")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.sessions.RevokeSession(ctx, access.Digest(session.ID)); err != nil {
		writeAPIError(w, 503, "session_unavailable", "session could not be revoked")
		return
	}
	s.setSessionCookie(w, "", time.Time{})
	writeJSON(w, 200, map[string]bool{"signed_out": true})
}
func (s *Server) cookiePrincipal(r *http.Request) (access.Principal, *http.Request, error) {
	id, ok := s.sessionID(r)
	if !ok || s.sessions == nil {
		return access.Principal{}, r, access.ErrUnauthenticated
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		tokens := r.Header.Values("X-Sentinel-CSRF")
		if !s.originAllowed(r) || len(tokens) != 1 || subtle.ConstantTimeCompare([]byte(tokens[0]), []byte(access.CSRFToken(id))) != 1 {
			return access.Principal{}, r, errCSRF
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	now := time.Now().UTC()
	session, err := s.sessions.TouchSession(ctx, access.Digest(id), now)
	if err != nil {
		return access.Principal{}, r, err
	}
	if !s.access.ValidateSession(session, now) {
		return access.Principal{}, r, access.ErrUnauthenticated
	}
	return session.Principal, r.WithContext(context.WithValue(r.Context(), sessionKey{}, sessionContext{ID: id, ExpiresAt: session.ExpiresAt})), nil
}

var errCSRF = errors.New("session CSRF check failed")
