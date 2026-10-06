# ADR-0003: Use PostgreSQL as the System of Record

## Status

Accepted.

## Context

Sentinel has strongly related durable entities including assets, identities, detections, findings, investigations, hunts, analyst actions, and audit events.

## Decision

Use PostgreSQL as the primary durable relational store.

## Consequences

Benefits:

- strong transactional semantics;
- relational integrity;
- mature indexing and query tooling;
- JSON support where flexibility is needed.

Trade-offs:

- very high-volume raw telemetry may eventually require a specialised storage strategy;
- schema migrations must be managed carefully.

PostgreSQL remains authoritative for application state even if additional stores are introduced later.
