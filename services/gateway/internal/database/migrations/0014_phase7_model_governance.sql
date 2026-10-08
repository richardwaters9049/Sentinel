CREATE TABLE IF NOT EXISTS behaviour_models (
    model_version TEXT PRIMARY KEY,
    model_kind TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'retired')),
    feature_schema JSONB NOT NULL DEFAULT '[]'::jsonb,
    training_source TEXT NOT NULL,
    random_seed INTEGER,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO behaviour_models (
    model_version,
    model_kind,
    status,
    feature_schema,
    training_source,
    random_seed
)
VALUES (
    'sentinel-behaviour-iforest-v1',
    'IsolationForest',
    'active',
    '[
      "hour_sin",
      "hour_cos",
      "weekend",
      "destination_port",
      "cross_zone",
      "auth_failure",
      "service_account",
      "ot_activity",
      "event_rate_60m",
      "destination_diversity_24h",
      "auth_failure_rate_60m",
      "ot_activity_rate_24h"
    ]'::jsonb,
    'deterministic synthetic Northstar baseline',
    707
)
ON CONFLICT (model_version) DO NOTHING;

CREATE TABLE IF NOT EXISTS behaviour_settings (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    anomaly_threshold INTEGER NOT NULL CHECK (anomaly_threshold BETWEEN 1 AND 99),
    updated_by TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO behaviour_settings (singleton, anomaly_threshold)
VALUES (TRUE, 65)
ON CONFLICT (singleton) DO NOTHING;

CREATE TABLE IF NOT EXISTS behaviour_evaluation_runs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    model_version TEXT NOT NULL REFERENCES behaviour_models(model_version) ON DELETE RESTRICT,
    dataset_name TEXT NOT NULL,
    threshold INTEGER NOT NULL CHECK (threshold BETWEEN 1 AND 99),
    normal_count INTEGER NOT NULL CHECK (normal_count >= 0),
    anomaly_count INTEGER NOT NULL CHECK (anomaly_count >= 0),
    true_positive INTEGER NOT NULL CHECK (true_positive >= 0),
    false_positive INTEGER NOT NULL CHECK (false_positive >= 0),
    true_negative INTEGER NOT NULL CHECK (true_negative >= 0),
    false_negative INTEGER NOT NULL CHECK (false_negative >= 0),
    precision DOUBLE PRECISION NOT NULL CHECK (precision BETWEEN 0 AND 1),
    recall DOUBLE PRECISION NOT NULL CHECK (recall BETWEEN 0 AND 1),
    false_positive_rate DOUBLE PRECISION NOT NULL CHECK (false_positive_rate BETWEEN 0 AND 1),
    actor_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_behaviour_evaluation_runs_model_created
    ON behaviour_evaluation_runs (model_version, created_at DESC);
