package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type DetectionMetrics struct {
	DetectionID        string     `json:"detection_id"`
	HitCount           int64      `json:"hit_count"`
	OpenCount          int64      `json:"open_count"`
	ConfirmedCount     int64      `json:"confirmed_count"`
	FalsePositiveCount int64      `json:"false_positive_count"`
	ClosedCount        int64      `json:"closed_count"`
	FalsePositiveRate  float64    `json:"false_positive_rate"`
	LastTriggeredAt    *time.Time `json:"last_triggered_at,omitempty"`
}

func (d *Database) DetectionMetrics(ctx context.Context) ([]DetectionMetrics, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			d.id,
			COUNT(f.id) AS hit_count,
			COUNT(f.id) FILTER (
				WHERE f.status IN ('new', 'triaged', 'investigating', 'confirmed', 'contained')
			) AS open_count,
			COUNT(f.id) FILTER (WHERE f.status = 'confirmed') AS confirmed_count,
			COUNT(f.id) FILTER (WHERE f.status = 'false_positive') AS false_positive_count,
			COUNT(f.id) FILTER (WHERE f.status = 'closed') AS closed_count,
			MAX(f.last_observed_at) AS last_triggered_at
		FROM detections d
		LEFT JOIN findings f ON f.detection_id = d.id
		GROUP BY d.id
		ORDER BY d.id
	`)
	if err != nil {
		return nil, fmt.Errorf("query detection metrics: %w", err)
	}
	defer rows.Close()

	metrics := make([]DetectionMetrics, 0)
	for rows.Next() {
		var metric DetectionMetrics
		if err := rows.Scan(
			&metric.DetectionID,
			&metric.HitCount,
			&metric.OpenCount,
			&metric.ConfirmedCount,
			&metric.FalsePositiveCount,
			&metric.ClosedCount,
			&metric.LastTriggeredAt,
		); err != nil {
			return nil, fmt.Errorf("scan detection metrics: %w", err)
		}

		if metric.HitCount > 0 {
			metric.FalsePositiveRate = float64(metric.FalsePositiveCount) / float64(metric.HitCount)
		}

		metrics = append(metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate detection metrics: %w", err)
	}

	return metrics, nil
}

type FindingEvidenceContext struct {
	FindingID      string                  `json:"finding_id"`
	LinkedEvents   []EventEvidenceRecord   `json:"linked_events"`
	ContextEvents  []EventEvidenceRecord   `json:"context_events"`
	Enrichments    []EventEnrichmentRecord `json:"enrichments"`
	ContextMinutes int                     `json:"context_minutes"`
}

func (d *Database) GetFindingEvidence(
	ctx context.Context,
	findingID string,
	contextMinutes int,
) (FindingEvidenceContext, error) {
	if d == nil || d.pool == nil {
		return FindingEvidenceContext{}, fmt.Errorf("database is not initialised")
	}
	if contextMinutes < 0 || contextMinutes > 60 {
		return FindingEvidenceContext{}, fmt.Errorf("context minutes must be between 0 and 60")
	}

	var (
		firstObserved time.Time
		lastObserved  time.Time
	)
	if err := d.pool.QueryRow(ctx, `
		SELECT first_observed_at, last_observed_at
		FROM findings
		WHERE id = $1
	`, findingID).Scan(&firstObserved, &lastObserved); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FindingEvidenceContext{}, ErrFindingNotFound
		}
		return FindingEvidenceContext{}, fmt.Errorf("query finding evidence bounds: %w", err)
	}

	linked, err := d.findingEvents(ctx, findingID)
	if err != nil {
		return FindingEvidenceContext{}, err
	}

	if contextMinutes == 0 || len(linked) == 0 {
		eventIDs := make([]string, 0, len(linked))
		for _, event := range linked {
			eventIDs = append(eventIDs, event.ID)
		}
		enrichments, err := d.eventEnrichmentsForEventIDs(ctx, eventIDs)
		if err != nil {
			return FindingEvidenceContext{}, err
		}

		return FindingEvidenceContext{
			FindingID:      findingID,
			LinkedEvents:   linked,
			ContextEvents:  []EventEvidenceRecord{},
			Enrichments:    enrichments,
			ContextMinutes: contextMinutes,
		}, nil
	}

	assetIDs := make([]string, 0, len(linked))
	identityIDs := make([]string, 0, len(linked))
	linkedIDs := make([]string, 0, len(linked))
	seenAssets := map[string]struct{}{}
	seenIdentities := map[string]struct{}{}

	for _, event := range linked {
		linkedIDs = append(linkedIDs, event.ID)
		if event.AssetID != nil {
			if _, seen := seenAssets[*event.AssetID]; !seen {
				seenAssets[*event.AssetID] = struct{}{}
				assetIDs = append(assetIDs, *event.AssetID)
			}
		}
		if event.IdentityID != nil {
			if _, seen := seenIdentities[*event.IdentityID]; !seen {
				seenIdentities[*event.IdentityID] = struct{}{}
				identityIDs = append(identityIDs, *event.IdentityID)
			}
		}
	}

	contextEvents, err := d.contextEvents(
		ctx,
		firstObserved.Add(-time.Duration(contextMinutes)*time.Minute),
		lastObserved.Add(time.Duration(contextMinutes)*time.Minute),
		assetIDs,
		identityIDs,
		linkedIDs,
	)
	if err != nil {
		return FindingEvidenceContext{}, err
	}

	enrichmentEventIDs := append([]string(nil), linkedIDs...)
	for _, event := range contextEvents {
		enrichmentEventIDs = append(enrichmentEventIDs, event.ID)
	}
	enrichments, err := d.eventEnrichmentsForEventIDs(ctx, enrichmentEventIDs)
	if err != nil {
		return FindingEvidenceContext{}, err
	}

	return FindingEvidenceContext{
		FindingID:      findingID,
		LinkedEvents:   linked,
		ContextEvents:  contextEvents,
		Enrichments:    enrichments,
		ContextMinutes: contextMinutes,
	}, nil
}

func (d *Database) contextEvents(
	ctx context.Context,
	start time.Time,
	end time.Time,
	assetIDs []string,
	identityIDs []string,
	excludedIDs []string,
) ([]EventEvidenceRecord, error) {
	if len(assetIDs) == 0 && len(identityIDs) == 0 {
		return []EventEvidenceRecord{}, nil
	}

	rows, err := d.pool.Query(ctx, `
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
		WHERE source_timestamp BETWEEN $1 AND $2
		  AND (
			(CARDINALITY($3::text[]) > 0 AND asset_id = ANY($3::text[]))
			OR
			(CARDINALITY($4::text[]) > 0 AND identity_id = ANY($4::text[]))
		  )
		  AND NOT (id = ANY($5::text[]))
		ORDER BY source_timestamp, id
		LIMIT 200
	`,
		start.UTC(),
		end.UTC(),
		assetIDs,
		identityIDs,
		excludedIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("query finding context events: %w", err)
	}
	defer rows.Close()

	records := make([]EventEvidenceRecord, 0)
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
			return nil, fmt.Errorf("scan finding context event: %w", err)
		}
		if err := json.Unmarshal(payload, &record.Payload); err != nil {
			return nil, fmt.Errorf("decode finding context event payload: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate finding context events: %w", err)
	}

	return records, nil
}
