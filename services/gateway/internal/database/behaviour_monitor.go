package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/richardwaters9049/Sentinel/services/gateway/internal/behaviour"
)

func (d *Database) BehaviourMonitor(ctx context.Context, q BehaviourQuery, now time.Time) (behaviour.Monitor, error) {
	if err := q.Validate(); err != nil {
		return behaviour.Monitor{}, err
	}
	if d == nil || d.pool == nil {
		return behaviour.Monitor{}, fmt.Errorf("database is not initialised")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	// A repeatable snapshot prevents concurrent ingestion changing the model boundary mid-read.
	tx, err := d.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return behaviour.Monitor{}, fmt.Errorf("begin monitor snapshot: %w", err)
	}
	defer tx.Rollback(ctx)
	var version string
	err = tx.QueryRow(ctx, `
 SELECT model_version FROM behavioural_scores
 WHERE ($1='' OR entity_type=$1) AND ($2='' OR entity_id=$2)
   AND scored_at < $3
 ORDER BY scored_at DESC, event_id LIMIT 1
`, q.EntityType, q.EntityID, now).Scan(&version)
	if err != nil && err != pgx.ErrNoRows {
		return behaviour.Monitor{}, fmt.Errorf("query monitor model: %w", err)
	}
	windows := make([][]behaviour.Score, 2)
	for i := 0; i < 2; i++ {
		end := now.Add(-time.Duration(i) * 24 * time.Hour)
		start := end.Add(-24 * time.Hour)
		rows, err := tx.Query(ctx, `
 SELECT anomaly_score, baseline_context, scored_at FROM behavioural_scores
 WHERE model_version=$1 AND ($2='' OR entity_type=$2) AND ($3='' OR entity_id=$3)
   AND scored_at >= $4 AND scored_at < $5
 ORDER BY scored_at DESC, event_id LIMIT $6
`, version, q.EntityType, q.EntityID, start, end, behaviour.MonitorSampleLimit+1)
		if err != nil {
			return behaviour.Monitor{}, fmt.Errorf("query monitor window: %w", err)
		}
		for rows.Next() {
			s := behaviour.Score{ModelVersion: version}
			var baseline []byte
			if err := rows.Scan(&s.AnomalyScore, &baseline, &s.ScoredAt); err != nil {
				rows.Close()
				return behaviour.Monitor{}, err
			}
			if err := json.Unmarshal(baseline, &s.Baseline); err != nil {
				rows.Close()
				return behaviour.Monitor{}, fmt.Errorf("decode monitor baseline: %w", err)
			}
			windows[i] = append(windows[i], s)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return behaviour.Monitor{}, err
		}
	}
	m := behaviour.MonitorScores(now, version, windows[0], windows[1])
	var recall, fpr float64
	var evaluated time.Time
	var threshold int
	var dataset string
	err = tx.QueryRow(ctx, `
 SELECT r.recall, r.false_positive_rate, r.created_at, r.threshold, r.dataset_name
 FROM behaviour_evaluation_runs r
 JOIN behaviour_settings s ON s.singleton=TRUE AND s.anomaly_threshold=r.threshold
 WHERE r.model_version=$1 AND r.normal_count>0 AND r.anomaly_count>0
 ORDER BY r.created_at DESC, r.id DESC LIMIT 1
`, version).Scan(&recall, &fpr, &evaluated, &threshold, &dataset)
	if err != nil && err != pgx.ErrNoRows {
		return behaviour.Monitor{}, fmt.Errorf("query monitor evaluation: %w", err)
	}
	if err == nil {
		status := "within_guardrails"
		if recall < 0.90 || fpr > 0.15 {
			status = "outside_guardrails"
		}
		if now.Sub(evaluated) > 7*24*time.Hour {
			status = "stale_evaluation"
		}
		m.Evaluation = behaviour.HealthSignal{Status: status, Explanation: fmt.Sprintf("Synthetic dataset %s at threshold %d: recall %.3f, false-positive rate %.3f; evaluated %s. Guardrails: recall >= 0.90, false-positive rate <= 0.15; stale after 7 days. This is development evidence only.", dataset, threshold, recall, fpr, evaluated.UTC().Format(time.RFC3339))}
	}
	if err := tx.Commit(ctx); err != nil {
		return behaviour.Monitor{}, err
	}
	return m, nil
}
