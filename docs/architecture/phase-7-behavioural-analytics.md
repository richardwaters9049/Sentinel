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

Phase 7 now exposes:

```text
GET   /api/v1/behaviour/catalogue
GET   /api/v1/behaviour/scores
GET   /api/v1/behaviour/metrics
GET   /api/v1/behaviour/monitor
GET   /api/v1/behaviour/settings
PATCH /api/v1/behaviour/settings
GET   /api/v1/behaviour/models
GET   /api/v1/behaviour/evaluations
POST  /api/v1/behaviour/evaluations
GET   /api/v1/events/{id}/behaviour
```

The score list is bounded to 200 records.

Metrics currently report:

- total scored events;
- anomalous scores;
- high-severity scores;
- average anomaly score;
- most recent scoring time.

## Model governance

Migration:

```text
0014_phase7_model_governance.sql
```

Phase 7 now separates model provenance, operational scoring policy, and evaluation history.

### Model registry

`behaviour_models` records:

- model version;
- model kind;
- lifecycle state;
- feature schema;
- training-source description;
- deterministic random seed;
- registration time.

The active v1 model is registered as:

```text
sentinel-behaviour-iforest-v1
IsolationForest
12 explicit features
synthetic Northstar baseline
seed 707
```

This gives analysts and future model-lifecycle code a durable provenance record instead of relying only on Python constants.

### Threshold settings

`behaviour_settings` stores the operational anomaly threshold independently from the trained model.

Changing the threshold does not retrain or mutate the model.

New scores use the latest persisted threshold. Existing scores keep the exact threshold used at their scoring time.

Threshold changes require the current development actor bridge and record who last changed the setting.

The initial allowed range is:

```text
1–99
```

### Evaluation history

`behaviour_evaluation_runs` stores reproducible validation runs with:

- model version;
- dataset identity;
- threshold used;
- confusion-matrix counts;
- precision;
- recall;
- false-positive rate;
- actor;
- timestamp.

The repository-owned workflow is:

```bash
make record-phase7-evaluation
```

It:

1. queries the live governed threshold;
2. evaluates the deterministic synthetic validation set with that threshold;
3. submits the metrics to the gateway;
4. persists the run against the registered model version.

The gateway validates confusion-matrix totals and rate bounds before accepting an evaluation.

Synthetic evaluation results remain development evidence only and are not represented as production SOC accuracy.

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

## Entity pivots

Scores and metrics accept an optional paired `entity_type` and `entity_id` query filter.
Namespaces remain distinct: an identity and an asset with the same ID are not combined.
IDs are bounded to 256 bytes and bound as SQL parameters. Unsupported types and incomplete
pairs return HTTP 400. The score list retains its 200-record cap; metrics cover all persisted
scores in the selected scope, rather than only the displayed rows.

The Behaviour workspace offers an entity pivot from the selected score's rolling baseline
and a clear action back to all entities. Loading a different scope cancels stale requests.
Historical baseline snapshots remain unchanged.

## Remaining Phase 7 work

- add the Phase 7 final regression and completion gate.

## Distribution monitoring and baseline health

`GET /api/v1/behaviour/monitor` uses the same paired entity filters as scores and metrics.
It returns independent signals, not an overall security verdict:

- **Rolling context:** recent sampled scores with fewer than five prior entity events in their
  persisted 24-hour baseline snapshot are labelled cold context. Five events is a context-depth
  heuristic, not proof that the baseline is representative or that collection coverage is adequate.
- **Scoring freshness:** latest persisted score for the selected model and scope within the
  comparison periods is fresh through 15 minutes, stale afterwards. No scores in 48 hours is
  reported explicitly. Sparse telemetry and pipeline failure cannot be distinguished by this signal.
- **Analytics availability:** the existing live ML readiness dependency check is reported separately;
  disabled, unknown and unavailable are distinct. A passing health check does not prove throughput.
  Database failures return HTTP 503 with the normal sanitised API error shape.
- **Observed distribution:** adjacent UTC scoring-time windows `[now-24h, now)` and
  `[now-48h, now-24h)` compare score-band proportions for 0–19, 20–39, 40–59, 60–79, 80–100.
  Total variation distance is half the sum of absolute differences in proportions, ranging from
  zero to one. A distance of at least 0.20 prompts review. Floating-point noise at that boundary
  is tolerated at 1e-12. Each window requires at least 30 scores. These are development defaults,
  not calibrated statistical significance or operational drift guarantees.
- **Synthetic evaluation:** latest non-empty evaluation matching the selected model and current
  operational threshold is compared with recall >= 0.90 and false-positive rate <= 0.15.
  Its dataset, threshold, metrics and timestamp are explained. Older than seven days is stale.
  Missing matching runs are not evaluated. Synthetic results do not estimate production accuracy.

The monitored model is the latest persisted scored model in the requested scope, selected
with deterministic timestamp/event-ID ordering. It may differ from the registry's active model.
Versions are never pooled. A new model needs sufficient samples in both windows. Historical
thresholds and anomalous booleans are excluded from distribution comparison: changing policy
alone cannot trigger a distribution-change signal. No anomaly-rate claim is used as drift proof.

