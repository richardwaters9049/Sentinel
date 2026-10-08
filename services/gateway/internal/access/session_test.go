package access

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSessionCredentialAndTimeBoundaries(t *testing.T) {
	now := time.Now().UTC()
	data, token := manifest(now)
	verifier, err := Parse(data, now)
	if err != nil {
		t.Fatal(err)
	}
	var entries []credential
	if err = json.Unmarshal(data, &entries); err != nil {
		t.Fatal(err)
	}
	session := Session{Hash: Digest("session"), CredentialHash: Digest(token), Principal: entries[0].Principal, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	if !verifier.ValidateSession(session, now) {
		t.Fatal("valid session rejected")
	}
	for _, change := range []func(*Session){
		func(s *Session) { s.Principal.Subject = "spoofed" }, func(s *Session) { s.Principal.Role = Administrator }, func(s *Session) { s.Principal.Role = Collector },
		func(s *Session) { s.CredentialHash = Digest("removed") }, func(s *Session) { s.LastSeenAt = now.Add(-SessionIdleTimeout) },
		func(s *Session) { s.CreatedAt = now.Add(time.Second) }, func(s *Session) { s.ExpiresAt = now },
		func(s *Session) { s.ExpiresAt = now.Add(2 * time.Hour) }, func(s *Session) { s.ExpiresAt = now.Add(9 * time.Hour) },
	} {
		s := session
		change(&s)
		if verifier.ValidateSession(s, now) {
			t.Fatal("invalid session accepted")
		}
	}
	id, err := RandomSessionID()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidSessionID(id) || ValidSessionID("bad") {
		t.Fatal("invalid session identifier contract")
	}
	if Digest(id) == CSRFToken(id) {
		t.Fatal("CSRF derivation lacks domain separation")
	}
}
