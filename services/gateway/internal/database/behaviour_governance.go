package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/richardwaters9049/Sentinel/services/gateway/internal/behaviour"
)

type BehaviourModelRecord struct {
	ModelVersion   string    `json:"model_version"`
	ModelKind      string    `json:"model_kind"`
	Status         string    `json:"status"`
	FeatureSchema  []string  `json:"feature_schema"`
	TrainingSource string    `json:"training_source"`
	RandomSeed     *int      `json:"random_seed,omitempty"`
	RegisteredAt   time.Time `json:"registered_at"`
}

type BehaviourEvaluationRun struct {
	ID                int64     `json:"id"`
	ModelVersion      string    `json:"model_version"`
	DatasetName       string    `json:"dataset_name"`
	Threshold         int       `json:"threshold"`
	NormalCount       int       `json:"normal_count"`
	AnomalyCount      int       `json:"anomaly_count"`
	TruePositive      int       `json:"true_positive"`
	FalsePositive     int       `json:"false_positive"`
	TrueNegative      int       `json:"true_negative"`
	FalseNegative     int       `json:"false_negative"`
	Precision         float64   `json:"precision"`
	Recall            float64   `json:"recall"`
	FalsePositiveRate float64   `json:"false_positive_rate"`
	ActorID           string    `json:"actor_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
}

type BehaviourEvaluationInput struct {
	ModelVersion      string  `json:"model_version"`
	DatasetName       string  `json:"dataset_name"`
	Threshold         int     `json:"threshold"`
	NormalCount       int     `json:"normal_count"`
	AnomalyCount      int     `json:"anomaly_count"`
	TruePositive      int     `json:"true_positive"`
	FalsePositive     int     `json:"false_positive"`
	TrueNegative      int     `json:"true_negative"`
	FalseNegative     int     `json:"false_negative"`
	Precision         float64 `json:"precision"`
	Recall            float64 `json:"recall"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
}

func (d *Database) GetBehaviourSettings(ctx context.Context) (behaviour.Settings, error) {
	if d == nil || d.pool == nil {
		return behaviour.Settings{}, fmt.Errorf("database is not initialised")
	}

	var settings behaviour.Settings
	if err := d.pool.QueryRow(ctx, `
		SELECT
			anomaly_threshold,
			COALESCE(updated_by, ''),
			updated_at
		FROM behaviour_settings
		WHERE singleton = TRUE
	`).Scan(
		&settings.AnomalyThreshold,
		&settings.UpdatedBy,
		&settings.UpdatedAt,
	); err != nil {
		return behaviour.Settings{}, fmt.Errorf("query behaviour settings: %w", err)
	}

	return settings, nil
}

func (d *Database) UpdateBehaviourSettings(
	ctx context.Context,
	threshold int,
	actorID string,
) (behaviour.Settings, error) {
	if d == nil || d.pool == nil {
		return behaviour.Settings{}, fmt.Errorf("database is not initialised")
	}
	if threshold < 1 || threshold > 99 {
		return behaviour.Settings{}, fmt.Errorf("anomaly threshold must be between 1 and 99")
	}

	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return behaviour.Settings{}, fmt.Errorf("actor id is required")
	}

	var settings behaviour.Settings
	if err := d.pool.QueryRow(ctx, `
		UPDATE behaviour_settings
		SET
			anomaly_threshold = $1,
			updated_by = $2,
			updated_at = NOW()
		WHERE singleton = TRUE
		RETURNING anomaly_threshold, updated_by, updated_at
	`, threshold, actorID).Scan(
		&settings.AnomalyThreshold,
		&settings.UpdatedBy,
		&settings.UpdatedAt,
	); err != nil {
		return behaviour.Settings{}, fmt.Errorf("update behaviour settings: %w", err)
	}

	return settings, nil
}

