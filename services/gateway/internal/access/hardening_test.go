package access

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestCollectorSignature(t *testing.T) {
	now := time.Now().UTC()
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	principal := Principal{Subject: "signed-collector", Role: Collector, ExpiresAt: now.Add(time.Hour)}
	raw, _ := json.Marshal([]credential{{Principal: principal, TokenSHA256: Digest("token"), CollectorPublicKey: base64.RawStdEncoding.EncodeToString(pub)}})
	v, err := Parse(raw, now)
	if err != nil || !v.CollectorsReady() {
		t.Fatal("collector configuration", err)
	}
	message := CollectorMessage("POST", "/api/v1/telemetry", "123", "nonce", []byte(`{"event_id":"synthetic"}`))
	signature := base64.RawStdEncoding.EncodeToString(ed25519.Sign(private, message))
	if hash, ok := v.VerifyCollector(principal, signature, message); !ok || hash != Digest(string(pub)) {
		t.Fatal("signature rejected")
	}
	for _, modified := range [][]byte{CollectorMessage("GET", "/api/v1/telemetry", "123", "nonce", nil), append(append([]byte{}, message...), ' ')} {
		if _, ok := v.VerifyCollector(principal, signature, modified); ok {
			t.Fatal("tampered message accepted")
		}
	}
	for _, invalid := range []string{"", signature + "=", "malformed"} {
		if _, ok := v.VerifyCollector(principal, invalid, message); ok {
			t.Fatal("invalid signature accepted")
		}
	}
	principal.Subject = "other"
	if _, ok := v.VerifyCollector(principal, signature, message); ok {
		t.Fatal("identity substitution accepted")
	}
}
