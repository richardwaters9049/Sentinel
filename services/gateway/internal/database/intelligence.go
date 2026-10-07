package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/enrichment"
)

type IntelligenceIndicatorQuery struct {
	Limit         int
	IndicatorType string
	Value         string
}

type EventEnrichmentRecord struct {
	ID                  int64                  `json:"id"`
	EventID             string                 `json:"event_id"`
	IndicatorID         string                 `json:"indicator_id"`
	SourceID            string                 `json:"source_id"`
	SourceName          string                 `json:"source_name"`
	SourceType          string                 `json:"source_type"`
	IndicatorType       string                 `json:"indicator_type"`
	IndicatorValue      string                 `json:"indicator_value"`
	EventField          string                 `json:"event_field"`
	ObservedValue       string                 `json:"observed_value"`
	SourceConfidence    int                    `json:"source_confidence"`
	IndicatorConfidence int                    `json:"indicator_confidence"`
	EffectiveConfidence int                    `json:"effective_confidence"`
	Tags                []string               `json:"tags"`
	Context             map[string]interface{} `json:"context"`
	Provenance          map[string]interface{} `json:"provenance"`
	MatchedAt           time.Time              `json:"matched_at"`
}

type IntelligenceMetrics struct {
	ActiveSources      int `json:"active_sources"`
	ActiveIndicators   int `json:"active_indicators"`
	EnrichedEvents     int `json:"enriched_events"`
	TotalMatches       int `json:"total_matches"`
	HighConfidenceHits int `json:"high_confidence_hits"`
}

func (d *Database) LookupIndicators(
	ctx context.Context,
	indicatorType string,
	value string,
	observedAt time.Time,
) ([]enrichment.Indicator, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

	indicatorType = strings.ToLower(strings.TrimSpace(indicatorType))
	value = strings.ToLower(strings.TrimSpace(value))
	if indicatorType == "" || value == "" {
		return nil, nil
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			i.id,
			i.source_id,
			s.name,
			s.source_type,
			i.indicator_type,
			i.value,
			i.normalized_value,
			s.default_confidence,
			i.confidence,
			i.valid_from,
			i.valid_until,
			i.tags,
			i.context,
			s.provenance
		FROM threat_indicators i
		JOIN threat_intel_sources s ON s.id = i.source_id
		WHERE s.active = TRUE
		  AND i.indicator_type = $1
		  AND i.normalized_value = $2
		  AND i.valid_from <= $3
		  AND (i.valid_until IS NULL OR i.valid_until >= $3)
		ORDER BY i.confidence DESC, i.id
		LIMIT 25
	`, indicatorType, value, observedAt.UTC())
	if err != nil {
		return nil, fmt.Errorf("lookup threat indicators: %w", err)
	}
	defer rows.Close()

	indicators := make([]enrichment.Indicator, 0, 4)
	for rows.Next() {
		var (
			record         enrichment.Indicator
			tagsJSON       []byte
			contextJSON    []byte
			provenanceJSON []byte
		)
		if err := rows.Scan(
			&record.ID,
			&record.SourceID,
			&record.SourceName,
			&record.SourceType,
			&record.IndicatorType,
			&record.Value,
			&record.NormalizedValue,
			&record.SourceConfidence,
			&record.Confidence,
			&record.ValidFrom,
			&record.ValidUntil,
			&tagsJSON,
			&contextJSON,
			&provenanceJSON,
		); err != nil {
			return nil, fmt.Errorf("scan threat indicator: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &record.Tags); err != nil {
			return nil, fmt.Errorf("decode threat indicator tags: %w", err)
		}
		if err := json.Unmarshal(contextJSON, &record.Context); err != nil {
			return nil, fmt.Errorf("decode threat indicator context: %w", err)
		}
		if err := json.Unmarshal(provenanceJSON, &record.Provenance); err != nil {
			return nil, fmt.Errorf("decode threat indicator provenance: %w", err)
		}

		indicators = append(indicators, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate threat indicators: %w", err)
	}

	return indicators, nil
}

func (d *Database) SaveEventEnrichments(
	ctx context.Context,
	eventID string,
	matches []enrichment.Match,
) error {
	if d == nil || d.pool == nil {
		return fmt.Errorf("database is not initialised")
	}
	if strings.TrimSpace(eventID) == "" {
		return fmt.Errorf("event id is required")
	}
	if len(matches) == 0 {
		return nil
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin event enrichment transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, match := range matches {
		provenance, err := json.Marshal(match.Provenance)
		if err != nil {
			return fmt.Errorf("marshal enrichment provenance: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO event_enrichments (
				event_id,
				indicator_id,
				source_id,
				event_field,
				observed_value,
				source_confidence,
				indicator_confidence,
				effective_confidence,
				provenance,
				matched_at
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (event_id, indicator_id, event_field) DO NOTHING
		`,
			eventID,
			match.IndicatorID,
			match.SourceID,
			match.EventField,
			match.ObservedValue,
			match.SourceConfidence,
			match.IndicatorConfidence,
			match.EffectiveConfidence,
			provenance,
			match.MatchedAt.UTC(),
		); err != nil {
			return fmt.Errorf("insert event enrichment: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit event enrichment transaction: %w", err)
	}
	return nil
}

