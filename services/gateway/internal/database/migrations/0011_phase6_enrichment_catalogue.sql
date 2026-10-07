INSERT INTO threat_intel_sources (
    id,
    name,
    source_type,
    description,
    default_confidence,
    provenance
)
VALUES (
    'intel-local-review',
    'Northstar Analyst Review Feed',
    'local_fixture',
    'Secondary synthetic source used to exercise confidence and disagreement handling.',
    70,
    '{
      "origin": "Sentinel repository",
      "collection_method": "synthetic analyst-review fixture",
      "network_scope": "documentation-only addresses and reserved example domains",
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
    'ioc-local-domain-002',
    'intel-local-review',
    'domain',
    'telemetry-sync.example',
    'telemetry-sync.example',
    92,
    '2026-01-01T00:00:00Z',
    NULL,
    '["synthetic", "domain", "command-and-control", "northstar-lab"]'::jsonb,
    '{
      "classification": "malicious",
      "summary": "Synthetic domain fixture used for DNS enrichment testing."
    }'::jsonb
),
(
    'ioc-local-sha256-001',
    'intel-local-northstar',
    'sha256',
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    90,
    '2026-01-01T00:00:00Z',
    NULL,
    '["synthetic", "file-hash", "northstar-lab"]'::jsonb,
    '{
      "classification": "malicious",
      "summary": "Synthetic SHA-256 fixture used for endpoint-file enrichment testing."
    }'::jsonb
),
(
    'ioc-review-ip-001',
    'intel-local-review',
    'ip',
    '198.51.100.66',
    '198.51.100.66',
    60,
    '2026-01-01T00:00:00Z',
    NULL,
    '["synthetic", "review", "northstar-lab"]'::jsonb,
    '{
      "classification": "suspicious",
      "summary": "Secondary synthetic review source with lower confidence for disagreement testing."
    }'::jsonb
),
(
    'ioc-local-internal-ip-001',
    'intel-local-northstar',
    'ip',
    '10.10.0.25',
    '10.10.0.25',
    90,
    '2026-01-01T00:00:00Z',
    NULL,
    '["synthetic", "internal-watchlist", "northstar-lab"]'::jsonb,
    '{
      "classification": "suspicious",
      "summary": "Synthetic internal watchlist address used to test finding and investigation enrichment."
    }'::jsonb
),
(
    'ioc-expired-ip-001',
    'intel-local-review',
    'ip',
    '192.0.2.44',
    '192.0.2.44',
    95,
    '2026-01-01T00:00:00Z',
    '2026-06-01T00:00:00Z',
    '["synthetic", "expired", "northstar-lab"]'::jsonb,
    '{
      "classification": "malicious",
      "summary": "Expired synthetic indicator used to verify validity filtering."
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
