# Sentinel Threat Model

## Status

Initial threat model, updated for the first Phase Eight API access-control slice.

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

Implemented in required authentication mode:

- explicit route permissions and denial of routes without a policy;
- gateway-verified expiring API credentials with server-provisioned roles;
- collector, analyst and administrator separation;
- authenticated actor attribution in domain audits and structured access-decision logs.

Interactive sign-in now uses hashed PostgreSQL sessions, bounded absolute/idle expiry,
login rotation, logout revocation, HttpOnly/SameSite cookies and exact-origin/CSRF checks.

Residual risks: federation/MFA, immediate credential-registry revocation and durable
access-log retention are outstanding. Development compatibility trusts loopback callers, including
local proxies. Manifest/host operators remain trusted. These controls do not establish
signed event-source provenance or per-investigation ownership enforcement.
See [ADR 0005](../adr/0005-api-authentication.md).

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
