# Phase 3 — Threat Hunting & Investigation

## Status

In progress on:

```text
feat/phase-3-threat-hunting
```

## Goal

Phase 3 turns Sentinel from a detection pipeline into an analyst-driven threat-hunting and investigation platform.

The first vertical slice adds two connected capabilities:

1. saved hunts with hypotheses and bounded event queries;
2. investigations that preserve findings, events, notes, audit history, and a unified timeline.

## Threat-hunting flow

```text
Analyst hypothesis
      ↓
Saved hunt definition
      ↓
Bounded event query
      ↓
Hunt execution record
      ↓
Relevant events
      ↓
Investigation
```

A hunt is intentionally explicit and reproducible. It stores both the analyst hypothesis and the query used to test it.

## Saved hunts

Saved hunts are stored in PostgreSQL.

A hunt contains:

- stable hunt ID;
- name;
- description;
- hypothesis;
- structured query;
- creator;
- creation/update timestamps.

Current query fields include:

- category;
- action;
- outcome;
- asset ID;
- identity ID;
- source IP;
- destination zone;
- from timestamp;
- to timestamp;
- result limit.

Result limits are bounded to a maximum of 500 events.

## Hunt APIs

```http
GET  /api/v1/hunts
POST /api/v1/hunts
GET  /api/v1/hunts/{id}
POST /api/v1/hunts/{id}/run
```

Creating and executing hunts requires an analyst actor header during the current pre-authentication development phase.

## Hunt execution history

Every successful run records:

- hunt ID;
- actor ID;
- start time;
- completion time;
- result count;
- effective query parameters.

This makes hunts reproducible and gives later UI work enough information to show hunt history and compare results over time.

## Investigations

An investigation provides a durable case around suspicious activity.

An investigation contains:

- stable investigation ID;
- title and description;
- status;
- priority;
- optional owner;
- creator;
- linked findings;
- linked telemetry events;
- analyst notes;
- audit records;
- unified timeline.

## Investigation APIs

```http
GET   /api/v1/investigations
POST  /api/v1/investigations
GET   /api/v1/investigations/{id}
POST  /api/v1/investigations/{id}/notes
PATCH /api/v1/investigations/{id}/status
```

## Investigation workflow

Current states:

```text
open
 ↓
investigating
 ├─→ contained
 │     ├─→ investigating
 │     └─→ closed
 └─→ closed
```

Closed investigations are terminal.

## Investigation timeline

The detail endpoint produces a single chronological timeline containing:

- findings;
- linked telemetry events;
- analyst notes;
- audit events.

The timeline keeps each item type explicit. It does not flatten evidence into ambiguous free text.

This creates a backend foundation for the later analyst investigation UI.

## Audit behaviour

Investigation creation, note creation, and status changes create append-only audit records.

Mutations require `X-Sentinel-Actor` during local development.

As with Phase 2, this is a temporary development mechanism. Production identity must come from authenticated server-side context.

## First Phase 3 verification scenario

The initial Phase 3 smoke test:

1. submits a synthetic successful service-account login;
2. waits for DET-AUTH-002;
3. saves a hunt for that identity and authentication pattern;
4. executes the saved hunt and verifies the event is returned;
5. confirms a hunt-run history record is stored;
6. creates an investigation containing the finding and event;
7. adds an analyst note;
8. moves the investigation from `open` to `investigating`;
9. verifies findings, events, notes, audit records, and the unified timeline.

## Analyst pivots

Phase 3 now supports direct pivots from an asset or identity into its recent telemetry and related findings.

```http
GET /api/v1/assets/{id}/pivot?limit=50
GET /api/v1/identities/{id}/pivot?limit=50
```

Each pivot returns the entity metadata plus:

- recent events;
- findings linked through those events.

The result limit is bounded from 1 to 200.

## Hunt-run history

Every hunt execution now receives a durable run ID and stores the exact event IDs returned by that run.

```http
GET /api/v1/hunts/{id}/runs
```

This means a later investigation can reference the exact historical result set rather than re-running a query against changed telemetry.

