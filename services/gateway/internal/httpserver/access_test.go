package httpserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
)

func localRequest(method, target string, body io.Reader) *http.Request {
	r := httptest.NewRequest(method, target, body)
	r.RemoteAddr = "127.0.0.1:12345"
	return r
}

func accessFixture(t *testing.T) (*access.Verifier, map[string]string) {
	t.Helper()
	tokens := make(map[string]string)
	var entries []map[string]interface{}
	for _, role := range []string{access.Analyst, access.Administrator, access.Collector} {
		hash := sha256.Sum256([]byte(role))
		token := base64.RawURLEncoding.EncodeToString(hash[:])
		digest := sha256.Sum256([]byte(token))
		tokens[role] = token
		entries = append(entries, map[string]interface{}{"subject": "verified-" + role, "role": role, "token_sha256": hex.EncodeToString(digest[:]), "expires_at": time.Now().UTC().Add(time.Hour)})
	}
	data, _ := json.Marshal(entries)
	v, err := access.Parse(data, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return v, tokens
}

func TestRoutePermissions(t *testing.T) {
	verifier, tokens := accessFixture(t)
	s := New(nil, nil, nil, nil).WithAccess(verifier, nil)
	// A newly added mux route without a policy must remain inaccessible.
	s.mux.HandleFunc("GET /api/v1/unreviewed", func(http.ResponseWriter, *http.Request) { t.Fatal("unreviewed handler executed") })
	for _, tc := range []struct {
		method, path string
		allowed      []string
	}{
		{"GET", "/api/v1/events", []string{access.Analyst, access.Administrator}},
		{"GET", "/api/v1/behaviour/monitor", []string{access.Analyst, access.Administrator}},
		{"GET", "/api/v1/session", []string{access.Analyst, access.Administrator, access.Collector}},
		{"POST", "/api/v1/telemetry", []string{access.Collector}},
		{"PATCH", "/api/v1/detections/DET-AUTH-001", []string{access.Administrator}},
		{"PATCH", "/api/v1/intelligence/sources/local", []string{access.Administrator}},
		{"PATCH", "/api/v1/behaviour/settings", []string{access.Administrator}},
		{"POST", "/api/v1/behaviour/evaluations", []string{access.Administrator}},
		{"POST", "/api/v1/hunts/hnt_test/run", []string{access.Analyst, access.Administrator}},
		{"POST", "/api/v1/investigations", []string{access.Analyst, access.Administrator}},
		{"GET", "/api/v1/unreviewed", nil}, {"DELETE", "/api/v1/events", nil},
		{"GET", "/api/v1/../events", nil}, {"GET", "/api/v1/events//private", nil},
	} {
		for _, role := range []string{"", access.Analyst, access.Administrator, access.Collector} {
			t.Run(tc.method+tc.path+role, func(t *testing.T) {
				r := localRequest(tc.method, tc.path, nil)
				r.Header.Set("X-Sentinel-Role", "administrator")
				r.Header.Set("X-Sentinel-Actor", "spoofed")
				if role != "" {
					r.Header.Set("Authorization", "Bearer "+tokens[role])
				}
				res := httptest.NewRecorder()
				s.authorise(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(res, r)
				expected := 403
				if role == "" {
					expected = 401
				}
				for _, allowed := range tc.allowed {
					if role == allowed {
						expected = 204
					}
				}
				if res.Code != expected {
					t.Fatalf("got %d want %d: %s", res.Code, expected, res.Body.String())
				}
			})
		}
	}
}

func TestAuthenticatedActorAndAudit(t *testing.T) {
	verifier, tokens := accessFixture(t)
	store := &stubFindingReader{}
	var logs bytes.Buffer
	s := New(nil, nil, nil, store).WithAccess(verifier, slog.New(slog.NewJSONHandler(&logs, nil)))
	r := localRequest("PATCH", "/api/v1/findings/fnd_test/status", strings.NewReader(`{"status":"triaged"}`))
	r.Header.Set("Authorization", "Bearer "+tokens[access.Analyst])
	r.Header.Set("X-Sentinel-Actor", "spoofed-admin")
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, r)
	if res.Code != 200 || store.actorID != "verified-analyst" {
		t.Fatalf("status=%d actor=%s", res.Code, store.actorID)
	}
	if !strings.Contains(logs.String(), `"actor_id":"verified-analyst"`) || strings.Contains(logs.String(), tokens[access.Analyst]) || strings.Contains(logs.String(), "spoofed-admin") {
		t.Fatal("invalid or secret-bearing audit log")
	}
	r = localRequest("GET", "/api/v1/session", nil)
	r.Header.Add("Authorization", "Bearer "+tokens[access.Analyst])
	r.Header.Add("Authorization", "Bearer "+tokens[access.Administrator])
	res = httptest.NewRecorder()
	s.Handler().ServeHTTP(res, r)
	if res.Code != 401 {
		t.Fatal("duplicate authorisation accepted")
	}
}

func TestDevelopmentLoopbackBoundary(t *testing.T) {
	for _, remote := range []string{"127.0.0.1:1234", "[::1]:1234", "192.0.2.4:1234", "invalid"} {
		r := localRequest("GET", "/api/v1/events", nil)
		r.RemoteAddr = remote
		r.Header.Set("X-Forwarded-For", "127.0.0.1")
		res := httptest.NewRecorder()
		New(nil, nil, nil, nil).Handler().ServeHTTP(res, r)
		expected := 403
		if strings.HasPrefix(remote, "127.") || strings.HasPrefix(remote, "[::1]") {
			expected = 503
		}
		if res.Code != expected {
			t.Fatalf("%s: %d", remote, res.Code)
		}
	}
}
