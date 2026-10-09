package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSignedCollectorTransport(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	raw, _ := json.Marshal(map[string]any{"tokens": map[string]string{"collector": token}, "collector_private_seed": base64.RawStdEncoding.EncodeToString(key.Seed()), "expires_at": time.Now().Add(time.Hour)})
	path := filepath.Join(t.TempDir(), "private.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	transport, err := collectorTransport(path)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		hash := sha256.Sum256(body)
		message := strings.Join([]string{"sentinel-collector-v1", r.Method, r.URL.Path, r.Header.Get("X-Sentinel-Timestamp"), r.Header.Get("X-Sentinel-Nonce"), hex.EncodeToString(hash[:])}, "\n")
		signature, err := base64.RawStdEncoding.DecodeString(r.Header.Get("X-Sentinel-Signature"))
		if err != nil || !ed25519.Verify(pub, []byte(message), signature) || r.Header.Get("Authorization") != "Bearer "+token {
			t.Error("signature invalid")
		}
		w.WriteHeader(202)
		w.Write([]byte(`{"accepted":true,"event_id":"synthetic"}`))
	}))
	defer server.Close()
	if _, err := postEvent(context.Background(), &http.Client{Transport: transport}, server.URL, map[string]interface{}{"event_id": "synthetic"}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := collectorTransport(path); err == nil {
		t.Fatal("public credentials accepted")
	}
	link := filepath.Join(filepath.Dir(path), "linked.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := collectorTransport(link); err == nil {
		t.Fatal("symlink credentials accepted")
	}
}
