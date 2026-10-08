package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration data and migrations are confined to a disposable schema.
func TestBehaviourMonitorIntegration(t *testing.T) {
	url := os.Getenv("SENTINEL_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("SENTINEL_TEST_DATABASE_URL is required for local PostgreSQL integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("monitor_test_%d", time.Now().UnixNano())
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Errorf("clean monitor schema: %v", err)
		}
	}()
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	d := &Database{pool: pool}
	if err = d.ApplyMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	if err = d.ApplyMigrations(ctx); err != nil {
		t.Fatal("migration rerun:", err)
	}
	now := time.Now().UTC()
	version := "sentinel-behaviour-iforest-v1"
	// Distinct namespaces share the same ID; legacy and new model records coexist.
	for _, kind := range []string{"identity", "asset", "collector", "unknown"} {
		for window := 0; window < 2; window++ {
			at := now.Add(-time.Minute - time.Duration(window)*24*time.Hour)
			for i := 0; i < 30; i++ {
				id := fmt.Sprintf("%s-%d-%d", kind, window, i)
				if _, err = pool.Exec(ctx, `INSERT INTO events (id,schema_version,source_timestamp,source_type,category,action,outcome,payload) VALUES ($1,'1.0',$2,'monitor-test','network','connect','success','{}')`, id, at); err != nil {
					t.Fatal(err)
				}
				score := 40
				if kind == "identity" && window == 0 {
					score = 100
				}
				threshold := 65
				if window == 0 {
					threshold = 99
				}
				if _, err = pool.Exec(ctx, `INSERT INTO behavioural_scores(event_id,entity_id,entity_type,baseline_context,model_version,model_kind,anomaly_score,severity,anomalous,threshold,scored_at) VALUES($1,'shared',$2,'{"prior_events_24h":6}',$3,'IsolationForest',$4,'low',false,$5,$6)`, id, kind, version, score, threshold, at); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, kind := range []string{"identity", "asset", "collector", "unknown"} {
		m, err := d.BehaviourMonitor(ctx, BehaviourQuery{EntityType: kind, EntityID: "shared"}, now)
		if err != nil {
			t.Fatal(err)
		}
		expected := "stable"
		if kind == "identity" {
			expected = "distribution_change"
		}
		if m.Current.Samples != 30 || m.Reference.Samples != 30 || m.Distribution.Status != expected {
			t.Fatalf("namespace %s: %+v", kind, m)
		}
	}
	empty, err := d.BehaviourMonitor(ctx, BehaviourQuery{EntityType: "asset", EntityID: "absent"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if empty.ModelVersion != "" || empty.Current.Samples != 0 {
		t.Fatal("empty scope contaminated")
	}
	// Evaluation must match the monitored model and current threshold, not list order.
	if _, err = pool.Exec(ctx, `INSERT INTO behaviour_evaluation_runs(model_version,dataset_name,threshold,normal_count,anomaly_count,true_positive,false_positive,true_negative,false_negative,precision,recall,false_positive_rate) VALUES($1,'synthetic-test',65,100,30,30,0,100,0,1,1,0)`, version); err != nil {
		t.Fatal(err)
	}
	m, err := d.BehaviourMonitor(ctx, BehaviourQuery{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if m.Evaluation.Status != "within_guardrails" {
		t.Fatalf("evaluation: %+v", m.Evaluation)
	}

	if _, err = pool.Exec(ctx, `UPDATE behaviour_evaluation_runs SET recall=0.5`); err != nil {
		t.Fatal(err)
	}
	m, err = d.BehaviourMonitor(ctx, BehaviourQuery{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if m.Evaluation.Status != "outside_guardrails" {
		t.Fatal("poor quality accepted")
	}
	if _, err = pool.Exec(ctx, `UPDATE behaviour_evaluation_runs SET created_at=$1`, now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	m, err = d.BehaviourMonitor(ctx, BehaviourQuery{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if m.Evaluation.Status != "stale_evaluation" {
		t.Fatal("stale evaluation accepted")
	}
	if _, err = pool.Exec(ctx, `UPDATE behaviour_settings SET anomaly_threshold=71`); err != nil {
		t.Fatal(err)
	}
	m, err = d.BehaviourMonitor(ctx, BehaviourQuery{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if m.Evaluation.Status != "not_evaluated" {
		t.Fatal("threshold mismatch accepted")
	}
	if _, err = pool.Exec(ctx, `UPDATE behavioural_scores SET model_version='v2' WHERE event_id='identity-0-0'`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE behavioural_scores SET scored_at=$1 WHERE event_id='identity-0-0'`, now.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	m, err = d.BehaviourMonitor(ctx, BehaviourQuery{EntityType: "identity", EntityID: "shared"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if m.ModelVersion != "v2" || m.Current.Samples != 1 || m.Reference.Samples != 0 || m.Distribution.Status != "insufficient_samples" {
		t.Fatalf("mixed model versions: %+v", m)
	}
}
