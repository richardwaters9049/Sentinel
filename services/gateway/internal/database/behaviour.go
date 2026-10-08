package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/behaviour"
)

func (d *Database) GetBehaviourBaselineContext(
	ctx context.Context,
	entityID string,
	entityType string,
	observedAt time.Time,
) (behaviour.BaselineContext, error) {
	if d == nil || d.pool == nil {
		return behaviour.BaselineContext{}, fmt.Errorf("database is not initialised")
	}

	entityID = strings.TrimSpace(entityID)
	entityType = strings.TrimSpace(entityType)
	if entityID == "" {
		return behaviour.BaselineContext{}, fmt.Errorf("behaviour entity id is required")
	}

	var entityPredicate string
	switch entityType {
	case "identity":
		entityPredicate = "identity_id = $1"
	case "asset":
		entityPredicate = "asset_id = $1"
	case "collector":
		entityPredicate = "payload #>> '{source,collector}' = $1"
	default:
		return behaviour.BaselineContext{}, fmt.Errorf("unsupported behaviour entity type %q", entityType)
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) FILTER (
				WHERE source_timestamp >= $2::timestamptz - INTERVAL '60 minutes'
			),
			COUNT(*) FILTER (
				WHERE source_timestamp >= $2::timestamptz - INTERVAL '24 hours'
			),
			COUNT(DISTINCT payload #>> '{network,destination_ip}') FILTER (
				WHERE source_timestamp >= $2::timestamptz - INTERVAL '24 hours'
				  AND COALESCE(payload #>> '{network,destination_ip}', '') <> ''
			),
			COUNT(DISTINCT payload #>> '{network,destination_port}') FILTER (
				WHERE source_timestamp >= $2::timestamptz - INTERVAL '24 hours'
				  AND COALESCE(payload #>> '{network,destination_port}', '') <> ''
			),
			COUNT(*) FILTER (
				WHERE source_timestamp >= $2::timestamptz - INTERVAL '60 minutes'
				  AND category = 'authentication'
				  AND outcome = 'failure'
			),
			COUNT(*) FILTER (
				WHERE source_timestamp >= $2::timestamptz - INTERVAL '24 hours'
				  AND (
					COALESCE(payload #>> '{asset,zone}', '') = 'ot'
					OR COALESCE(payload #>> '{network,destination_zone}', '') = 'ot'
				  )
			)
		FROM events
		WHERE %s
		  AND source_timestamp < $2::timestamptz
		  AND source_timestamp >= $2::timestamptz - INTERVAL '24 hours'
	`, entityPredicate)

	var baseline behaviour.BaselineContext
	if err := d.pool.QueryRow(
		ctx,
		query,
		entityID,
		observedAt.UTC(),
	).Scan(
		&baseline.PriorEvents60m,
		&baseline.PriorEvents24h,
		&baseline.UniqueDestinationIPs24h,
		&baseline.UniqueDestinationPorts24h,
		&baseline.AuthFailures60m,
		&baseline.OTEvents24h,
	); err != nil {
		return behaviour.BaselineContext{}, fmt.Errorf("query behavioural baseline context: %w", err)
	}

	baseline.EventRate60m = float64(baseline.PriorEvents60m)
	if baseline.PriorEvents24h > 0 {
		baseline.DestinationDiversity24h =
			float64(baseline.UniqueDestinationIPs24h) / float64(baseline.PriorEvents24h)
		baseline.OTActivityRate24h =
			float64(baseline.OTEvents24h) / float64(baseline.PriorEvents24h)
	}
	if baseline.PriorEvents60m > 0 {
		baseline.AuthFailureRate60m =
			float64(baseline.AuthFailures60m) / float64(baseline.PriorEvents60m)
	}

	return baseline, nil
}

// BehaviourQuery scopes scores and metrics to the same entity namespace.
type BehaviourQuery struct {
	Limit      int
	EntityID   string
	EntityType string
}

func (q BehaviourQuery) Validate() error {
	if (q.EntityID == "") != (q.EntityType == "") {
		return fmt.Errorf("entity_id and entity_type must be supplied together")
	}
	if len(q.EntityID) > 256 {
		return fmt.Errorf("entity_id must be at most 256 bytes")
	}
	switch q.EntityType {
	case "", "identity", "asset", "collector", "unknown":
		return nil
	default:
		return fmt.Errorf("unsupported entity_type")
	}
}

type BehaviourMetrics struct {
	TotalScores        int64      `json:"total_scores"`
	AnomalousScores    int64      `json:"anomalous_scores"`
	HighSeverityScores int64      `json:"high_severity_scores"`
	AverageScore       float64    `json:"average_score"`
	LastScoredAt       *time.Time `json:"last_scored_at,omitempty"`
}

func (d *Database) SaveBehaviourScore(
	ctx context.Context,
	score behaviour.Score,
) error {
	if d == nil || d.pool == nil {
		return fmt.Errorf("database is not initialised")
	}

	explanations, err := json.Marshal(score.Explanations)
	if err != nil {
		return fmt.Errorf("marshal behavioural explanations: %w", err)
	}
	baseline, err := json.Marshal(score.Baseline)
	if err != nil {
		return fmt.Errorf("marshal behavioural baseline context: %w", err)
	}

	_, err = d.pool.Exec(ctx, `
		INSERT INTO behavioural_scores (
			event_id,
			entity_id,
			entity_type,
			baseline_context,
			model_version,
			model_kind,
			anomaly_score,
			severity,
			anomalous,
			threshold,
			explanations,
			scored_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (event_id) DO UPDATE SET
			entity_id = EXCLUDED.entity_id,
			entity_type = EXCLUDED.entity_type,
			baseline_context = EXCLUDED.baseline_context,
			model_version = EXCLUDED.model_version,
			model_kind = EXCLUDED.model_kind,
			anomaly_score = EXCLUDED.anomaly_score,
			severity = EXCLUDED.severity,
			anomalous = EXCLUDED.anomalous,
			threshold = EXCLUDED.threshold,
			explanations = EXCLUDED.explanations,
			scored_at = EXCLUDED.scored_at
	`,
		score.EventID,
		score.EntityID,
		score.EntityType,
		baseline,
		score.ModelVersion,
		score.ModelKind,
		score.AnomalyScore,
		score.Severity,
		score.Anomalous,
		score.Threshold,
		explanations,
		score.ScoredAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("save behavioural score: %w", err)
	}
	return nil
}

func (d *Database) GetEventBehaviourScore(
	ctx context.Context,
	eventID string,
) (behaviour.Score, error) {
	if d == nil || d.pool == nil {
		return behaviour.Score{}, fmt.Errorf("database is not initialised")
	}

	var (
		score        behaviour.Score
		baseline     []byte
		explanations []byte
	)
	if err := d.pool.QueryRow(ctx, `
		SELECT
			event_id,
			entity_id,
			entity_type,
			baseline_context,
			model_version,
			model_kind,
			anomaly_score,
			severity,
			anomalous,
			threshold,
			explanations,
			scored_at
		FROM behavioural_scores
		WHERE event_id = $1
	`, eventID).Scan(
		&score.EventID,
		&score.EntityID,
		&score.EntityType,
		&baseline,
		&score.ModelVersion,
		&score.ModelKind,
		&score.AnomalyScore,
		&score.Severity,
		&score.Anomalous,
		&score.Threshold,
		&explanations,
		&score.ScoredAt,
	); err != nil {
		return behaviour.Score{}, fmt.Errorf("query behavioural score: %w", err)
	}
	if err := json.Unmarshal(baseline, &score.Baseline); err != nil {
		return behaviour.Score{}, fmt.Errorf("decode behavioural baseline context: %w", err)
	}
	if err := json.Unmarshal(explanations, &score.Explanations); err != nil {
		return behaviour.Score{}, fmt.Errorf("decode behavioural explanations: %w", err)
	}
	return score, nil
}

func (d *Database) ListBehaviourScores(
	ctx context.Context,
	query BehaviourQuery,
) ([]behaviour.Score, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if err := query.Validate(); err != nil {
		return nil, err
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			event_id,
			entity_id,
			entity_type,
			baseline_context,
			model_version,
			model_kind,
			anomaly_score,
			severity,
			anomalous,
			threshold,
			explanations,
			scored_at
		FROM behavioural_scores
		WHERE ($2 = '' OR entity_type = $2)
		  AND ($3 = '' OR entity_id = $3)
		ORDER BY scored_at DESC, event_id
		LIMIT $1
	`, limit, query.EntityType, query.EntityID)
	if err != nil {
		return nil, fmt.Errorf("query behavioural scores: %w", err)
	}
	defer rows.Close()

	result := make([]behaviour.Score, 0, limit)
	for rows.Next() {
		var (
			score        behaviour.Score
			baseline     []byte
			explanations []byte
		)
		if err := rows.Scan(
			&score.EventID,
			&score.EntityID,
			&score.EntityType,
			&baseline,
			&score.ModelVersion,
			&score.ModelKind,
			&score.AnomalyScore,
			&score.Severity,
			&score.Anomalous,
			&score.Threshold,
			&explanations,
			&score.ScoredAt,
		); err != nil {
			return nil, fmt.Errorf("scan behavioural score: %w", err)
		}
		if err := json.Unmarshal(baseline, &score.Baseline); err != nil {
			return nil, fmt.Errorf("decode behavioural baseline context: %w", err)
		}
		if err := json.Unmarshal(explanations, &score.Explanations); err != nil {
			return nil, fmt.Errorf("decode behavioural explanations: %w", err)
		}
		result = append(result, score)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate behavioural scores: %w", err)
	}
	return result, nil
}

func (d *Database) behaviourScoresForEventIDs(
	ctx context.Context,
	eventIDs []string,
) ([]behaviour.Score, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if len(eventIDs) == 0 {
		return []behaviour.Score{}, nil
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			event_id,
			entity_id,
			entity_type,
			baseline_context,
			model_version,
			model_kind,
			anomaly_score,
			severity,
			anomalous,
			threshold,
			explanations,
			scored_at
		FROM behavioural_scores
		WHERE event_id = ANY($1)
		ORDER BY anomaly_score DESC, scored_at DESC, event_id
		LIMIT 500
	`, eventIDs)
	if err != nil {
		return nil, fmt.Errorf("query behavioural scores for evidence: %w", err)
	}
	defer rows.Close()

	result := make([]behaviour.Score, 0, len(eventIDs))
	for rows.Next() {
		var (
			score        behaviour.Score
			baseline     []byte
			explanations []byte
		)
		if err := rows.Scan(
			&score.EventID,
			&score.EntityID,
			&score.EntityType,
			&baseline,
			&score.ModelVersion,
			&score.ModelKind,
			&score.AnomalyScore,
			&score.Severity,
			&score.Anomalous,
			&score.Threshold,
			&explanations,
			&score.ScoredAt,
		); err != nil {
			return nil, fmt.Errorf("scan behavioural evidence score: %w", err)
		}
		if err := json.Unmarshal(baseline, &score.Baseline); err != nil {
			return nil, fmt.Errorf("decode behavioural evidence baseline: %w", err)
		}
		if err := json.Unmarshal(explanations, &score.Explanations); err != nil {
			return nil, fmt.Errorf("decode behavioural evidence explanations: %w", err)
		}
		result = append(result, score)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate behavioural evidence scores: %w", err)
	}
	return result, nil
}

func (d *Database) BehaviourMetrics(
	ctx context.Context,
	query BehaviourQuery,
) (BehaviourMetrics, error) {
	if d == nil || d.pool == nil {
		return BehaviourMetrics{}, fmt.Errorf("database is not initialised")
	}

	if err := query.Validate(); err != nil {
		return BehaviourMetrics{}, err
	}

	var metrics BehaviourMetrics
	if err := d.pool.QueryRow(ctx, `
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE anomalous = TRUE),
			COUNT(*) FILTER (WHERE severity = 'high'),
			COALESCE(AVG(anomaly_score), 0),
			MAX(scored_at)
		FROM behavioural_scores
		WHERE ($1 = '' OR entity_type = $1)
		  AND ($2 = '' OR entity_id = $2)
	`, query.EntityType, query.EntityID).Scan(
		&metrics.TotalScores,
		&metrics.AnomalousScores,
		&metrics.HighSeverityScores,
		&metrics.AverageScore,
		&metrics.LastScoredAt,
	); err != nil {
		return BehaviourMetrics{}, fmt.Errorf("query behavioural metrics: %w", err)
	}
	return metrics, nil
}
