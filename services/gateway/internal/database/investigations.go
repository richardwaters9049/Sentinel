package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/investigation"
)

var ErrInvestigationNotFound = errors.New("investigation not found")

type InvestigationCreateInput struct {
	Title       string
	Description string
	Priority    string
	OwnerID     string
	CreatedBy   string
	FindingIDs  []string
	EventIDs    []string
	RequestID   string
}

func (d *Database) CreateInvestigation(
	ctx context.Context,
	input InvestigationCreateInput,
) (InvestigationDetail, error) {
	if d == nil || d.pool == nil {
		return InvestigationDetail{}, fmt.Errorf("database is not initialised")
	}

	priority := strings.ToLower(strings.TrimSpace(input.Priority))
	if priority == "" {
		priority = "medium"
	}
	if err := investigation.ValidateCreate(input.Title, priority); err != nil {
		return InvestigationDetail{}, err
	}
	if strings.TrimSpace(input.CreatedBy) == "" {
		return InvestigationDetail{}, investigation.ErrInvalidInvestigation
	}

	id, err := newInvestigationID()
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("generate investigation id: %w", err)
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("begin investigation transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO investigations (
			id, title, description, status, priority, owner_id, created_by
		)
		VALUES ($1, $2, $3, 'open', $4, NULLIF($5, ''), $6)
	`,
		id,
		strings.TrimSpace(input.Title),
		strings.TrimSpace(input.Description),
		priority,
		strings.TrimSpace(input.OwnerID),
		strings.TrimSpace(input.CreatedBy),
	); err != nil {
		return InvestigationDetail{}, fmt.Errorf("insert investigation: %w", err)
	}

	for _, findingID := range uniqueNonEmpty(input.FindingIDs) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO investigation_findings (investigation_id, finding_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, id, findingID); err != nil {
			return InvestigationDetail{}, fmt.Errorf("attach finding %s: %w", findingID, err)
		}
	}

	for _, eventID := range uniqueNonEmpty(input.EventIDs) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO investigation_events (investigation_id, event_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, id, eventID); err != nil {
			return InvestigationDetail{}, fmt.Errorf("attach event %s: %w", eventID, err)
		}
	}

	details, err := json.Marshal(map[string]any{
		"priority":    priority,
		"finding_ids": uniqueNonEmpty(input.FindingIDs),
		"event_ids":   uniqueNonEmpty(input.EventIDs),
	})
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("marshal investigation audit details: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, action, resource_type, resource_id, request_id, details
		)
		VALUES ($1, 'investigation.created', 'investigation', $2, NULLIF($3, ''), $4)
	`, input.CreatedBy, id, input.RequestID, details); err != nil {
		return InvestigationDetail{}, fmt.Errorf("insert investigation audit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return InvestigationDetail{}, fmt.Errorf("commit investigation transaction: %w", err)
	}

	return d.GetInvestigation(ctx, id)
}

func (d *Database) ListInvestigations(
	ctx context.Context,
	status string,
	limit int,
) ([]investigation.Record, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	args := []any{}
	sql := "SELECT id, title, description, status, priority, owner_id, created_by, created_at, updated_at FROM investigations"
	if value := strings.TrimSpace(status); value != "" {
		args = append(args, value)
		sql += " WHERE status = $1"
	}
	args = append(args, limit)
	sql += fmt.Sprintf(" ORDER BY updated_at DESC, id DESC LIMIT $%d", len(args))

	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query investigations: %w", err)
	}
	defer rows.Close()

	records := make([]investigation.Record, 0, limit)
	for rows.Next() {
		var record investigation.Record
		if err := rows.Scan(
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
			return nil, fmt.Errorf("scan investigation: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate investigations: %w", err)
	}
	return records, nil
}

