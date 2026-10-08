# Sentinel

A defensive threat-hunting, detection-engineering and security-analytics platform for
**simulated critical-infrastructure environments**.

Sentinel models a fictional Northstar Energy lab, combining enterprise and OT telemetry
with explainable findings, analyst investigations and reproducible scenarios. It is a
portfolio project focused on secure engineering, event-driven systems and defensive workflows.

## Current capabilities

- Validated, versioned synthetic telemetry with durable NATS JetStream delivery and PostgreSQL persistence.
- Deterministic detections with evidence, rule controls and audited finding triage.
- Versioned hypothesis-driven hunts, asset/identity pivots and investigation timelines.
- A Next.js analyst console for findings, hunts, investigations, detection health and environment visibility.
- Safe OT simulation, telemetry-gap context and local threat-intelligence enrichment.
- Behavioural analytics with model versions, analyst explanations and reproducible synthetic evaluation.
- API authentication/RBAC and interactive sign-in with revocable PostgreSQL sessions and CSRF protection.

**Status:** Phases 0–7 are complete within their simulated scope. Phase 8 platform
hardening is in progress. Behavioural results have documented coverage limitations;
synthetic completion does not establish production detection accuracy.
See the [development roadmap](docs/ROADMAP.md), [Phase Seven completion evidence](docs/architecture/phase-7-completion.md)
and [current hardening work](docs/architecture/phase-8-platform-hardening.md).

## Architecture

```text
Synthetic telemetry → Go gateway → NATS JetStream → persistence / detection
                                                       ↓
                                                   PostgreSQL
                                                       ↓
                                            Go API → Next.js console
```

The current stack uses **Go, PostgreSQL, NATS JetStream, Next.js/React/TypeScript,
Python behavioural analytics and Docker Compose**. Components grow around real
responsibilities; optional collectors and deployment complexity remain future work.
See [event flow](docs/architecture/event-flow.md) and [architecture decisions](docs/README.md#architecture-and-security).

## Run locally

Requirements: Go 1.27.1, Bun 1.3.14, Docker with Compose, and Python 3 for the
Phase Eight provisioning/smoke scripts. ML dependencies run in their container.

From `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
make help
```

Follow the [local development guide](docs/LOCAL_DEVELOPMENT.md) to install console
dependencies, start the lab and enable sign-in. The canonical console origin is
`http://127.0.0.1:3000`. Keep local development and synthetic fixtures off public networks.

## Safety

Sentinel uses synthetic data, fictional assets and controlled local simulations.
It does not target third parties, manipulate real controllers or perform disruptive
remediation. Never commit credentials, private telemetry or sensitive infrastructure data.
ML scores support analyst judgement; they are not security verdicts.

## Documentation

- [Documentation index](docs/README.md) — architecture, contracts, detections, hunts and decisions.
- [Development roadmap](docs/ROADMAP.md) — phase status, scope and verification gates.
- [Project vision](docs/PROJECT_VISION.md) — motivation, planned telemetry, scenarios and long-term goals.
- [Engineering and contribution guide](docs/ENGINEERING.md) — security, testing, observability and workflow.
- [Threat model](docs/threat-model/THREAT_MODEL.md) and [trust boundaries](docs/architecture/trust-boundaries.md).
- [Agent instructions](AGENTS.md) — detailed repository operating rules.

## Author and licence

**Richard Waters** — Senior Software Engineer focused on secure systems, cybersecurity,
backend platforms, cloud infrastructure and AI-enabled products.

A licence has not yet been selected. The repository does not grant broad reuse rights.
