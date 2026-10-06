CREATE TABLE IF NOT EXISTS hunt_run_events (
    hunt_run_id BIGINT NOT NULL REFERENCES hunt_runs(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    PRIMARY KEY (hunt_run_id, event_id)
);

CREATE INDEX IF NOT EXISTS idx_hunt_run_events_event
    ON hunt_run_events (event_id);

CREATE INDEX IF NOT EXISTS idx_investigation_findings_finding
    ON investigation_findings (finding_id);

CREATE INDEX IF NOT EXISTS idx_investigation_events_event
    ON investigation_events (event_id);
