# Phase Eight and Nine — Completion evidence

Completed on 9 October 2026 within Sentinel's defensive, synthetic local-lab scope.
This record covers implemented platform hardening and production-shaped packaging;
it does not certify public deployment, high availability or detection accuracy.

## Delivered

- Expiring credentials, server-side RBAC and secure, revocable console sessions.
- Ed25519 collector signatures, verified provenance and durable nonce replay rejection.
- Bounded peer/principal rate limits, durable access records and explicit 90-day retention.
- Patched runtime dependencies, reachable vulnerability checks, secret/policy scans and eight runtime SBOMs.
- Non-root, read-only runtime containers; private runtime secrets and internal dependencies.
- Prometheus alerts, authenticated Grafana queries and linked gateway/ML OTLP traces.
- Measured synthetic ingestion, dependency failure/recovery and PostgreSQL backup restore.
- A validated Kubernetes base and operational credential/retention/recovery documentation.

## Checks actually run

From `/Users/richy/Documents/Github/Sentinel`, using Go 1.27.2:

| Check | Result |
| --- | --- |
| `make final-phase7` | Passed inherited Phase One–Six and behavioural regressions; 23 ML tests passed |
| `make final-phase8` | Passed gateway/simulator race tests and vet, PostgreSQL integration tests, console lint, 14 tests, TypeScript and production build, API/session smokes |
| Additional collector regression tests | Passed duplicate-header, query, future-timestamp, body-size, private-file and symlink cases |
| `make final-phase9` | Passed fresh full-stack deployment, replay after restart, provenance, rate limiting, outages, restore, Grafana backend queries, linked traces and all eight runtime security controls |
| `make security-gate` | Passed reachable Go, runtime HIGH/CRITICAL, secret and deployment-policy checks; generated eight CycloneDX SBOMs |
| Kubernetes strict schema validation | Eight valid resources; zero invalid, errored or skipped |
| Prometheus alert validation | Four rules accepted by promtool |
| Local launcher unit tests | 17 passed |

The Phase Seven normal fixture now uses a deterministic weekday business-hour time,
with an explicit simulator `-at` option. Previously its current-clock timestamp could
correctly score as unusual during evening runs, making the regression clock-dependent.
Model thresholds and detection assertions remain intact. BA-001, BA-003 and BA-004 still
miss all twelve changed cases in their documented catalogue evaluation; this milestone
does not conceal or resolve those analytical limitations.

## Local load and recovery result

The final isolated deployment run accepted 64 unique events at concurrency four:

| Measurement | Result |
| --- | --- |
| HTTP ingestion throughput | 116.34 events/second |
| Request latency p50 / p95 / p99 | 25.94 / 75.03 / 87.44 ms |
| Persisted and restored unique events | 64 / 64 |
| Burst request outcomes | 69 accepted, 111 rate limited |

These measurements include local signing and console-proxy overhead over a small burst.
They are not a sustained capacity benchmark or end-to-end detection-latency claim.
The report is written to `/tmp/sentinel-phase9-report.json`; SBOMs are in
`/tmp/sentinel-security-artifacts`. These generated artifacts are excluded from Git.
The gate removed only its own disposable containers, volumes and private credentials.

## Remaining boundaries

Kubernetes was rendered and schema/policy validated; no cluster or public endpoint was
deployed. Hosted CodeQL, GitHub Actions and branch-protection enforcement are configured
but are not claimed as locally executed. An unfixed braces advisory remains in development
tooling, separate from the scanned standalone runtime. Local dependency trust rests on an
isolated network; distributed deployment needs authenticated, encrypted dependencies.
Rate limits are process-local, identity is lab-provisioned, and monitoring credentials
expire after eight hours. Retention and credential rotation require operator action.

See [Phase Eight](phase-8-platform-hardening.md), [Phase Nine](phase-9-deployment.md)
and [ADR 0007](../adr/0007-collector-signatures-and-access-ledger.md) for contracts,
trust assumptions and residual risks.
