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
    'DET-OT-001',
    1,
    'PLC or Controller Parameter Change',
    'Detects a synthetic OT parameter-change event on a PLC, RTU, or PAC in the OT zone.',
    'high',
    TRUE,
    '{
      "kind": "single_event",
      "category": "ot",
      "action": "parameter_change",
      "source_zone": "ot",
      "device_types": ["plc", "rtu", "pac"],
      "safety_note": "Synthetic telemetry observation only."
    }'::jsonb,
    '[
      {
        "framework": "MITRE ATT&CK for ICS",
        "technique_id": "T0836",
        "technique": "Modify Parameter"
      }
    ]'::jsonb
),
(
    'DET-OT-002',
    1,
    'Unauthorized OT Command Message',
    'Detects a synthetic OT command-message observation explicitly labelled as unauthorized.',
    'critical',
    TRUE,
    '{
      "kind": "single_event",
      "category": "ot",
      "action": "command_message",
      "source_zone": "ot",
      "authorized": false,
      "safety_note": "Synthetic telemetry observation only."
    }'::jsonb,
    '[
      {
        "framework": "MITRE ATT&CK for ICS",
        "technique_id": "T1692.001",
        "technique": "Unauthorized Message: Command Message"
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
