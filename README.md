# Sentinel

> A defensive cyber-threat hunting and detection engineering platform for simulated critical-infrastructure environments.

Sentinel is a portfolio-grade security engineering project focused on **threat hunting, detection engineering, event correlation, telemetry analysis, and IT/OT visibility**. It is designed to model the kinds of defensive workflows used in Security Operations Centres (SOCs), critical national infrastructure, industrial environments, and high-assurance organisations.

The project is deliberately more ambitious than a conventional CRUD or dashboard application. Sentinel is intended to demonstrate systems thinking across secure software engineering, distributed services, event-driven architecture, observability, data engineering, defensive cybersecurity, and applied machine learning.

---

## Project status

**Status:** Phase 1 telemetry pipeline complete; Phase 2 detection engineering complete; Phase 3 threat hunting and investigation complete; Phase 4 analyst console complete; Phase 5 OT simulation complete; Phase 6 intelligence and enrichment complete; Phase 7 behavioural analytics complete (simulated scope; documented model-coverage limitations); Phase 8 platform hardening in progress.

The first production-shaped vertical slice is now implemented:

1. generate safe synthetic telemetry;
2. ingest it through `POST /api/v1/telemetry`;
3. validate and normalise events against schema v1;
4. publish accepted events through NATS JetStream;
5. persist events, assets, and identities in PostgreSQL;
6. query stored telemetry through `GET /api/v1/events`;
7. verify idempotency and dependency recovery with repeatable smoke/resilience tests.

Phase 2 adds a multi-rule detection engine, explainable findings, audited analyst workflows, and runtime detection controls. The verified catalogue includes DET-AUTH-001, DET-AUTH-002, and DET-NET-001. Phase 3 now adds versioned hypothesis-driven hunts, richer bounded event searches, durable hunt-run history, asset/identity pivots, durable investigations, analyst notes, exact hunt-result attachment, owner/priority management, workflow metrics, and unified investigation timelines.

The system should grow incrementally from that foundation. Every major component must be testable in isolation and understandable without requiring the full stack to run.

---

## Why Sentinel exists

Modern defensive security teams rarely work from one log source or one simple alert stream. Analysts must reason across identity events, process activity, network flows, service behaviour, endpoint telemetry, configuration changes, threat intelligence, and contextual information about assets.

Industrial and critical-infrastructure environments add another layer of difficulty:

- business IT and operational technology have different availability and safety constraints;
- legacy systems may not support conventional endpoint tooling;
- telemetry may be incomplete or inconsistent;
- normal traffic can include unusual protocols and long-lived connections;
- false positives can be costly;
- a defensive action that is reasonable on a laptop may be unacceptable on an industrial controller.

Sentinel exists to model these problems safely using **synthetic data and simulated assets**.

It is **not** intended to interact with real industrial equipment, bypass protections, deliver payloads, or provide offensive capability against third-party systems.

---

## Core objectives

Sentinel should demonstrate the following capabilities:

- secure ingestion of high-volume security telemetry;
- normalisation of heterogeneous event formats;
- deterministic detection rules;
- behavioural and anomaly-based detections;
- analyst-driven threat hunting;
- event correlation across multiple sources;
- MITRE ATT&CK and MITRE ATT&CK for ICS mappings;
- asset and identity context;
- investigation timelines;
- finding triage and evidence handling;
- detection quality metrics;
- telemetry-gap identification;
- threat-intelligence enrichment;
- auditable analyst actions;
- reproducible simulation scenarios;
- resilient distributed-service design;
- secure-by-default engineering practices;
- observability for the platform itself.

---

## Safety and scope

Sentinel is a **defensive research and simulation platform**.

All demonstration environments must be synthetic, local, containerised, explicitly authorised, or otherwise controlled by the project owner.

The repository must not contain:

