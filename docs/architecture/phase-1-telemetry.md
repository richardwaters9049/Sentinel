# Phase 1 — Telemetry Vertical Slice

## Status

Complete on:

```text
feat/phase-1-telemetry
```

## Goal

Create the first complete security-data path through Sentinel.

```text
Synthetic telemetry
      ↓
POST /api/v1/telemetry
      ↓
validation + normalisation
      ↓
NATS JetStream
      ↓
durable persistence consumer
      ↓
PostgreSQL
      ↓
GET /api/v1/events
```

## Implemented in the first Phase 1 slice

### Versioned event contract

- schema version `1.0.0`;
- machine-readable JSON Schema;
- typed Go domain model;
- cryptographically random event IDs;
- canonical timestamps;
- bounded labels;
- IP and port validation;
- future-clock protection;
- normalised categorical fields.

### Ingestion API

```text
POST /api/v1/telemetry
```

Behaviour:

- strict JSON decoding;
- unknown-field rejection;
- one event per request;
- request size bounded by the gateway;
- validation failures return HTTP 400;
- accepted events return HTTP 202;
- successful response occurs only after JetStream acknowledges the publish.

### Event transport

JetStream configuration:

```text
stream:  SENTINEL_TELEMETRY
subject: sentinel.telemetry.v1
storage: file
max age: 24 hours
```

The event ID is supplied as the JetStream message ID.

### Durable persistence

Consumer:

```text
sentinel-event-store-workers-v1
```

The consumer:

- uses explicit acknowledgements;
- acknowledges only after persistence succeeds;
- negatively acknowledges failed persistence attempts;
- applies bounded processing time;
- supports idempotent event insertion.

### Database behaviour

On event persistence Sentinel:

1. upserts asset context when supplied;
2. upserts identity context when supplied;
3. updates first/last-seen times;
4. inserts the normalised event;
5. commits the operation transactionally.

### Event query

```text
GET /api/v1/events
```

Filters currently include:

- limit;
- before timestamp;
- category;
- asset ID;
- identity ID.

### Synthetic simulator

A separate Go module now exists under:

```text
simulator/
```

Current scenarios:

- `normal-login`
- `auth-burst`

`auth-burst` emits four failed authentication events followed by a successful login from the same synthetic source.

This is intentionally safe synthetic telemetry. It will later become the first useful input for deterministic detection engineering.

## Verified end-to-end behaviour

The first real vertical slice has been manually verified:

1. gateway readiness reported PostgreSQL and NATS ready;
2. simulator emitted five authentication events;
3. every event received HTTP 202;
4. JetStream delivered the events to the durable consumer;
5. PostgreSQL stored all five events;
6. `GET /api/v1/events?category=authentication` returned the complete normalised records.

This proves the first meaningful Sentinel data path works across process boundaries.

## Operational failure semantics

### NATS unavailable

If NATS is unavailable, the gateway remains alive but `/ready` returns HTTP 503.

Telemetry ingestion also returns HTTP 503 because Sentinel cannot truthfully acknowledge an event that has not been accepted by JetStream.

The NATS client is configured to reconnect indefinitely, so readiness and ingestion recover automatically when the broker returns.

### PostgreSQL unavailable

If PostgreSQL becomes unavailable after startup, `/ready` returns HTTP 503.

Telemetry ingestion may still return HTTP 202 while NATS is healthy because JetStream has durably accepted the event. The persistence consumer negatively acknowledges transient database failures so JetStream can redeliver the event after PostgreSQL recovers.

Database insertion is idempotent on `events.id`, preventing redelivery from creating duplicate records.

### Duplicate telemetry

The event ID is used both as the JetStream message ID and as the PostgreSQL primary key.

Within the JetStream duplicate window, repeated publishes using the same event ID are suppressed by the broker. If a duplicate reaches the database outside that window, `ON CONFLICT DO NOTHING` still prevents duplicate event rows.

### Permanently invalid queued messages

Malformed queued telemetry or an unsupported schema version is treated as a permanent consumer failure. Such a message is terminated rather than retried indefinitely.

Transient storage failures remain retryable.

### Gateway restart

Accepted telemetry survives a gateway restart because the event is stored in JetStream before the ingestion endpoint returns HTTP 202. The durable consumer can resume delivery after the service reconnects.

## Phase 1 remaining work

Before Phase 1 is considered complete:

- [x] define event schema v1;
- [x] implement normalisation;
- [x] implement validation;
- [x] create ingestion endpoint;
- [x] publish through JetStream;
- [x] create durable persistence consumer;
- [x] persist assets, identities, and events;
- [x] query stored events;
- [x] create synthetic telemetry generator;
- [x] unit-test event, service, and HTTP behaviour;
- [x] add repeatable Phase 1 integration smoke test to the normal verification workflow;
- [x] verify duplicate event handling end to end;
- [x] verify NATS interruption and recovery during telemetry ingestion;
- [x] verify PostgreSQL interruption and redelivery behaviour;
- [x] document operational failure semantics;
- [x] complete final Phase 1 regression pass.

## Non-goals for Phase 1

Do not add detection rules yet.

Do not add machine learning yet.

Do not build the analyst console yet.

Phase 1 should end with a trustworthy event pipeline. Detection engineering belongs to Phase 2.
