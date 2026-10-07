# Phase 6 — Intelligence and Enrichment

## Status

Complete on:

```text
feat/phase-6-intelligence-enrichment
```

## Objective

Phase 6 adds provenance-aware threat intelligence to Sentinel without treating external or local reputation as unquestioned truth.

The implementation is local-first and deterministic. Repository-owned synthetic fixtures keep development, demonstrations, and regression testing independent of third-party feeds.

The completed pipeline is:

```text
persisted telemetry
      ↓
IOC candidate extraction
      ↓
strict normalisation
      ↓
revision-aware bounded cache
      ↓
active + valid indicator lookup
      ↓
confidence calculation
      ↓
persisted enrichment
      ↓
findings / investigations / analyst console
```

## Intelligence storage

Phase 6 introduces:

```text
threat_intel_sources
threat_indicators
event_enrichments
```

Sources retain:

- stable ID and source type;
- default confidence;
- active/inactive runtime state;
- provenance metadata;
- timestamps used to invalidate enrichment-cache entries after source changes.

Indicators retain:

- type: IP, domain, or SHA-256;
- original and normalised value;
- source;
- confidence;
- validity window;
- tags and contextual classification.

Event enrichments retain a snapshot of the intelligence used at match time:

- event and indicator IDs;
- matched event field and observed value;
- source confidence;
- indicator confidence;
- effective confidence;
- source provenance;
- match timestamp.

The uniqueness boundary is:

```text
(event_id, indicator_id, event_field)
```

so retries remain idempotent.

## Local intelligence catalogue

The completed phase uses two synthetic sources:

```text
intel-local-northstar
Northstar Local Threat Intelligence
default confidence = 85

intel-local-review
Northstar Analyst Review Feed
default confidence = 70
```

Both are repository-owned local fixtures with no external dependency.

The catalogue contains safe fixtures for:

- documentation-only external IP addresses;
- a synthetic internal watchlist IP;
- a reserved `.example` domain;
- a synthetic SHA-256 value;
- an intentionally expired indicator.

The same IP/domain can appear in more than one source so Sentinel can preserve disagreement instead of collapsing conflicting intelligence into one truth value.

## IOC matching

The enrichment engine evaluates:

```text
network.source_ip
network.destination_ip
labels.dns.query
labels.file.sha256
```

Normalisation is type-specific:

- IP values use Go's IP parser;
- domains are lower-cased, trailing-dot normalised, and structurally validated;
- SHA-256 values must be exactly 64 hexadecimal characters and are normalised to lower case.

Invalid candidates are ignored rather than queried.

## Confidence model

Sentinel preserves separate confidence dimensions:

```text
source confidence
indicator confidence
effective confidence
```

Effective confidence is:

```text
round(source_confidence × indicator_confidence / 100)
```

For example:

```text
source confidence     = 85
indicator confidence  = 95
effective confidence  = 81
```

Different sources can classify the same IOC differently. The analyst console surfaces this as source disagreement rather than silently choosing one source.

## Validity and runtime source state

Runtime lookups require:

- source `active = true`;
- `valid_from <= observed event time`;
- no `valid_until`, or the indicator remains valid for the event time.

The analyst indicator catalogue also excludes expired indicators using the current time.

Threat-intelligence sources can be enabled or disabled through:

```text
GET   /api/v1/intelligence/sources
PATCH /api/v1/intelligence/sources/{id}
```

State changes:

- require the temporary development actor header;
- reject unchanged state;
- use a row-locked transaction;
- write `intelligence_source.enabled_changed` audit events;
- update the source revision so cached results are invalidated immediately.

## Enrichment cache

The enrichment engine uses a bounded in-memory cache:

- five-minute TTL;
- 1,024-entry maximum;
- positive and negative results;
- key = indicator type + normalised value;
- each entry stores the intelligence-source revision.

A cache hit requires both a valid TTL and the same intelligence revision. Source-state changes therefore invalidate cached IOC results without waiting for TTL expiry.

Unit and integration tests cover positive caching, negative caching, and revision-triggered invalidation.

Detailed cache hit/miss telemetry is intentionally deferred to the broader observability/platform work rather than added as Phase 6 product scope.

## Telemetry processing

Persisted telemetry passes through an ordered processor chain:

```text
event persisted
      ↓
deterministic detection
      ↓
threat-intelligence enrichment
```

A processor failure stops the chain and preserves the existing JetStream retry behaviour.

## Analyst APIs

Phase 6 provides:

```text
GET   /api/v1/intelligence/indicators
GET   /api/v1/intelligence/sources
PATCH /api/v1/intelligence/sources/{id}
GET   /api/v1/intelligence/matches
GET   /api/v1/intelligence/metrics
GET   /api/v1/events/{id}/enrichments
```

Indicator queries are bounded and support optional type/value filters.

Metrics include:

- active sources;
- active indicators;
- enriched events;
- total enrichment matches;
- high-confidence matches.

## Finding and investigation integration

Threat intelligence is now part of the analyst workflow rather than isolated on a separate page.

Finding detail includes enrichments attached to linked evidence events.

Finding evidence context includes enrichments across both:

- direct linked evidence;
- nearby contextual evidence returned by the bounded context window.

Investigation detail includes enrichments for evidence already attached to the case.

The database uses one bounded bulk query for related event IDs rather than one enrichment query per event.

## Synthetic simulator scenarios

Phase 6 includes:

```text
intel-ioc-match
intel-domain-match
intel-sha256-match
intel-finding-match
```

They exercise IP, domain, SHA-256, and finding/investigation propagation using only synthetic metadata and reserved/documentation values.

No real threat infrastructure is contacted.

## Analyst console

The Intelligence workspace is available at:

```text
/intelligence
```

It shows:

- live intelligence metrics;
- indicator catalogue;
- source / indicator / effective confidence;
- tags and classifications;
- source provenance;
- audited source enable/disable controls;
- recent enrichment matches;
- explicit source-disagreement indicators.

Findings and Investigations now display relevant IOC context in their existing workflows.

## Verification

Phase 6 provides:

```text
scripts/phase6-intelligence-smoke.sh
scripts/phase6-final-regression.sh
```

with:

```bash
make smoke-phase6
make final-phase6
```

The Phase 6 workflow regression verifies:

1. two-source local intelligence catalogue;
2. expired-indicator filtering;
3. IP enrichment;
4. domain enrichment;
5. SHA-256 enrichment;
6. provenance preservation;
7. confidence calculation;
8. multi-source disagreement;
9. audited source disable/enable;
10. immediate cache invalidation after source-state change;
11. finding enrichment propagation;
12. finding-evidence enrichment propagation;
13. investigation enrichment propagation;
14. intelligence metrics.

The complete Phase 6 final regression also runs the full Phase 5 baseline and frontend lint/production build.

The complete Phase 6 final regression passes.

## Phase 6 completion criteria

- [x] local threat-intelligence fixtures;
- [x] IP IOC enrichment;
- [x] DNS/domain enrichment;
- [x] SHA-256 enrichment;
- [x] provenance model;
- [x] source / indicator / effective confidence model;
- [x] bounded positive and negative enrichment caching;
- [x] revision-aware cache invalidation;
- [x] source enable/disable and expiry behaviour;
- [x] multi-source disagreement handling;
- [x] finding enrichment;
- [x] investigation enrichment;
- [x] analyst-console intelligence workflow;
- [x] Phase 6 end-to-end smoke test;
- [x] Phase 6 full regression suite.