- malware;
- credential theft tooling;
- persistence mechanisms intended for real systems;
- destructive industrial-control logic;
- exploit chains targeting third parties;
- real production secrets;
- live customer telemetry;
- unauthorised scanning targets;
- instructions whose primary purpose is to compromise real infrastructure.

Attack scenarios should be represented as safe event generators, replay files, mocks, protocol simulators, or clearly bounded lab activity.

When there is a choice between realism and safety, choose safety.

---

## Conceptual environment

Sentinel models a fictional critical-infrastructure organisation containing conventional enterprise systems and a small simulated operational-technology network.

```text
                         ┌──────────────────────────────┐
                         │        Analyst Console       │
                         │       Next.js / React        │
                         └──────────────┬───────────────┘
                                        │
                                 HTTPS / WebSocket
                                        │
                         ┌──────────────▼───────────────┐
                         │       Sentinel API Layer      │
                         │              Go               │
                         └──────────────┬───────────────┘
                                        │
                                  Event / Query APIs
                                        │
        ┌───────────────────────────────┼──────────────────────────────┐
        │                               │                              │
┌───────▼────────┐             ┌────────▼─────────┐          ┌────────▼─────────┐
│ Detection      │             │ Hunt / Correlate │          │ Enrichment       │
│ Engine         │             │ Engine           │          │ Service          │
│ Go             │             │ Go               │          │ Go / Python      │
└───────┬────────┘             └────────┬─────────┘          └────────┬─────────┘
        │                               │                              │
        └───────────────────────────────┼──────────────────────────────┘
                                        │
                              ┌─────────▼─────────┐
                              │ Event Bus         │
                              │ NATS JetStream or │
                              │ Apache Kafka      │
                              └─────────┬─────────┘
                                        │
                    ┌───────────────────┼───────────────────┐
                    │                   │                   │
           ┌────────▼────────┐ ┌────────▼────────┐ ┌────────▼────────┐
           │ PostgreSQL      │ │ Redis           │ │ Object Storage  │
           │ events/findings │ │ hot state/cache │ │ evidence/replay │
           └─────────────────┘ └─────────────────┘ └─────────────────┘
                                        ▲
                                        │
                              ┌─────────┴─────────┐
                              │ Telemetry Gateway │
                              │ Go / Rust         │
                              └─────────┬─────────┘
                                        │
        ┌───────────────────────────────┼───────────────────────────────┐
        │                               │                               │
┌───────▼────────┐             ┌────────▼────────┐             ┌────────▼────────┐
│ Enterprise IT  │             │ Identity / IAM  │             │ OT Simulation  │
│ synthetic logs │             │ synthetic logs  │             │ synthetic logs │
└────────────────┘             └─────────────────┘             └─────────────────┘
```

This is the target architecture, not a requirement that every service exist on day one.

---

## Simulated asset model

The initial lab can model the following fictional environment:

```text
Northstar Energy Facility
│
├── Corporate IT
│   ├── employee-workstation-01
│   ├── employee-workstation-02
│   ├── identity-server-01
│   ├── application-server-01
│   └── database-server-01
│
├── Security / DMZ
│   ├── jump-host-01
│   ├── historian-01
│   ├── telemetry-gateway-01
│   └── update-server-01
│
└── OT Lab
    ├── engineering-workstation-01
    ├── hmi-01
    ├── plc-sim-01
    ├── plc-sim-02
    └── sensor-sim-01
```

All systems are synthetic. The initial OT components should be logical simulations rather than real industrial devices.

---

## Planned telemetry sources

Sentinel should eventually ingest several event families:

### Identity

- successful authentication;
- failed authentication;
- privilege changes;
- MFA events;
- account creation;
- account disablement;
- role assignment;
- service-account use;
- abnormal login time;
- new device association.

### Endpoint

- process start;
- process stop;
- file creation;
- file modification;
- executable hash;
- parent/child process relationship;
- scheduled task simulation;
- service change simulation;
- removable-media event;
- user session event.

