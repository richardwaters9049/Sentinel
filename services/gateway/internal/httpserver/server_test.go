package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	New().Handler().ServeHTTP(res, req)

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

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	New().Handler().ServeHTTP(res, req)

	if got := res.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}

	if got := res.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected DENY frame policy, got %q", got)
	}
}
