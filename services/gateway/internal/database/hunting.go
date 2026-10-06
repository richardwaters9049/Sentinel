package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/hunting"
)

var ErrHuntNotFound = errors.New("hunt not found")

func (d *Database) CreateHunt(
	ctx context.Context,
	name string,
	description string,
	hypothesis string,
	query hunting.Query,
	actorID string,
) (hunting.Definition, error) {
	if d == nil || d.pool == nil {
		return hunting.Definition{}, fmt.Errorf("database is not initialised")
	}
	if err := hunting.ValidateDefinition(name, hypothesis, query); err != nil {
		return hunting.Definition{}, err
	}

	id, err := newHuntID()
	if err != nil {
		return hunting.Definition{}, fmt.Errorf("generate hunt id: %w", err)
	}

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return hunting.Definition{}, fmt.Errorf("marshal hunt query: %w", err)
	}

	var definition hunting.Definition
	var storedQuery []byte

	err = d.pool.QueryRow(ctx, `
		INSERT INTO hunts (
			id, name, description, hypothesis, query, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, description, hypothesis, query, created_by, created_at, updated_at
	`,
		id,
		strings.TrimSpace(name),
		strings.TrimSpace(description),
		strings.TrimSpace(hypothesis),
		queryJSON,
		actorID,
	).Scan(
		&definition.ID,
		&definition.Name,
		&definition.Description,
		&definition.Hypothesis,
		&storedQuery,
		&definition.CreatedBy,
		&definition.CreatedAt,
		&definition.UpdatedAt,
	)
	if err != nil {
		return hunting.Definition{}, fmt.Errorf("insert hunt: %w", err)
	}
	if err := json.Unmarshal(storedQuery, &definition.Query); err != nil {
		return hunting.Definition{}, fmt.Errorf("decode hunt query: %w", err)
	}

	return definition, nil
}

