# DET-OT-003 — Controller Mode Change Followed by Parameter Change

## Purpose

Correlate two synthetic OT telemetry observations on the same controller:

1. a controller mode-change event;
2. a parameter-change event within ten minutes.

This adds a temporal correlation pattern to Phase 5 without interacting with real industrial equipment.

## Correlation key

The rule groups by:

- `asset_id`.

The query is bounded to:

- category `ot`;
- action `controller_mode_change`;
- the ten minutes preceding the terminal parameter-change event;
- at most eight candidate events.

## Trigger

The terminal event must have:

- `event.category = ot`;
- `event.action = parameter_change`;
- `asset.zone = ot`.

A recent controller mode-change event for the same asset must exist inside the correlation window.

## Finding

- severity: critical
- confidence: 94
- status: new
- evidence: both event IDs in chronological order, the ten-minute window, OT device/protocol/operation metadata, and safety-impact annotation

## ATT&CK for ICS

- T0836 — Modify Parameter

The mapping is contextual. The sequence indicates a higher-risk operational change pattern but is not, by itself, proof of malicious activity.

## Negative cases

The rule does not fire when:

- a parameter change occurs without a recent controller mode change;
- the source asset is outside the OT zone;
- the relevant controller mode change falls outside the bounded correlation window.
