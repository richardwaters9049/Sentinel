package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/detection"
)

var (
	ErrFindingNotFound    = errors.New("finding not found")
	ErrInvalidTransition  = errors.New("invalid finding status transition")
	ErrDetectionNotFound  = errors.New("detection not found")
	ErrDetectionUnchanged = errors.New("detection enabled state unchanged")
)

type FindingDetail struct {
	FindingRecord
	Events      []EventEvidenceRecord   `json:"events"`
	Enrichments []EventEnrichmentRecord `json:"enrichments"`
	Audit       []AuditRecord           `json:"audit"`
}

type EventEvidenceRecord struct {
	ID              string    `json:"id"`
	SourceTimestamp time.Time `json:"source_timestamp"`
	Category        string    `json:"category"`
	Action          string    `json:"action"`
	Outcome         string    `json:"outcome,omitempty"`
	AssetID         *string   `json:"asset_id,omitempty"`
	IdentityID      *string   `json:"identity_id,omitempty"`
	Payload         any       `json:"payload"`
}

type AuditRecord struct {
	ID           int64          `json:"id"`
	OccurredAt   time.Time      `json:"occurred_at"`
	ActorID      *string        `json:"actor_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *string        `json:"resource_id,omitempty"`
	RequestID    *string        `json:"request_id,omitempty"`
	Details      map[string]any `json:"details"`
}

type DetectionRecord struct {
	ID          string           `json:"id"`
	Version     int              `json:"version"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Severity    string           `json:"severity"`
	Enabled     bool             `json:"enabled"`
	Definition  map[string]any   `json:"definition"`
	MITRE       []map[string]any `json:"mitre"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func (d *Database) GetFinding(ctx context.Context, id string) (FindingDetail, error) {
	if d == nil || d.pool == nil {
		return FindingDetail{}, fmt.Errorf("database is not initialised")
	}

	var (
		record   FindingRecord
		evidence []byte
	)

	err := d.pool.QueryRow(ctx, `
		SELECT
			id, detection_id, detection_version, title, severity, confidence,
			status, first_observed_at, last_observed_at, evidence, created_at, updated_at
		FROM findings
		WHERE id = $1
	`, id).Scan(
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
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return FindingDetail{}, ErrFindingNotFound
	}
	if err != nil {
		return FindingDetail{}, fmt.Errorf("query finding: %w", err)
	}
	if err := json.Unmarshal(evidence, &record.Evidence); err != nil {
		return FindingDetail{}, fmt.Errorf("decode finding evidence: %w", err)
	}

	events, err := d.findingEvents(ctx, id)
	if err != nil {
		return FindingDetail{}, err
	}
	eventIDs := make([]string, 0, len(events))
	for _, event := range events {
		eventIDs = append(eventIDs, event.ID)
	}
	enrichments, err := d.eventEnrichmentsForEventIDs(ctx, eventIDs)
	if err != nil {
		return FindingDetail{}, err
	}

	audit, err := d.auditForResource(ctx, "finding", id)
	if err != nil {
		return FindingDetail{}, err
	}

	return FindingDetail{
		FindingRecord: record,
		Events:        events,
		Enrichments:   enrichments,
		Audit:         audit,
	}, nil
}

func (d *Database) UpdateFindingStatus(
	ctx context.Context,
	id string,
	targetStatus string,
	actorID string,
	requestID string,
) (FindingDetail, error) {
	if d == nil || d.pool == nil {
		return FindingDetail{}, fmt.Errorf("database is not initialised")
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return FindingDetail{}, fmt.Errorf("begin finding status transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	if err := tx.QueryRow(ctx, "SELECT status FROM findings WHERE id = $1 FOR UPDATE", id).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FindingDetail{}, ErrFindingNotFound
		}
		return FindingDetail{}, fmt.Errorf("query finding status: %w", err)
	}

	if err := detection.ValidateFindingTransition(current, targetStatus); err != nil {
		return FindingDetail{}, fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE findings
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, id, targetStatus); err != nil {
		return FindingDetail{}, fmt.Errorf("update finding status: %w", err)
	}

	details, err := json.Marshal(map[string]string{
		"from": current,
		"to":   targetStatus,
	})
	if err != nil {
		return FindingDetail{}, fmt.Errorf("marshal finding audit details: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, action, resource_type, resource_id, request_id, details
		)
		VALUES (NULLIF($1, ''), 'finding.status_changed', 'finding', $2, NULLIF($3, ''), $4)
	`, actorID, id, requestID, details); err != nil {
		return FindingDetail{}, fmt.Errorf("insert finding audit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return FindingDetail{}, fmt.Errorf("commit finding status transaction: %w", err)
	}

	return d.GetFinding(ctx, id)
}

func (d *Database) IsDetectionEnabled(ctx context.Context, id string) (bool, error) {
	if d == nil || d.pool == nil {
		return false, fmt.Errorf("database is not initialised")
	}

	var enabled bool
	if err := d.pool.QueryRow(ctx, "SELECT enabled FROM detections WHERE id = $1", id).Scan(&enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrDetectionNotFound
		}
		return false, fmt.Errorf("query detection enabled state: %w", err)
	}

	return enabled, nil
}

