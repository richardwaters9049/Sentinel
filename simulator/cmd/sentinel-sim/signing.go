package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type signedTransport struct {
	token string
	key   ed25519.PrivateKey
	next  http.RoundTripper
}

func collectorTransport(path string) (http.RoundTripper, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("collector credentials require a private regular file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("collector credentials unavailable")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() || opened.Mode().Perm()&0077 != 0 {
		return nil, errors.New("collector credential file changed or is not private")
	}
	raw, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(raw) > 65536 {
		return nil, errors.New("invalid collector credential file")
	}
	var value struct {
		Tokens  map[string]string `json:"tokens"`
		Seed    string            `json:"collector_private_seed"`
		Expires time.Time         `json:"expires_at"`
	}
	if json.Unmarshal(raw, &value) != nil || !value.Expires.After(time.Now()) {
		return nil, errors.New("invalid or expired collector credentials")
	}
	seed, err := base64.RawStdEncoding.Strict().DecodeString(value.Seed)
	token := value.Tokens["collector"]
	decoded, tokenErr := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(seed) != 32 || tokenErr != nil || len(decoded) != 32 {
		return nil, errors.New("invalid collector signing material")
	}
	return &signedTransport{token: token, key: ed25519.NewKeyFromSeed(seed), next: http.DefaultTransport}, nil
}
func (s *signedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme != "https" && r.URL.Hostname() != "127.0.0.1" && r.URL.Hostname() != "localhost" && r.URL.Hostname() != "::1" {
		return nil, errors.New("collector credentials require HTTPS outside loopback")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20+1))
	if err != nil || len(body) > 1<<20 {
		return nil, errors.New("invalid collector request body")
	}
	r.Body.Close()
	clone := r.Clone(r.Context())
	clone.Body = io.NopCloser(bytes.NewReader(body))
	clone.ContentLength = int64(len(body))
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	stamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceText := hex.EncodeToString(nonce[:])
	digest := sha256.Sum256(body)
	message := strings.Join([]string{"sentinel-collector-v1", r.Method, r.URL.Path, stamp, nonceText, hex.EncodeToString(digest[:])}, "\n")
	clone.Header.Set("Authorization", "Bearer "+s.token)
	clone.Header.Set("X-Sentinel-Timestamp", stamp)
	clone.Header.Set("X-Sentinel-Nonce", nonceText)
	clone.Header.Set("X-Sentinel-Signature", base64.RawStdEncoding.EncodeToString(ed25519.Sign(s.key, []byte(message))))
	return s.next.RoundTrip(clone)
}