func (d *Database) ListBehaviourModels(
	ctx context.Context,
) ([]BehaviourModelRecord, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			model_version,
			model_kind,
			status,
			feature_schema,
			training_source,
			random_seed,
			registered_at
		FROM behaviour_models
		ORDER BY registered_at DESC, model_version
	`)
	if err != nil {
		return nil, fmt.Errorf("query behaviour models: %w", err)
	}
	defer rows.Close()

	records := make([]BehaviourModelRecord, 0)
	for rows.Next() {
		var (
			record   BehaviourModelRecord
			features []byte
		)
		if err := rows.Scan(
			&record.ModelVersion,
			&record.ModelKind,
			&record.Status,
			&features,
			&record.TrainingSource,
			&record.RandomSeed,
			&record.RegisteredAt,
		); err != nil {
			return nil, fmt.Errorf("scan behaviour model: %w", err)
		}
		if err := json.Unmarshal(features, &record.FeatureSchema); err != nil {
			return nil, fmt.Errorf("decode behaviour model feature schema: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate behaviour models: %w", err)
	}

	return records, nil
}

func (d *Database) SaveBehaviourEvaluationRun(
	ctx context.Context,
	input BehaviourEvaluationInput,
	actorID string,
) (BehaviourEvaluationRun, error) {
	if d == nil || d.pool == nil {
		return BehaviourEvaluationRun{}, fmt.Errorf("database is not initialised")
	}

	input.ModelVersion = strings.TrimSpace(input.ModelVersion)
	input.DatasetName = strings.TrimSpace(input.DatasetName)
	actorID = strings.TrimSpace(actorID)

	if input.ModelVersion == "" {
		return BehaviourEvaluationRun{}, fmt.Errorf("model version is required")
	}
	if input.DatasetName == "" {
		return BehaviourEvaluationRun{}, fmt.Errorf("dataset name is required")
	}
	if input.Threshold < 1 || input.Threshold > 99 {
		return BehaviourEvaluationRun{}, fmt.Errorf("threshold must be between 1 and 99")
	}
	if input.NormalCount < 0 || input.AnomalyCount < 0 ||
		input.TruePositive < 0 || input.FalsePositive < 0 ||
		input.TrueNegative < 0 || input.FalseNegative < 0 {
		return BehaviourEvaluationRun{}, fmt.Errorf("evaluation counts must not be negative")
	}
	if input.TruePositive+input.FalseNegative != input.AnomalyCount {
		return BehaviourEvaluationRun{}, fmt.Errorf("anomaly confusion-matrix counts do not match anomaly_count")
	}
	if input.TrueNegative+input.FalsePositive != input.NormalCount {
		return BehaviourEvaluationRun{}, fmt.Errorf("normal confusion-matrix counts do not match normal_count")
	}
	if input.Precision < 0 || input.Precision > 1 ||
		input.Recall < 0 || input.Recall > 1 ||
		input.FalsePositiveRate < 0 || input.FalsePositiveRate > 1 {
		return BehaviourEvaluationRun{}, fmt.Errorf("evaluation rates must be between 0 and 1")
	}
	if actorID == "" {
		return BehaviourEvaluationRun{}, fmt.Errorf("actor id is required")
	}

	var run BehaviourEvaluationRun
	if err := d.pool.QueryRow(ctx, `
		INSERT INTO behaviour_evaluation_runs (
			model_version,
			dataset_name,
			threshold,
			normal_count,
			anomaly_count,
			true_positive,
			false_positive,
			true_negative,
			false_negative,
			precision,
			recall,
			false_positive_rate,
			actor_id
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING
			id,
			model_version,
			dataset_name,
			threshold,
			normal_count,
			anomaly_count,
			true_positive,
			false_positive,
			true_negative,
			false_negative,
			precision,
			recall,
			false_positive_rate,
			COALESCE(actor_id, ''),
			created_at
	`,
		input.ModelVersion,
		input.DatasetName,
		input.Threshold,
		input.NormalCount,
		input.AnomalyCount,
		input.TruePositive,
		input.FalsePositive,
		input.TrueNegative,
		input.FalseNegative,
		input.Precision,
		input.Recall,
		input.FalsePositiveRate,
		actorID,
	).Scan(
		&run.ID,
		&run.ModelVersion,
		&run.DatasetName,
		&run.Threshold,
		&run.NormalCount,
		&run.AnomalyCount,
		&run.TruePositive,
		&run.FalsePositive,
		&run.TrueNegative,
		&run.FalseNegative,
		&run.Precision,
		&run.Recall,
		&run.FalsePositiveRate,
		&run.ActorID,
		&run.CreatedAt,
	); err != nil {
		return BehaviourEvaluationRun{}, fmt.Errorf("save behaviour evaluation run: %w", err)
	}

	return run, nil
}

func (d *Database) ListBehaviourEvaluationRuns(
	ctx context.Context,
	limit int,
) ([]BehaviourEvaluationRun, error) {
	if d == nil || d.pool == nil {
		return nil, fmt.Errorf("database is not initialised")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := d.pool.Query(ctx, `
		SELECT
			id,
			model_version,
			dataset_name,
			threshold,
			normal_count,
			anomaly_count,
			true_positive,
			false_positive,
			true_negative,
			false_negative,
			precision,
			recall,
			false_positive_rate,
			COALESCE(actor_id, ''),
			created_at
		FROM behaviour_evaluation_runs
		ORDER BY created_at DESC, id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query behaviour evaluation runs: %w", err)
	}
	defer rows.Close()

	runs := make([]BehaviourEvaluationRun, 0, limit)
	for rows.Next() {
		var run BehaviourEvaluationRun
		if err := rows.Scan(
			&run.ID,
			&run.ModelVersion,
			&run.DatasetName,
			&run.Threshold,
			&run.NormalCount,
			&run.AnomalyCount,
			&run.TruePositive,
			&run.FalsePositive,
			&run.TrueNegative,
			&run.FalseNegative,
			&run.Precision,
			&run.Recall,
			&run.FalsePositiveRate,
			&run.ActorID,
			&run.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan behaviour evaluation run: %w", err)
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate behaviour evaluation runs: %w", err)
	}

	return runs, nil
}
