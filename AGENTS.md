# AGENTS.md

## Purpose

This file defines the operating rules for AI coding agents and automated assistants working in the **Sentinel** repository.

Sentinel is a defensive cyber-threat hunting, detection engineering, and security-analytics platform for a **simulated critical-infrastructure environment**.

Agents must treat this repository as a security-sensitive system. Correctness, safety, auditability, and maintainability take priority over speed.

---

## Project mission

Build a portfolio-grade defensive platform that can:

- ingest synthetic security telemetry;
- normalise events;
- evaluate detections;
- correlate evidence;
- create findings;
- support analyst investigations;
- support threat-hunting hypotheses;
- model enterprise and OT/ICS visibility;
- identify telemetry gaps;
- enrich findings with threat context;
- expose system health and detection quality;
- remain safe, reproducible, and explainable.

The project must demonstrate real engineering depth without becoming an offensive-security toolkit.

---

## Safety boundary

Sentinel is a defensive and simulated system.

Agents MUST NOT add functionality whose primary purpose is to:

- compromise third-party systems;
- steal credentials;
- establish persistence on real systems;
- evade endpoint or network defences;
- deploy malware;
- deliver destructive payloads;
- manipulate real industrial controllers;
- scan unauthorised public targets;
- exfiltrate real data;
- bypass authentication or authorisation on systems not owned by the developer.

Safe attack simulation should use:

- generated events;
- fixture files;
- local containers;
- mocked protocols;
- replay datasets;
- fictional assets;
- explicitly authorised lab systems.

If realism conflicts with safety, prefer the safe simulation.

---

## Working directory

The project root is:

```text
/Users/richy/Documents/Github/Sentinel
```

Before giving the user terminal commands, always state the directory from which the command should be run.

For project-wide commands, use:

```bash
cd /Users/richy/Documents/Github/Sentinel
```

Never assume the user's current working directory.

---

## Repository principles

Agents should preserve these priorities, in order:

1. security;
2. correctness;
3. data integrity;
4. observability;
5. maintainability;
6. testability;
7. performance;
8. developer convenience;
9. visual polish.

Performance optimisation must not weaken validation or correctness.

---

## Target architecture

The long-term architecture is expected to contain:

```text
apps/
  console/                 Next.js / React analyst console

services/
  gateway/                 Go ingestion/API gateway
  detection-engine/        Go deterministic detection engine
  hunt-engine/             Go hunting/correlation workflows
  enrichment/              threat/intelligence enrichment
  ml/                      Python anomaly-analysis service

collectors/
  rust-agent/              optional Rust telemetry collector

packages/
  schemas/                 shared event contracts
  detections/              rule definitions and fixtures
  testdata/                synthetic test datasets

simulator/
  generators/              synthetic event generation
  scenarios/               safe scenario definitions
  assets/                  fictional asset inventory

infra/
  docker/
  kubernetes/
  monitoring/
  migrations/

docs/
  architecture/
  detections/
  threat-model/
  security/
  scenarios/
  adr/

scripts/
```

Do not create empty structure merely to make the repository look complete. Add directories when implementation requires them.

---

## Technology guidance

### Go

Use Go for core backend and event-processing services.

Expected practices:

- standard `context.Context` propagation;
- explicit timeouts;
- graceful shutdown;
- bounded concurrency;
- clear interfaces at real seams;
- constructor-based dependency injection;
- small packages;
- structured errors;
- table-driven tests where suitable;
- avoid global mutable state;
- avoid unnecessary frameworks.

Use the standard library where practical.

### Rust

Rust is intended for low-level collector work where memory safety and a small native runtime are useful.

Do not introduce Rust into a service simply because the project already uses it.

Expected practices:

- no unnecessary `unsafe`;
- explicit error types;
- bounded queues;
- retry/backoff;
- backpressure;
- secure defaults;
- no secret logging.

### TypeScript / Next.js

Use TypeScript strictly.

Expected practices:

- avoid `any`;
- keep server-side authorisation server-side;
- validate external data;
- prefer typed API contracts;
- accessible UI;
- responsive layout;
- reduced-motion support;
- error and loading states;
- do not expose internal secrets through client bundles;
- avoid fetching sensitive data from browser code when a server boundary is more appropriate.

