package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/behaviour"
)

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

	_, err = d.pool.Exec(ctx, `
		INSERT INTO behavioural_scores (
			event_id,
			entity_id,
			model_version,
			model_kind,
			anomaly_score,
			severity,
			anomalous,
			threshold,
			explanations,
			scored_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (event_id) DO UPDATE SET
			entity_id = EXCLUDED.entity_id,
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
		explanations []byte
	)
	if err := d.pool.QueryRow(ctx, `
		SELECT
			event_id,
			entity_id,
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
	if err := json.Unmarshal(explanations, &score.Explanations); err != nil {
		return behaviour.Score{}, fmt.Errorf("decode behavioural explanations: %w", err)
	}
	return score, nil
}

func (d *Database) ListBehaviourScores(
	ctx context.Context,
	limit int,
) ([]behaviour.Score, error) {
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
			event_id,
			entity_id,
			model_version,
			model_kind,
			anomaly_score,
			severity,
			anomalous,
			threshold,
			explanations,
			scored_at
		FROM behavioural_scores
		ORDER BY scored_at DESC, event_id
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query behavioural scores: %w", err)
	}
	defer rows.Close()

	result := make([]behaviour.Score, 0, limit)
	for rows.Next() {
		var (
			score        behaviour.Score
			explanations []byte
		)
		if err := rows.Scan(
			&score.EventID,
			&score.EntityID,
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

func (d *Database) BehaviourMetrics(
	ctx context.Context,
) (BehaviourMetrics, error) {
	if d == nil || d.pool == nil {
		return BehaviourMetrics{}, fmt.Errorf("database is not initialised")
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
	`).Scan(
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
