# Phase 6 — Intelligence and Enrichment

## Status

In progress on:

```text
feat/phase-6-intelligence-enrichment
```

## Objective

Phase 6 adds provenance-aware threat intelligence to Sentinel without turning external reputation into unquestioned truth.

The first vertical slice is deliberately local-first. It uses repository-owned synthetic threat-intelligence fixtures so development, testing, and demonstrations remain deterministic and do not depend on third-party feeds.

The pipeline is:

```text
persisted telemetry
      ↓
network IOC extraction
      ↓
normalisation
      ↓
bounded enrichment cache
      ↓
local indicator lookup
      ↓
confidence calculation
      ↓
persisted event enrichment
      ↓
analyst API / console
```

## Local threat-intelligence fixtures

Phase 6 introduces:

```text
threat_intel_sources
threat_indicators
event_enrichments
```

The initial source is:

```text
intel-local-northstar
Northstar Local Threat Intelligence
source_type = local_fixture
default_confidence = 85
```

Its provenance explicitly states that it is:

- repository-owned;
- curated synthetic data;
- based on documentation-only network addresses;
- independent of external services.

Initial fixtures include:

- `198.51.100.66` — synthetic command-and-control context;
- `203.0.113.77` — synthetic staging context;
- `telemetry-sync.example` — reserved synthetic domain fixture for later DNS enrichment.

Only IP matching is active in the first runtime slice. The schema already supports domain and SHA-256 indicators for later work.

## Confidence model

Sentinel retains two independent confidence values:

- source confidence;
- indicator confidence.

The runtime computes:

```text
effective_confidence =
    round(source_confidence × indicator_confidence / 100)
```

For the primary fixture:

```text
source confidence     = 85
indicator confidence  = 95
effective confidence  = 81
```

This deliberately avoids presenting feed data as binary truth.

## Provenance

Every event enrichment stores a provenance snapshot from the source that produced the match.

Analysts can therefore see:

- source identifier;
- source name/type;
- source confidence;
- indicator confidence;
- effective confidence;
- tags;
- indicator context;
- source provenance;
- matched event field and value;
- match timestamp.

## Enrichment cache

The first enrichment engine includes a bounded in-memory cache:

- default TTL: five minutes;
- maximum entries: 1,024;
- caches positive lookups;
- caches negative lookups;
- cache keys include indicator type and normalised value.

Caching occurs after normalisation and before repository lookup.

Unit tests verify repeated negative lookups do not repeatedly query PostgreSQL.

## Event matching

The first slice evaluates:

```text
network.source_ip
network.destination_ip
```

IP values are normalised with Go's IP parser before lookup.

Matches are persisted idempotently using:

```text
(event_id, indicator_id, event_field)
```

as the uniqueness boundary.

## Telemetry processing

The existing telemetry persistence flow now uses an ordered processor chain:

```text
event persisted
      ↓
detection engine
      ↓
enrichment engine
```

If a processor fails, the chain stops and the JetStream message remains retryable under the existing consumer behaviour.

## Analyst APIs

Phase 6 adds:

```text
GET /api/v1/intelligence/indicators
GET /api/v1/intelligence/matches
GET /api/v1/intelligence/metrics
GET /api/v1/events/{id}/enrichments
```

Indicator queries support bounded limits and optional type/value filters.

The metrics endpoint reports:

- active sources;
- active indicators;
- enriched events;
- total matches;
- high-confidence hits.

## Synthetic scenario

The simulator now includes:

```text
intel-ioc-match
```

It produces a synthetic network connection to the documentation-only address:

```text
198.51.100.66
```

No real threat-infrastructure contact occurs.

## Analyst console

Phase 6 adds:

```text
/intelligence
```

The workspace shows:

- live intelligence metrics;
- indicator catalogue;
- source/indicator/effective confidence;
- indicator tags and classification;
- provenance model;
- recent enrichment matches.

The shared drawer navigation now exposes the Intelligence workspace.

## Verification

The first Phase 6 smoke test is:

```bash
make smoke-phase6
```

It runs with an isolated temporary NATS JetStream instance and verifies:

1. local intelligence fixtures are migrated;
2. the indicator API returns source provenance and confidence;
3. synthetic IOC-match telemetry is persisted;
4. the enrichment engine matches the IOC;
5. the match is linked to the correct event field;
6. source and indicator confidence are preserved;
7. effective confidence is calculated correctly;
8. provenance and classification context are preserved;
9. intelligence metrics reflect the enrichment.

## Remaining Phase 6 work

- add DNS/domain enrichment against the existing local fixture model;
- add SHA-256 enrichment fixtures and safe endpoint-file metadata scenarios;
- expose enrichment context directly inside finding and investigation workflows;
- add source enable/disable and expiry behaviour;
- add cache observability;
- add richer provenance conflict handling when multiple sources disagree;
- add Phase 6 final regression once the enrichment catalogue is mature.