### Python

Python is reserved mainly for ML/data-analysis components.

Expected practices:

- type hints;
- deterministic seeds for tests;
- explicit dependency versions;
- model-evaluation code;
- clear separation between training and inference;
- no unvalidated deserialisation;
- no unsafe loading of arbitrary pickle/joblib artefacts from untrusted sources.

---

## Event architecture rules

Events are core domain objects.

Every event must have:

- stable event identifier;
- timestamp;
- receive time;
- source metadata;
- schema version;
- event category;
- event action;
- outcome where relevant;
- asset context where available;
- actor/identity context where available;
- network context where available;
- source provenance.

Rules:

- preserve original source identifiers where safe;
- do not silently discard malformed events;
- reject or quarantine invalid events explicitly;
- record validation failures;
- make schema versioning visible;
- normalise timestamps to UTC internally;
- do not trust producer clocks;
- support idempotency where ingestion can be retried;
- protect against duplicate event amplification.

---

## Detection-engine rules

Detections must be explainable.

A detection implementation should normally include:

- stable detection ID;
- title;
- description;
- severity;
- confidence rules;
- version;
- MITRE mapping where justified;
- event conditions;
- correlation window;
- grouping key;
- suppression/deduplication behaviour;
- positive test fixtures;
- negative test fixtures;
- edge-case tests.

A finding must contain enough evidence for an analyst to understand why it exists.

Do not create opaque detections that only return a numerical score.

---

## Machine-learning rules

Machine learning is a secondary analytical capability.

Agents must not replace reliable deterministic detections with ML merely to increase perceived sophistication.

Any model added should document:

- problem definition;
- dataset;
- feature set;
- train/validation split;
- metrics;
- baseline;
- failure cases;
- false-positive behaviour;
- false-negative behaviour;
- reproducibility;
- model version;
- inference contract.

ML output should be presented as supportive evidence, not absolute truth.

---

## OT / ICS rules

Operational-technology simulation requires additional caution.

Agents must assume:

- availability may be more important than immediate remediation;
- normal enterprise response actions may be inappropriate;
- device behaviour may be legacy or constrained;
- active probing can be unsafe in real environments.

Therefore:

- prefer passive/synthetic telemetry;
- never add automatic disruptive remediation;
- never add real controller-write logic;
- use simulation fixtures for configuration changes;
- label all OT demonstrations as simulated.

---

## Authentication and authorisation

When authentication is introduced:

- use server-side authorisation;
- deny by default;
- separate analyst and administrative privileges;
- audit privileged actions;
- avoid long-lived credentials;
- never trust client-provided role claims without verification.

Authentication state must not be stored only in UI state.

---

## Secret handling

Never commit:

- passwords;
- API tokens;
- private keys;
- access keys;
- JWT signing secrets;
- database production credentials;
- cloud credentials;
- personal access tokens.

Use environment variables and documented placeholders.

Any `.env.example` must contain fake values only.

If a secret is discovered in repository history, do not merely delete it from the latest file; advise rotation and history remediation.

---

## Logging

Logs must be structured.

Never log:

- passwords;
- session secrets;
- bearer tokens;
- full API keys;
- private keys;
- unnecessary personal information.

Useful log fields may include:

- service;
- version;
- request ID;
- trace ID;
- event ID;
- detection ID;
- finding ID;
- asset ID;
- duration;
- outcome;
- error class.

Do not log entire event payloads by default.

---

## Error handling

Do not swallow errors.

Errors should be categorised where useful:

- validation;
- authentication;
- authorisation;
- not found;
- conflict;
- dependency unavailable;
- persistence;
- queue;
- timeout;
- internal.

User-facing API responses must avoid leaking sensitive internals.

Internal logs should preserve sufficient context for debugging.

---

## Database rules

For schema changes:

- use migrations;
- make destructive migrations explicit;
- prefer backward-compatible rollout where possible;
- index based on query patterns, not guesswork;
- use transactions for multi-step consistency boundaries;
- avoid unbounded queries;
- paginate list endpoints;
- test migrations.

