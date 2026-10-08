package access

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"
)

const SessionLifetime = 8 * time.Hour
const SessionIdleTimeout = 30 * time.Minute

var ErrSessionNotFound = errors.New("session not found")
var ErrSessionCapacity = errors.New("session capacity reached")

type Session struct {
	Hash           string
	CredentialHash string
	Principal      Principal
	CreatedAt      time.Time
	LastSeenAt     time.Time
	ExpiresAt      time.Time
}

type SessionStore interface {
	CreateSession(context.Context, Session, string) error
	TouchSession(context.Context, string, time.Time) (Session, error)
	RevokeSession(context.Context, string) error
}

func RandomSessionID() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}
func ValidSessionID(id string) bool {
	if len(id) != 43 {
		return false
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(id)
	return err == nil && len(b) == 32
}
func Digest(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
func CSRFToken(id string) string { return Digest("sentinel-session-csrf:" + id) }

// A session cannot outlive or gain permissions beyond its provisioned credential.
func (v *Verifier) ValidateSession(s Session, now time.Time) bool {
	if (s.Principal.Role != Analyst && s.Principal.Role != Administrator) || v == nil || !s.ExpiresAt.After(now) || s.ExpiresAt.After(s.CreatedAt.Add(SessionLifetime)) || s.CreatedAt.After(now) || !s.LastSeenAt.After(now.Add(-SessionIdleTimeout)) {
		return false
	}
	for _, c := range v.credentials {
		if c.TokenSHA256 == s.CredentialHash && c.Principal.Subject == s.Principal.Subject && c.Principal.Role == s.Principal.Role && c.ExpiresAt.After(now) && !s.ExpiresAt.After(c.ExpiresAt) {
			return true
		}
	}
	return false
}
