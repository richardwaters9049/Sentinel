# Sentinel

**A defensive threat-hunting, detection-engineering and security-analytics platform for simulated critical-infrastructure environments.**

Sentinel models a fictional Northstar Energy facility, bringing enterprise IT and operational-technology telemetry into one analyst workflow. It turns synthetic events into explainable findings, supports hypothesis-driven hunts, and preserves evidence in investigation timelines.

The project demonstrates secure backend engineering, durable event processing and practical defensive-security workflows. Its focus is evidence that analysts can inspect, decisions they can explain, and scenarios they can reproduce.

## What it does

- **Telemetry ingestion:** validates and normalises versioned events, handles duplicate delivery, and persists asset and identity context.
- **Detection engineering:** evaluates deterministic rules, preserves supporting evidence, and provides rule controls and audited finding triage.
- **Hunting and investigations:** supports versioned hypotheses, bounded searches, asset/identity pivots, analyst notes and unified timelines.
- **IT/OT visibility:** models fictional industrial assets, boundary activity and telemetry gaps through safe simulation.
- **Context and analytics:** enriches findings using local threat-intelligence fixtures and adds behavioural scores with model versions and analyst explanations.
- **Analyst console:** brings these workflows together with authenticated sign-in, role-based API permissions and revocable sessions.

## How it works

```text
Synthetic events → Go gateway → NATS JetStream → persistence and detection
                                                        ↓
                                                    PostgreSQL
                                                        ↓
                                              API → Analyst console
```

**Stack:** Go for the gateway and core processing; PostgreSQL for durable evidence; NATS JetStream for event delivery; Next.js, React and TypeScript for the console; Python for supporting behavioural analytics. Docker Compose runs the local infrastructure.

## Current status

Phases 0–7 are complete within the simulated lab scope. **Phase 8 platform hardening is underway**, with API authentication, role checks and secure console sessions implemented. Further work includes role-aware console controls, rate limiting, collector provenance and security scanning.

Behavioural analytics supplement deterministic detections. Their synthetic evaluations expose known coverage limitations; model scores are supporting evidence, not proof of compromise or production detection accuracy. Progress and verification are recorded in the [development roadmap](docs/ROADMAP.md).

## Getting started

The local lab uses Go 1.27.1, Bun 1.3.14 and Docker Compose, with Python 3 for authentication provisioning and smoke scripts. The [local development guide](docs/LOCAL_DEVELOPMENT.md) covers installation, service startup, private lab credentials and checks.

For architecture, event contracts, detection rules and engineering decisions, see the [project docs](docs/README.md).

## Safety and scope

All demonstrations use synthetic telemetry, fictional assets and controlled simulations. Sentinel does not target third-party systems, manipulate real industrial controllers or perform disruptive remediation. Keep credentials and private telemetry out of the repository.

## Author and licence

Created by **Richard Waters**, a Senior Software Engineer focused on secure systems and defensive security platforms.

No licence has been selected; broad reuse rights are not granted.
