CREATE TABLE IF NOT EXISTS threat_intel_sources (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    default_confidence SMALLINT NOT NULL CHECK (default_confidence BETWEEN 0 AND 100),
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS threat_indicators (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL REFERENCES threat_intel_sources(id) ON DELETE RESTRICT,
    indicator_type TEXT NOT NULL CHECK (indicator_type IN ('ip', 'domain', 'sha256')),
    value TEXT NOT NULL,
    normalized_value TEXT NOT NULL,
    confidence SMALLINT NOT NULL CHECK (confidence BETWEEN 0 AND 100),
    valid_from TIMESTAMPTZ NOT NULL,
    valid_until TIMESTAMPTZ,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    context JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (source_id, indicator_type, normalized_value)
);

CREATE INDEX IF NOT EXISTS idx_threat_indicators_lookup
    ON threat_indicators (indicator_type, normalized_value);

CREATE INDEX IF NOT EXISTS idx_threat_indicators_validity
    ON threat_indicators (valid_from, valid_until);

CREATE TABLE IF NOT EXISTS event_enrichments (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    indicator_id TEXT NOT NULL REFERENCES threat_indicators(id) ON DELETE RESTRICT,
    source_id TEXT NOT NULL REFERENCES threat_intel_sources(id) ON DELETE RESTRICT,
    event_field TEXT NOT NULL,
    observed_value TEXT NOT NULL,
    source_confidence SMALLINT NOT NULL CHECK (source_confidence BETWEEN 0 AND 100),
    indicator_confidence SMALLINT NOT NULL CHECK (indicator_confidence BETWEEN 0 AND 100),
    effective_confidence SMALLINT NOT NULL CHECK (effective_confidence BETWEEN 0 AND 100),
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
    matched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (event_id, indicator_id, event_field)
);

CREATE INDEX IF NOT EXISTS idx_event_enrichments_event
    ON event_enrichments (event_id, matched_at DESC);

CREATE INDEX IF NOT EXISTS idx_event_enrichments_indicator
    ON event_enrichments (indicator_id, matched_at DESC);

INSERT INTO threat_intel_sources (
    id,
    name,
    source_type,
    description,
    default_confidence,
    provenance
)
VALUES (
    'intel-local-northstar',
    'Northstar Local Threat Intelligence',
    'local_fixture',
    'Synthetic, offline-only threat-intelligence fixtures for Sentinel development and regression testing.',
    85,
    '{
      "origin": "Sentinel repository",
      "collection_method": "curated synthetic fixture",
      "network_scope": "documentation-only addresses",
      "external_dependency": false
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    source_type = EXCLUDED.source_type,
    description = EXCLUDED.description,
    default_confidence = EXCLUDED.default_confidence,
    provenance = EXCLUDED.provenance,
    active = TRUE,
    updated_at = NOW();

INSERT INTO threat_indicators (
    id,
    source_id,
    indicator_type,
    value,
    normalized_value,
    confidence,
    valid_from,
    valid_until,
    tags,
    context
)
VALUES
(
    'ioc-local-ip-001',
    'intel-local-northstar',
    'ip',
    '198.51.100.66',
    '198.51.100.66',
    95,
    '2026-01-01T00:00:00Z',
    NULL,
    '["synthetic", "command-and-control", "northstar-lab"]'::jsonb,
    '{
      "classification": "malicious",
      "summary": "Synthetic command-and-control address used only by Sentinel scenarios.",
      "reference": "RFC 5737 TEST-NET-2"
    }'::jsonb
),
(
    'ioc-local-ip-002',
    'intel-local-northstar',
    'ip',
    '203.0.113.77',
    '203.0.113.77',
    88,
    '2026-01-01T00:00:00Z',
    NULL,
    '["synthetic", "staging", "northstar-lab"]'::jsonb,
    '{
      "classification": "suspicious",
      "summary": "Synthetic staging address used only by Sentinel scenarios.",
      "reference": "RFC 5737 TEST-NET-3"
    }'::jsonb
),
(
    'ioc-local-domain-001',
    'intel-local-northstar',
    'domain',
    'telemetry-sync.example',
    'telemetry-sync.example',
    80,
    '2026-01-01T00:00:00Z',
    NULL,
    '["synthetic", "domain", "northstar-lab"]'::jsonb,
    '{
      "classification": "suspicious",
      "summary": "Reserved synthetic domain fixture for future DNS telemetry enrichment."
    }'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    source_id = EXCLUDED.source_id,
    indicator_type = EXCLUDED.indicator_type,
    value = EXCLUDED.value,
    normalized_value = EXCLUDED.normalized_value,
    confidence = EXCLUDED.confidence,
    valid_from = EXCLUDED.valid_from,
    valid_until = EXCLUDED.valid_until,
    tags = EXCLUDED.tags,
    context = EXCLUDED.context;
