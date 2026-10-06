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
VALUES
(
    'DET-AUTH-002',
    1,
    'Interactive Login Using a Service Account',
    'Detects a successful interactive login where the actor is classified as a service account.',
    'high',
    TRUE,
    '{
      "kind": "single_event",
      "category": "authentication",
      "action": "login",
      "outcome": "success",
      "actor_type": "service_account"
    }'::jsonb,
    '[
      {
        "framework": "MITRE ATT&CK",
        "technique_id": "T1078",
        "technique": "Valid Accounts"
      }
    ]'::jsonb
),
(
    'DET-NET-001',
    1,
    'Unexpected Corporate-to-OT Network Connection',
    'Detects a network connection originating from a corporate-zone asset and targeting the OT zone.',
    'high',
    TRUE,
    '{
      "kind": "single_event",
      "category": "network",
      "action": "connection",
      "source_zone": "corporate",
      "destination_zone": "ot"
    }'::jsonb,
    '[]'::jsonb
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
