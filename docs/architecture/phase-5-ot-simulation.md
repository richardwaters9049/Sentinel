# Phase 5 — OT Simulation

## Status

In progress on:

```text
feat/phase-5-ot-simulation
```

## Objective

Phase 5 extends Sentinel's synthetic Northstar Energy lab with safe operational-technology telemetry and explainable defensive detections.

This phase remains simulation-only.

Sentinel must not connect to real controllers, alter industrial equipment, or generate operational command sequences. OT behaviour is represented as normalised metadata events that analysts can inspect, hunt, correlate, and investigate.

## First vertical slice

The first Phase 5 slice adds three deterministic simulator scenarios:

```text
ot-hmi-read-baseline
ot-plc-parameter-change
ot-unauthorized-command
```

The baseline represents expected HMI/controller observation and must not create an OT finding.

The parameter-change and unauthorized-message scenarios represent suspicious **telemetry observations** only. They include descriptive labels such as:

```text
ot.device_type
ot.protocol
ot.operation
ot.authorized
ot.safety_impact
ot.simulated
```

These labels keep the existing telemetry schema stable while allowing Phase 5 to carry OT-specific context.

## Northstar OT assets

The first slice uses the existing fictional Northstar OT model:

- engineering-workstation-01;
- hmi-01;
- plc-sim-01.

Additional simulated assets can be added in later Phase 5 slices, including plc-sim-02 and sensor-sim-01.

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

The ATT&CK mappings describe analyst context. A Sentinel finding is evidence that the deterministic telemetry conditions matched; it is not proof of malicious intent.

## Safety annotations

Every Phase 5 simulator event includes:

```text
ot.simulated = true
```

Suspicious OT observations also carry a high-level safety-impact classification.

The simulator deliberately models event metadata rather than process-control instructions or device payloads.

## Explainable evidence

Phase 5 extends finding evidence with OT-specific fields:

- OT protocol family;
- device type;
- high-level operation;
- authorization classification;
- safety-impact annotation.

This context flows into the existing finding, hunt, investigation, and analyst-console workflows.

## Verification

The initial Phase 5 smoke test is:

```bash
make smoke-phase5
```

It verifies:

1. both OT detections are present and enabled;
2. ATT&CK for ICS mappings are persisted;
3. the normal HMI baseline does not create an OT finding;
4. parameter-change telemetry creates DET-OT-001;
5. unauthorized-message telemetry creates DET-OT-002;
6. OT-specific evidence is preserved;
7. OT events are queryable through the normal event API;
8. all simulator events are explicitly marked synthetic.

## Remaining Phase 5 work

- broaden the fictional OT asset inventory;
- add controller mode-change and historian interaction scenarios;
- add operational-safety annotations to the analyst UI;
- add richer normal-behaviour fixtures to protect against noisy rules;
- expose ATT&CK for ICS mappings in finding and detection detail views;
- add a complete Phase 5 regression suite once the OT scenario catalogue is mature.
