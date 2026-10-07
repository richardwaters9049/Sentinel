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
    'DET-OT-003',
    1,
    'Controller Mode Change Followed by Parameter Change',
    'Correlates a synthetic controller mode change with a parameter change on the same OT asset within ten minutes.',
    'critical',
    TRUE,
    '{
      "kind": "temporal_correlation",
      "category": "ot",
      "sequence": ["controller_mode_change", "parameter_change"],
      "group_by": "asset_id",
      "window_seconds": 600,
      "source_zone": "ot",
      "safety_note": "Synthetic telemetry correlation only."
    }'::jsonb,
    '[
      {
        "framework": "MITRE ATT&CK for ICS",
        "technique_id": "T0836",
        "technique": "Modify Parameter"
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
