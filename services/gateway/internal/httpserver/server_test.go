package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/readiness"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	New(readyChecker()).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	if got := res.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected application/json content type, got %q", got)
	}

	if !strings.Contains(res.Body.String(), "\"service\":\"gateway\"") {
		t.Fatalf("expected gateway service in response, got %s", res.Body.String())
	}
}

func TestReadyReturnsOKWhenDependenciesPass(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	res := httptest.NewRecorder()

	New(readyChecker()).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	if !strings.Contains(res.Body.String(), "\"ready\":true") {
		t.Fatalf("expected ready=true, got %s", res.Body.String())
	}
}

func TestReadyReturnsUnavailableWhenDependencyFails(t *testing.T) {
	t.Parallel()

	checker := readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return errors.New("down") },
	})

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	res := httptest.NewRecorder()

	New(checker).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}

	if !strings.Contains(res.Body.String(), "\"ready\":false") {
		t.Fatalf("expected ready=false, got %s", res.Body.String())
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	New(readyChecker()).Handler().ServeHTTP(res, req)

	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}

	if got := res.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected DENY frame policy, got %q", got)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	res := httptest.NewRecorder()

	New(readyChecker()).Handler().ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}
}

func readyChecker() *readiness.Checker {
	return readiness.New(time.Second, map[string]readiness.CheckFunc{
		"postgres": func(context.Context) error { return nil },
		"nats":     func(context.Context) error { return nil },
	})
}
