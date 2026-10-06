# Sentinel Architecture Overview

## Purpose

Sentinel is a defensive threat-hunting and detection-engineering platform built around synthetic security telemetry. The architecture should support a small local lab first and scale conceptually toward production-shaped distributed services.

## Architectural priorities

1. Security and explicit trust boundaries.
2. Explainable detections.
3. Reliable event processing.
4. Testability.
5. Observability.
6. Incremental complexity.

## Phase 0 components

The initial foundation contains:

- a Go gateway service;
- PostgreSQL for durable state;
- NATS JetStream for event transport;
- Docker Compose for local infrastructure;
- CI for formatting, vetting, and tests.

No frontend, ML service, or Rust collector is required in Phase 0.

## Service boundaries

### Gateway

Responsibilities:

- expose HTTP endpoints;
- apply request limits and baseline security headers;
- host health/readiness endpoints;
- later receive telemetry and analyst API requests.

The gateway must not absorb detection logic or persistence details as the system grows.

### NATS JetStream

NATS is the initial event backbone. It is chosen to keep local development lightweight while still allowing durable streams, consumer groups, retries, and decoupled services.

### PostgreSQL

PostgreSQL will be the system of record for durable application state such as assets, identities, findings, detection metadata, hunts, investigations, and audit records.

## Growth path

The next service boundaries are expected to be:

- telemetry ingestion and normalisation;
- detection engine;
- hunt and correlation engine;
- enrichment;
- analyst console;
- optional anomaly-analysis service;
- optional native collector.

Services should only be split when there is a real responsibility or operational boundary.