### Network

- source and destination;
- port and protocol;
- connection duration;
- bytes transferred;
- DNS query;
- TLS metadata;
- east-west connection;
- IT-to-OT boundary crossing;
- blocked or denied connection.

### OT / ICS simulation

- HMI-to-controller communication;
- engineering workstation access;
- PLC configuration-change event;
- command-frequency changes;
- tag/value change;
- historian interaction;
- device mode change;
- controller restart;
- firmware-update simulation;
- unexpected protocol usage.

### Cloud / platform

- deployment event;
- secret access;
- role assumption;
- container start;
- image digest;
- CI/CD execution;
- infrastructure-change event.

---

## Normalised event schema

A normalised event should be expressive enough to support correlation without forcing every producer into the same raw structure.

Illustrative shape:

```json
{
  "event_id": "evt_01J...",
  "timestamp": "2026-10-06T20:00:00Z",
  "received_at": "2026-10-06T20:00:01Z",
  "source": {
    "type": "identity",
    "vendor": "sentinel-sim",
    "collector": "collector-01"
  },
  "asset": {
    "id": "asset-ew-01",
    "hostname": "engineering-workstation-01",
    "zone": "ot"
  },
  "actor": {
    "id": "user-042",
    "type": "human",
    "name": "operator.demo"
  },
  "event": {
    "category": "authentication",
    "action": "login",
    "outcome": "success"
  },
  "network": {
    "source_ip": "10.20.1.14",
    "destination_ip": "10.30.2.20",
    "destination_port": 443,
    "protocol": "tcp"
  },
  "labels": {
    "environment": "lab",
    "scenario": "lateral-movement-01"
  }
}
```

The real schema should use typed domain models and versioning.

---

## Detection model

Detections should initially be deterministic and explainable.

A detection should contain:

- stable ID;
- title;
- description;
- severity;
- confidence;
- status;
- author;
- version;
- event selectors;
- conditions;
- time window;
- grouping key;
- suppression logic;
- evidence requirements;
- MITRE mapping;
- ATT&CK for ICS mapping where applicable;
- test fixtures;
- expected positive cases;
- expected negative cases.

Example detection ideas:

- repeated failed authentication followed by success;
- new privileged role assignment;
- privileged account used from an unseen host;
- workstation communicating directly with an OT controller;
- engineering workstation initiating unusual controller interaction;
- unusual process spawning a network-capable child process;
- abnormal service-account use;
- unexpected protocol crossing an IT/OT boundary;
- configuration change outside an approved maintenance window;
- sudden increase in controller-write operations;
- identity activity followed by lateral movement;
- new executable followed by unusual outbound connection.

Every detection should explain **why** it fired.

---

## Threat hunting

A core differentiator for Sentinel is that it should support analyst-created hunting hypotheses rather than only predefined alerts.

Example hypothesis:

> Identify systems showing lateral movement after suspicious authentication activity.

The hunt engine could:

1. find unusual authentication events;
2. identify affected identities;
3. trace those identities across hosts;
4. identify first-seen network relationships;
5. identify privilege or process activity around the same time;
6. build an evidence timeline;
7. calculate a confidence score;
8. return candidate investigations.

Initial hunts can be implemented as saved queries or code-defined strategies. A later phase may add a query language.

---

## MITRE ATT&CK integration

Detections and investigations should be mapped to MITRE ATT&CK where appropriate.

Potential enterprise techniques include:

- Valid Accounts;
- Remote Services;
- Command and Scripting Interpreter;
- Account Discovery;
- Network Service Scanning;
- Remote System Discovery;
- Impair Defences.

ICS-specific mappings should be added only where the simulated behaviour actually supports them.

Mappings are context for analysts, not proof of malicious activity.

---

## Telemetry-gap analysis

Sentinel should eventually answer not only:

> What suspicious behaviour did we detect?

but also:

