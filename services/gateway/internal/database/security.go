package database

import (
	"context"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
	"time"
)

func (d *Database) RecordAccess(ctx context.Context, r access.AuditRecord) error {
	_, err := d.pool.Exec(ctx, `INSERT INTO access_audit(trace_id,method,route,subject,role,outcome,status) VALUES($1,$2,$3,$4,$5,$6,$7)`, r.TraceID, r.Method, r.Route, r.Subject, r.Role, r.Outcome, r.Status)
	return err
}
func (d *Database) ListAccess(ctx context.Context, before int64, limit int) ([]access.AuditRecord, error) {
	rows, err := d.pool.Query(ctx, `SELECT id,occurred_at,trace_id,method,route,subject,role,outcome,status FROM access_audit WHERE ($1::BIGINT=0 OR id<$1) ORDER BY id DESC LIMIT $2`, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]access.AuditRecord, 0)
	for rows.Next() {
		var r access.AuditRecord
		if err = rows.Scan(&r.ID, &r.OccurredAt, &r.TraceID, &r.Method, &r.Route, &r.Subject, &r.Role, &r.Outcome, &r.Status); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// Retention is explicit, bounded and applies only to access records, never domain evidence.
func (d *Database) PruneAccess(ctx context.Context) (int64, error) {
	result, err := d.pool.Exec(ctx, `DELETE FROM access_audit WHERE id IN (SELECT id FROM access_audit WHERE occurred_at < NOW()-INTERVAL '90 days' ORDER BY occurred_at,id LIMIT 1000)`)
	return result.RowsAffected(), err
}
func (d *Database) ReserveCollectorNonce(ctx context.Context, subject, nonce, digest string, expires time.Time) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(83160917)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM collector_nonces WHERE expires_at < NOW()`); err != nil {
		return err
	}
	var count int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM collector_nonces`).Scan(&count); err != nil {
		return err
	}
	if count >= 100000 {
		return access.ErrReplay
	}
	result, err := tx.Exec(ctx, `INSERT INTO collector_nonces(subject,nonce,body_sha256,expires_at) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, subject, nonce, digest, expires)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return access.ErrReplay
	}
	return tx.Commit(ctx)
}