func (d *Database) ListIntelligenceIndicators(
	ctx context.Context,
	query IntelligenceIndicatorQuery,
) ([]enrichment.Indicator, error) {
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
	conditions := []string{"s.active = TRUE"}
	addArg := func(value interface{}) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if value := strings.ToLower(strings.TrimSpace(query.IndicatorType)); value != "" {
		conditions = append(conditions, "i.indicator_type = "+addArg(value))
	}
	if value := strings.ToLower(strings.TrimSpace(query.Value)); value != "" {
		conditions = append(conditions, "i.normalized_value = "+addArg(value))
	}

	sql := `
		SELECT
			i.id,
			i.source_id,
			s.name,
			s.source_type,
			i.indicator_type,
			i.value,
			i.normalized_value,
			s.default_confidence,
			i.confidence,
			i.valid_from,
			i.valid_until,
			i.tags,
			i.context,
			s.provenance
		FROM threat_indicators i
		JOIN threat_intel_sources s ON s.id = i.source_id
		WHERE ` + strings.Join(conditions, " AND ") +
		" ORDER BY i.confidence DESC, i.id LIMIT " + addArg(limit)

	rows, err := d.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query threat indicators: %w", err)
	}
	defer rows.Close()

	result := make([]enrichment.Indicator, 0, limit)
	for rows.Next() {
		var (
			record         enrichment.Indicator
			tagsJSON       []byte
			contextJSON    []byte
			provenanceJSON []byte
		)
		if err := rows.Scan(
			&record.ID,
			&record.SourceID,
			&record.SourceName,
			&record.SourceType,
			&record.IndicatorType,
			&record.Value,
			&record.NormalizedValue,
			&record.SourceConfidence,
			&record.Confidence,
			&record.ValidFrom,
			&record.ValidUntil,
			&tagsJSON,
			&contextJSON,
			&provenanceJSON,
		); err != nil {
			return nil, fmt.Errorf("scan threat indicator list: %w", err)
		}
		if err := json.Unmarshal(tagsJSON, &record.Tags); err != nil {
			return nil, fmt.Errorf("decode threat indicator tags: %w", err)
		}
		if err := json.Unmarshal(contextJSON, &record.Context); err != nil {
			return nil, fmt.Errorf("decode threat indicator context: %w", err)
		}
		if err := json.Unmarshal(provenanceJSON, &record.Provenance); err != nil {
			return nil, fmt.Errorf("decode threat indicator provenance: %w", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate threat indicator list: %w", err)
	}

	return result, nil
}

func (d *Database) GetEventEnrichments(
	ctx context.Context,
	eventID string,
) ([]EventEnrichmentRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			e.id,
			e.event_id,
			e.indicator_id,
			e.source_id,
			s.name,
			s.source_type,
			i.indicator_type,
			i.value,
			e.event_field,
			e.observed_value,
			e.source_confidence,
			e.indicator_confidence,
			e.effective_confidence,
			i.tags,
			i.context,
			e.provenance,
			e.matched_at
		FROM event_enrichments e
		JOIN threat_indicators i ON i.id = e.indicator_id
		JOIN threat_intel_sources s ON s.id = e.source_id
		WHERE e.event_id = $1
		ORDER BY e.effective_confidence DESC, e.id
	`, strings.TrimSpace(eventID))
	if err != nil {
		return nil, fmt.Errorf("query event enrichments: %w", err)
	}
	defer rows.Close()

	result := make([]EventEnrichmentRecord, 0, 4)
	for rows.Next() {
		var (
			record         EventEnrichmentRecord
			tagsJSON       []byte
			contextJSON    []byte
			provenanceJSON []byte
		)
		if err := rows.Scan(
			&record.ID,
			&record.EventID,
			&record.IndicatorID,
			&record.SourceID,
			&record.SourceName,
			&record.SourceType,
			&record.IndicatorType,
			&record.IndicatorValue,
			&record.EventField,
			&record.ObservedValue,
			&record.SourceConfidence,
			&record.IndicatorConfidence,
			&record.EffectiveConfidence,
			&tagsJSON,
			&contextJSON,
			&provenanceJSON,
			&record.MatchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan event enrichment: %w", err)
		}
		if err := json.Unmarshal(tagsJSON, &record.Tags); err != nil {
			return nil, fmt.Errorf("decode event enrichment tags: %w", err)
		}
		if err := json.Unmarshal(contextJSON, &record.Context); err != nil {
			return nil, fmt.Errorf("decode event enrichment context: %w", err)
		}
		if err := json.Unmarshal(provenanceJSON, &record.Provenance); err != nil {
			return nil, fmt.Errorf("decode event enrichment provenance: %w", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate event enrichments: %w", err)
	}

	return result, nil
}

func (d *Database) ListRecentEventEnrichments(
	ctx context.Context,
	limit int,
) ([]EventEnrichmentRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			e.id,
			e.event_id,
			e.indicator_id,
			e.source_id,
			s.name,
			s.source_type,
			i.indicator_type,
			i.value,
			e.event_field,
			e.observed_value,
			e.source_confidence,
			e.indicator_confidence,
			e.effective_confidence,
			i.tags,
			i.context,
			e.provenance,
			e.matched_at
		FROM event_enrichments e
		JOIN threat_indicators i ON i.id = e.indicator_id
		JOIN threat_intel_sources s ON s.id = e.source_id
		ORDER BY e.matched_at DESC, e.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent event enrichments: %w", err)
	}
	defer rows.Close()

	result := make([]EventEnrichmentRecord, 0, limit)
	for rows.Next() {
		record, err := scanEventEnrichment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent event enrichments: %w", err)
	}

	return result, nil
}

type enrichmentScanner interface {
	Scan(...interface{}) error
}

func scanEventEnrichment(scanner enrichmentScanner) (EventEnrichmentRecord, error) {
	var (
		record         EventEnrichmentRecord
		tagsJSON       []byte
		contextJSON    []byte
		provenanceJSON []byte
	)

	if err := scanner.Scan(
		&record.ID,
		&record.EventID,
		&record.IndicatorID,
		&record.SourceID,
		&record.SourceName,
		&record.SourceType,
		&record.IndicatorType,
		&record.IndicatorValue,
		&record.EventField,
		&record.ObservedValue,
		&record.SourceConfidence,
		&record.IndicatorConfidence,
		&record.EffectiveConfidence,
		&tagsJSON,
		&contextJSON,
		&provenanceJSON,
		&record.MatchedAt,
	); err != nil {
		return EventEnrichmentRecord{}, fmt.Errorf("scan event enrichment: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &record.Tags); err != nil {
		return EventEnrichmentRecord{}, fmt.Errorf("decode event enrichment tags: %w", err)
	}
	if err := json.Unmarshal(contextJSON, &record.Context); err != nil {
		return EventEnrichmentRecord{}, fmt.Errorf("decode event enrichment context: %w", err)
	}
	if err := json.Unmarshal(provenanceJSON, &record.Provenance); err != nil {
		return EventEnrichmentRecord{}, fmt.Errorf("decode event enrichment provenance: %w", err)
	}

	return record, nil
}

func (d *Database) IntelligenceMetrics(ctx context.Context) (IntelligenceMetrics, error) {
	if d == nil || d.pool == nil {
		return IntelligenceMetrics{}, fmt.Errorf("database is not initialised")
	}

	var metrics IntelligenceMetrics
	if err := d.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM threat_intel_sources WHERE active = TRUE),
			(SELECT COUNT(*)
			 FROM threat_indicators i
			 JOIN threat_intel_sources s ON s.id = i.source_id
			 WHERE s.active = TRUE
			   AND i.valid_from <= NOW()
			   AND (i.valid_until IS NULL OR i.valid_until >= NOW())),
			(SELECT COUNT(DISTINCT event_id) FROM event_enrichments),
			(SELECT COUNT(*) FROM event_enrichments),
			(SELECT COUNT(*) FROM event_enrichments WHERE effective_confidence >= 80)
	`).Scan(
		&metrics.ActiveSources,
		&metrics.ActiveIndicators,
		&metrics.EnrichedEvents,
		&metrics.TotalMatches,
		&metrics.HighConfidenceHits,
	); err != nil {
		return IntelligenceMetrics{}, fmt.Errorf("query intelligence metrics: %w", err)
	}

	return metrics, nil
}