> What important behaviour would we be unable to detect with our current telemetry?

Examples:

- no process telemetry from an engineering workstation;
- network flow available but identity context absent;
- controller interactions visible but change events unavailable;
- authentication logs arriving late;
- collector stopped sending events;
- event source clock drift.

This feature is especially relevant to detection engineering and critical-infrastructure defence.

---

## Finding lifecycle

A finding can move through states such as:

```text
new
  ↓
triaged
  ↓
investigating
  ├──→ false_positive
  ├──→ benign_expected
  ├──→ duplicate
  └──→ confirmed
          ↓
       contained
          ↓
        closed
```

Every state transition should be auditable.

Analyst notes, evidence links, timestamps, and ownership changes should be preserved.

---

## Analyst console

The UI should feel like an operational security product rather than a generic admin dashboard.

Initial screens:

### Overview

- recent findings;
- findings by severity;
- ingestion health;
- event volume;
- active scenarios;
- telemetry coverage;
- recent high-confidence correlations.

### Findings

- sortable finding queue;
- severity and confidence filters;
- detection name;
- asset;
- identity;
- first and last observed time;
- event count;
- status;
- MITRE mapping.

### Investigation

- narrative summary;
- event timeline;
- related assets;
- related identities;
- network relationships;
- detection evidence;
- enrichment;
- analyst notes;
- MITRE techniques;
- raw event inspection.

### Hunts

- saved hunting hypotheses;
- execution history;
- matching entities;
- timeline;
- confidence;
- query details.

### Assets

- hostname;
- zone;
- operating-system family;
- owner / role;
- criticality;
- first seen;
- last seen;
- telemetry sources;
- risk context.

### Detection engineering

- rule list;
- rule version;
- enabled state;
- validation results;
- positive/negative fixtures;
- hit count;
- false-positive rate;
- last triggered.

### Platform health

- collectors;
- queue lag;
- storage health;
- processing latency;
- event rejection rate;
- service versions.

---

## Proposed technology stack

### Backend

**Go**

Primary responsibilities:

- ingestion APIs;
- event normalisation;
- detection evaluation;
- correlation;
- hunting;
- API services;
- service health;
- background processing.

Why Go:

- predictable concurrency;
- strong standard library;
- low runtime overhead;
- simple deployment;
- good fit for high-throughput event systems;
- relevant to modern infrastructure/security engineering.

### Sensor / collector

**Rust**

Potential responsibilities:

- lightweight local collector;
- safe parsing;
- event batching;
- backpressure handling;
- signed/encrypted transport;
- local buffering;
- integrity metadata.

Rust should only be introduced where it adds value. The project should not become multi-language for appearance alone.

### Frontend

**Next.js + React + TypeScript**

Responsibilities:

- analyst dashboard;
- finding triage;
- investigation timelines;
- threat-hunt interface;
- asset exploration;
- detection management;
- platform-health views.

### Data / state

**PostgreSQL**

Durable storage for:

- events;
- findings;
- assets;
- identities;
- detections;
- investigations;
- analyst actions;
- scenario metadata.

**Redis**

Short-lived state for:

- rule windows;
- deduplication;
- rate limiting;
- temporary correlation state;
- caching.

### Event transport

Start with **NATS JetStream** unless requirements later justify Kafka.

NATS provides a lighter operational footprint for local development while still demonstrating durable event-driven architecture.

Kafka can be introduced later if the project requires partition-heavy streaming semantics or Kafka-specific experience.

### Machine learning

**Python**

Potential later-phase use:

- anomaly scoring;
- baseline modelling;
- feature extraction;
- clustering;
- offline model evaluation.

ML must not replace explainable deterministic detection logic. Models should support analysts, not create opaque security conclusions.

### Infrastructure

- Docker / Docker Compose;
- GitHub Actions;
- OpenTelemetry;
- Prometheus;
- Grafana;
- structured JSON logging;
- optional Kubernetes deployment in a later phase.

