CREATE INDEX IF NOT EXISTS idx_audit_events_resource
    ON audit_events (resource_type, resource_id, occurred_at, id);

CREATE INDEX IF NOT EXISTS idx_finding_events_event_id
    ON finding_events (event_id);

CREATE INDEX IF NOT EXISTS idx_findings_detection_status
    ON findings (detection_id, status, last_observed_at DESC);
