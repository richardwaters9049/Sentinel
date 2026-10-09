# Sentinel documentation

Start with the [root README](../README.md) for capabilities, status and safety scope.
These documents separate current implementation from longer-term design goals.

## Getting started and planning

- [Local development](LOCAL_DEVELOPMENT.md) — toolchain, services, sign-in and checks.
- [Development roadmap](ROADMAP.md) — Phases 0–9, current priorities and completion gates.
- [Project vision](PROJECT_VISION.md) — motivation, telemetry families, console goals and safe scenarios.
- [Engineering and contribution guide](ENGINEERING.md) — security, testing, observability and workflow.
- [Agent operating rules](../AGENTS.md).

## Architecture and security

- [Architecture overview](architecture/overview.md) — original foundation and growth boundaries.
- [Event flow and invariants](architecture/event-flow.md).
- [Trust boundaries](architecture/trust-boundaries.md).
- [Threat model](threat-model/THREAT_MODEL.md).
- [Platform hardening](architecture/phase-8-platform-hardening.md) — authentication, roles, sessions and residual risk.

Architecture decisions:

- [0001 — Go for the core backend](adr/0001-core-backend-go.md).
- [0002 — NATS JetStream](adr/0002-nats-jetstream.md).
- [0003 — PostgreSQL as the system of record](adr/0003-postgresql-system-of-record.md).
- [0004 — Behavioural monitoring](adr/0004-behaviour-monitoring.md).
- [0005 — API authentication](adr/0005-api-authentication.md).
- [0006 — Console sessions](adr/0006-console-sessions.md).

## Telemetry contracts

- [Event schema v1](telemetry/EVENT_SCHEMA_V1.md).
- [Machine-readable event schema](../packages/schemas/telemetry-event-v1.schema.json).
- [Failure, delivery and persistence semantics](telemetry/FAILURE_SEMANTICS.md).

## Detections and analyst workflows

- [Finding lifecycle and analyst workflow](detections/ANALYST_WORKFLOW.md).
- [Hunt model](hunting/HUNT_MODEL.md).
- [DET-AUTH-001](detections/DET-AUTH-001.md).
- [DET-AUTH-002](detections/DET-AUTH-002.md).
- [DET-NET-001](detections/DET-NET-001.md).
- [DET-OT-001](detections/DET-OT-001.md).
- [DET-OT-002](detections/DET-OT-002.md).
- [DET-OT-003](detections/DET-OT-003.md).

## Implemented phases

- [Phase 0 — Foundation](architecture/phase-0-foundation.md).
- [Phase 1 — Telemetry](architecture/phase-1-telemetry.md).
- [Phase 2 — Detection engineering](architecture/phase-2-detection-engineering.md).
- [Phase 3 — Threat hunting and investigation](architecture/phase-3-threat-hunting.md).
- [Phase 4 — Analyst console](architecture/phase-4-analyst-console.md).
- [Phase 5 — OT simulation](architecture/phase-5-ot-simulation.md).
- [Phase 6 — Intelligence and enrichment](architecture/phase-6-intelligence-enrichment.md).
- [Phase 7 — Behavioural analytics](architecture/phase-7-behavioural-analytics.md).
- [Phase 7 — Completion evidence and limitations](architecture/phase-7-completion.md).
- [Phase 8 — Platform hardening](architecture/phase-8-platform-hardening.md).
- [Phase 9 — Deployment and operations](architecture/phase-9-deployment.md).
- [Phase Eight/Nine completion evidence](architecture/phase-8-9-completion.md).
- [Collector signatures and access ledger](adr/0007-collector-signatures-and-access-ledger.md).
- [Behavioural analytics service](../services/ml/README.md) — model features and reproducibility.

Phase documents preserve historical milestones and verification. Current authentication
requirements are described in Phase Eight; earlier development examples may rely on
explicit local compatibility mode. Design aspirations in the project vision do not
supersede implemented contracts or imply production readiness.
