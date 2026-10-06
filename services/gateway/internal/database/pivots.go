package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrAssetNotFound    = errors.New("asset not found")
	ErrIdentityNotFound = errors.New("identity not found")
	ErrHuntRunNotFound  = errors.New("hunt run not found")
)

type AssetPivot struct {
	ID           string                `json:"id"`
	Hostname     string                `json:"hostname"`
	Zone         string                `json:"zone"`
	Criticality  string                `json:"criticality"`
	FirstSeenAt  *time.Time            `json:"first_seen_at,omitempty"`
	LastSeenAt   *time.Time            `json:"last_seen_at,omitempty"`
	RecentEvents []EventEvidenceRecord `json:"recent_events"`
	Findings     []FindingRecord       `json:"findings"`
}

type IdentityPivot struct {
	ID           string                `json:"id"`
	IdentityType string                `json:"identity_type"`
	Name         string                `json:"name"`
	FirstSeenAt  *time.Time            `json:"first_seen_at,omitempty"`
	LastSeenAt   *time.Time            `json:"last_seen_at,omitempty"`
	RecentEvents []EventEvidenceRecord `json:"recent_events"`
	Findings     []FindingRecord       `json:"findings"`
}

func (d *Database) GetAssetPivot(ctx context.Context, id string, limit int) (AssetPivot, error) {
	if d == nil || d.pool == nil {
		return AssetPivot{}, fmt.Errorf("database is not initialised")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var pivot AssetPivot
	if err := d.pool.QueryRow(ctx, `
		SELECT id, hostname, zone, criticality, first_seen_at, last_seen_at
		FROM assets
		WHERE id = $1
	`, id).Scan(
		&pivot.ID,
		&pivot.Hostname,
		&pivot.Zone,
		&pivot.Criticality,
		&pivot.FirstSeenAt,
		&pivot.LastSeenAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AssetPivot{}, ErrAssetNotFound
		}
		return AssetPivot{}, fmt.Errorf("query asset pivot: %w", err)
	}

	events, err := d.pivotEvents(ctx, "asset_id", id, limit)
	if err != nil {
		return AssetPivot{}, err
	}
	findings, err := d.pivotFindings(ctx, "asset_id", id, limit)
	if err != nil {
		return AssetPivot{}, err
	}

	pivot.RecentEvents = events
	pivot.Findings = findings
	return pivot, nil
}

func (d *Database) GetIdentityPivot(ctx context.Context, id string, limit int) (IdentityPivot, error) {
	if d == nil || d.pool == nil {
		return IdentityPivot{}, fmt.Errorf("database is not initialised")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	var pivot IdentityPivot
	if err := d.pool.QueryRow(ctx, `
		SELECT id, identity_type, name, first_seen_at, last_seen_at
		FROM identities
		WHERE id = $1
	`, id).Scan(
		&pivot.ID,
		&pivot.IdentityType,
		&pivot.Name,
		&pivot.FirstSeenAt,
		&pivot.LastSeenAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return IdentityPivot{}, ErrIdentityNotFound
		}
		return IdentityPivot{}, fmt.Errorf("query identity pivot: %w", err)
	}

	events, err := d.pivotEvents(ctx, "identity_id", id, limit)
	if err != nil {
		return IdentityPivot{}, err
	}
	findings, err := d.pivotFindings(ctx, "identity_id", id, limit)
	if err != nil {
		return IdentityPivot{}, err
	}

	pivot.RecentEvents = events
	pivot.Findings = findings
	return pivot, nil
}

func (d *Database) pivotEvents(
	ctx context.Context,
	column string,
	id string,
	limit int,
) ([]EventEvidenceRecord, error) {
	if column != "asset_id" && column != "identity_id" {
		return nil, fmt.Errorf("unsupported pivot column")
	}

	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT
			id,
			source_timestamp,
			category,
			action,
			COALESCE(outcome, ''),
			asset_id,
			identity_id,
			payload
		FROM events
		WHERE %s = $1
		ORDER BY source_timestamp DESC, id DESC
		LIMIT $2
	`, column), id, limit)
	if err != nil {
		return nil, fmt.Errorf("query pivot events: %w", err)
	}
	defer rows.Close()

	events := make([]EventEvidenceRecord, 0, limit)
	for rows.Next() {
		var (
			event   EventEvidenceRecord
			payload []byte
		)
		if err := rows.Scan(
			&event.ID,
			&event.SourceTimestamp,
			&event.Category,
			&event.Action,
			&event.Outcome,
			&event.AssetID,
			&event.IdentityID,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("scan pivot event: %w", err)
		}
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, fmt.Errorf("decode pivot event payload: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pivot events: %w", err)
	}
	return events, nil
}

func (d *Database) pivotFindings(
	ctx context.Context,
	column string,
	id string,
	limit int,
) ([]FindingRecord, error) {
	if column != "asset_id" && column != "identity_id" {
		return nil, fmt.Errorf("unsupported pivot column")
	}

	rows, err := d.pool.Query(ctx, fmt.Sprintf(`
		SELECT DISTINCT
			f.id,
			f.detection_id,
			f.detection_version,
			f.title,
			f.severity,
			f.confidence,
			f.status,
			f.first_observed_at,
			f.last_observed_at,
			f.evidence,
			f.created_at,
			f.updated_at
		FROM findings f
		JOIN finding_events fe ON fe.finding_id = f.id
		JOIN events e ON e.id = fe.event_id
		WHERE e.%s = $1
		ORDER BY f.last_observed_at DESC, f.id DESC
		LIMIT $2
	`, column), id, limit)
	if err != nil {
		return nil, fmt.Errorf("query pivot findings: %w", err)
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
			return nil, fmt.Errorf("scan pivot finding: %w", err)
		}
		if err := json.Unmarshal(evidence, &record.Evidence); err != nil {
			return nil, fmt.Errorf("decode pivot finding evidence: %w", err)
		}
		findings = append(findings, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pivot findings: %w", err)
	}
	return findings, nil
}
