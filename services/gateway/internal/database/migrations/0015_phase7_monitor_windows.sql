-- Bounded recent windows and entity/model isolation for behavioural monitoring.
CREATE INDEX IF NOT EXISTS idx_behaviour_monitor_model_time
 ON behavioural_scores (model_version, scored_at DESC, event_id);
CREATE INDEX IF NOT EXISTS idx_behaviour_monitor_entity_time
 ON behavioural_scores (entity_type, entity_id, scored_at DESC, event_id);
CREATE INDEX IF NOT EXISTS idx_behaviour_monitor_time
 ON behavioural_scores (scored_at DESC, event_id);
