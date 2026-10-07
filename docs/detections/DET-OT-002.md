# DET-OT-002 — Unauthorized OT Command Message

## Purpose

Detect a synthetic OT command-message observation explicitly classified as unauthorized in the Northstar lab.

Sentinel consumes descriptive telemetry only.

## Match conditions

The rule requires:

- `event.category = ot`;
- `event.action = command_message`;
- `asset.zone = ot`;
- `labels["ot.authorized"] = false`.

The simulator adds descriptive context including device type, protocol family, source/destination addresses, and safety-impact classification.

## Finding

- severity: critical
- confidence: 96
- status: new

Evidence records source/destination context, device type, protocol, operation, authorization classification, safety-impact annotation, and the triggering event ID.

## ATT&CK for ICS

- T1692.001 — Unauthorized Message: Command Message

The mapping represents defensive analytic context only.

## Negative case

Command-message observations classified as authorized do not create this finding.
