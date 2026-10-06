# Sentinel Threat Model

## Status

Initial threat model for Phase 0. This document must evolve with the architecture.

## Security objectives

Sentinel should protect:

- integrity of telemetry;
- integrity of findings and investigations;
- confidentiality of secrets;
- analyst authentication state;
- availability of ingestion and analysis;
- audit history;
- detection definitions;
- provenance of simulated and future collected events.

## Assets

- event data;
- finding data;
- detection rules;
- analyst notes;
- service credentials;
- database credentials;
- event-bus credentials;
- audit records;
- configuration;
- source code and CI pipeline.

## Trust assumptions

Trusted:

- the authorised developer;
- controlled CI;
- approved local simulation components.

Partially trusted or untrusted:

- incoming telemetry;
- browser/client input;
- malformed fixture data;
- external dependencies;
- future collectors until authenticated.

## Initial risks and controls

### Fabricated or replayed telemetry

Risk: invalid or repeated events could distort findings.

Controls planned:

- stable event IDs;
- deduplication;
- receive-time tracking;
- source provenance;
- authenticated producers;
- idempotent consumers.

### Excessive event volume

Risk: memory, queue capacity, database connections, or disk may be exhausted.

Controls:

- request-size limits;
- bounded concurrency;
- queue backpressure;
- rate controls;
- metrics;
- retention policies.

### Malformed input

Risk: bad data may break parsers or downstream processing.

Controls:

- typed schemas;
- strict validation;
- explicit rejection;
- safe parsing;
- focused tests.

### Privilege misuse

Risk: a user or service could perform actions outside its intended role.

Controls planned:

- deny-by-default RBAC;
- server-side authorisation;
- service-specific credentials;
- auditable privileged actions.

### Evidence integrity

Risk: findings or related evidence could be modified incorrectly.

Controls planned:

- stable identifiers;
- controlled state transitions;
- append-oriented audit history;
- restricted update paths.

### Secret leakage

Risk: credentials or keys may leak through source control, logs, crashes, or client bundles.

Controls:

- no committed secrets;
- environment-based local configuration;
- log minimisation;
- future CI secret scanning;
- server-only secret handling.

### Dependency and build risk

Risk: third-party dependencies, images, or CI components may introduce vulnerabilities.

Controls planned:

- minimal dependencies;
- pinned versions where practical;
- dependency review;
- code scanning;
- container scanning;
- SBOM generation.

### Service availability

Risk: expensive requests or dependency failures may reduce availability.

Controls planned:

- timeouts;
- bounded workloads;
- pagination;
- health/readiness probes;
- queue controls;
- graceful shutdown.

## OT simulation safety

Sentinel uses synthetic OT telemetry and does not require direct access to real industrial equipment.

No project feature should depend on disruptive interaction with a real controller or unauthorised third-party system.

## Phase 0 residual risk

The current environment is development-only. PostgreSQL and NATS use local development configuration and do not yet implement production authentication, TLS, network segmentation, or full secret-management integration.

Those limitations are acceptable only for local development and must be revisited before any non-local deployment.
