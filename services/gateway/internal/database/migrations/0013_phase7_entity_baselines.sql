ALTER TABLE behavioural_scores
    ADD COLUMN IF NOT EXISTS entity_type TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS baseline_context JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE behavioural_scores
    DROP CONSTRAINT IF EXISTS behavioural_scores_entity_type_check;

ALTER TABLE behavioural_scores
    ADD CONSTRAINT behavioural_scores_entity_type_check
    CHECK (entity_type IN ('identity', 'asset', 'collector', 'unknown'));

CREATE INDEX IF NOT EXISTS idx_behavioural_scores_entity_type_entity
    ON behavioural_scores (entity_type, entity_id, scored_at DESC);
