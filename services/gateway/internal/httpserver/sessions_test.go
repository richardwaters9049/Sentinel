package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
)

type sessionMemory struct {
	rows map[string]access.Session
	fail bool
}

func (m *sessionMemory) CreateSession(_ context.Context, s access.Session, old string) error {
	if m.fail {
		return errors.New("private store error")
	}
	delete(m.rows, old)
	m.rows[s.Hash] = s
	return nil
}
func (m *sessionMemory) TouchSession(_ context.Context, hash string, now time.Time) (access.Session, error) {
	if m.fail {
		return access.Session{}, errors.New("private store error")
	}
	s, ok := m.rows[hash]
	if !ok || !s.ExpiresAt.After(now) || !s.LastSeenAt.After(now.Add(-access.SessionIdleTimeout)) {
		return s, access.ErrSessionNotFound
	}
	s.LastSeenAt = now
	m.rows[hash] = s
	s.Principal.ExpiresAt = s.ExpiresAt
	return s, nil
}
func (m *sessionMemory) RevokeSession(_ context.Context, hash string) error {
	if m.fail {
		return errors.New("private store error")
	}
	delete(m.rows, hash)
	return nil
}

func TestConsoleSessionLifecycle(t *testing.T) {
	verifier, tokens := accessFixture(t)
	store := &sessionMemory{rows: make(map[string]access.Session)}
	s := New(nil, nil, nil, &stubFindingReader{}).WithAccess(verifier, nil).WithSessions(store, "http://127.0.0.1:3000")
	perform := func(method, path, body string, cookie *http.Cookie, csrf, origin string) *httptest.ResponseRecorder {
		r := localRequest(method, path, strings.NewReader(body))
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if csrf != "" {
			r.Header.Set("X-Sentinel-CSRF", csrf)
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	login := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		return perform("POST", "/api/v1/auth/login", `{"token":"`+tokens[access.Analyst]+`"}`, cookie, "", "http://127.0.0.1:3000")
	}
	for _, origin := range []string{"", "null", "https://attacker.invalid", "http://127.0.0.1:3000/"} {
		if w := perform("POST", "/api/v1/auth/login", `{"token":"`+tokens[access.Analyst]+`"}`, nil, "", origin); w.Code != 403 {
			t.Fatalf("origin %q accepted: %d", origin, w.Code)
		}
	}
	if w := perform("POST", "/api/v1/auth/login", `{"token":"`+tokens[access.Collector]+`"}`, nil, "", "http://127.0.0.1:3000"); w.Code != 403 {
		t.Fatal("collector login accepted")
	}
	first := login(nil)
	if first.Code != 201 {
		t.Fatal(first.Code, first.Body.String())
	}
	cookie := first.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" || cookie.Domain != "" || cookie.Secure {
		t.Fatal("invalid local cookie properties")
	}
	var body struct {
		CSRF string `json:"csrf_token"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(store.rows) != 1 || store.rows[access.Digest(cookie.Value)].CredentialHash != access.Digest(tokens[access.Analyst]) {
		t.Fatal("session persistence contains incorrect credential hash")
	}
	if strings.Contains(first.Body.String(), cookie.Value) || strings.Contains(first.Body.String(), tokens[access.Analyst]) {
		t.Fatal("secret returned in session JSON")
	}
	if w := perform("GET", "/api/v1/session", "", cookie, "", ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, tc := range []struct{ csrf, origin string }{{"", "http://127.0.0.1:3000"}, {body.CSRF, ""}, {body.CSRF, "https://attacker.invalid"}, {"wrong", "http://127.0.0.1:3000"}} {
		if w := perform("PATCH", "/api/v1/findings/fnd_test/status", `{"status":"triaged"}`, cookie, tc.csrf, tc.origin); w.Code != 403 {
			t.Fatal("CSRF check bypassed", w.Code)
		}
	}
	if w := perform("PATCH", "/api/v1/findings/fnd_test/status", `{"status":"triaged"}`, cookie, body.CSRF, "http://127.0.0.1:3000"); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := perform("PATCH", "/api/v1/detections/DET-AUTH-001", `{"enabled":false}`, cookie, body.CSRF, "http://127.0.0.1:3000"); w.Code != 403 {
		t.Fatal("session role escalation")
	}
	rotated := login(cookie)
	if rotated.Code != 201 {
		t.Fatal(rotated.Code)
	}
	newCookie := rotated.Result().Cookies()[0]
	if cookie.Value == newCookie.Value || len(store.rows) != 1 {
		t.Fatal("session fixation/rotation failure")
	}
	if w := perform("GET", "/api/v1/session", "", cookie, "", ""); w.Code != 401 {
		t.Fatal("old session replayed")
	}
	if err := json.Unmarshal(rotated.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	logout := perform("POST", "/api/v1/auth/logout", "", newCookie, body.CSRF, "http://127.0.0.1:3000")
	if logout.Code != 200 || logout.Result().Cookies()[0].MaxAge != -1 || len(store.rows) != 0 {
		t.Fatal("logout did not revoke and clear session")
	}
	if w := perform("GET", "/api/v1/session", "", newCookie, "", ""); w.Code != 401 {
		t.Fatal("logged-out session replayed")
	}
	fresh := login(nil)
	newCookie = fresh.Result().Cookies()[0]
	record := store.rows[access.Digest(newCookie.Value)]
	record.LastSeenAt = time.Now().Add(-31 * time.Minute)
	store.rows[record.Hash] = record
	if w := perform("GET", "/api/v1/session", "", newCookie, "", ""); w.Code != 401 {
		t.Fatal("idle session accepted")
	}
	fresh = login(nil)
	newCookie = fresh.Result().Cookies()[0]
	record = store.rows[access.Digest(newCookie.Value)]
	record.CredentialHash = access.Digest("removed credential")
	store.rows[record.Hash] = record
	if w := perform("GET", "/api/v1/session", "", newCookie, "", ""); w.Code != 401 {
		t.Fatal("revoked credential session accepted")
	}
	store.fail = true
	if w := perform("GET", "/api/v1/session", "", newCookie, "", ""); w.Code != 503 || strings.Contains(w.Body.String(), "private store error") {
		t.Fatal("store failure did not fail closed")
	}
}

func TestSecureSessionCookie(t *testing.T) {
	s := New(nil, nil, nil, nil).WithSessions(nil, "https://console.example.test")
	w := httptest.NewRecorder()
	s.setSessionCookie(w, "synthetic", time.Now().Add(time.Hour))
	cookie := w.Result().Cookies()[0]
	if cookie.Name != "__Host-sentinel_session" || !cookie.Secure || !cookie.HttpOnly || cookie.Domain != "" || cookie.Path != "/" {
		t.Fatal(cookie)
	}
}
