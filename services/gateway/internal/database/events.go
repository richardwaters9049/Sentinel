package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/telemetry"
)

type EventQuery struct {
	Limit      int
	Before     *time.Time
	Category   string
	AssetID    string
	IdentityID string
}

func (d *Database) SaveEvent(ctx context.Context, event telemetry.Event) (bool, error) {
	if d == nil || d.pool == nil {
		return false, fmt.Errorf("database is not initialised")
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return false, fmt.Errorf("marshal event payload: %w", err)
	}

	labels, err := json.Marshal(event.Labels)
	if err != nil {
		return false, fmt.Errorf("marshal event labels: %w", err)
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin event transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var assetID interface{}
	if event.Asset != nil {
		assetID = event.Asset.ID
		if _, err := tx.Exec(ctx, `
			INSERT INTO assets (
				id, hostname, zone, first_seen_at, last_seen_at
			)
			VALUES ($1, $2, $3, $4, $4)
			ON CONFLICT (id) DO UPDATE SET
				hostname = EXCLUDED.hostname,
				zone = EXCLUDED.zone,
				first_seen_at = LEAST(
					COALESCE(assets.first_seen_at, EXCLUDED.first_seen_at),
					EXCLUDED.first_seen_at
				),
				last_seen_at = GREATEST(
					COALESCE(assets.last_seen_at, EXCLUDED.last_seen_at),
					EXCLUDED.last_seen_at
				),
				updated_at = NOW()
		`, event.Asset.ID, event.Asset.Hostname, event.Asset.Zone, event.Timestamp); err != nil {
			return false, fmt.Errorf("upsert event asset: %w", err)
		}
	}

	var identityID interface{}
	if event.Actor != nil {
		identityID = event.Actor.ID
		if _, err := tx.Exec(ctx, `
			INSERT INTO identities (
				id, identity_type, name, first_seen_at, last_seen_at
			)
			VALUES ($1, $2, $3, $4, $4)
			ON CONFLICT (id) DO UPDATE SET
				identity_type = EXCLUDED.identity_type,
				name = EXCLUDED.name,
				first_seen_at = LEAST(
					COALESCE(identities.first_seen_at, EXCLUDED.first_seen_at),
					EXCLUDED.first_seen_at
				),
				last_seen_at = GREATEST(
					COALESCE(identities.last_seen_at, EXCLUDED.last_seen_at),
					EXCLUDED.last_seen_at
				),
				updated_at = NOW()
		`, event.Actor.ID, event.Actor.Type, event.Actor.Name, event.Timestamp); err != nil {
			return false, fmt.Errorf("upsert event identity: %w", err)
		}
	}

	commandTag, err := tx.Exec(ctx, `
		INSERT INTO events (
			id,
			schema_version,
			source_timestamp,
			received_at,
			source_type,
			category,
			action,
			outcome,
			asset_id,
			identity_id,
			payload,
			labels
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''),
			$9, $10, $11, $12
		)
		ON CONFLICT (id) DO NOTHING
	`,
		event.EventID,
		event.SchemaVersion,
		event.Timestamp,
		event.ReceivedAt,
		event.Source.Type,
		event.Event.Category,
		event.Event.Action,
		event.Event.Outcome,
		assetID,
		identityID,
		payload,
		labels,
	)
	if err != nil {
		return false, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit event transaction: %w", err)
	}

	return commandTag.RowsAffected() == 1, nil
}

func (d *Database) ListEvents(ctx context.Context, query EventQuery) ([]telemetry.Event, error) {
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

	args := make([]interface{}, 0, 5)
	conditions := make([]string, 0, 4)

	addArg := func(value interface{}) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if query.Before != nil {
		conditions = append(conditions, "source_timestamp < "+addArg(query.Before.UTC()))
	}
	if value := strings.TrimSpace(query.Category); value != "" {
		conditions = append(conditions, "category = "+addArg(strings.ToLower(value)))
	}
	if value := strings.TrimSpace(query.AssetID); value != "" {
		conditions = append(conditions, "asset_id = "+addArg(value))
	}
	if value := strings.TrimSpace(query.IdentityID); value != "" {
		conditions = append(conditions, "identity_id = "+addArg(value))
	}

	sql := "SELECT payload FROM events"
	if len(conditions) > 0 {
		sql += " WHERE " + strings.Join(conditions, " AND ")
	}
	sql += " ORDER BY source_timestamp DESC, id DESC LIMIT " + addArg(limit)

	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	events := make([]telemetry.Event, 0, limit)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan event payload: %w", err)
		}

		var event telemetry.Event
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("decode stored event %d: %w", len(events), err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}
