package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
	"os"
	"testing"
	"time"
)

func TestSessionIntegration(t *testing.T) {
	url := os.Getenv("SENTINEL_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("SENTINEL_TEST_DATABASE_URL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := pgx.Identifier{fmt.Sprintf("session_test_%d", time.Now().UnixNano())}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	d := &Database{pool: pool}
	for i := 0; i < 2; i++ {
		if err = d.ApplyMigrations(ctx); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	create := func(id, subject string) access.Session {
		return access.Session{Hash: access.Digest(id), CredentialHash: access.Digest("credential"), Principal: access.Principal{Subject: subject, Role: access.Analyst}, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	}
	for i := 0; i < 8; i++ {
		if err = d.CreateSession(ctx, create(fmt.Sprint(i), "analyst"), ""); err != nil {
			t.Fatal(err)
		}
	}
	if err = d.CreateSession(ctx, create("overflow", "analyst"), ""); !errors.Is(err, access.ErrSessionCapacity) {
		t.Fatal("per-subject cap not enforced", err)
	}
	if err = d.CreateSession(ctx, create("rotation", "analyst"), access.Digest("0")); err != nil {
		t.Fatal(err)
	}
	if _, err = d.TouchSession(ctx, access.Digest("0"), now); !errors.Is(err, access.ErrSessionNotFound) {
		t.Fatal("rotated session survived")
	}
	touched, err := d.TouchSession(ctx, access.Digest("rotation"), now.Add(time.Minute))
	if err != nil || !touched.LastSeenAt.Equal(now.Add(time.Minute)) {
		t.Fatal("touch failed", err)
	}
	if _, err = d.TouchSession(ctx, touched.Hash, now.Add(31*time.Minute)); !errors.Is(err, access.ErrSessionNotFound) {
		t.Fatal("idle boundary not enforced")
	}
	if err = d.RevokeSession(ctx, access.Digest("1")); err != nil {
		t.Fatal(err)
	}
	if _, err = d.TouchSession(ctx, access.Digest("1"), now); !errors.Is(err, access.ErrSessionNotFound) {
		t.Fatal("revoked session survived")
	}
	if _, err = d.TouchSession(ctx, access.Digest("2"), now.Add(time.Hour)); !errors.Is(err, access.ErrSessionNotFound) {
		t.Fatal("absolute expiry not enforced")
	}
	// Capacity is global as well as per subject; simulate records across many principals.
	if _, err = pool.Exec(ctx, `INSERT INTO console_sessions SELECT md5(i::text)||md5(i::text),$1,'principal-'||i,'analyst',$2,$2,$3 FROM generate_series(1,9993) AS i`, access.Digest("credential"), now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = d.CreateSession(ctx, create("global-overflow", "another"), ""); !errors.Is(err, access.ErrSessionCapacity) {
		t.Fatal("global cap not enforced", err)
	}
}