func (d *Database) AddInvestigationNote(
	ctx context.Context,
	id string,
	actorID string,
	body string,
	requestID string,
) (InvestigationDetail, error) {
	if d == nil || d.pool == nil {
		return InvestigationDetail{}, fmt.Errorf("database is not initialised")
	}
	if strings.TrimSpace(actorID) == "" || strings.TrimSpace(body) == "" || len(strings.TrimSpace(body)) > 5000 {
		return InvestigationDetail{}, investigation.ErrInvalidInvestigation
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("begin investigation note transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM investigations WHERE id = $1)", id).Scan(&exists); err != nil {
		return InvestigationDetail{}, fmt.Errorf("check investigation: %w", err)
	}
	if !exists {
		return InvestigationDetail{}, ErrInvestigationNotFound
	}

	if _, err := tx.Exec(ctx, "INSERT INTO investigation_notes (investigation_id, actor_id, body) VALUES ($1, $2, $3)", id, actorID, strings.TrimSpace(body)); err != nil {
		return InvestigationDetail{}, fmt.Errorf("insert investigation note: %w", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE investigations SET updated_at = NOW() WHERE id = $1", id); err != nil {
		return InvestigationDetail{}, fmt.Errorf("touch investigation: %w", err)
	}

	details, err := json.Marshal(map[string]string{"body": strings.TrimSpace(body)})
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("marshal note audit details: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, action, resource_type, resource_id, request_id, details
		)
		VALUES ($1, 'investigation.note_added', 'investigation', $2, NULLIF($3, ''), $4)
	`, actorID, id, requestID, details); err != nil {
		return InvestigationDetail{}, fmt.Errorf("insert note audit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return InvestigationDetail{}, fmt.Errorf("commit investigation note transaction: %w", err)
	}
	return d.GetInvestigation(ctx, id)
}

func (d *Database) UpdateInvestigationStatus(
	ctx context.Context,
	id string,
	targetStatus string,
	actorID string,
	requestID string,
) (InvestigationDetail, error) {
	if d == nil || d.pool == nil {
		return InvestigationDetail{}, fmt.Errorf("database is not initialised")
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("begin investigation status transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current string
	if err := tx.QueryRow(ctx, "SELECT status FROM investigations WHERE id = $1 FOR UPDATE", id).Scan(&current); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InvestigationDetail{}, ErrInvestigationNotFound
		}
		return InvestigationDetail{}, fmt.Errorf("query investigation status: %w", err)
	}

	targetStatus = strings.TrimSpace(targetStatus)
	if err := investigation.ValidateStatusTransition(current, targetStatus); err != nil {
		return InvestigationDetail{}, err
	}

	if _, err := tx.Exec(ctx, "UPDATE investigations SET status = $2, updated_at = NOW() WHERE id = $1", id, targetStatus); err != nil {
		return InvestigationDetail{}, fmt.Errorf("update investigation status: %w", err)
	}

	details, err := json.Marshal(map[string]string{"from": current, "to": targetStatus})
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("marshal investigation status audit: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, action, resource_type, resource_id, request_id, details
		)
		VALUES ($1, 'investigation.status_changed', 'investigation', $2, NULLIF($3, ''), $4)
	`, actorID, id, requestID, details); err != nil {
		return InvestigationDetail{}, fmt.Errorf("insert investigation status audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return InvestigationDetail{}, fmt.Errorf("commit investigation status transaction: %w", err)
	}
	return d.GetInvestigation(ctx, id)
}

func (d *Database) AttachHuntRun(
	ctx context.Context,
	investigationID string,
	runID int64,
	actorID string,
	requestID string,
) (InvestigationDetail, error) {
	if d == nil || d.pool == nil {
		return InvestigationDetail{}, fmt.Errorf("database is not initialised")
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("begin hunt-run attachment transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var investigationExists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM investigations WHERE id = $1)", investigationID).Scan(&investigationExists); err != nil {
		return InvestigationDetail{}, fmt.Errorf("check investigation: %w", err)
	}
	if !investigationExists {
		return InvestigationDetail{}, ErrInvestigationNotFound
	}

	var runExists bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM hunt_runs WHERE id = $1)", runID).Scan(&runExists); err != nil {
		return InvestigationDetail{}, fmt.Errorf("check hunt run: %w", err)
	}
	if !runExists {
		return InvestigationDetail{}, ErrHuntRunNotFound
	}

	commandTag, err := tx.Exec(ctx, `
		INSERT INTO investigation_events (investigation_id, event_id)
		SELECT $1, event_id
		FROM hunt_run_events
		WHERE hunt_run_id = $2
		ON CONFLICT DO NOTHING
	`, investigationID, runID)
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("attach hunt run events: %w", err)
	}

	if _, err := tx.Exec(ctx, "UPDATE investigations SET updated_at = NOW() WHERE id = $1", investigationID); err != nil {
		return InvestigationDetail{}, fmt.Errorf("touch investigation: %w", err)
	}

	details, err := json.Marshal(map[string]any{
		"hunt_run_id":  runID,
		"events_added": commandTag.RowsAffected(),
	})
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("marshal hunt-run audit details: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, action, resource_type, resource_id, request_id, details
		)
		VALUES ($1, 'investigation.hunt_run_attached', 'investigation', $2, NULLIF($3, ''), $4)
	`, actorID, investigationID, requestID, details); err != nil {
		return InvestigationDetail{}, fmt.Errorf("insert hunt-run attachment audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return InvestigationDetail{}, fmt.Errorf("commit hunt-run attachment: %w", err)
	}
	return d.GetInvestigation(ctx, investigationID)
}

func (d *Database) UpdateInvestigationMetadata(
	ctx context.Context,
	id string,
	ownerID *string,
	priority *string,
	actorID string,
	requestID string,
) (InvestigationDetail, error) {
	if d == nil || d.pool == nil {
		return InvestigationDetail{}, fmt.Errorf("database is not initialised")
	}
	if ownerID == nil && priority == nil {
		return InvestigationDetail{}, investigation.ErrInvalidInvestigation
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("begin investigation metadata transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		currentOwner    *string
		currentPriority string
	)
	if err := tx.QueryRow(ctx, `
		SELECT owner_id, priority
		FROM investigations
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&currentOwner, &currentPriority); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return InvestigationDetail{}, ErrInvestigationNotFound
		}
		return InvestigationDetail{}, fmt.Errorf("query investigation metadata: %w", err)
	}

	nextOwner := currentOwner
	nextPriority := currentPriority
	if ownerID != nil {
		value := strings.TrimSpace(*ownerID)
		if value == "" {
			nextOwner = nil
		} else {
			nextOwner = &value
		}
	}
	if priority != nil {
		value := strings.ToLower(strings.TrimSpace(*priority))
		if err := investigation.ValidateCreate("metadata-update", value); err != nil {
			return InvestigationDetail{}, err
		}
		nextPriority = value
	}

	if _, err := tx.Exec(ctx, `
		UPDATE investigations
		SET owner_id = $2, priority = $3, updated_at = NOW()
		WHERE id = $1
	`, id, nextOwner, nextPriority); err != nil {
		return InvestigationDetail{}, fmt.Errorf("update investigation metadata: %w", err)
	}

	details, err := json.Marshal(map[string]any{
		"owner_id": map[string]any{
			"from": currentOwner,
			"to":   nextOwner,
		},
		"priority": map[string]string{
			"from": currentPriority,
			"to":   nextPriority,
		},
	})
	if err != nil {
		return InvestigationDetail{}, fmt.Errorf("marshal investigation metadata audit: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, action, resource_type, resource_id, request_id, details
		)
		VALUES ($1, 'investigation.metadata_changed', 'investigation', $2, NULLIF($3, ''), $4)
	`, actorID, id, requestID, details); err != nil {
		return InvestigationDetail{}, fmt.Errorf("insert investigation metadata audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return InvestigationDetail{}, fmt.Errorf("commit investigation metadata transaction: %w", err)
	}
	return d.GetInvestigation(ctx, id)
}

func uniqueNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func newInvestigationID() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "inv_" + hex.EncodeToString(random[:]), nil
}
