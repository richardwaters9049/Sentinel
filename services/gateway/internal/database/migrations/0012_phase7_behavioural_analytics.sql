CREATE TABLE IF NOT EXISTS behavioural_scores (
    event_id TEXT PRIMARY KEY REFERENCES events(id) ON DELETE CASCADE,
    entity_id TEXT NOT NULL,
    model_version TEXT NOT NULL,
    model_kind TEXT NOT NULL,
    anomaly_score INTEGER NOT NULL CHECK (anomaly_score BETWEEN 0 AND 100),
    severity TEXT NOT NULL CHECK (severity IN ('low', 'medium', 'high')),
    anomalous BOOLEAN NOT NULL,
    threshold INTEGER NOT NULL CHECK (threshold BETWEEN 0 AND 100),
    explanations JSONB NOT NULL DEFAULT '[]'::jsonb,
    scored_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_behavioural_scores_entity_score
    ON behavioural_scores (entity_id, anomaly_score DESC, scored_at DESC);

CREATE INDEX IF NOT EXISTS idx_behavioural_scores_anomalous_scored
    ON behavioural_scores (anomalous, scored_at DESC);