Each window reads at most 2,001 rows under a repeatable-read snapshot and a three-second
query deadline. At most 2,000 are summarised; the extra row detects overflow and withholds
comparison, rather than interpreting a biased recent sample as the entire window. Counts and
cold-context counts then describe the capped sample. Migration 0015 adds query-pattern indexes.
The API has a five-second context deadline; existing bounded readiness checks remain in use.

Monitoring is calculated on demand. No monitoring records are persisted in this slice because
there is no scheduled monitor or audit-history requirement yet. Historical scores, thresholds
and baseline snapshots are preserved. This endpoint does not change the existing development
actor trust boundary or add remediation actions. The console cancels stale scope requests,
loads monitoring independently so its failure does not hide existing scores, and refreshes health
when threshold policy changes.

Limitations: score distributions can change with entity mix, telemetry composition, synthetic
scenario activity, seasonality, replay timing or feature changes. Score-band comparison cannot
identify which input feature changed and is not a model-performance test. Scoring timestamps
avoid trusting producer clocks for monitoring windows but reflect replay/processing time.
Thirty correlated events are not thirty independent observations. Cold context uses historical
snapshot event counts, whose source timestamps still affect their coverage. Review these
signals alongside explanations, deterministic findings and collection health.

Verification includes monitor unit and API tests, optional PostgreSQL integration tests in a
disposable schema (`SENTINEL_TEST_DATABASE_URL`), migration rerun checks, and monitoring
contracts/entity isolation in the Phase Seven smoke test. Phase Seven remains in progress.

## Behavioural catalogue maturity

`GET /api/v1/behaviour/catalogue` proxies repository-owned ML profile metadata and
per-profile synthetic validation. It is model-wide and rejects query filters with HTTP 400.
ML being disabled, unavailable or returning an invalid catalogue produces a sanitised HTTP 503.
The gateway caps upstream responses at 1 MiB, applies a three-second context deadline, bounds
profile counts and text lengths, rejects duplicate IDs, and verifies validation provenance,
count/rate consistency and finite metrics before exposing them to the console.

Catalogue version: `sentinel-behaviour-catalogue-v1`.
Dataset: `synthetic-behaviour-catalogue-v1`.
Model: `sentinel-behaviour-iforest-v1` (unchanged training and inference).

| ID | Profile | Features/context | Suggested prior events |
| --- | --- | --- | --- |
| BA-001 | Unusual activity timing | UTC cyclic hour and weekend; current event | 0 |
| BA-002 | Authentication failure concentration | Failure indicator and failures / all entity events in 60 minutes | 5 in 60 minutes |
| BA-003 | Entity activity burst | Prior entity event count in 60 minutes, capped at 60 for the model feature | 5 in 60 minutes |
| BA-004 | Destination diversity | Distinct destination IPs / prior entity events in 24 hours | 5 in 24 hours |
| BA-005 | Simulated enterprise-to-OT context | Cross-zone, service account, OT indicator and prior OT proportion in 24 hours | 5 in 24 hours |

These are analytical profiles, not new detection rules, causal attributions, MITRE mappings or
findings. The suggested context depths guide review and do not change scoring eligibility.
The catalogue links a selected score only when its model version matches and a profile shares
features with its persisted top explanations. This is a feature link, not a rule match; absence
from the top three explanations does not establish absence of that behaviour.

Each profile has twelve reference-like and twelve changed-context synthetic cases, dated in
March 2026, separate from January training and the existing February validation dataset.
Reference cases are reused across profiles, so totals do not represent independent observations.
Changed-context labels mean feature changes, not maliciousness. Activity-burst cases also
adjust total counts and derived destination ratios to keep context internally consistent.
The OT cohort combines features and does not measure each feature's independent contribution.

The API computes this deterministic validation report once at ML service startup using its
existing model instance and a fixed threshold of 65. It reports counts flagged, flag rates,
mean scores and explanation-feature coverage. It is not persisted evaluation history and does
not replace the governed evaluation workflow. Operational threshold changes do not alter this
reference report; the console explicitly shows policy differences. Other thresholds can be
examined with `make ml-catalogue-evaluate` and `SENTINEL_BEHAVIOUR_THRESHOLD` passed into
the evaluator container, without changing the model.

At threshold 65 in the pinned runtime:

- BA-001, BA-003 and BA-004: 0/12 changed-context cases flagged, despite higher mean scores
  and relevant explanations. Isolated coverage remains limited; the console shows this warning.
- BA-002 and BA-005: 12/12 changed-context cases flagged.
- All five reference cohorts: 0/12 cases flagged.

These measurements describe tiny synthetic cohorts only. The existing aggregate recall and
false-positive guardrails remain scoped to their original dataset and cannot be generalised
from these results. Known limitations include shift work, maintenance, source-clock errors,
collection/replay bursts, missing destination fields, small denominators and the
enterprise-dominant reference scoring authorised OT activity highly.

No model retraining, historical score mutation, remediation, database migration or new
runtime dependency is introduced. The trust boundary remains gateway-to-configured ML
service with bounded, validated metadata. The catalogue's native expandable panels support
keyboard interaction and introduce no decorative motion. Tests cover malformed/oversized
responses, invalid provenance and metrics, cancellation, disabled services, fixed-threshold
separation, deterministic contrast cases and the model-wide proxy contract.

The remaining Phase Seven gate must assess these limitations explicitly before declaring
completion. Catalogue maturity does not mean all profiles have reliable detection coverage.
