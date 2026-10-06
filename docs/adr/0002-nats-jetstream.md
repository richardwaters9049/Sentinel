# ADR-0002: Use NATS JetStream as the Initial Event Backbone

## Status

Accepted.

## Context

Sentinel needs durable asynchronous event delivery without making the local development environment unnecessarily heavy.

## Decision

Start with NATS JetStream.

## Rationale

It provides:

- durable streams;
- consumer acknowledgements;
- replay;
- subject-based routing;
- a small operational footprint;
- strong local-development ergonomics.

## Consequences

Kafka-specific features are not available initially. If future scale, partitioning, retention, or ecosystem requirements justify Kafka, this decision should be revisited through a new ADR.
