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

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return hunting.Definition{}, fmt.Errorf("begin hunt transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO hunts (
			id, current_version, name, description, hypothesis, query, created_by
		)
		VALUES ($1, 1, $2, $3, $4, $5, $6)
	`,
		id,
		strings.TrimSpace(name),
		strings.TrimSpace(description),
		strings.TrimSpace(hypothesis),
		queryJSON,
		actorID,
	); err != nil {
		return hunting.Definition{}, fmt.Errorf("insert hunt: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO hunt_versions (
			hunt_id, version, name, description, hypothesis, query, changed_by
		)
		VALUES ($1, 1, $2, $3, $4, $5, $6)
	`,
		id,
		strings.TrimSpace(name),
		strings.TrimSpace(description),
		strings.TrimSpace(hypothesis),
		queryJSON,
		actorID,
	); err != nil {
		return hunting.Definition{}, fmt.Errorf("insert initial hunt version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return hunting.Definition{}, fmt.Errorf("commit hunt transaction: %w", err)
	}
	return d.GetHunt(ctx, id)
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
		SELECT id, current_version, name, description, hypothesis, query, created_by, created_at, updated_at
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

func (d *Database) UpdateHunt(
	ctx context.Context,
	id string,
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

	queryJSON, err := json.Marshal(query)
	if err != nil {
		return hunting.Definition{}, fmt.Errorf("marshal hunt query: %w", err)
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return hunting.Definition{}, fmt.Errorf("begin hunt update transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentVersion int
	if err := tx.QueryRow(ctx, "SELECT current_version FROM hunts WHERE id = $1 FOR UPDATE", id).Scan(&currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return hunting.Definition{}, ErrHuntNotFound
		}
		return hunting.Definition{}, fmt.Errorf("query hunt version: %w", err)
	}
	nextVersion := currentVersion + 1

	if _, err := tx.Exec(ctx, `
		UPDATE hunts
		SET current_version = $2,
		    name = $3,
		    description = $4,
		    hypothesis = $5,
		    query = $6,
		    updated_at = NOW()
		WHERE id = $1
	`,
		id,
		nextVersion,
		strings.TrimSpace(name),
		strings.TrimSpace(description),
		strings.TrimSpace(hypothesis),
		queryJSON,
	); err != nil {
		return hunting.Definition{}, fmt.Errorf("update hunt: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO hunt_versions (
			hunt_id, version, name, description, hypothesis, query, changed_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		id,
		nextVersion,
		strings.TrimSpace(name),
		strings.TrimSpace(description),
		strings.TrimSpace(hypothesis),
		queryJSON,
		actorID,
	); err != nil {
		return hunting.Definition{}, fmt.Errorf("insert hunt version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return hunting.Definition{}, fmt.Errorf("commit hunt update: %w", err)
	}
	return d.GetHunt(ctx, id)
}

func (d *Database) ListHuntVersions(
	ctx context.Context,
	huntID string,
) ([]hunting.VersionRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if _, err := d.GetHunt(ctx, huntID); err != nil {
		return nil, err
	}

	rows, err := d.pool.Query(ctx, `
		SELECT hunt_id, version, name, description, hypothesis, query, changed_by, changed_at
		FROM hunt_versions
		WHERE hunt_id = $1
		ORDER BY version DESC
	`, huntID)
	if err != nil {
		return nil, fmt.Errorf("query hunt versions: %w", err)
	}
	defer rows.Close()

	versions := make([]hunting.VersionRecord, 0)
	for rows.Next() {
		var (
			record    hunting.VersionRecord
			queryJSON []byte
		)
		if err := rows.Scan(
			&record.HuntID,
			&record.Version,
			&record.Name,
			&record.Description,
			&record.Hypothesis,
			&queryJSON,
			&record.ChangedBy,
			&record.ChangedAt,
		); err != nil {
			return nil, fmt.Errorf("scan hunt version: %w", err)
		}
		if err := json.Unmarshal(queryJSON, &record.Query); err != nil {
			return nil, fmt.Errorf("decode hunt version query: %w", err)
		}
		versions = append(versions, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hunt versions: %w", err)
	}
	return versions, nil
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

	args := make([]any, 0, 20)
	conditions := make([]string, 0, 18)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if value := strings.TrimSpace(query.Category); value != "" {
		conditions = append(conditions, "category = "+addArg(strings.ToLower(value)))
	}
	if values := normaliseStrings(query.Categories); len(values) > 0 {
		conditions = append(conditions, "category = ANY("+addArg(values)+"::text[])")
	}
	if value := strings.TrimSpace(query.Action); value != "" {
		conditions = append(conditions, "action = "+addArg(strings.ToLower(value)))
	}
	if values := normaliseStrings(query.Actions); len(values) > 0 {
		conditions = append(conditions, "action = ANY("+addArg(values)+"::text[])")
	}
	if value := strings.TrimSpace(query.Outcome); value != "" {
		conditions = append(conditions, "outcome = "+addArg(strings.ToLower(value)))
	}
	if values := normaliseStrings(query.Outcomes); len(values) > 0 {
		conditions = append(conditions, "outcome = ANY("+addArg(values)+"::text[])")
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
	if value := strings.TrimSpace(query.DestinationIP); value != "" {
		conditions = append(conditions, "payload #>> '{network,destination_ip}' = "+addArg(value))
	}
	if values := normaliseStrings(query.SourceZones); len(values) > 0 {
		conditions = append(conditions, "payload #>> '{asset,zone}' = ANY("+addArg(values)+"::text[])")
	}
	if value := strings.TrimSpace(query.DestinationZone); value != "" {
		conditions = append(conditions, "payload #>> '{network,destination_zone}' = "+addArg(strings.ToLower(value)))
	}
	if values := normaliseStrings(query.DestinationZones); len(values) > 0 {
		conditions = append(conditions, "payload #>> '{network,destination_zone}' = ANY("+addArg(values)+"::text[])")
	}
	if len(query.DestinationPorts) > 0 {
		conditions = append(conditions, "(payload #>> '{network,destination_port}')::int = ANY("+addArg(query.DestinationPorts)+"::int[])")
	}
	if len(query.Labels) > 0 {
		labelsJSON, err := json.Marshal(query.Labels)
		if err != nil {
			return nil, fmt.Errorf("marshal hunt labels: %w", err)
		}
		conditions = append(conditions, "labels @> "+addArg(labelsJSON)+"::jsonb")
	}
	if query.LastMinutes > 0 {
		conditions = append(conditions, "source_timestamp >= "+addArg(time.Now().UTC().Add(-time.Duration(query.LastMinutes)*time.Minute)))
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
		&definition.Version,
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
	if len(override.Categories) > 0 {
		base.Categories = append([]string(nil), override.Categories...)
	}
	if override.Action != "" {
		base.Action = override.Action
	}
	if len(override.Actions) > 0 {
		base.Actions = append([]string(nil), override.Actions...)
	}
	if override.Outcome != "" {
		base.Outcome = override.Outcome
	}
	if len(override.Outcomes) > 0 {
		base.Outcomes = append([]string(nil), override.Outcomes...)
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
	if override.DestinationIP != "" {
		base.DestinationIP = override.DestinationIP
	}
	if len(override.SourceZones) > 0 {
		base.SourceZones = append([]string(nil), override.SourceZones...)
	}
	if override.DestinationZone != "" {
		base.DestinationZone = override.DestinationZone
	}
	if len(override.DestinationZones) > 0 {
		base.DestinationZones = append([]string(nil), override.DestinationZones...)
	}
	if len(override.DestinationPorts) > 0 {
		base.DestinationPorts = append([]int(nil), override.DestinationPorts...)
	}
	if len(override.Labels) > 0 {
		base.Labels = make(map[string]string, len(override.Labels))
		for key, value := range override.Labels {
			base.Labels[key] = value
		}
	}
	if override.LastMinutes != 0 {
		base.LastMinutes = override.LastMinutes
		base.From = nil
		base.To = nil
	}
	if override.From != nil {
		base.LastMinutes = 0
		base.From = override.From
	}
	if override.To != nil {
		base.LastMinutes = 0
		base.To = override.To
	}
	if override.Limit != 0 {
		base.Limit = override.Limit
	}
}

func normaliseStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
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

func newHuntID() (string, error) {
	var random [12]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "hunt_" + hex.EncodeToString(random[:]), nil
}
