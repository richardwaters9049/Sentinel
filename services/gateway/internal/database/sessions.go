package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/access"
)

func (d *Database) CreateSession(ctx context.Context, s access.Session, replaceHash string) error {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialise only session creation so the global/per-subject caps cannot race.
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(83160916)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM console_sessions WHERE expires_at <= $1 OR last_seen_at <= $2 OR session_hash = $3`, s.CreatedAt, s.CreatedAt.Add(-access.SessionIdleTimeout), replaceHash); err != nil {
		return err
	}
	var total, subjectCount int
	if err = tx.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE subject=$1) FROM console_sessions`, s.Principal.Subject).Scan(&total, &subjectCount); err != nil {
		return err
	}
	if total >= 10000 || subjectCount >= 8 {
		return access.ErrSessionCapacity
	}
	if _, err = tx.Exec(ctx, `INSERT INTO console_sessions (session_hash,credential_hash,subject,role,created_at,last_seen_at,expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, s.Hash, s.CredentialHash, s.Principal.Subject, s.Principal.Role, s.CreatedAt, s.LastSeenAt, s.ExpiresAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (d *Database) TouchSession(ctx context.Context, hash string, now time.Time) (access.Session, error) {
	var s access.Session
	err := d.pool.QueryRow(ctx, `UPDATE console_sessions SET last_seen_at=$2 WHERE session_hash=$1 AND expires_at>$2 AND last_seen_at>$3 RETURNING session_hash,credential_hash,subject,role,created_at,last_seen_at,expires_at`, hash, now, now.Add(-access.SessionIdleTimeout)).Scan(&s.Hash, &s.CredentialHash, &s.Principal.Subject, &s.Principal.Role, &s.CreatedAt, &s.LastSeenAt, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, access.ErrSessionNotFound
	}
	s.Principal.ExpiresAt = s.ExpiresAt
	return s, err
}
func (d *Database) RevokeSession(ctx context.Context, hash string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM console_sessions WHERE session_hash=$1`, hash)
	return err
}
