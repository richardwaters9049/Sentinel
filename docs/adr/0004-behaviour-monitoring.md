# ADR-0004: Separate behavioural distribution signals from model drift claims

## Status

Accepted.

## Context

Sentinel has persisted anomaly scores and rolling context snapshots, but lacks real-world
labels and calibrated drift ground truth. Operational threshold policy changes independently
of the model. Synthetic evaluation describes development quality only.

## Decision

Compare fixed score-band proportions across adjacent 24-hour scoring-time windows using
explainable total variation distance. Monitor one model version and optional entity namespace
at a time. Require 30 scores per window and withhold comparisons beyond 2,000 rows per
window. Report context depth, score freshness, live availability and matching synthetic
evaluation separately. Compute on demand under a consistent read snapshot; do not add a
scheduled worker or persisted monitoring history yet.

See [Phase Seven architecture](../architecture/phase-7-behavioural-analytics.md#distribution-monitoring-and-baseline-health)
for thresholds, boundaries and interpretation.

## Consequences

No dependencies, historical score mutations or automated remediation are introduced.
Operational threshold changes cannot themselves trigger distribution change. Windows and
sample counts make missing coverage visible. Read-only monitoring preserves the existing
trust boundary and uses validated, parameterised entity filters.

The signal cannot establish model drift, feature-level causes, compromise or production
accuracy. Data composition, entity mix, processing delays and replay can change distributions.
Sample and distance thresholds remain development defaults. Higher-volume comparison and
monitoring history require a later design, rather than silently interpreting capped samples.
