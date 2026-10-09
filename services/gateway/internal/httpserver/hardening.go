package httpserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/observability"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type collectorKey struct{}
type collectorProof struct{ KeyHash, BodyHash string }

var noncePattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

func (s *Server) WithHardening(store access.SecurityStore, ops *observability.Operations) *Server {
	s.security = store
	s.operations = ops
	s.limits = newLimiter(4096)
	return s
}
func (s *Server) WithQueueStats(stats func(context.Context) (uint64, int, error)) *Server {
	s.queueStats = stats
	return s
}
func routeName(pattern string) string {
	if pattern == "" {
		return "unmatched"
	}
	return pattern
}
func methodName(method string) string {
	if len(method) > 16 {
		return "OTHER"
	}
	return method
}
func (s *Server) operational(next http.Handler) http.Handler {
	if s.operations == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, trace, err := observability.NewTrace(r.Context(), r.Header.Get("traceparent"))
		if err != nil {
			writeAPIError(w, 503, "request_unavailable", "request could not be started")
			return
		}
		r = r.WithContext(ctx)
		w.Header().Set("X-Sentinel-Request-ID", trace.ID)
		w.Header().Set("traceparent", observability.Header(ctx))
		response := &auditResponse{ResponseWriter: w, status: 200}
		_, pattern := s.mux.Handler(r)
		defer func() { s.operations.Complete(trace, routeName(pattern), methodName(r.Method), response.status) }()
		if s.access != nil && !(r.Method == "GET" && (r.URL.Path == "/health" || r.URL.Path == "/ready")) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = "unknown"
			}
			rate, burst := float64(300), float64(600)
			key := "peer:" + host
			if r.URL.Path == "/api/v1/auth/login" {
				rate = 1.0 / 6
				burst = 20
				key = "login:" + host
			}
			if !s.limits.allow(key, rate, burst, time.Now()) {
				s.operations.RateDenied.Add(1)
				w.Header().Set("Retry-After", "6")
				writeAPIError(response, 429, "rate_limited", "request limit reached")
				return
			}
			if !s.persistAccess(r, pattern, access.Principal{}, "received", 0) {
				writeAPIError(response, 503, "audit_unavailable", "security audit storage is unavailable")
				return
			}
		}
		next.ServeHTTP(response, r)
	})
}
func (s *Server) persistAccess(r *http.Request, pattern string, p access.Principal, outcome string, status int) bool {
	if s.security == nil {
		return s.operations == nil || s.access == nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	err := s.security.RecordAccess(ctx, access.AuditRecord{TraceID: observability.Current(ctx).ID, Method: methodName(r.Method), Route: routeName(pattern), Subject: p.Subject, Role: p.Role, Outcome: outcome, Status: status})
	if err != nil {
		if s.operations != nil {
			s.operations.AuditFailures.Add(1)
		}
		if s.auditLogger != nil {
			s.auditLogger.Error("access audit persistence failed", "trace_id", observability.Current(ctx).ID, "outcome", outcome)
		}
		return false
	}
	return true
}
func (s *Server) signedCollector(r *http.Request, p access.Principal) (*http.Request, error) {
	one := func(name string) string {
		values := r.Header.Values(name)
		if len(values) != 1 {
			return ""
		}
		return values[0]
	}
	stamp, nonce, sig := one("X-Sentinel-Timestamp"), one("X-Sentinel-Nonce"), one("X-Sentinel-Signature")
	now := time.Now().UTC()
	seconds, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || strconv.FormatInt(seconds, 10) != stamp || !noncePattern.MatchString(nonce) || time.Unix(seconds, 0).Before(now.Add(-30*time.Second)) || time.Unix(seconds, 0).After(now.Add(30*time.Second)) || r.URL.RawQuery != "" {
		return r, access.ErrUnauthenticated
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var oversized *http.MaxBytesError
		if errors.As(err, &oversized) {
			return r, err
		}
		return r, access.ErrUnauthenticated
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	keyHash, valid := s.access.VerifyCollector(p, sig, access.CollectorMessage(r.Method, r.URL.Path, stamp, nonce, body))
	if !valid {
		return r, access.ErrUnauthenticated
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err = s.security.ReserveCollectorNonce(ctx, p.Subject, nonce, access.Digest(string(body)), now.Add(2*time.Minute)); err != nil {
		return r, err
	}
	return r.WithContext(context.WithValue(r.Context(), collectorKey{}, collectorProof{keyHash, access.Digest(string(body))})), nil
}
func (s *Server) handleAccessAudit(w http.ResponseWriter, r *http.Request) {
	if s.security == nil {
		writeAPIError(w, 503, "audit_unavailable", "security audit storage is unavailable")
		return
	}
	limit := 100
	before := int64(0)
	var err error
	for key, values := range r.URL.Query() {
		if len(values) != 1 || (key != "limit" && key != "before") {
			writeAPIError(w, 400, "invalid_query", "invalid audit pagination")
			return
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 200 {
			writeAPIError(w, 400, "invalid_query", "limit must be 1–200")
			return
		}
	}
	if raw := r.URL.Query().Get("before"); raw != "" {
		before, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || before < 1 {
			writeAPIError(w, 400, "invalid_query", "before must be a positive record ID")
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	rows, err := s.security.ListAccess(ctx, before, limit)
	if err != nil {
		writeAPIError(w, 503, "audit_unavailable", "security audit storage is unavailable")
		return
	}
	writeJSON(w, 200, map[string]any{"records": rows, "count": len(rows)})
}
func (s *Server) handleAuditRetention(w http.ResponseWriter, r *http.Request) {
	if s.security == nil {
		writeAPIError(w, 503, "audit_unavailable", "security audit storage is unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	n, err := s.security.PruneAccess(ctx)
	if err != nil {
		writeAPIError(w, 503, "audit_unavailable", "retention could not be applied")
		return
	}
	writeJSON(w, 200, map[string]any{"deleted": n, "retention_days": 90, "batch_limit": 1000})
}
func (s *Server) handlePlatformMetrics(w http.ResponseWriter, r *http.Request) {
	if s.operations == nil {
		writeAPIError(w, 503, "metrics_unavailable", "metrics are unavailable")
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	s.operations.Metrics(w)
	if s.queueStats != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		pending, unacked, err := s.queueStats(ctx)
		if err != nil {
			fmt.Fprintln(w, "sentinel_consumer_available 0")
			return
		}
		fmt.Fprintf(w, "sentinel_consumer_available 1\nsentinel_consumer_pending %d\nsentinel_consumer_unacknowledged %d\n", pending, unacked)
	}
}

func (s *Server) requireCollector(r *http.Request, p access.Principal) (*http.Request, string, int) {
	updated, err := s.signedCollector(r, p)
	if err == nil {
		return updated, "", 0
	}
	s.operations.SignatureDenied.Add(1)
	var oversized *http.MaxBytesError
	if errors.As(err, &oversized) {
		return r, "body_too_large", 413
	}
	if errors.Is(err, access.ErrReplay) {
		return r, "collector_replay", 409
	}
	if errors.Is(err, access.ErrUnauthenticated) {
		return r, "invalid_collector_signature", 401
	}
	return r, "collector_storage_unavailable", 503
}
