package httpserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/observability"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type securityFixture struct {
	records []access.AuditRecord
	nonces  map[string]bool
	fail    bool
}

func (s *securityFixture) RecordAccess(_ context.Context, r access.AuditRecord) error {
	if s.fail {
		return errors.New("offline")
	}
	s.records = append(s.records, r)
	return nil
}
func (s *securityFixture) ListAccess(context.Context, int64, int) ([]access.AuditRecord, error) {
	return s.records, nil
}
func (s *securityFixture) PruneAccess(context.Context) (int64, error) { return 0, nil }
func (s *securityFixture) ReserveCollectorNonce(_ context.Context, subject, nonce, digest string, _ time.Time) error {
	if s.fail {
		return errors.New("offline")
	}
	if s.nonces[subject+nonce] {
		return access.ErrReplay
	}
	s.nonces[subject+nonce] = true
	return nil
}

func TestAuditFailsClosedAndOmitsSecrets(t *testing.T) {
	verifier, tokens := accessFixture(t)
	store := &securityFixture{}
	ops := observability.New(nil, "", "test")
	defer ops.Close(context.Background())
	s := New(nil, nil, nil, nil).WithAccess(verifier, nil).WithHardening(store, ops)
	call := func() int {
		r := localRequest("GET", "/api/v1/session?secret=never-log", nil)
		r.Header.Set("Authorization", "Bearer "+tokens[access.Analyst])
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w.Code
	}
	if status := call(); status != 200 || len(store.records) != 3 {
		t.Fatalf("status %d records %d", status, len(store.records))
	}
	raw, _ := json.Marshal(store.records)
	if strings.Contains(string(raw), "never-log") || strings.Contains(string(raw), tokens[access.Analyst]) {
		t.Fatal("audit exposed request secrets")
	}
	store.fail = true
	if call() != 503 || ops.AuditFailures.Load() != 1 {
		t.Fatal("audit outage did not deny dispatch")
	}
}
func TestCollectorReplayAndTampering(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	expires := time.Now().UTC().Add(time.Hour)
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	principal := access.Principal{Subject: "collector", Role: access.Collector, ExpiresAt: expires}
	raw, _ := json.Marshal([]map[string]any{{"subject": principal.Subject, "role": principal.Role, "expires_at": expires, "token_sha256": access.Digest(token), "collector_public_key": base64.RawStdEncoding.EncodeToString(pub)}})
	verifier, err := access.Parse(raw, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	store := &securityFixture{nonces: map[string]bool{}}
	ops := observability.New(nil, "", "test")
	defer ops.Close(context.Background())
	s := New(nil, nil, nil, nil).WithAccess(verifier, nil).WithHardening(store, ops)
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := strings.Repeat("a", 32)
	body := `{"synthetic":true}`
	sig := base64.RawStdEncoding.EncodeToString(ed25519.Sign(key, access.CollectorMessage("POST", "/api/v1/telemetry", stamp, nonce, []byte(body))))
	invoke := func(payload, signature, timestamp string, mutate ...func(*http.Request)) int {
		r := localRequest("POST", "/api/v1/telemetry", strings.NewReader(payload))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Sentinel-Timestamp", timestamp)
		r.Header.Set("X-Sentinel-Nonce", nonce)
		r.Header.Set("X-Sentinel-Signature", signature)
		for _, change := range mutate {
			change(r)
		}
		w := httptest.NewRecorder()
		limitRequestBody(s.operational(s.authorise(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requestActor(r) != "collector" {
				t.Error("unverified actor")
			}
			if _, ok := r.Context().Value(collectorKey{}).(collectorProof); !ok {
				t.Error("missing provenance")
			}
			w.WriteHeader(204)
		})))).ServeHTTP(w, r)
		return w.Code
	}
	if invoke(body, "", stamp) != 401 || invoke(body+" ", sig, stamp) != 401 || invoke(body, sig, "1") != 401 {
		t.Fatal("invalid signature accepted")
	}
	for name, change := range map[string]func(*http.Request){
		"duplicate signature": func(r *http.Request) { r.Header.Add("X-Sentinel-Signature", sig) },
		"query":               func(r *http.Request) { r.URL.RawQuery = "unexpected=true" },
		"future timestamp": func(r *http.Request) {
			r.Header.Set("X-Sentinel-Timestamp", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
		},
	} {
		t.Run(name, func(t *testing.T) {
			if invoke(body, sig, stamp, change) != 401 {
				t.Fatal("invalid collector request accepted")
			}
		})
	}
	if invoke(strings.Repeat("x", int(maxRequestBodyBytes)+1), sig, stamp) != 413 {
		t.Fatal("oversized signed request did not preserve body limit")
	}
	if invoke(body, sig, stamp) != 204 || invoke(body, sig, stamp) != 409 {
		t.Fatal("durable nonce replay not enforced")
	}
}
