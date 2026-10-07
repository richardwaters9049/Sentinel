# Phase 7 — Behavioural Analytics

## Status

In progress on:

```text
feat/phase-7-behavioural-analytics
```

## Objective

Phase 7 introduces behavioural baselines and explainable anomaly scoring without replacing Sentinel's deterministic detections.

Machine-learning output is supporting evidence only. An anomaly score does not mean an event is malicious, compromised, or confirmed.

The first vertical slice is:

```text
persisted telemetry
      ↓
deterministic detections
      ↓
threat-intelligence enrichment
      ↓
behaviour feature extraction
      ↓
Python Isolation Forest service
      ↓
durable behavioural score
      ↓
analyst API / Behaviour workspace
```

## Python analytics service

The first dedicated Python service lives at:

```text
services/ml
```

Runtime:

- Python 3.12 container;
- FastAPI;
- scikit-learn Isolation Forest;
- NumPy;
- pinned dependency versions;
- deterministic random seed `707`.

The service exposes:

```text
GET  /health
POST /v1/score
```

The model is trained at service startup from deterministic synthetic Northstar baseline rows.

No pickle, joblib, or externally supplied model artefacts are deserialised.

## Initial feature set

The first model evaluates:

- cyclic UTC hour;
- weekend activity;
- destination port;
- source/destination zone transition;
- authentication failure;
- service-account activity;
- OT-zone activity.

These are intentionally simple and analyst-readable.

Feature engineering is explicit in code rather than hidden inside a serialized pipeline.

## Baseline

The first model retains a deterministic synthetic Northstar reference baseline:

- weekday activity;
- typical daytime hours;
- common service ports;
- mostly same-zone corporate traffic;
- low failure frequency;
- low service-account frequency;
- rare corporate-to-DMZ activity;
- no assumption that anomaly equals maliciousness.

Phase 7 now also derives rolling entity context directly from persisted Sentinel telemetry before the current event is scored.

The entity key is selected in this order:

1. identity;
2. asset;
3. collector.

The bounded rolling windows are:

```text
60 minutes
24 hours
```

The current rolling context includes:

- prior events in 60 minutes;
- prior events in 24 hours;
- unique destination IPs in 24 hours;
- unique destination ports in 24 hours;
- authentication failures in 60 minutes;
- OT-related events in 24 hours;
- event rate;
- destination diversity;
- authentication-failure rate;
- OT-activity rate.

The current event is excluded from the historical window.

The exact context used at inference time is persisted with the score so an analyst can reconstruct what Sentinel knew when the event was evaluated.

## Model output

The inference response includes:

- event ID;
- entity ID;
- model version;
- model kind;
- anomaly score from 0–100;
- analytical severity;
- anomaly threshold;
- anomalous boolean;
- top feature deviations;
- analyst-readable explanation messages.

Initial threshold:

```text
65 / 100
```

Severity bands:

```text
0–64   low
65–84  medium
85–100 high
```

These are analytical bands, not incident severity or compromise probability.

## Explainability

Isolation Forest supplies the anomaly score.

Sentinel separately calculates feature deviations against the learned synthetic baseline and returns the top deviations.

This means an analyst can see statements such as:

- cross-zone activity is uncommon;
- OT activity differs from the enterprise baseline;
- authentication failures are uncommon;
- service-account behaviour is less common;
- event timing differs from normal daily activity.

The explanation layer does not claim those features caused the Isolation Forest decision in a formal causal sense. It provides transparent supporting context around the same feature vector.

## Gateway integration

Behavioural analytics are opt-in:

```env
SENTINEL_BEHAVIOUR_ENABLED=false
SENTINEL_ML_URL=http://127.0.0.1:8090
```

When enabled, the Go gateway adds the behavioural processor after deterministic detection and threat-intelligence enrichment.

The ML service also becomes part of readiness checks.

Keeping the feature opt-in means existing deterministic telemetry processing does not silently acquire a mandatory Python dependency.

## Persistence

Migration:

```text
0012_phase7_behavioural_analytics.sql
```

Table:

```text
behavioural_scores
```

Persisted fields include:

- event ID;
- entity ID;
- model version/kind;
- anomaly score;
- analytical severity;
- threshold;
- anomalous flag;
- explanation payload;
- scoring timestamp.

The event ID is the primary key, so retries remain idempotent.

## Entity selection

The first processor chooses one entity key in this order:

1. actor ID;
2. asset ID;
3. source collector.

This keeps the first runtime slice deterministic while leaving room for richer entity-specific baseline modelling later.

## Analyst APIs

Phase 7 begins with:

```text
GET /api/v1/behaviour/scores
GET /api/v1/behaviour/metrics
GET /api/v1/events/{id}/behaviour
```

The score list is bounded to 200 records.

Metrics currently report:

- total scored events;
- anomalous scores;
- high-severity scores;
- average anomaly score;
- most recent scoring time.

## Analyst console

The new workspace is:

```text
/behaviour
```

It shows:

- scored-event volume;
- baseline-deviation count;
- high-score count;
- deviation rate;
- recent behavioural scores;
- model version/kind;
- anomaly threshold;
- feature-level explanations;
- observed and baseline values.

The page deliberately states that ML output is supporting evidence rather than a security verdict.

## Model evaluation

Phase 7 includes a deterministic synthetic evaluation set:

- 120 baseline-like validation events;
- 30 intentionally anomalous validation events.

Evaluation reports:

- true positives;
- false positives;
- true negatives;
- false negatives;
- precision;
- recall;
- false-positive rate.

The first guardrails are:

```text
recall >= 0.90
false_positive_rate <= 0.15
```

These metrics only describe the synthetic validation set and must not be presented as real-world SOC performance.

## Synthetic simulator scenarios

The simulator adds:

```text
behaviour-normal
behaviour-anomaly
```

The anomalous scenario combines:

- service-account activity;
- authentication failure;
- corporate-to-OT transition;
- TCP/502 destination.

It remains synthetic and passive.

## Verification

Initial commands:

```bash
make ml-test
make ml-evaluate
make smoke-phase7
```

The Phase 7 smoke test verifies:

1. Python ML service health;
2. model version/kind;
3. baseline-like activity scoring;
4. anomalous activity scoring;
5. durable PostgreSQL persistence;
6. per-event behavioural API;
7. bounded score-list API;
8. behavioural metrics;
9. analyst-visible explanations;
10. synthetic model evaluation guardrails.

## Remaining Phase 7 work

- add model lifecycle/version registry;
- persist evaluation runs;
- add threshold/configuration controls;
- add richer analyst pivots by entity;
- add drift monitoring;
- mature the behavioural catalogue and add the Phase 7 final regression.