func (d *Database) ListDetections(ctx context.Context) ([]DetectionRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

	rows, err := d.pool.Query(ctx, `
		SELECT id, version, title, description, severity, enabled, definition, mitre, created_at, updated_at
		FROM detections
		ORDER BY id, version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query detections: %w", err)
	}
	defer rows.Close()

	var records []DetectionRecord
	for rows.Next() {
		record, err := scanDetection(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate detections: %w", err)
	}

	return records, nil
}

func (d *Database) SetDetectionEnabled(
	ctx context.Context,
	id string,
	enabled bool,
	actorID string,
	requestID string,
) (DetectionRecord, error) {
	if d == nil || d.pool == nil {
		return DetectionRecord{}, fmt.Errorf("database is not initialised")
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return DetectionRecord{}, fmt.Errorf("begin detection state transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current bool
	if err := tx.QueryRow(ctx, "SELECT enabled FROM detections WHERE id = $1 FOR UPDATE", id).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DetectionRecord{}, ErrDetectionNotFound
		}
		return DetectionRecord{}, fmt.Errorf("query detection state: %w", err)
	}
	if current == enabled {
		return DetectionRecord{}, ErrDetectionUnchanged
	}

	if _, err := tx.Exec(ctx, `
		UPDATE detections
		SET enabled = $2, updated_at = NOW()
		WHERE id = $1
	`, id, enabled); err != nil {
		return DetectionRecord{}, fmt.Errorf("update detection state: %w", err)
	}

	details, err := json.Marshal(map[string]bool{
		"from": current,
		"to":   enabled,
	})
	if err != nil {
		return DetectionRecord{}, fmt.Errorf("marshal detection audit details: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, action, resource_type, resource_id, request_id, details
		)
		VALUES (NULLIF($1, ''), 'detection.enabled_changed', 'detection', $2, NULLIF($3, ''), $4)
	`, actorID, id, requestID, details); err != nil {
		return DetectionRecord{}, fmt.Errorf("insert detection audit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return DetectionRecord{}, fmt.Errorf("commit detection state transaction: %w", err)
	}

	return d.getDetection(ctx, id)
}

func (d *Database) findingEvents(ctx context.Context, findingID string) ([]EventEvidenceRecord, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT
			e.id,
			e.source_timestamp,
			e.category,
			e.action,
			COALESCE(e.outcome, ''),
			e.asset_id,
			e.identity_id,
			e.payload
		FROM finding_events fe
		JOIN events e ON e.id = fe.event_id
		WHERE fe.finding_id = $1
		ORDER BY e.source_timestamp, e.id
	`, findingID)
	if err != nil {
		return nil, fmt.Errorf("query finding events: %w", err)
	}
	defer rows.Close()

	var records []EventEvidenceRecord
	for rows.Next() {
		var (
			record  EventEvidenceRecord
			payload []byte
		)
		if err := rows.Scan(
			&record.ID,
			&record.SourceTimestamp,
			&record.Category,
			&record.Action,
			&record.Outcome,
			&record.AssetID,
			&record.IdentityID,
			&payload,
		); err != nil {
			return nil, fmt.Errorf("scan finding event: %w", err)
		}
		if err := json.Unmarshal(payload, &record.Payload); err != nil {
			return nil, fmt.Errorf("decode finding event payload: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate finding events: %w", err)
	}
	return records, nil
}

func (d *Database) auditForResource(ctx context.Context, resourceType, resourceID string) ([]AuditRecord, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, occurred_at, actor_id, action, resource_type, resource_id, request_id, details
		FROM audit_events
		WHERE resource_type = $1 AND resource_id = $2
		ORDER BY occurred_at, id
	`, resourceType, resourceID)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	var records []AuditRecord
	for rows.Next() {
		var (
			record  AuditRecord
			details []byte
		)
		if err := rows.Scan(
			&record.ID,
			&record.OccurredAt,
			&record.ActorID,
			&record.Action,
			&record.ResourceType,
			&record.ResourceID,
			&record.RequestID,
			&details,
		); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		if err := json.Unmarshal(details, &record.Details); err != nil {
			return nil, fmt.Errorf("decode audit details: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}

	return records, nil
}

func (d *Database) getDetection(ctx context.Context, id string) (DetectionRecord, error) {
	row := d.pool.QueryRow(ctx, `
		SELECT id, version, title, description, severity, enabled, definition, mitre, created_at, updated_at
		FROM detections
		WHERE id = $1
	`, id)

	record, err := scanDetection(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DetectionRecord{}, ErrDetectionNotFound
	}
	return record, err
}

type detectionScanner interface {
	Scan(...any) error
}

func scanDetection(scanner detectionScanner) (DetectionRecord, error) {
	var (
		record     DetectionRecord
		definition []byte
		mitre      []byte
	)
	if err := scanner.Scan(
		&record.ID,
		&record.Version,
		&record.Title,
		&record.Description,
		&record.Severity,
		&record.Enabled,
		&definition,
		&mitre,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return DetectionRecord{}, err
	}
	if err := json.Unmarshal(definition, &record.Definition); err != nil {
		return DetectionRecord{}, fmt.Errorf("decode detection definition: %w", err)
	}
	if err := json.Unmarshal(mitre, &record.MITRE); err != nil {
		return DetectionRecord{}, fmt.Errorf("decode detection MITRE mapping: %w", err)
	}
	return record, nil
}
