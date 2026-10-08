// Package access verifies bounded, operator-provisioned lab credentials.
package access

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	Analyst       = "analyst"
	Administrator = "administrator"
	Collector     = "collector"
)

var ErrUnauthenticated = errors.New("valid bearer credential required")
var subjectPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:@-]{0,127}$`)

type Principal struct {
	Subject   string    `json:"subject"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

type credential struct {
	Principal
	TokenSHA256 string `json:"token_sha256"`
	digest      [32]byte
}

type Verifier struct{ credentials []credential }

func LoadFile(path string, now time.Time) (*Verifier, error) {
	// Reject shared-readable manifests and non-regular files before consuming them.
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("credential file must be a private regular file (mode 0600 or stricter)")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("credential file could not be opened")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 {
		return nil, errors.New("credential file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return nil, errors.New("credential file could not be read")
	}
	return Parse(data, now)
}

func Parse(data []byte, now time.Time) (*Verifier, error) {
	if len(data) > 65536 {
		return nil, errors.New("credential manifest exceeds 64 KiB")
	}
	var entries []credential
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&entries); err != nil {
		return nil, errors.New("invalid credential manifest")
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, errors.New("credential manifest must contain one JSON value")
	}
	if len(entries) == 0 || len(entries) > 128 {
		return nil, errors.New("credential manifest requires 1–128 entries")
	}
	seen := make(map[string]bool)
	for i := range entries {
		c := &entries[i]
		if !subjectPattern.MatchString(c.Subject) || (c.Role != Analyst && c.Role != Administrator && c.Role != Collector) {
			return nil, errors.New("invalid credential subject or role")
		}
		if !c.ExpiresAt.After(now) || c.ExpiresAt.After(now.Add(24*time.Hour)) {
			return nil, errors.New("credentials must expire within the next 24 hours")
		}
		digest, err := hex.DecodeString(c.TokenSHA256)
		if err != nil || len(digest) != sha256.Size || c.TokenSHA256 != strings.ToLower(c.TokenSHA256) || seen[c.TokenSHA256] {
			return nil, errors.New("credential digests must be unique lowercase SHA-256 values")
		}
		seen[c.TokenSHA256] = true
		copy(c.digest[:], digest)
	}
	return &Verifier{credentials: entries}, nil
}

func (v *Verifier) Authenticate(header string, now time.Time) (Principal, error) {
	if v == nil || len(header) > 128 {
		return Principal{}, ErrUnauthenticated
	}
	parts := strings.Split(header, " ")
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || len(parts[1]) != 43 {
		return Principal{}, ErrUnauthenticated
	}
	token, err := base64.RawURLEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(token) != 32 {
		return Principal{}, ErrUnauthenticated
	}
	digest := sha256.Sum256([]byte(parts[1]))
	var principal Principal
	found := false
	// Compare every configured digest so a matching entry does not shorten the scan.
	for _, c := range v.credentials {
		match := subtle.ConstantTimeCompare(digest[:], c.digest[:]) == 1
		if match && c.ExpiresAt.After(now) {
			principal = c.Principal
			found = true
		}
	}
	if !found {
		return Principal{}, ErrUnauthenticated
	}
	return principal, nil
}
