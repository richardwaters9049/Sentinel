# Phase 2 — Detection Engineering

## Status

In progress on:

```text
feat/phase-2-detection-engineering
```

The first deterministic detection vertical slice is implemented and verified.

## Goal

Turn trusted normalised telemetry into explainable security findings.

The Phase 2 processing path is:

```text
telemetry accepted
      ↓
JetStream
      ↓
event persistence
      ↓
post-persistence detection engine
      ↓
temporal correlation query
      ↓
finding + evidence links
      ↓
GET /api/v1/findings
```

Detection evaluation deliberately happens only after an event has been inserted successfully.

Duplicate events do not run detection processing again.

## First detection

Phase 2 begins with:

```text
DET-AUTH-001
Repeated Authentication Failures Followed by Success
```

Logic:

```text
same identity
+ same source IP
+ 4 failed logins
+ within 5 minutes
+ followed by successful login
= high-severity finding
```

The rule is intentionally deterministic and inspectable.

## Detection registry

The detection metadata is stored in PostgreSQL and seeded through migration `0002_detection_engineering.sql`.

Stored metadata includes:

- stable detection ID;
- version;
- title;
- description;
- severity;
- enabled state;
- structured rule definition;
- MITRE ATT&CK context.

## Correlation

The current engine queries persisted telemetry for the four most recent matching authentication failures before a successful login.

The query is bounded by:

- identity ID;
- source IP;
- authentication category;
- login action;
- failure outcome;
- five-minute time window;
- maximum result count.

A targeted partial database index supports the authentication correlation path.

## Findings

A finding contains:

- stable deterministic ID;
- detection ID and version;
- deduplication key;
- title;
- severity;
- confidence;
- workflow status;
- first/last observed times;
- structured evidence;
- links to all contributing event records.

Finding creation and evidence linking are transactional.

## Deduplication

The database contains a unique partial index on `findings.dedup_key`.

Finding inserts use conflict-safe semantics.

This protects against duplicate findings if processing is retried.

## Findings API

Phase 2 adds:

```http
GET /api/v1/findings
```

Supported filters:

- `limit` — 1 to 200;
- `status`;
- `severity`.

Example:

```http
GET /api/v1/findings?status=new&severity=high&limit=25
```

## Verified first vertical slice

The Phase 2 smoke test submits:

1. failed authentication;
2. failed authentication;
3. failed authentication;
4. failed authentication;
5. successful authentication.

All events use the same synthetic identity and source address.

The test verifies:

- all events are accepted through the real HTTP API;
- JetStream transports them;
- PostgreSQL persists them;
- the detector correlates the preceding failures;
- exactly one `DET-AUTH-001` finding is created;
- the finding contains five evidence-event links;
- the finding is visible through the findings API.

## Phase 2 roadmap

The remaining Phase 2 work should build on the current engine rather than adding unrelated product features.

Planned work:

- [x] finding detail endpoint;
- [x] finding workflow/status transitions;
- [x] append-only audit records for analyst actions;
- [x] detection enable/disable state;
- [x] detection metadata/query API;
- [ ] additional deterministic rules;
- negative scenario fixtures;
- multi-detection regression suite;
- detection quality metrics;
- richer evidence retrieval;
- final Phase 2 resilience and regression testing.

The analyst UI remains a later concern. The backend finding model and workflow should be stable first.
