# Sentinel behavioural analytics service

Phase 7 introduces a Python service that scores already-normalised Sentinel telemetry against a deterministic synthetic Northstar baseline.

The first model is an Isolation Forest. It is deliberately used as supportive analyst evidence rather than an authoritative malicious/benign classifier.

## Reproducibility

- Python 3.12 in the container.
- Exact dependency versions in `requirements.txt`.
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
