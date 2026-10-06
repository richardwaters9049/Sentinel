# Phase 2 — Detection Engineering

## Status

Complete on:

```text
feat/phase-2-detection-engineering
```

The multi-detection framework, analyst workflow, first three deterministic detections, quality metrics, contextual evidence retrieval, and final resilience/regression suite are implemented and verified.

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

## Multi-detection engine

The detection engine now operates over a registry of independent rule implementations rather than one hard-coded detector.

Each rule:

- exposes a stable detection ID;
- evaluates one normalised event;
- may query bounded historical context when required;
- returns either no match or one explainable finding;
- is checked against persisted enabled/disabled state before evaluation.

The current registry contains:

```text
DET-AUTH-001  Repeated Authentication Failures Followed by Success
DET-AUTH-002  Interactive Login Using a Service Account
DET-NET-001   Unexpected Corporate-to-OT Network Connection
```

This allows stateless single-event detections and stateful temporal detections to coexist behind the same engine contract.

### DET-AUTH-002

`DET-AUTH-002` detects a successful interactive login where `actor.type` is `service_account`.

It is a single-event rule and does not require a historical query.

### DET-NET-001

`DET-NET-001` detects a `network/connection` event where:

```text
source asset zone      = corporate
destination network zone = ot
```

The telemetry contract now carries an optional `network.destination_zone` field to support explicit boundary-aware reasoning.

No ATT&CK technique is assigned to this rule yet because a zone-crossing connection alone is not sufficient evidence for a specific adversary technique.

### Runtime control

Every rule checks its persisted detection state before evaluation.

Disabling one rule does not disable unrelated detections.

The engine regression suite verifies independent per-rule enable state and prevents an unrelated rule from invoking unnecessary historical correlation queries.

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

## Detection quality metrics

Phase 2 exposes derived quality data through:

```http
GET /api/v1/detections/metrics
```

Metrics are calculated from persisted findings and currently include:

- total hit count;
- currently open finding count;
- currently confirmed count;
- false-positive count;
- closed count;
- false-positive rate;
- last-triggered timestamp.

The false-positive rate is:

```text
false-positive findings / total findings
```

These metrics are deliberately transparent database-derived values rather than opaque scoring.

They provide the first feedback loop for tuning rules as scenario coverage grows.

## Contextual evidence retrieval

Finding detail already returns the events directly linked to a finding.

Phase 2 now also supports bounded surrounding context:

```http
GET /api/v1/findings/{id}/evidence?context_minutes=5
```

The response separates:

- `linked_events` — evidence that directly caused or contributed to the finding;
- `context_events` — nearby telemetry associated with the same assets or identities;
- `context_minutes` — the requested window.

The context window is bounded from 0 to 60 minutes and context results are capped.

Direct evidence is never mixed silently with contextual evidence. This distinction is important for analyst reasoning and later explainability work.

## Final resilience and regression pass

The final Phase 2 regression command is:

```bash
make final-phase2
```

It verifies:

1. formatting, vet, unit tests, and Go race detection;
2. NATS outage and automatic recovery;
3. PostgreSQL outage, JetStream buffering, and redelivery;
4. DET-AUTH-001 temporal correlation;
5. analyst status workflow and append-only audit history;
6. runtime detection disable/re-enable behaviour;
7. DET-AUTH-002 and DET-NET-001 end-to-end;
8. contextual evidence retrieval;
9. detection quality metrics.

This suite deliberately reuses the Phase 1 resilience checks because Phase 2 consumes the same durable event path. A detection system cannot be considered reliable if its underlying event delivery is not reliable.

## Phase 2 roadmap

The remaining Phase 2 work should build on the current engine rather than adding unrelated product features.

Planned work:

- [x] finding detail endpoint;
- [x] finding workflow/status transitions;
- [x] append-only audit records for analyst actions;
- [x] detection enable/disable state;
- [x] detection metadata/query API;
- [x] additional deterministic rules;
- [x] negative rule fixtures for non-matching behaviour;
- [x] multi-detection regression suite;
- [x] detection quality metrics;
- [x] richer evidence retrieval;
- [x] final Phase 2 resilience and regression testing.

Phase 2 is complete.

The next phase can build analyst-facing product surfaces and deeper hunting/correlation capabilities on top of a tested event, detection, finding, evidence, and audit foundation.