---

## Proposed repository layout

```text
Sentinel/
├── README.md
├── AGENTS.md
├── LICENSE
├── .gitignore
├── .editorconfig
├── .env.example
├── Makefile
├── compose.yaml
│
├── apps/
│   └── console/                  # Next.js analyst console
│
├── services/
│   ├── gateway/                  # Go ingestion/API gateway
│   ├── detection-engine/         # Go detection evaluation
│   ├── hunt-engine/              # Go correlation and hunt workflows
│   ├── enrichment/               # Enrichment service
│   └── ml/                       # Python anomaly-analysis service
│
├── collectors/
│   └── rust-agent/               # Optional Rust telemetry collector
│
├── packages/
│   ├── schemas/                  # shared event schemas/contracts
│   ├── detections/               # rule definitions and fixtures
│   └── testdata/                 # synthetic fixtures only
│
├── simulator/
│   ├── scenarios/                # safe simulation definitions
│   ├── generators/               # synthetic telemetry generators
│   └── assets/                   # fictional asset definitions
│
├── infra/
│   ├── docker/
│   ├── kubernetes/
│   ├── monitoring/
│   └── migrations/
│
├── docs/
│   ├── architecture/
│   ├── detections/
│   ├── threat-model/
│   ├── security/
│   ├── scenarios/
│   └── adr/
│
└── scripts/
```

This layout is a target. Empty directories should not be created until they serve an actual implementation need.

---

## Security architecture principles

Sentinel should follow these principles from the start.

### Least privilege

Services should receive only the permissions they require.

### Explicit trust boundaries

Document boundaries between:

- browser and API;
- external producer and ingestion service;
- ingestion and event bus;
- event bus and consumers;
- services and database;
- simulation environment and host;
- analyst actions and privileged administration.

### Input validation

Every externally supplied event, request, identifier, query, and configuration value must be validated.

### Authentication and authorisation

The first local prototype may use simplified authentication, but production-shaped architecture should support:

- authenticated users;
- role-based access control;
- server-side authorisation;
- separation between analyst and administrator actions.

### Secret management

Secrets must never be committed.

Use:

- environment variables for local development;
- secret stores in deployed environments;
- example placeholders in `.env.example`.

### Auditability

Security-sensitive actions should generate immutable or append-only audit events where practical.

### Secure defaults

Services should:

- bind only where necessary;
- reject malformed events;
- use sensible request size limits;
- use timeouts;
- fail closed for authorisation;
- avoid leaking secrets in logs;
- use explicit CORS policy;
- disable debug endpoints outside development.

### Dependency security

CI should eventually include:

- dependency review;
- vulnerability scanning;
- secret scanning;
- static analysis;
- container-image scanning;
- SBOM generation.

---

## Threat model

Sentinel must have its own threat model because a security product can itself become a high-value target.

Initial threats include:

- forged telemetry;
- replayed events;
- event flooding;
- malformed events;
- parser abuse;
- queue exhaustion;
- database exhaustion;
- analyst-session compromise;
- privilege escalation inside the platform;
- malicious detection definitions;
- rule bypass through field manipulation;
- log injection;
- evidence tampering;
- secret leakage;
- vulnerable dependencies;
- compromised collector;
- supply-chain compromise;
- insecure local development defaults.

Threat-model documents should identify:

- assets;
- actors;
- trust boundaries;
- entry points;
- abuse cases;
- controls;
- residual risks.

---

## Engineering standards

The project should optimise for clarity and correctness before cleverness.

Code should:

- use explicit domain models;
- avoid giant utility modules;
- keep service boundaries meaningful;
- separate transport concerns from business logic;
- return structured errors;
- use context cancellation in Go;
- use bounded concurrency;
- use timeouts for external operations;
- avoid hidden global state;
- make dependencies explicit;
- keep functions focused;
- include concise comments where intent is not obvious;
- favour tests over explanatory comments for behavioural guarantees.

