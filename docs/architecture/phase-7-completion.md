# Phase Seven completion gate

**Passed locally on 8 October 2026** with `make final-phase7`. The run included
22 passing Python tests, four passing console API-client tests, and the uncached
PostgreSQL integration test under the race detector. The preflight rejection was also
verified against the restored development gateway. Gateway, ML and console readiness
were checked afterwards, with the anomaly threshold restored to 65. Console TypeScript checking
(`bunx tsc --noEmit`) also passed separately. The validation dataset returned recall
1.0 and false-positive rate 0.0 at threshold 65; the governed runtime evaluation at
threshold 71 also met the required guardrails. These results apply only to the
synthetic datasets described below.

Run from `/Users/richy/Documents/Github/Sentinel`:

```bash
make final-phase7
```

This is a local, synthetic regression gate. It requires Go, Bun, Docker, curl and
Python 3. Stop the development gateway before running: inherited smoke tests use
exclusive workers on the local NATS broker. The preflight rejects a connected
Sentinel gateway rather than allowing competing consumers to invalidate correlation
checks. Do not run concurrent smoke tests or start another gateway during the gate.

The inherited resilience tests briefly stop and restart local PostgreSQL and NATS.
The smoke tests retain generated evidence, findings, investigations and audit records
in the development database. No historical evidence is deleted. The PostgreSQL
monitor integration test uses a disposable schema and drops only that schema.
`SENTINEL_TEST_DATABASE_URL` can override its default local development connection;
use only a database authorised for tests with schema creation privileges.

## Required checks

| Gate | Evidence |
| --- | --- |
| Cleanup regression | Actual smoke-test EXIT trap tested for success, failure, SIGTERM, untouched settings and failed restoration |
| Phases One to Six | Existing final Phase Six gate, including formatting, Go tests/vet/race checks, telemetry recovery, detection controls/catalogue/quality, hunts, investigations, console lint/build, simulated OT and intelligence enrichment |
| PostgreSQL monitor integration | Uncached race test; migrations applied twice; identity/asset/collector/unknown namespace isolation, empty scope, evaluation guardrails, evaluation age, threshold and model-version boundaries |
| Console API contracts | Monitor and catalogue client tests, including scoped requests and error handling |
| Phase Seven runtime | Isolated gateway/broker, model health, durable scores, normal/changed activity, rolling baseline snapshots, finding/investigation evidence, entity filters, monitor contract, threshold restoration and governed evaluation history |
| Synthetic model quality | Python tests and deterministic validation: 120 normal cases, 30 anomalous cases, recall at least 0.90 and false-positive rate at most 0.15 |
| Catalogue limitations | Fixed threshold 65 and expected per-profile counts are asserted, independently of the live governed threshold |

CI also supplies PostgreSQL 16 to Go tests, so the monitor integration test no longer
silently skips there. CI runs the cleanup regression and shell syntax checks alongside
the existing Python, console and Go checks. The complete Docker runtime gate remains
a local command; changing CI does not establish that a remote CI run has passed.

Threshold cleanup attempts restoration before shutting down the isolated gateway,
preserves the failing test's exit status, and reports restoration failure as a failed
run. An unreachable gateway or database can still prevent restoration; the script
prints the original threshold for recovery. Successful restoration is checked against
the saved setting. Synthetic evaluation history remains append-only through this gate.

## Accepted limitations

At reference threshold 65, BA-001 (timing), BA-003 (burst) and BA-004 (destination
diversity) each flag **0/12 changed-context cases**. BA-002 (authentication failures)
and BA-005 (simulated enterprise-to-OT context) each flag **12/12**. Every reference
cohort flags **0/12**. These tiny, partly shared cohorts measure feature contrasts,
not maliciousness or independent real-world observations. Aggregate validation
metrics do not override these isolated coverage gaps.

Distribution change means a shift between adjacent scoring-time windows, not proven
model drift. Comparisons need 30 samples per window, remain isolated by model version
and entity namespace, and are withheld when the bounded sample query overflows.
Cold context, stale scoring, unavailable analytics and synthetic evaluation quality
remain separate signals. Scores are supporting evidence, never compromise probabilities.

Completion covers the documented simulated Phase Seven capabilities. It does not
certify production accuracy, introduce authentication/RBAC, retrain the model, change
historical scores or authorise real OT actions. Platform hardening remains Phase Eight.
