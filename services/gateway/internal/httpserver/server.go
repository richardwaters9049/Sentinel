package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/readiness"
)

const maxRequestBodyBytes int64 = 1 << 20

type Server struct {
	mux       *http.ServeMux
	readiness *readiness.Checker
}

func New(readinessChecker *readiness.Checker) *Server {
	s := &Server{
		mux:       http.NewServeMux(),
		readiness: readinessChecker,
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return securityHeaders(limitRequestBody(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", method(http.MethodGet, s.handleHealth))
	s.mux.HandleFunc("/ready", method(http.MethodGet, s.handleReady))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"service":   "gateway",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if s.readiness == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status": "not_ready",
			"ready":  false,
		})
		return
	}

	result := s.readiness.Check(r.Context())
	status := http.StatusOK
	statusText := "ready"

	if !result.Ready {
		status = http.StatusServiceUnavailable
		statusText = "not_ready"
	}

	writeJSON(w, status, map[string]interface{}{
		"status":       statusText,
		"ready":        result.Ready,
		"dependencies": result.Dependencies,
	})
}

func method(allowed string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != allowed {
			w.Header().Set("Allow", allowed)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}
