package access

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type AuditRecord struct {
	ID         int64     `json:"id"`
	OccurredAt time.Time `json:"occurred_at"`
	TraceID    string    `json:"trace_id"`
	Method     string    `json:"method"`
	Route      string    `json:"route"`
	Subject    string    `json:"subject,omitempty"`
	Role       string    `json:"role,omitempty"`
	Outcome    string    `json:"outcome"`
	Status     int       `json:"status"`
}

type SecurityStore interface {
	RecordAccess(context.Context, AuditRecord) error
	ListAccess(context.Context, int64, int) ([]AuditRecord, error)
	PruneAccess(context.Context) (int64, error)
	ReserveCollectorNonce(context.Context, string, string, string, time.Time) error
}

var ErrReplay = errors.New("collector request replayed or replay storage at capacity")

// Collector signatures cover exact bytes, route, method, time and a one-use nonce.
func CollectorMessage(method, path, timestamp, nonce string, body []byte) []byte {
	digest := sha256.Sum256(body)
	return []byte(strings.Join([]string{"sentinel-collector-v1", method, path, timestamp, nonce, hex.EncodeToString(digest[:])}, "\n"))
}

func (v *Verifier) CollectorsReady() bool {
	seen := make(map[string]bool)
	for _, c := range v.credentials {
		if c.Role == Collector {
			key, err := base64.RawStdEncoding.Strict().DecodeString(c.CollectorPublicKey)
			if err != nil || len(key) != ed25519.PublicKeySize || seen[c.Subject] {
				return false
			}
			seen[c.Subject] = true
		}
	}
	return true
}

func (v *Verifier) VerifyCollector(principal Principal, signature string, message []byte) (string, bool) {
	if principal.Role != Collector {
		return "", false
	}
	sig, err := base64.RawStdEncoding.Strict().DecodeString(signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return "", false
	}
	for _, c := range v.credentials {
		if c.Role == Collector && c.Subject == principal.Subject && c.ExpiresAt.Equal(principal.ExpiresAt) {
			key, err := base64.RawStdEncoding.Strict().DecodeString(c.CollectorPublicKey)
			if err == nil && len(key) == ed25519.PublicKeySize && ed25519.Verify(key, message, sig) {
				return Digest(string(key)), true
			}
		}
	}
	return "", false
}
