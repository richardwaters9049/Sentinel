package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/detection"
)

type FindingQuery struct {
	Limit    int
	Status   string
	Severity string
}

type FindingRecord struct {
	ID               string                 `json:"id"`
	DetectionID      string                 `json:"detection_id"`
	DetectionVersion int                    `json:"detection_version"`
	Title            string                 `json:"title"`
	Severity         string                 `json:"severity"`
	Confidence       int                    `json:"confidence"`
	Status           string                 `json:"status"`
	FirstObservedAt  time.Time              `json:"first_observed_at"`
	LastObservedAt   time.Time              `json:"last_observed_at"`
	Evidence         map[string]interface{} `json:"evidence"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

func (d *Database) RecentAuthenticationFailures(
	ctx context.Context,
	identityID string,
	sourceIP string,
	before time.Time,
	window time.Duration,
	limit int,
) ([]detection.AuthFailure, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("failure query limit must be greater than zero")
	}

	rows, err := d.pool.Query(ctx, `
		SELECT id, source_timestamp
		FROM events
		WHERE identity_id = $1
		  AND category = 'authentication'
		  AND action = 'login'
		  AND outcome = 'failure'
		  AND payload #>> '{network,source_ip}' = $2
		  AND source_timestamp < $3
		  AND source_timestamp >= $3 - ($4 * INTERVAL '1 second')
		ORDER BY source_timestamp DESC, id DESC
		LIMIT $5
	`,
		identityID,
		sourceIP,
		before.UTC(),
		int(window.Seconds()),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query recent authentication failures: %w", err)
	}
	defer rows.Close()

	failures := make([]detection.AuthFailure, 0, limit)
	for rows.Next() {
		var failure detection.AuthFailure
		if err := rows.Scan(&failure.EventID, &failure.Timestamp); err != nil {
			return nil, fmt.Errorf("scan authentication failure: %w", err)
		}
		failures = append(failures, failure)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate authentication failures: %w", err)
	}

	return failures, nil
}

func (d *Database) CreateFinding(ctx context.Context, finding detection.Finding) (bool, error) {
	if d == nil || d.pool == nil {
		return false, fmt.Errorf("database is not initialised")
	}

	evidence, err := json.Marshal(finding.Evidence)
	if err != nil {
		return false, fmt.Errorf("marshal finding evidence: %w", err)
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin finding transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	commandTag, err := tx.Exec(ctx, `
		INSERT INTO findings (
			id,
			detection_id,
			detection_version,
			dedup_key,
			title,
			severity,
			confidence,
			status,
			first_observed_at,
			last_observed_at,
			evidence
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT DO NOTHING
	`,
		finding.ID,
		finding.DetectionID,
		finding.DetectionVersion,
		finding.DedupKey,
		finding.Title,
		finding.Severity,
		finding.Confidence,
		finding.Status,
		finding.FirstObservedAt.UTC(),
		finding.LastObservedAt.UTC(),
		evidence,
	)
	if err != nil {
		return false, fmt.Errorf("insert finding: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit duplicate finding transaction: %w", err)
		}
		return false, nil
	}

	for _, eventID := range finding.EventIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO finding_events (finding_id, event_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, finding.ID, eventID); err != nil {
			return false, fmt.Errorf("link finding event %s: %w", eventID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit finding transaction: %w", err)
	}

	return true, nil
}

func (d *Database) ListFindings(ctx context.Context, query FindingQuery) ([]FindingRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	args := make([]interface{}, 0, 3)
	conditions := make([]string, 0, 2)
	addArg := func(value interface{}) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if value := strings.TrimSpace(query.Status); value != "" {
		conditions = append(conditions, "status = "+addArg(value))
	}
	if value := strings.TrimSpace(query.Severity); value != "" {
		conditions = append(conditions, "severity = "+addArg(value))
	}

	sql := `
		SELECT
			id,
			detection_id,
			detection_version,
			title,
			severity,
			confidence,
			status,
			first_observed_at,
			last_observed_at,
			evidence,
			created_at,
			updated_at
		FROM findings
	`
	if len(conditions) > 0 {
		sql += " WHERE " + strings.Join(conditions, " AND ")
	}
	sql += " ORDER BY last_observed_at DESC, id DESC LIMIT " + addArg(limit)

	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query findings: %w", err)
	}
	defer rows.Close()

	findings := make([]FindingRecord, 0, limit)
	for rows.Next() {
		var (
			record   FindingRecord
			evidence []byte
		)

		if err := rows.Scan(
			&record.ID,
			&record.DetectionID,
			&record.DetectionVersion,
			&record.Title,
			&record.Severity,
			&record.Confidence,
			&record.Status,
			&record.FirstObservedAt,
			&record.LastObservedAt,
			&evidence,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}

		if err := json.Unmarshal(evidence, &record.Evidence); err != nil {
			return nil, fmt.Errorf("decode finding evidence: %w", err)
		}

		findings = append(findings, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate findings: %w", err)
	}

	return findings, nil
}
