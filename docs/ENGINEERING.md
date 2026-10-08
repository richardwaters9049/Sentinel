# Engineering and contribution guide

These principles apply to work on Sentinel. Some observability, scanning and
deployment controls are longer-term requirements rather than completed features.
[AGENTS.md](../AGENTS.md) contains the detailed repository operating rules.
The [threat model](threat-model/THREAT_MODEL.md),
[trust boundaries](architecture/trust-boundaries.md) and
[platform-hardening document](architecture/phase-8-platform-hardening.md) record
implemented controls and residual risks.

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

Required mode now provides credential authentication and console sessions. Preserve:

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
- stateful correlation windows;
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

## Database design

The original domain design includes the following concepts. This is not an exact table
inventory; the [gateway migrations](../services/gateway/internal/database/migrations/)
are the authoritative schema:

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
