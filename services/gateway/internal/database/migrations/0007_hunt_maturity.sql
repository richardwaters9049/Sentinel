ALTER TABLE hunts
    ADD COLUMN IF NOT EXISTS current_version INTEGER NOT NULL DEFAULT 1
        CHECK (current_version > 0);

CREATE TABLE IF NOT EXISTS hunt_versions (
    hunt_id TEXT NOT NULL REFERENCES hunts(id) ON DELETE CASCADE,
    version INTEGER NOT NULL CHECK (version > 0),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    hypothesis TEXT NOT NULL DEFAULT '',
    query JSONB NOT NULL,
    changed_by TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (hunt_id, version)
);

INSERT INTO hunt_versions (
    hunt_id, version, name, description, hypothesis, query, changed_by, changed_at
)
SELECT
    id, current_version, name, description, hypothesis, query, created_by, created_at
FROM hunts
ON CONFLICT (hunt_id, version) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_hunt_versions_changed
    ON hunt_versions (hunt_id, changed_at DESC);

CREATE INDEX IF NOT EXISTS idx_hunt_runs_completed
    ON hunt_runs (completed_at DESC);

CREATE INDEX IF NOT EXISTS idx_investigations_owner_status
    ON investigations (owner_id, status, updated_at DESC);
