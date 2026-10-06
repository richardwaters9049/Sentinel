# Telemetry Failure Semantics

## Purpose

Phase 1 establishes explicit behaviour for dependency failures so that accepted telemetry is not silently lost and callers can distinguish an unavailable ingestion path from a delayed persistence path.

## Acceptance boundary

The HTTP ingestion endpoint returns **202 Accepted** only after NATS JetStream acknowledges the normalised event publish.

This is Sentinel's current acceptance boundary.

A 202 therefore means:

- the request passed validation;
- the event was normalised;
- JetStream accepted the message.

It does **not** mean PostgreSQL has already persisted the event.

## NATS unavailable

When NATS is unavailable:

- `/ready` returns HTTP 503;
- PostgreSQL may still report ready;
- telemetry publish cannot reach the acceptance boundary;
- `POST /api/v1/telemetry` returns HTTP 503 with `publish_failed`;
- the API must not claim the event was accepted.

The NATS client uses unlimited reconnect attempts with a one-second reconnect interval after an established connection is lost.

Once NATS returns, the existing gateway connection recovers without requiring a process restart.

## PostgreSQL unavailable

When PostgreSQL is unavailable but NATS remains available:

- `/ready` returns HTTP 503;
- telemetry can still be accepted into JetStream;
- `POST /api/v1/telemetry` can return HTTP 202;
- the persistence consumer fails the database write;
- the message is negatively acknowledged;
- JetStream redelivers the event;
- PostgreSQL connection pooling reconnects when the database returns;
- the event is persisted after recovery.

This separates durable ingestion from database availability.

A deployment platform may choose not to route new requests to an instance whose readiness endpoint is failing. The underlying event pipeline nevertheless preserves already accepted messages.

## Duplicate delivery

JetStream is an at-least-once delivery system.

Sentinel therefore assumes duplicate delivery can happen.

Two layers currently protect event identity:

1. the event ID is used as the JetStream message ID, enabling duplicate suppression inside the configured JetStream duplicate window;
2. PostgreSQL uses `events.id` as a primary key and event insertion uses `ON CONFLICT DO NOTHING`.

The persistence path is therefore idempotent for repeated delivery of the same event ID.

## Transient consumer failures

Examples:

- PostgreSQL temporarily unavailable;
- transaction timeout;
- temporary connection failure.

Transient errors are negatively acknowledged with a delay and remain eligible for redelivery.

The Phase 1 consumer does not impose a finite delivery-attempt limit because a database outage should not permanently discard accepted telemetry.

## Permanent consumer failures

Examples:

- malformed payload already present on the internal telemetry subject;
- unsupported telemetry schema version.

These failures cannot be fixed by repeatedly retrying the same message.

The persistence handler marks them as permanent. The JetStream consumer terminates delivery of that message instead of creating an infinite retry loop.

A later phase should add an explicit quarantine/dead-letter stream with operator visibility. Phase 1 terminates poison-message delivery but logs the failure.

## Process shutdown

On controlled shutdown:

- HTTP serving stops through `http.Server.Shutdown`;
- the telemetry subscription is unsubscribed;
- the NATS connection drains;
- the database pool closes.

Context and timeout boundaries prevent shutdown from waiting indefinitely.

## Verified Phase 1 behaviour

The local resilience test verifies:

- duplicate event IDs result in one PostgreSQL event;
- NATS outage causes readiness failure and ingestion rejection;
- the NATS connection recovers after the server restarts;
- PostgreSQL outage causes readiness failure;
- JetStream still accepts telemetry while PostgreSQL is unavailable;
- queued telemetry is persisted after PostgreSQL restarts.

Run from the project root:

```bash
make resilience-phase1
```

The test intentionally stops and restarts the local Sentinel NATS and PostgreSQL containers.