---

## Error handling

Errors should be typed or classified where useful.

At minimum, distinguish:

- validation errors;
- authentication errors;
- authorisation errors;
- not-found errors;
- conflict errors;
- transient dependency errors;
- storage errors;
- queue/publish errors;
- internal logic errors.

Do not expose sensitive internals in API responses.

Log enough context to diagnose the problem while avoiding credentials, secrets, full tokens, or unnecessary personal data.

---

## Observability

Every service should eventually expose:

- structured logs;
- health endpoint;
- readiness endpoint;
- metrics;
- traces;
- version/build metadata.

Useful metrics include:

- events received;
- events rejected;
- events normalised;
- queue lag;
- detection evaluation latency;
- findings generated;
- duplicate findings suppressed;
- database query latency;
- HTTP request duration;
- failed authentication count;
- collector heartbeat age.

---

## Testing strategy

### Unit tests

Use for:

- parsers;
- normalisers;
- rule conditions;
- correlation logic;
- scoring functions;
- state transitions;
- utility functions.

### Integration tests

Use for:

- PostgreSQL repositories;
- Redis-backed windows;
- event-bus publishing/consumption;
- service boundaries;
- migrations.

### Contract tests

Ensure producers and consumers agree on schema versions.

### Detection tests

Every detection should have:

- positive fixtures;
- negative fixtures;
- edge cases;
- expected evidence;
- expected severity;
- expected MITRE mapping.

### End-to-end tests

Exercise:

```text
scenario generator
    → ingestion
    → queue
    → detection
    → finding persistence
    → API
    → analyst UI
```

### Security tests

Include:

- malformed payloads;
- oversized requests;
- unauthorised access;
- privilege boundaries;
- replay scenarios;
- dependency scanning;
- secret scanning.

### Performance tests

Measure:

- ingestion throughput;
- detection latency;
- queue lag;
- memory growth;
- database behaviour under sustained event volume.

---

## Synthetic scenario framework

Scenarios should be repeatable and deterministic where possible.

A scenario definition may include:

- name;
- description;
- participating assets;
- participating identities;
- start offset;
- normal background events;
- suspicious event sequence;
- expected detections;
- expected non-detections;
- MITRE mappings;
- seed.

Example scenarios:

1. **Credential misuse and lateral movement**
2. **Unexpected engineering workstation access**
3. **IT-to-OT boundary violation**
4. **Abnormal controller write activity**
5. **Service account used interactively**
6. **Suspicious process and outbound connection**
7. **Maintenance-window policy violation**
8. **Telemetry collector loss**
9. **False-positive baseline scenario**
10. **Multi-stage correlated intrusion simulation**

Scenarios must remain non-destructive and lab-safe.

---

## Data quality

Security analytics are only as useful as the telemetry beneath them.

Sentinel should track:

- missing fields;
- parser failure rate;
- clock skew;
- duplicate events;
- unknown assets;
- unknown identities;
- stale collectors;
- schema versions;
- source reliability;
- event delay;
- dropped events.

Data quality should be visible to analysts rather than silently ignored.

---

## Privacy and data minimisation

Although the project uses synthetic data, its architecture should model responsible handling of real-world security telemetry.

Avoid unnecessary storage of:

- raw secrets;
- authentication tokens;
- content unrelated to the detection goal;
- excessive personal data.

Retention should eventually be configurable by event class.

---

## API design

API endpoints should be versioned once external consumers exist.

Potential resources:

```text
GET    /api/health
GET    /api/v1/findings
GET    /api/v1/findings/{id}
PATCH  /api/v1/findings/{id}
GET    /api/v1/events
GET    /api/v1/assets
GET    /api/v1/assets/{id}
GET    /api/v1/identities
GET    /api/v1/detections
POST   /api/v1/hunts
GET    /api/v1/hunts/{id}
POST   /api/v1/telemetry
```