The new `hunt_run_events` relationship preserves this membership explicitly.

## Investigation enrichment

An existing investigation can attach the exact events from a previous hunt run:

```http
POST /api/v1/investigations/{id}/hunt-runs/{run_id}
```

This operation is idempotent at the event-link level and creates an append-only audit event recording the run ID and number of newly attached events.

Investigation ownership and priority can also be updated:

```http
PATCH /api/v1/investigations/{id}
```

Supported fields are:

- `owner_id`;
- `priority`.

Setting `owner_id` to an empty string clears the owner. Priority remains constrained to `low`, `medium`, `high`, or `critical`.

Metadata changes are audited transactionally.

## Verification

The Phase 3 pivot smoke test verifies:

- durable hunt-run IDs;
- exact hunt-run event membership;
- hunt-run history retrieval;
- asset pivots;
- identity pivots;
- attaching hunt-run results to an investigation;
- ownership changes;
- priority changes;
- audit records for investigation enrichment.

Run it with:

```bash
make pivots-phase3
```

## Hunt editing and version history

Saved hunts can now be edited without losing their previous definitions.

```http
PATCH /api/v1/hunts/{id}
GET   /api/v1/hunts/{id}/versions
```

Every successful edit:

- increments the hunt version;
- updates the current definition;
- appends an immutable `hunt_versions` record;
- preserves who made the change and when.

Existing hunts are backfilled into version 1 when the migration is applied.

This means hunt tuning remains reviewable instead of silently rewriting analyst intent.

## Richer hunt operators

The typed hunt model now supports both single-value and bounded multi-value filters.

Additional operators include:

- categories;
- actions;
- outcomes;
- destination IP;
- source zones;
- destination zones;
- destination ports;
- exact label matches;
- relative time through `last_minutes`.

Example:

```json
{
  "categories": ["authentication", "network"],
  "outcomes": ["success"],
  "source_zones": ["corporate"],
  "destination_ports": [22, 443],
  "labels": {
    "environment": "lab"
  },
  "last_minutes": 60,
  "limit": 100
}
```

Multi-value fields use OR semantics within the field and AND semantics across different fields.

Relative time is deliberately incompatible with absolute `from`/`to` bounds in one query. This keeps the effective temporal window explicit.

Safety limits include:

- maximum 500 results;
- maximum 20 values for most multi-value fields;
- maximum 50 destination ports;
- maximum 20 label predicates;
- maximum relative window of seven days.

## Hunt and investigation metrics

Phase 3 now exposes operational metrics for the analyst workflow.

```http
GET /api/v1/hunts/metrics
GET /api/v1/investigations/metrics
```

Hunt metrics include:

- hunt count;
- total executions;
- total results;
- per-hunt run count;
- per-hunt total and average result count;
- current hunt version;
- last-run timestamp.

Investigation metrics include:

- total investigations;
- unassigned investigations;
- counts by status;
- counts by priority;
- oldest currently open investigation timestamp;
- latest investigation update timestamp.

These metrics are derived directly from persisted workflow data and are intended for transparent operational visibility rather than opaque scoring.

## Hunt-maturity verification

Run:

```bash
make maturity-phase3
```

The test verifies:

- version-one hunt creation;
- edit to version two;
- append-only version history;
- multi-value filtering;
- destination-port filtering;
- label filtering;
- relative-time filtering;
- hunt metrics;
- investigation metrics.

## Phase 3 roadmap

The first slice deliberately focuses on durable analyst primitives.

Next work should add:

- [x] asset and identity pivot endpoints;
- [x] hunt-run listing/history;
- [x] attaching additional hunt results to an existing investigation;
- [x] investigation ownership changes;
- [x] investigation priority changes;
- [x] saved hunt editing/versioning;
- [x] richer temporal and multi-field hunt operators;
- [x] investigation/hunt metrics;
- [ ] final Phase 3 resilience and regression testing.

The web analyst console should consume these stable APIs rather than invent its own investigation state.
