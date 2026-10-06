ALTER TABLE findings
    ADD COLUMN IF NOT EXISTS dedup_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_findings_dedup_key
    ON findings (dedup_key)
    WHERE dedup_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_events_auth_lookup
    ON events (identity_id, outcome, source_timestamp DESC)
    WHERE category = 'authentication' AND action = 'login';

INSERT INTO detections (
    id,
    version,
    title,
    description,
    severity,
    enabled,
    definition,
    mitre
)
VALUES (
    'DET-AUTH-001',
    1,
    'Repeated Authentication Failures Followed by Success',
    'Detects four or more failed login attempts followed by a successful login for the same identity and source address within five minutes.',
    'high',
    TRUE,
    '{
      "kind": "temporal_threshold",
      "category": "authentication",
      "action": "login",
      "failure_outcome": "failure",
      "success_outcome": "success",
      "failure_threshold": 4,
      "window_seconds": 300,
      "group_by": ["identity_id", "network.source_ip"]
    }'::jsonb,
    '[
      {
        "framework": "MITRE ATT&CK",
        "technique_id": "T1110",
        "technique": "Brute Force"
      },
      {
        "framework": "MITRE ATT&CK",
        "technique_id": "T1078",
        "technique": "Valid Accounts"
      }
    ]'::jsonb
)
ON CONFLICT (id) DO UPDATE SET
    version = EXCLUDED.version,
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    severity = EXCLUDED.severity,
    enabled = EXCLUDED.enabled,
    definition = EXCLUDED.definition,
    mitre = EXCLUDED.mitre,
    updated_at = NOW();