Ingestion and analyst APIs may eventually be separated.

---

## Database design

Initial domain entities are likely to include:

- `events`
- `assets`
- `identities`
- `detections`
- `detection_versions`
- `findings`
- `finding_events`
- `investigations`
- `analyst_notes`
- `hunt_definitions`
- `hunt_runs`
- `audit_events`
- `collectors`
- `telemetry_sources`

Raw event storage and normalised event storage may be separated later if volume requires it.

---

## Development phases

### Phase 0 — Foundation

- repository structure;
- architecture decision records;
- threat model;
- coding standards;
- CI baseline;
- Docker Compose;
- local PostgreSQL;
- local NATS;
- initial Go service.

### Phase 1 — Telemetry vertical slice

- event schema v1;
- synthetic generator;
- ingestion endpoint;
- validation;
- persistence;
- basic event explorer.

**Exit criteria:** a deterministic simulation event appears in the UI through the complete pipeline.

### Phase 2 — Detection engineering

- detection model;
- rule engine;
- first 5–10 detections;
- fixture-based rule tests;
- finding lifecycle;
- analyst finding queue.

**Exit criteria:** scenario events reliably create explainable findings.

### Phase 3 — Investigation workflow

- investigation page;
- event timeline;
- asset context;
- identity context;
- related-event search;
- analyst notes;
- audit log.

### Phase 4 — Threat hunting

- saved hypotheses;
- hunt execution;
- correlation engine;
- candidate entity ranking;
- evidence timeline.

### Phase 5 — OT simulation

- fictional OT assets;
- synthetic ICS-style telemetry;
- IT/OT boundary rules;
- ATT&CK for ICS mappings;
- operational-safety annotations.

### Phase 6 — Intelligence and enrichment

- local threat-intel fixtures;
- IOC enrichment;
- confidence / provenance model;
- enrichment caching.

### Phase 7 — Behavioural analytics

- baselines;
- anomaly features;
- Python ML service;
- model evaluation;
- analyst-visible explanations;
- entity-scoped score exploration and matching metrics (identity, asset or collector);
- model-isolated score-distribution monitoring, cold-context and freshness signals, live analytics availability and matching synthetic evaluation health;
- five versioned behavioural profiles with per-profile synthetic coverage, known limitations and analyst review guidance;
- a passing combined regression gate: `make final-phase7` (see [completion evidence](docs/architecture/phase-7-completion.md)).

### Phase 8 — Platform hardening

The initial [API authentication/RBAC slice](docs/architecture/phase-8-platform-hardening.md)
is implemented with short-lived lab credentials, verified audit actors and local-only
development compatibility. Interactive console sign-in remains outstanding.

- authentication;
- RBAC;
- rate limiting;
- secure headers;
- signed collector identity;
- replay resistance;
- SBOM;
- dependency scanning;
- container scanning.

### Phase 9 — Production-shaped deployment

- OpenTelemetry;
- Prometheus;
- Grafana;
- load tests;
- failure testing;
- Kubernetes manifests;
- deployment documentation.

---

## Architecture decision records

Important design choices should be documented under `docs/adr/`.

Example ADRs:

- ADR-001 — use Go for core backend services;
- ADR-002 — use NATS JetStream before Kafka;
- ADR-003 — PostgreSQL as system of record;
- ADR-004 — synthetic-only OT environment;
- ADR-005 — deterministic detections before ML;
- ADR-006 — event schema versioning;
- ADR-007 — separate raw telemetry from findings;
- ADR-008 — audit analyst actions.

---

## Coding workflow

Recommended branch naming:

```text
feat/<short-name>
fix/<short-name>
chore/<short-name>
docs/<short-name>
security/<short-name>
refactor/<short-name>
test/<short-name>
```

Examples:

```text
feat/telemetry-ingestion
security/request-size-limits
docs/threat-model
test/detection-fixtures
```

