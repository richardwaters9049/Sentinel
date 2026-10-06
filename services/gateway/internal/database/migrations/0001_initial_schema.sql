CREATE TABLE IF NOT EXISTS assets (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    zone TEXT NOT NULL,
    criticality TEXT NOT NULL DEFAULT 'medium',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS identities (
    id TEXT PRIMARY KEY,
    identity_type TEXT NOT NULL,
    name TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    schema_version TEXT NOT NULL,
    source_timestamp TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source_type TEXT NOT NULL,
    category TEXT NOT NULL,
    action TEXT NOT NULL,
    outcome TEXT,
    asset_id TEXT REFERENCES assets(id) ON DELETE SET NULL,
    identity_id TEXT REFERENCES identities(id) ON DELETE SET NULL,
    payload JSONB NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_source_timestamp
    ON events (source_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_events_asset_time
    ON events (asset_id, source_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_events_identity_time
    ON events (identity_id, source_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_events_category_action
    ON events (category, action);

CREATE TABLE IF NOT EXISTS detections (
    id TEXT PRIMARY KEY,
    version INTEGER NOT NULL CHECK (version > 0),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('informational', 'low', 'medium', 'high', 'critical')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    definition JSONB NOT NULL,
    mitre JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, version)
);

CREATE TABLE IF NOT EXISTS findings (
    id TEXT PRIMARY KEY,
    detection_id TEXT NOT NULL,
    detection_version INTEGER NOT NULL,
    title TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('informational', 'low', 'medium', 'high', 'critical')),
    confidence SMALLINT NOT NULL CHECK (confidence BETWEEN 0 AND 100),
    status TEXT NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'triaged', 'investigating', 'false_positive', 'benign_expected', 'duplicate', 'confirmed', 'contained', 'closed')),
    first_observed_at TIMESTAMPTZ NOT NULL,
    last_observed_at TIMESTAMPTZ NOT NULL,
    evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (detection_id, detection_version)
        REFERENCES detections(id, version)
);

CREATE INDEX IF NOT EXISTS idx_findings_status_severity
    ON findings (status, severity);

CREATE INDEX IF NOT EXISTS idx_findings_last_observed
    ON findings (last_observed_at DESC);

CREATE TABLE IF NOT EXISTS finding_events (
    finding_id TEXT NOT NULL REFERENCES findings(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (finding_id, event_id)
);

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_id TEXT,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    request_id TEXT,
    details JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_audit_events_occurred_at
    ON audit_events (occurred_at DESC);