Do not delete historical security evidence casually.

Retention should be explicit.

---

## Queue / event-bus rules

Consumers should be designed for:

- retries;
- duplicate delivery;
- temporary downstream failure;
- graceful shutdown;
- backpressure;
- dead-letter or quarantine handling when justified.

Do not assume exactly-once delivery.

Use idempotent processing where feasible.

---

## API rules

APIs should:

- validate inputs;
- return consistent error shapes;
- use appropriate HTTP status codes;
- have explicit timeouts;
- apply request-size limits;
- paginate large responses;
- avoid leaking internal database structure;
- use stable resource IDs;
- document authentication requirements;
- version public contracts when breaking changes become possible.

---

## Frontend design rules

The analyst console should look like a serious operational security tool, not a generic template.

Priorities:

- information hierarchy;
- scanability;
- evidence clarity;
- dense but readable data;
- consistent severity representation;
- useful empty states;
- meaningful loading states;
- keyboard accessibility;
- responsive behaviour;
- reduced-motion support.

Animations should support comprehension, not distract from data.

Avoid decorative motion in critical tables and incident workflows.

---

## Testing requirements

A change is not complete without appropriate tests.

### Go

Use:

- unit tests;
- table-driven tests;
- integration tests for repositories/queues;
- race detector where practical.

### TypeScript

Use:

- component tests for important interaction;
- API/client contract tests;
- end-to-end tests for critical analyst flows.

### Rust

Use:

- unit tests;
- property tests where parsers benefit;
- integration tests for queueing/buffering behaviour.

### Python

Use:

- unit tests;
- dataset validation;
- deterministic evaluation tests.

### Detection rules

Every rule must include:

- at least one positive case;
- at least one negative case;
- meaningful edge cases.

---

## Security testing

Where relevant, test:

- malformed JSON;
- oversized input;
- invalid enum values;
- missing required fields;
- replayed events;
- duplicate events;
- unauthenticated requests;
- unauthorised requests;
- path traversal attempts;
- injection payloads;
- invalid identifiers;
- queue flooding behaviour;
- timeout behaviour.

Tests must remain safe and local.

---

## Performance rules

Do not make performance claims without measurement.

When optimising:

1. establish a baseline;
2. identify the bottleneck;
3. implement the change;
4. rerun measurement;
5. document material trade-offs.

For ingestion, useful measures include:

- events/sec;
- p50/p95/p99 latency;
- memory usage;
- queue lag;
- database write latency;
- rejection rate.

---

## Observability requirements

Services should eventually expose:

- structured logs;
- health endpoint;
- readiness endpoint;
- metrics;
- distributed traces;
- build version.

Agent-created background workers must expose enough telemetry to diagnose stalled or failed processing.

---

## Documentation rules

Update documentation whenever a change invalidates existing docs.

Important architecture decisions belong in ADRs.

Security-sensitive changes should document:

- changed trust boundary;
- threat addressed;
- assumptions;
- residual risk.

Do not allow `README.md` to become inaccurate marketing copy.

---

## Code comments

Comments should be short and useful.

Good comments explain:

- why a security check exists;
- non-obvious invariants;
- concurrency assumptions;
- protocol quirks;
- intentional trade-offs.

Avoid comments that simply repeat code.

---

## Refactoring rules

When refactoring:

- preserve behaviour with tests;
- remove dead code;
- remove unused dependencies;
- avoid creating abstraction layers with only one trivial implementation;
- reduce duplication where it improves clarity;
- prefer explicit code over overly generic helpers.

Do not mix a large refactor with unrelated feature work unless necessary.

---

## Dependency policy

Before adding a dependency, ask:

- can the standard library reasonably handle this?
- is the project maintained?
- is the licence appropriate?
- does it create a large transitive dependency tree?
- does it meaningfully simplify the implementation?
- does it introduce native/build complexity?

Security-sensitive dependencies should be pinned and scanned.

---

## Git workflow

Preferred branch names:

```text
feat/<name>
fix/<name>
security/<name>
docs/<name>
test/<name>
refactor/<name>
chore/<name>
```

