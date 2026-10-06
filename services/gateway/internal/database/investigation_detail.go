package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/investigation"
)

type InvestigationDetail struct {
	investigation.Record
	Findings []FindingRecord              `json:"findings"`
	Events   []EventEvidenceRecord        `json:"events"`
	Notes    []investigation.Note         `json:"notes"`
	Audit    []AuditRecord                `json:"audit"`
	Timeline []InvestigationTimelineEntry `json:"timeline"`
}

type InvestigationTimelineEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"`
	ID        string         `json:"id"`
	Summary   string         `json:"summary"`
	Data      map[string]any `json:"data,omitempty"`
}

func (d *Database) GetInvestigation(
	ctx context.Context,
	id string,
) (InvestigationDetail, error) {
	if d == nil || d.pool == nil {
		return InvestigationDetail{}, fmt.Errorf("database is not initialised")
	}

	var record investigation.Record
	if err := d.pool.QueryRow(ctx, `
		SELECT id, title, description, status, priority, owner_id, created_by, created_at, updated_at
		FROM investigations
		WHERE id = $1
	`, id).Scan(
		&record.ID,
		&record.Title,
		&record.Description,
		&record.Status,
		&record.Priority,
		&record.OwnerID,
		&record.CreatedBy,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InvestigationDetail{}, ErrInvestigationNotFound
		}
		return InvestigationDetail{}, fmt.Errorf("query investigation: %w", err)
	}

	findings, err := d.investigationFindings(ctx, id)
	if err != nil {
		return InvestigationDetail{}, err
	}
	events, err := d.investigationEvents(ctx, id)
	if err != nil {
		return InvestigationDetail{}, err
	}
	notes, err := d.investigationNotes(ctx, id)
	if err != nil {
		return InvestigationDetail{}, err
	}
	audit, err := d.auditForResource(ctx, "investigation", id)
	if err != nil {
		return InvestigationDetail{}, err
	}

	return InvestigationDetail{
		Record:   record,
		Findings: findings,
		Events:   events,
		Notes:    notes,
		Audit:    audit,
		Timeline: buildInvestigationTimeline(findings, events, notes, audit),
	}, nil
}

func (d *Database) investigationFindings(ctx context.Context, id string) ([]FindingRecord, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT
			f.id, f.detection_id, f.detection_version, f.title, f.severity,
			f.confidence, f.status, f.first_observed_at, f.last_observed_at,
			f.evidence, f.created_at, f.updated_at
		FROM investigation_findings i
		JOIN findings f ON f.id = i.finding_id
		WHERE i.investigation_id = $1
		ORDER BY f.last_observed_at, f.id
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query investigation findings: %w", err)
	}
	defer rows.Close()

	records := make([]FindingRecord, 0)
	for rows.Next() {
		var record FindingRecord
		var evidence []byte
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
			return nil, fmt.Errorf("scan investigation finding: %w", err)
		}
		if err := json.Unmarshal(evidence, &record.Evidence); err != nil {
			return nil, fmt.Errorf("decode investigation finding evidence: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate investigation findings: %w", err)
	}
	return records, nil
}

func (d *Database) investigationEvents(ctx context.Context, id string) ([]EventEvidenceRecord, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT
			e.id, e.source_timestamp, e.category, e.action, COALESCE(e.outcome, ''),
			e.asset_id, e.identity_id, e.payload
		FROM investigation_events i
		JOIN events e ON e.id = i.event_id
		WHERE i.investigation_id = $1
		ORDER BY e.source_timestamp, e.id
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query investigation events: %w", err)
	}
	defer rows.Close()

	records := make([]EventEvidenceRecord, 0)
	for rows.Next() {
		var event EventEvidenceRecord
		var payload []byte
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
			return nil, fmt.Errorf("scan investigation event: %w", err)
		}
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, fmt.Errorf("decode investigation event payload: %w", err)
		}
		records = append(records, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate investigation events: %w", err)
	}
	return records, nil
}

func (d *Database) investigationNotes(ctx context.Context, id string) ([]investigation.Note, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, investigation_id, actor_id, body, created_at
		FROM investigation_notes
		WHERE investigation_id = $1
		ORDER BY created_at, id
	`, id)
	if err != nil {
		return nil, fmt.Errorf("query investigation notes: %w", err)
	}
	defer rows.Close()

	notes := make([]investigation.Note, 0)
	for rows.Next() {
		var note investigation.Note
		if err := rows.Scan(&note.ID, &note.InvestigationID, &note.ActorID, &note.Body, &note.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan investigation note: %w", err)
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate investigation notes: %w", err)
	}
	return notes, nil
}

func buildInvestigationTimeline(
	findings []FindingRecord,
	events []EventEvidenceRecord,
	notes []investigation.Note,
	audit []AuditRecord,
) []InvestigationTimelineEntry {
	timeline := make([]InvestigationTimelineEntry, 0, len(findings)+len(events)+len(notes)+len(audit))

	for _, finding := range findings {
		timeline = append(timeline, InvestigationTimelineEntry{
			Timestamp: finding.FirstObservedAt,
			Type:      "finding",
			ID:        finding.ID,
			Summary:   finding.Title,
			Data: map[string]any{
				"detection_id": finding.DetectionID,
				"severity":     finding.Severity,
				"status":       finding.Status,
			},
		})
	}
	for _, event := range events {
		timeline = append(timeline, InvestigationTimelineEntry{
			Timestamp: event.SourceTimestamp,
			Type:      "event",
			ID:        event.ID,
			Summary:   event.Category + "/" + event.Action,
			Data:      map[string]any{"outcome": event.Outcome},
		})
	}
	for _, note := range notes {
		timeline = append(timeline, InvestigationTimelineEntry{
			Timestamp: note.CreatedAt,
			Type:      "note",
			ID:        fmt.Sprintf("note_%d", note.ID),
			Summary:   note.Body,
			Data:      map[string]any{"actor_id": note.ActorID},
		})
	}
	for _, item := range audit {
		timeline = append(timeline, InvestigationTimelineEntry{
			Timestamp: item.OccurredAt,
			Type:      "audit",
			ID:        fmt.Sprintf("audit_%d", item.ID),
			Summary:   item.Action,
			Data:      item.Details,
		})
	}

	sort.SliceStable(timeline, func(i, j int) bool {
		if timeline[i].Timestamp.Equal(timeline[j].Timestamp) {
			return timeline[i].ID < timeline[j].ID
		}
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})
	return timeline
}
