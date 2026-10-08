package access

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func manifest(now time.Time) ([]byte, string) {
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	hash := sha256.Sum256([]byte(token))
	data, _ := json.Marshal([]credential{{Principal: Principal{Subject: "lab-analyst", Role: Analyst, ExpiresAt: now.Add(time.Hour)}, TokenSHA256: hex.EncodeToString(hash[:])}})
	return data, token
}

func TestAuthenticate(t *testing.T) {
	now := time.Now().UTC()
	data, token := manifest(now)
	verifier, err := Parse(data, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, header string
		at           time.Time
		valid        bool
	}{
		{"valid", "Bearer " + token, now, true}, {"case insensitive scheme", "bearer " + token, now, true},
		{"expired", "Bearer " + token, now.Add(time.Hour), false}, {"missing", "", now, false},
		{"wrong", "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 32))), now, false},
		{"extra whitespace", "Bearer  " + token, now, false}, {"oversized", strings.Repeat("x", 129), now, false},
		{"malformed base64", "Bearer " + strings.Repeat("!", 43), now, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := verifier.Authenticate(tc.header, tc.at)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
			if tc.valid && (p.Subject != "lab-analyst" || p.Role != Analyst) {
				t.Fatal(p)
			}
		})
	}
}

func TestManifestValidation(t *testing.T) {
	now := time.Now().UTC()
	data, _ := manifest(now)
	for _, bad := range [][]byte{nil, []byte(`[]`), []byte(`null`), []byte(`{}`), append(append([]byte{}, data...), []byte(` {}`)...), []byte(strings.Repeat("x", 65537))} {
		if _, err := Parse(bad, now); err == nil {
			t.Fatal("invalid manifest accepted")
		}
	}
	for _, modify := range []func(*credential){
		func(c *credential) { c.Subject = "spoof\nadmin" }, func(c *credential) { c.Role = "superadmin" },
		func(c *credential) { c.ExpiresAt = now }, func(c *credential) { c.ExpiresAt = now.Add(25 * time.Hour) },
		func(c *credential) { c.TokenSHA256 = "bad" },
	} {
		var entries []credential
		_ = json.Unmarshal(data, &entries)
		modify(&entries[0])
		bad, _ := json.Marshal(entries)
		if _, err := Parse(bad, now); err == nil {
			t.Fatal("invalid credential accepted")
		}
	}
	var entries []credential
	_ = json.Unmarshal(data, &entries)
	entries = append(entries, entries[0])
	duplicate, _ := json.Marshal(entries)
	if _, err := Parse(duplicate, now); err == nil {
		t.Fatal("duplicate token accepted")
	}
	unknown := strings.Replace(string(data), `"subject":`, `"extra":true,"subject":`, 1)
	if _, err := Parse([]byte(unknown), now); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestPrivateCredentialFile(t *testing.T) {
	now := time.Now().UTC()
	data, _ := manifest(now)
	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path, now); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(path, now); err == nil {
		t.Fatal("shared-readable file accepted")
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFile(link, now); err == nil {
		t.Fatal("symlink accepted")
	}
}
