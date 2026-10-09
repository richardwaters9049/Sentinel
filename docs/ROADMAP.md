# Development roadmap

Phases describe delivered vertical slices, not production certification. Phases 0–9
are complete within the simulated lab scope. The linked phase
documents contain implementation details, contracts, limitations and exit criteria.
This roadmap follows the implemented phase names: hunting and investigations are
Phase 3, and the analyst console is Phase 4.

| Phase | Scope | Status / details |
| --- | --- | --- |
| 0 — Foundation | Go service, Compose, PostgreSQL, NATS, CI, threat model and ADRs | [Complete](architecture/phase-0-foundation.md) |
| 1 — Telemetry | Schema v1, synthetic generator, validation/normalisation, durable transport, persistence and queries | [Complete](architecture/phase-1-telemetry.md) |
| 2 — Detection engineering | Multi-rule engine, explainable findings, triage/audits, runtime controls and quality metrics | [Complete](architecture/phase-2-detection-engineering.md) |
| 3 — Threat hunting and investigation | Versioned hypotheses, bounded queries, run history, pivots, cases, ownership, notes, evidence and timelines | [Complete](architecture/phase-3-threat-hunting.md) |
| 4 — Analyst console | Operational workspaces, finding/investigation workflows, hunt execution and detection controls | [Complete](architecture/phase-4-analyst-console.md) |
| 5 — OT simulation | Fictional OT inventory, synthetic ICS-style events, boundary rules, coverage gaps and safety context | [Complete](architecture/phase-5-ot-simulation.md) |
| 6 — Intelligence and enrichment | Local intel fixtures, IOC matching, confidence, provenance and enrichment workflows | [Complete](architecture/phase-6-intelligence-enrichment.md) |
| 7 — Behavioural analytics | Python model, baselines/features, explanations, entity-scoped scores, distribution monitoring and governed evaluation | [Complete, simulated scope](architecture/phase-7-behavioural-analytics.md) |
| 8 — Platform hardening | Authentication, RBAC, sessions, collector trust and security tooling | [Complete, simulated scope](architecture/phase-8-platform-hardening.md) |
| 9 — Production-shaped deployment | Tracing/monitoring, measured load and failure tests, container deployment and validated Kubernetes base | [Complete, local simulated scope](architecture/phase-9-deployment.md) |

## Completed milestone

The initial end-to-end story is implemented across the delivered slices:

1. Generate a deterministic, safe scenario.
2. Ingest, validate, normalise and persist telemetry.
3. Evaluate deterministic rules and create explainable findings.
4. Inspect evidence, assets, identities and ATT&CK context in the console.
5. Hunt related activity and preserve it in investigations.
6. Record analyst decisions and audit attribution.

The deterministic catalogue includes DET-AUTH-001, DET-AUTH-002, DET-NET-001 and
three simulated OT rules; see the [rule documentation](README.md#detections-and-analyst-workflows).
Behavioural analytics supplement these rules. Five versioned profiles expose synthetic
coverage and review guidance. BA-001, BA-003 and BA-004 remain below threshold on
all twelve changed cases in the catalogue evaluation; see the
[Phase Seven completion evidence](architecture/phase-7-completion.md).

## Final milestones

Phase Eight adds signed collector provenance, durable nonce replay controls, bounded
rate limiting, access-audit retention and security scanning to the existing API/session
boundary. Phase Nine packages the local stack in hardened containers, connects metrics
and HTTP traces, verifies recovery and backup restore, and supplies a validated
Kubernetes base. See the [completion evidence](architecture/phase-8-9-completion.md).

The planned phases are delivered. Subsequent work should be driven by an explicit
deployment target or analytical requirement: central identity, shared multi-replica
limits, authenticated/encrypted dependencies, hosted CI enforcement and improvement of
the documented behavioural coverage gaps remain outside the completed lab scope.

## Verification gates

From `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
make final-phase7
make final-phase8
make final-phase9
make security-gate
```

`final-phase7` combines the inherited Phase One–Six baseline with behavioural model,
API, persistence and monitoring checks. The Phase Eight gate exercises isolated
required-mode API and console services. Phase Nine uses disposable credentials,
containers and volumes; security scanning produces eight runtime SBOMs outside the
repository. It needs Trivy 0.75.0 and govulncheck 1.8.0 on PATH. These commands build images/services and
use local PostgreSQL; their synthetic investigation evidence is retained. Check the
linked phase documents for ports and prerequisites before running alongside a lab.
The Phase Seven gate requires exclusive gateway/NATS workers and cycles local
dependencies: stop the development gateway first and free its console test port.
Earlier phase-specific commands remain discoverable through `make help`.