func (d *Database) ListHunts(ctx context.Context) ([]hunting.Definition, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

	rows, err := d.pool.Query(ctx, `
		SELECT id, name, description, hypothesis, query, created_by, created_at, updated_at
		FROM hunts
		ORDER BY created_at DESC, id
	`)
	if err != nil {
		return nil, fmt.Errorf("query hunts: %w", err)
	}
	defer rows.Close()

	hunts := make([]hunting.Definition, 0)
	for rows.Next() {
		definition, err := scanHunt(rows)
		if err != nil {
			return nil, err
		}
		hunts = append(hunts, definition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hunts: %w", err)
	}

	return hunts, nil
}

func (d *Database) GetHunt(ctx context.Context, id string) (hunting.Definition, error) {
	if d == nil || d.pool == nil {
		return hunting.Definition{}, fmt.Errorf("database is not initialised")
	}

	row := d.pool.QueryRow(ctx, `
		SELECT id, name, description, hypothesis, query, created_by, created_at, updated_at
		FROM hunts
		WHERE id = $1
	`, id)

	definition, err := scanHunt(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return hunting.Definition{}, ErrHuntNotFound
	}
	if err != nil {
		return hunting.Definition{}, err
	}
	return definition, nil
}

func (d *Database) RunHunt(
	ctx context.Context,
	id string,
	actorID string,
	override hunting.Query,
) (hunting.RunResult, error) {
	definition, err := d.GetHunt(ctx, id)
	if err != nil {
		return hunting.RunResult{}, err
	}

	query := definition.Query
	mergeHuntQuery(&query, override)

	if err := hunting.ValidateQuery(query); err != nil {
		return hunting.RunResult{}, err
	}

	startedAt := time.Now().UTC()
	events, err := d.executeHuntQuery(ctx, query)
	if err != nil {
		return hunting.RunResult{}, err
	}

	parameters, err := json.Marshal(query)
	if err != nil {
		return hunting.RunResult{}, fmt.Errorf("marshal hunt run parameters: %w", err)
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return hunting.RunResult{}, fmt.Errorf("begin hunt run transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var runID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO hunt_runs (
			hunt_id, actor_id, started_at, completed_at, result_count, parameters
		)
		VALUES ($1, $2, $3, NOW(), $4, $5)
		RETURNING id
	`, id, actorID, startedAt, len(events), parameters).Scan(&runID); err != nil {
		return hunting.RunResult{}, fmt.Errorf("record hunt run: %w", err)
	}

	for _, event := range events {
		if _, err := tx.Exec(ctx, `
			INSERT INTO hunt_run_events (hunt_run_id, event_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, runID, event.ID); err != nil {
			return hunting.RunResult{}, fmt.Errorf("record hunt result event %s: %w", event.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return hunting.RunResult{}, fmt.Errorf("commit hunt run transaction: %w", err)
	}

	return hunting.RunResult{
		RunID:       runID,
		HuntID:      id,
		ResultCount: len(events),
		Events:      events,
		ExecutedAt:  time.Now().UTC(),
	}, nil
}

func (d *Database) ListHuntRuns(
	ctx context.Context,
	huntID string,
	limit int,
) ([]hunting.RunRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if _, err := d.GetHunt(ctx, huntID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := d.pool.Query(ctx, `
		SELECT id, hunt_id, actor_id, started_at, completed_at, result_count, parameters
		FROM hunt_runs
		WHERE hunt_id = $1
		ORDER BY started_at DESC, id DESC
		LIMIT $2
	`, huntID, limit)
	if err != nil {
		return nil, fmt.Errorf("query hunt runs: %w", err)
	}
	defer rows.Close()

	runs := make([]hunting.RunRecord, 0, limit)
	for rows.Next() {
		var (
			run        hunting.RunRecord
			parameters []byte
		)
		if err := rows.Scan(
			&run.ID,
			&run.HuntID,
			&run.ActorID,
			&run.StartedAt,
			&run.CompletedAt,
			&run.ResultCount,
			&parameters,
		); err != nil {
			return nil, fmt.Errorf("scan hunt run: %w", err)
		}
		if err := json.Unmarshal(parameters, &run.Parameters); err != nil {
			return nil, fmt.Errorf("decode hunt run parameters: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hunt runs: %w", err)
	}
	return runs, nil
}

func (d *Database) HuntRunEvents(
	ctx context.Context,
	runID int64,
) ([]EventEvidenceRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

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
		FROM hunt_run_events hre
		JOIN events e ON e.id = hre.event_id
		WHERE hre.hunt_run_id = $1
		ORDER BY e.source_timestamp DESC, e.id DESC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("query hunt run events: %w", err)
	}
	defer rows.Close()

	events := make([]EventEvidenceRecord, 0)
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
			return nil, fmt.Errorf("scan hunt run event: %w", err)
		}
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, fmt.Errorf("decode hunt run event payload: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hunt run events: %w", err)
	}
	return events, nil
}

func (d *Database) executeHuntQuery(
	ctx context.Context,
	query hunting.Query,
) ([]EventEvidenceRecord, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}

	args := make([]any, 0, 10)
	conditions := make([]string, 0, 9)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if value := strings.TrimSpace(query.Category); value != "" {
		conditions = append(conditions, "category = "+addArg(strings.ToLower(value)))
	}
	if value := strings.TrimSpace(query.Action); value != "" {
		conditions = append(conditions, "action = "+addArg(strings.ToLower(value)))
	}
	if value := strings.TrimSpace(query.Outcome); value != "" {
		conditions = append(conditions, "outcome = "+addArg(strings.ToLower(value)))
	}
	if value := strings.TrimSpace(query.AssetID); value != "" {
		conditions = append(conditions, "asset_id = "+addArg(value))
	}
	if value := strings.TrimSpace(query.IdentityID); value != "" {
		conditions = append(conditions, "identity_id = "+addArg(value))
	}
	if value := strings.TrimSpace(query.SourceIP); value != "" {
		conditions = append(conditions, "payload #>> '{network,source_ip}' = "+addArg(value))
	}
	if value := strings.TrimSpace(query.DestinationZone); value != "" {
		conditions = append(conditions, "payload #>> '{network,destination_zone}' = "+addArg(strings.ToLower(value)))
	}
	if query.From != nil {
		conditions = append(conditions, "source_timestamp >= "+addArg(query.From.UTC()))
	}
	if query.To != nil {
		conditions = append(conditions, "source_timestamp <= "+addArg(query.To.UTC()))
	}

	sql := `
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
	`
	if len(conditions) > 0 {
		sql += " WHERE " + strings.Join(conditions, " AND ")
	}
	sql += " ORDER BY source_timestamp DESC, id DESC LIMIT " + addArg(limit)

	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("execute hunt query: %w", err)
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
			return nil, fmt.Errorf("scan hunt event: %w", err)
		}
		if err := json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, fmt.Errorf("decode hunt event payload: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hunt events: %w", err)
	}

	return events, nil
}

type huntScanner interface {
	Scan(...any) error
}

func scanHunt(scanner huntScanner) (hunting.Definition, error) {
	var (
		definition hunting.Definition
		queryJSON  []byte
	)

	if err := scanner.Scan(
		&definition.ID,
		&definition.Name,
		&definition.Description,
		&definition.Hypothesis,
		&queryJSON,
		&definition.CreatedBy,
		&definition.CreatedAt,
		&definition.UpdatedAt,
	); err != nil {
		return hunting.Definition{}, err
	}

	if err := json.Unmarshal(queryJSON, &definition.Query); err != nil {
		return hunting.Definition{}, fmt.Errorf("decode hunt query: %w", err)
	}

	return definition, nil
}

func mergeHuntQuery(base *hunting.Query, override hunting.Query) {
	if override.Category != "" {
		base.Category = override.Category
	}
	if override.Action != "" {
		base.Action = override.Action
	}
	if override.Outcome != "" {
		base.Outcome = override.Outcome
	}
	if override.AssetID != "" {
		base.AssetID = override.AssetID
	}
	if override.IdentityID != "" {
		base.IdentityID = override.IdentityID
	}
	if override.SourceIP != "" {
		base.SourceIP = override.SourceIP
	}
	if override.DestinationZone != "" {
		base.DestinationZone = override.DestinationZone
	}
	if override.From != nil {
		base.From = override.From
	}
	if override.To != nil {
		base.To = override.To
	}
	if override.Limit != 0 {
		base.Limit = override.Limit
	}
}

func newHuntID() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "hunt_" + hex.EncodeToString(random[:]), nil
}