Commit messages should be concise and descriptive:

```text
feat: add telemetry ingestion endpoint
security: reject oversized event payloads
test: cover failed-authentication correlation
docs: define Sentinel trust boundaries
```

---

## Definition of done

A feature is not complete merely because it works locally.

Where applicable, it should include:

- implementation;
- validation;
- error handling;
- tests;
- security review;
- observability;
- documentation;
- migration or schema updates;
- graceful failure behaviour;
- no known dead code;
- no committed secrets.

---

## Initial success criteria

The first meaningful demo should be able to show this story:

1. start the Sentinel development environment;
2. launch a named synthetic scenario;
3. watch events enter the platform;
4. see a detection trigger;
5. open the finding;
6. inspect the evidence timeline;
7. view asset and identity context;
8. see MITRE mapping;
9. change the finding status;
10. verify the action appears in the audit log.

That is a small enough target to build properly while still demonstrating the larger vision.

---

## Example first scenario

### Suspicious engineering workstation access

Normal behaviour:

- engineering workstation communicates with approved OT systems;
- activity occurs during a maintenance window;
- known operator identity is present.

Suspicious behaviour:

- unusual identity logs into the engineering workstation;
- login occurs outside the normal maintenance window;
- workstation initiates a first-seen connection to a controller simulation;
- a controller configuration-change event follows.

Expected Sentinel behaviour:

- identity anomaly contributes context;
- first-seen OT relationship is detected;
- maintenance-window violation is detected;
- events are correlated into a higher-confidence finding;
- analyst receives an ordered evidence timeline;
- relevant ATT&CK / ATT&CK for ICS context is attached.

No real PLC commands are required.

---

## Portfolio value

Sentinel is intentionally designed to demonstrate experience relevant to roles such as:

- Senior Software Engineer;
- Security Engineer;
- Threat Hunter;
- Detection Engineer;
- Security Platform Engineer;
- DevSecOps Engineer;
- Cloud Security Engineer;
- Security Automation Engineer;
- Backend / Distributed Systems Engineer;
- Critical-Infrastructure Cybersecurity Engineer;
- SOC Engineering;
- Threat Intelligence Engineering.

The project should show engineering depth rather than an excessive number of superficial features.

---

## Non-goals

Sentinel is not intended to become:

- a commercial SIEM clone;
- an EDR agent competing with mature endpoint vendors;
- a real industrial-control exploitation framework;
- a vulnerability scanner;
- an automated penetration-testing system;
- an AI chatbot wrapped around logs;
- an opaque "AI detects everything" demo.

The value is in a clear, inspectable defensive-security system.

---

## Documentation map

As implementation grows, documentation should include:

```text
docs/
├── architecture/
│   ├── overview.md
│   ├── event-flow.md
│   └── trust-boundaries.md
├── threat-model/
│   └── THREAT_MODEL.md
├── security/
│   ├── SECURITY_ARCHITECTURE.md
│   └── SECURE_DEVELOPMENT.md
├── detections/
│   ├── DETECTION_MODEL.md
│   └── TESTING.md
├── scenarios/
│   └── SCENARIO_MODEL.md
└── adr/
    └── ...
```

Documentation must evolve with the code. Stale design documents are worse than concise, accurate ones.

---

## Contributing

Sentinel is currently a personal engineering project.

Contributions should preserve:

- defensive scope;
- synthetic or authorised test data;
- secure defaults;
- strong tests;
- clear architecture;
- concise documentation;
- reproducible development.

Do not submit real credentials, private telemetry, or sensitive infrastructure information.

---

## Licence

A licence has not yet been selected.

Until a licence is added, the repository should not be treated as granting broad reuse rights.

---

## Author

**Richard Waters**

Senior Software Engineer focused on secure systems, cybersecurity, backend platforms, cloud infrastructure, and AI-enabled products.
