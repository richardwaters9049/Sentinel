package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSecurityStoreIntegration(t *testing.T) {
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
	schema := pgx.Identifier{fmt.Sprintf("security_test_%d", time.Now().UnixNano())}.Sanitize()
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

	nonce := strings.Repeat("a", 32)
	var accepted atomic.Int64
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() {
			err := d.ReserveCollectorNonce(ctx, "collector", nonce, access.Digest("body"), time.Now().Add(time.Minute))
			if err == nil {
				accepted.Add(1)
			} else if !errors.Is(err, access.ErrReplay) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("concurrent replay accepted %d", accepted.Load())
	}
	if err := d.ReserveCollectorNonce(ctx, "collector", nonce, access.Digest("changed"), time.Now().Add(time.Minute)); !errors.Is(err, access.ErrReplay) {
		t.Fatal("replay survived", err)
	}
	for range 3 {
		if err := d.RecordAccess(ctx, access.AuditRecord{TraceID: strings.Repeat("a", 32), Method: "GET", Route: "GET /api/v1/session", Subject: "analyst", Role: access.Analyst, Outcome: "completed", Status: 200}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := d.ListAccess(ctx, 0, 2)
	if err != nil || len(rows) != 2 || rows[0].ID <= rows[1].ID {
		t.Fatal("cursor order", err)
	}
	next, err := d.ListAccess(ctx, rows[1].ID, 2)
	if err != nil || len(next) != 1 {
		t.Fatal("cursor pagination", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE access_audit SET occurred_at=NOW()-INTERVAL '91 days' WHERE id=$1`, next[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO audit_events(occurred_at,action,resource_type) VALUES(NOW()-INTERVAL '91 days','synthetic-retention-check','test')`); err != nil {
		t.Fatal(err)
	}
	removed, err := d.PruneAccess(ctx)
	if err != nil || removed != 1 {
		t.Fatal("retention", removed, err)
	}
	var domainCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE action='synthetic-retention-check'`).Scan(&domainCount); err != nil || domainCount != 1 {
		t.Fatal("domain audit was removed", err)
	}
	rows, err = d.ListAccess(ctx, 0, 100)
	if err != nil || len(rows) != 2 {
		t.Fatal("retained recent rows", err)
	}
}