Preferred commit style:

```text
feat: add event ingestion
fix: prevent duplicate finding creation
security: validate collector identity
docs: document event trust boundary
test: add lateral-movement fixtures
refactor: isolate finding repository
chore: update development tooling
```

Do not commit generated secrets, local databases, build artefacts, `node_modules`, Python virtual environments, IDE caches, or OS metadata.

---

## Before committing

Run the checks relevant to the files changed.

Typical future command set:

```bash
go test ./...
go vet ./...
go test -race ./...
cargo test
cargo clippy --all-targets --all-features -- -D warnings
bun run lint
bun run test
bun run build
pytest
```

Do not invent commands before the corresponding tooling exists.

---

## CI expectations

CI should eventually include:

- formatting;
- linting;
- unit tests;
- integration tests;
- build;
- secret scanning;
- dependency scanning;
- code scanning;
- container scanning;
- SBOM generation.

CI must fail on meaningful test or security failures rather than hiding them.

---

## Definition of done

For a normal feature, confirm as applicable:

- code implemented;
- validation added;
- error cases handled;
- tests added;
- security impact reviewed;
- logs/metrics considered;
- docs updated;
- dead code removed;
- formatting/linting clean;
- no secrets committed.

---

## Agent behaviour

Agents working on Sentinel should:

- inspect existing code before modifying it;
- understand the current architecture before introducing new patterns;
- prefer small, reviewable changes;
- keep changes scoped to the request;
- preserve working behaviour unless the change intentionally replaces it;
- explain meaningful architectural choices;
- report tests actually run;
- distinguish tests run from tests merely recommended;
- never claim a command passed without executing it;
- never claim a file exists without verifying it;
- never claim deployment succeeded without verifying the result;
- avoid generated boilerplate that adds little value.

---

## File editing

When changing a file, first read the relevant current version.

Avoid destructive rewrites unless the file genuinely needs replacement.

If an automated edit touches security-sensitive code, inspect the resulting diff.

Do not allow formatters or generators to create unrelated large diffs without a reason.

---

## User communication

When giving terminal commands:

1. state the directory first;
2. use commands suitable for macOS/zsh unless the environment indicates otherwise;
3. keep commands copyable;
4. explain destructive commands before execution;
5. do not hide important side effects.

For this project, default directory:

```bash
cd /Users/richy/Documents/Github/Sentinel
```

---

## Initial implementation priorities

Agents should favour this build order unless requirements change:

1. repository foundation;
2. architecture documentation;
3. event schema;
4. PostgreSQL + migrations;
5. NATS development environment;
6. Go ingestion service;
7. synthetic event generator;
8. event persistence;
9. first deterministic detection;
10. finding model;
11. minimal analyst console;
12. first complete scenario;
13. correlation;
14. investigation timeline;
15. authentication/RBAC;
16. deeper hardening;
17. ML features;
18. Kubernetes/deployment complexity.

This order is intended to produce demonstrable vertical slices early.

---

## First vertical slice

The first end-to-end milestone should be:

```text
Synthetic event
      ↓
Go ingestion endpoint
      ↓
Validation + normalisation
      ↓
NATS
      ↓
Detection engine
      ↓
PostgreSQL finding
      ↓
Go query API
      ↓
Next.js finding view
```

The first slice should be small enough to understand completely.

---

## Suggested first detection

A useful first detection is:

**Repeated authentication failure followed by a successful login from the same identity within a short window.**

Why:

- simple synthetic telemetry;
- demonstrates temporal correlation;
- supports positive and negative fixtures;
- produces clear evidence;
- does not require OT-specific behaviour to establish the architecture.

A later scenario can build from that identity activity into simulated lateral movement.

---

## Long-term quality bar

Sentinel should eventually be credible enough that a security engineer reviewing the repository can see evidence of:

- intentional architecture;
- secure engineering;
- threat modelling;
- detection engineering;
- distributed-systems knowledge;
- event-driven design;
- observability;
- testing discipline;
- operational awareness;
- documented trade-offs.

Do not trade that quality bar for feature count.
