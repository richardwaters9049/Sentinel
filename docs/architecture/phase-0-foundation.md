# Phase 0 — Foundation

## Status

Implemented on the `feat/phase-0-foundation` branch.

## Goals

Phase 0 establishes the minimum trustworthy platform needed before security telemetry is introduced.

## Implemented

### Toolchain

- Go 1.27.1
- root Go workspace
- Docker Compose
- PostgreSQL 16
- NATS 2.10 with JetStream

### Gateway

- environment-driven configuration;
- structured JSON logging;
- health endpoint;
- dependency-aware readiness endpoint;
- request-size limit;
- baseline response security headers;
- explicit HTTP method handling;
- graceful shutdown;
- startup dependency validation.

### PostgreSQL

- isolated local host port `55432` to avoid collision with the developer's existing PostgreSQL instance;
- pooled pgx connection;
- startup connectivity check;
- embedded, ordered, transactional migrations;
- migration history table;
- initial relational model.

Initial tables:

- `assets`
- `identities`
- `events`
- `detections`
- `findings`
- `finding_events`
- `audit_events`
- `schema_migrations`

### NATS

- client connection lifecycle;
- JetStream capability verification;
- reconnect policy;
- readiness integration;
- graceful drain on shutdown.

### Testing and quality

- configuration tests;
- HTTP handler tests;
- readiness tests;
- `go test ./...`;
- `go vet ./...`;
- race-detector test run;
- repeatable Phase 0 smoke script;
- GitHub Actions baseline CI.

### Documentation

- architecture overview;
- event-flow overview;
- trust boundaries;
- initial threat model;
- ADR for Go;
- ADR for NATS JetStream;
- ADR for PostgreSQL.

## Verified behaviour

The running gateway has been tested against the real local Docker dependencies.

When PostgreSQL and NATS are available:

```json
{
  "dependencies": {
    "nats": "ready",
    "postgres": "ready"
  },
  "ready": true,
  "status": "ready"
}
```

When NATS is deliberately stopped, `/ready` returns HTTP 503 and reports NATS as unavailable while PostgreSQL remains ready.

After NATS restarts, readiness returns to HTTP 200 without restarting the gateway.

## Exit criteria

Phase 0 is complete when:

- [x] current Go toolchain is configured;
- [x] local infrastructure is reproducible;
- [x] gateway starts only with required dependencies;
- [x] database migrations apply automatically and idempotently;
- [x] dependency readiness is observable;
- [x] graceful shutdown is implemented;
- [x] unit tests pass;
- [x] race-detector run passes;
- [x] smoke test proves the local stack;
- [x] foundational architecture and threat model are documented.

## Next phase

Phase 1 introduces the first real security-data vertical slice:

```text
synthetic event
  -> ingestion API
  -> validation
  -> normalisation
  -> NATS JetStream
  -> persistence
  -> event query
```

No detection logic should be added until the event contract and ingestion path are stable.
