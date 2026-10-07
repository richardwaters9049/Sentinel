# Phase 5 — OT Simulation

## Status

Complete on:

```text
feat/phase-5-ot-simulation
```

## Objective

Phase 5 extends Sentinel's synthetic Northstar Energy lab with safe operational-technology telemetry, richer OT asset context, deterministic detections, and bounded temporal correlation.

This phase remains simulation-only.

Sentinel does not connect to real controllers, alter industrial equipment, or generate operational command sequences. OT behaviour is represented as normalised metadata events that analysts can inspect, hunt, correlate, and investigate.

## Northstar OT scenario catalogue

Phase 5 now includes:

```text
ot-hmi-read-baseline
ot-historian-read-baseline
ot-sensor-telemetry-baseline
ot-controller-mode-change
ot-plc-parameter-change
ot-unauthorized-command
ot-change-sequence
```

The normal-behaviour fixtures model expected OT observation without creating OT findings:

- HMI read activity;
- historian read activity;
- sensor telemetry;
- a standalone controller mode change.

The suspicious scenarios model metadata observations only:

- controller parameter change;
- explicitly unauthorized command message;
- controller mode change followed by parameter change on the same asset.

## Northstar OT asset inventory

Phase 5 exercises:

- engineering-workstation-01;
- hmi-01;
- historian-01;
- plc-sim-01;
- plc-sim-02;
- sensor-sim-01.

All are fictional, local, and synthetic.

## OT metadata

Phase 5 keeps the event schema stable by carrying OT-specific context through bounded string labels:

```text
ot.device_type
ot.protocol
ot.operation
ot.authorized
ot.safety_impact
ot.simulated
```

Every Phase 5 simulator event includes:

```text
ot.simulated = true
```

Suspicious observations also carry a high-level safety-impact classification.

## Detection catalogue

### DET-OT-001 — PLC or Controller Parameter Change

Matches:

- category `ot`;
- action `parameter_change`;
- source asset in the `ot` zone;
- controller type `plc`, `rtu`, or `pac`.

Finding:

- severity: high;
- confidence: 92.

ATT&CK for ICS context:

- T0836 — Modify Parameter.

### DET-OT-002 — Unauthorized OT Command Message

Matches:

- category `ot`;
- action `command_message`;
- source asset in the `ot` zone;
- `ot.authorized = false`.

Finding:

- severity: critical;
- confidence: 96.

ATT&CK for ICS context:

- T1692.001 — Unauthorized Message: Command Message.

### DET-OT-003 — Controller Mode Change Followed by Parameter Change

This is Phase 5's first OT temporal-correlation rule.

The terminal event must be an OT parameter change. Sentinel then queries only the same asset for recent OT `controller_mode_change` events.

The query is bounded to:

- the preceding 10 minutes;
- category `ot`;
- action `controller_mode_change`;
- the same `asset_id`;
- at most eight candidate events.

Finding:

- severity: critical;
- confidence: 94;
- evidence includes both sides of the correlated sequence in chronological order.

ATT&CK for ICS context:

- T0836 — Modify Parameter.

The mappings describe analyst context. A Sentinel finding means the deterministic telemetry conditions matched; it is not proof of malicious intent.

## Explainable evidence

Phase 5 findings preserve OT-specific evidence including:

- protocol family;
- device type;
- high-level operation;
- authorization classification;
- safety-impact annotation;
- correlation window;
- linked event IDs.

This context flows through the existing finding, hunt, investigation, and analyst-console workflows.

## Analyst console

Phase 5 extends the Phase 4 console rather than creating another standalone dashboard.

The Findings workspace now shows an OT operational-safety panel for `DET-OT-*` findings with:

- device type;
- protocol;
- operation;
- safety-impact annotation;
- clear synthetic-lab wording.

Finding explainability also includes OT protocol, device, operation, authorization, and correlation-window details when present.

The Detection Engineering workspace now displays persisted ATT&CK / ATT&CK for ICS mappings and any safety note stored in a detection definition.

## Regression isolation

The Phase 5 smoke test uses its own temporary NATS JetStream container and ports.

This matters because local development gateways share the normal telemetry durable consumer. Without an isolated NATS instance, another running gateway can legally consume a smoke-test message first, causing nondeterministic test results.

The temporary JetStream container is removed automatically when the test exits.

## Verification

Phase 5 provides:

```text
scripts/phase5-ot-smoke.sh
scripts/phase5-final-regression.sh
```

and Make targets:

```bash
make smoke-phase5
make final-phase5
```

The OT smoke test verifies:

1. DET-OT-001, DET-OT-002, and DET-OT-003 are present and enabled;
2. ATT&CK for ICS mappings are persisted;
3. normal HMI, historian, sensor, and standalone mode-change telemetry do not create OT findings;
4. parameter-change telemetry creates DET-OT-001;
5. unauthorized-message telemetry creates DET-OT-002;
6. a mode-change → parameter-change sequence creates DET-OT-003;
7. OT-specific evidence and correlation windows are preserved;
8. the expanded OT asset inventory is persisted;
9. OT events remain queryable through the normal event API;
10. all Phase 5 simulator events are explicitly marked synthetic.

The final Phase 5 regression runs the complete Phase 4 regression baseline first and then the full OT smoke suite.

## Phase 5 completion criteria

- [x] broaden the fictional OT asset inventory;
- [x] add controller mode-change and historian interaction scenarios;
- [x] add sensor and other normal-behaviour fixtures;
- [x] add operational-safety annotations to the analyst UI;
- [x] expose ATT&CK for ICS mappings in detection detail;
- [x] add explainable OT finding context;
- [x] add a bounded temporal OT correlation;
- [x] isolate OT smoke testing from local NATS consumers;
- [x] add a complete Phase 5 regression suite.
