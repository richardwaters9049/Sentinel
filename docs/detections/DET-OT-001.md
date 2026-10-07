# DET-OT-001 — PLC or Controller Parameter Change

## Purpose

Detect a synthetic parameter-change event affecting a PLC, RTU, or PAC in the Northstar OT lab.

Sentinel observes normalised telemetry only. It does not interact with industrial controllers.

## Match conditions

The rule requires:

- `event.category = ot`;
- `event.action = parameter_change`;
- `asset.zone = ot`;
- `labels["ot.device_type"]` is `plc`, `rtu`, or `pac`.

The simulator adds descriptive context such as protocol family, operation name, and safety-impact classification.

## Finding

- severity: high
- confidence: 92
- status: new

Evidence records the source zone, device type, protocol, operation, safety-impact annotation, and triggering event ID.

## ATT&CK for ICS

- T0836 — Modify Parameter

The mapping is contextual: the rule detects telemetry consistent with a parameter-change activity, not proof of malicious intent.

## Negative case

The rule ignores events outside the OT zone and events without a supported controller device type.
