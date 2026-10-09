# Sentinel behavioural analytics service

Phase 7 introduces a Python service that scores already-normalised Sentinel telemetry against a deterministic synthetic Northstar baseline.

The first model is an Isolation Forest. It is deliberately used as supportive analyst evidence rather than an authoritative malicious/benign classifier.

## Reproducibility

- Python 3.12 in the container.
- Pinned direct dependency versions in `requirements.txt`; runtime SBOMs record the resolved transitive versions.
- Fixed random seed: `707`.
- Model is trained from deterministic synthetic baseline rows at service startup.
- No pickle/joblib artefacts are loaded.

## Initial features

- cyclic UTC hour;
- weekend activity;
- destination port;
- cross-zone traffic;
- authentication failure;
- service-account use;
- OT-zone activity.

## Output

The service returns:

- anomaly score from 0–100;
- low/medium/high analytical severity;
- threshold used;
- model kind/version;
- top feature deviations with analyst-readable explanations.

The output is evidence, not a security verdict.

## Behavioural catalogue

`GET /v1/catalogue` returns five versioned analytical profiles, their telemetry requirements,
context-depth guidance, limitations and analyst review steps. The gateway exposes this through
`GET /api/v1/behaviour/catalogue`; the Behaviour workspace shows expandable profile panels
and links matching-model explanation features.

The catalogue includes a deterministic per-profile report computed once at startup from the
existing model, at fixed threshold 65. It uses twelve reference and twelve changed-context
cases per profile, with March fixtures separate from training and the original evaluation.
The original model version and persisted evaluation history remain unchanged.

From `/Users/richy/Documents/Github/Sentinel`, run `make ml-catalogue-evaluate` to inspect
per-profile score sensitivity. After building the image, an alternative policy can be measured
with `docker compose run --rm -e SENTINEL_BEHAVIOUR_THRESHOLD=71 ml python -m app.catalogue_evaluation`.

Timing, activity bursts and destination diversity alone do not cross threshold 65 in these
fixtures. Failure concentration and combined simulated OT context do. These are small
synthetic sensitivity measurements, not production recall or compromise probabilities.
See the [Phase Seven architecture](../../docs/architecture/phase-7-behavioural-analytics.md#behavioural-catalogue-maturity)
for interpretation and remaining limitations.

## Deployment and tracing

The runtime uses a digest-pinned Alpine Python 3.12 base and UID 10001. Build tools
compile the pinned scikit-learn wheel in a separate stage and do not enter the runtime.
The model seed/version remain unchanged; the regression gate verifies synthetic scores
and catalogue behaviour after framework/runtime upgrades.

`OTEL_EXPORTER_OTLP_ENDPOINT` optionally enables bounded OpenTelemetry export. Incoming
W3C trace context joins gateway HTTP calls. Spans contain method, registered route and
status, without request bodies, header values, query strings or exception messages.
Health remains available when the exporter is unavailable. See
[deployment operations](../../docs/architecture/phase-9-deployment.md).
