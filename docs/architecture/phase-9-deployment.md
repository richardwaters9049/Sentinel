# Phase Nine — Production-shaped deployment

## Status

Complete within the local simulated scope on 9 October 2026. The complete container
stack passed its deployment, recovery and security gates. See the
[completion evidence](phase-8-9-completion.md).

## Scope

This milestone packages and verifies the simulated platform in local containers.
It provides a deployment boundary, monitoring, distributed HTTP traces, measured
synthetic load, dependency recovery and backup/restore checks. It does not certify
Internet exposure, operational OT use, high availability or production detection accuracy.
Kubernetes resources are a validated deployment base; no live cluster is implied.

## Run the deployment

Prerequisites: Docker Compose, Go 1.27.2, Bun 1.3.14, Python 3 and OpenSSL with Ed25519.
From `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
python3 scripts/provision-deployment.py
export SENTINEL_DEPLOY_SECRETS_DIR=/absolute/secrets_dir/from/provisioning
export SENTINEL_BUILD_VERSION=$(git rev-parse --short HEAD)
docker compose -f infra/docker/compose.production.yaml up -d --build
```

Provisioning prints only a new private directory and expiry. Files include a random
PostgreSQL password, private database URL, eight-hour principal registry, collector
signing seed, monitoring credential and Grafana password. Read the analyst token from
`lab-tokens.json` privately to sign in at `http://127.0.0.1:13006`. Grafana is on port
13000 and Prometheus on 13090, both bound to loopback. Grafana disables anonymous access
and uses its generated password. Prometheus has no web authentication in this local
configuration; its API is restricted to loopback and contains operational metrics.

An isolated init service copies only runtime secrets into a private named volume owned
by UID 10001. Gateway and monitoring read that volume; the collector's private key and
raw analyst credentials remain outside it. Application containers run as UID/GID 10001,
with read-only roots, dropped capabilities, no privilege escalation and bounded tmpfs.
PostgreSQL runs as its dedicated UID/GID 70 with a read-only root and durable
volume. Init prepares volume ownership and its private password file; the unused
privilege-switching binary is removed. NATS uses a non-root process and durable JetStream volume. These dependencies
are accessible only on the internal backend network, without host-published ports.

The backend relies on this isolated network for PostgreSQL/NATS/ML trust. There is no
mutual TLS or broker principal policy in this local deployment. Use authenticated,
encrypted managed dependencies before distributing services across untrusted networks.
Expose a real deployment only through an operator-managed HTTPS ingress and set the
same exact HTTPS `SENTINEL_CONSOLE_ORIGIN` for console and gateway; HTTPS then enables
Secure host-prefixed session cookies. The shipped loopback origin is a lab exception.

## Observability

Gateway request logs include trace/span IDs, stable registered routes, status, duration
and build version. W3C trace context links gateway catalogue/score HTTP calls to ML
spans. Caller trace IDs are correlation hints, never authenticated identity. Export
queues are bounded to 256 spans and receiver timeouts to two seconds. Gateway export
failures/drops are counted; export failure does not block API work. ML uses the pinned
OpenTelemetry SDK and records route, method and status without headers, query strings,
bodies or automatic exception messages.

The collector accepts OTLP on the internal network and writes rotating trace files:
10 MiB per file, three backups. Trace storage is diagnostic, separate from the durable
security ledger; volume deletion removes it. Prometheus retains local metrics for
24 hours in tmpfs, so its restart loses history. Grafana is provisioned with the Sentinel
Prometheus data source; dashboard definitions can be added without changing backend
security. Metrics/trace history are intentionally limited for this reproducible lab.

The administrator-only `/api/v1/platform/metrics` exports status-class request counters,
latency buckets, access-storage failures, rate/signature denials, trace exporter health,
and JetStream pending/unacknowledged counts. Alerts cover an unavailable gateway or
expired monitoring credential, audit failures, trace failure and consumer backlog.
Grafana Explore can query these series. Its provisioned Prometheus data source is
checked with authenticated backend health queries. Alertmanager/paging is outside the lab boundary.

## Credential rotation and retention

Credentials expire after eight hours; expiry also affects Prometheus's scraper. Rotate
before expiry by provisioning a replacement principal registry, preserving the existing
PostgreSQL password/database URL and Grafana password. Point the secret-init input at
the replacement directory and recreate init, gateway and Prometheus. Do not replace the
PostgreSQL password file alone on an existing volume: PostgreSQL reads it only during
first initialization. Use a deliberate database role/password rotation for that change.
Changing registry digests revokes their sessions on gateway restart. Expired/replaced
collector keys deny new signatures. Keep clocks synchronized within the 30-second
signature allowance. Archive/remove obsolete private files only after verifying rotation.

Access records have a 90-day retention policy, applied explicitly through administrator
`POST /api/v1/security/audit/retention`, up to 1,000 records per call. Repeat until fewer
than 1,000 are removed and schedule according to actual volume. Existing events, findings,
investigations and transactional domain audits are not deleted by this endpoint.
Backups contain security evidence and identity metadata: store them privately, encrypt
external copies, restrict restore permissions and choose a separate evidence policy.

## Backup and recovery

From `/Users/richy/Documents/Github/Sentinel`, after exporting the same deployment
secret-directory setting:

```bash
cd /Users/richy/Documents/Github/Sentinel
umask 077
docker compose -f infra/docker/compose.production.yaml exec -T postgres pg_dump -U sentinel -d sentinel -Fc > /private/path/sentinel.dump
```

Restore into a **new empty database**, never overwrite evidence casually. The automated
gate creates `sentinel_recovery` in its own disposable PostgreSQL container, runs
`pg_restore --exit-on-error`, and verifies the exact persisted event count. A real recovery
also needs compatible service images, configuration, a protected credential registry
and a NATS volume/stream recovery plan. PostgreSQL contains durable evidence; JetStream's
existing stream retention is 24 hours and is not a replacement for database backups.

## Kubernetes base

`infra/kubernetes` defines namespace restrictions, ClusterIP services, one replica per
application, probes, CPU/memory bounds, read-only/non-root containers and network policy.
The gateway init container copies group-readable projected secrets into owner-only
files on a memory volume; runtime file validation remains strict. Tokens never appear
in checked-in resources. Supply a `sentinel-runtime` Secret with `credentials.json`,
`database-url`, `nats-url`, `console-origin` and `otlp-endpoint`; provide external trusted
PostgreSQL/NATS/OTLP endpoints and push immutable application images to your registry.
Replace the local `phase9` image tags through an environment-specific Kustomize overlay.

The base allows DNS, application traffic within the namespace and backend ports
5432/4222/4318. Restrict those external destinations to your actual dependency endpoints.
Ingress is not exposed: add an explicit TLS ingress and a corresponding ingress-network
policy in the environment overlay. No cluster, cloud database or public service is
created by the repository or regression gate.

From `/Users/richy/Documents/Github/Sentinel`:

```bash
cd /Users/richy/Documents/Github/Sentinel
kubectl kustomize infra/kubernetes
```

## Verification

`make final-phase9` builds the stack under a unique test project, provisions fresh private
credentials and tests synthetic ingestion, provenance, deduplication, signed replay after
restart, rate limits, dependency outage/recovery, trace outage, OTLP delivery, authenticated
Prometheus scraping, Grafana data-source queries and non-root/read-only controls on
all eight runtime containers. It verifies a gateway-to-ML parent/child trace link. It measures
64 events at concurrency four and reports p50/p95/p99 request latency and measured
throughput. These are local measurements, not sustained capacity claims. Signing and console
proxy overhead are included; persistence is checked separately. Results go to
`/tmp/sentinel-phase9-report.json`.

The gate binds test ports 13006/13090/13000 and removes only its own disposable containers,
volumes and private files. Stop a deployment using those ports before running it. The
normal lab/database are preserved. Run `make security-gate` separately with Trivy 0.75.0
and govulncheck 1.8.0 on PATH after building images; it fails on reachable Go vulnerabilities,
runtime HIGH/CRITICAL findings, secrets and deployment policy failures, and produces
CycloneDX SBOMs for all eight runtime images outside the repository. NATS, the minimal
OTLP collector, Prometheus and Grafana backend binaries are built from pinned upstream
sources with the patched Go toolchain. Grafana keeps only its required Prometheus plugin;
that backend is rebuilt with patched gRPC. Rebuilding invalidates its upstream
signature, so the image removes that manifest and explicitly allows only the
operator-built prometheus plugin ID. Plugin installation/preinstallation is disabled;
trust rests on the pinned source/image build, not an upstream signature for modified bytes. Initial builds take longer than ordinary
image pulls. CI adds scheduled scanning and CodeQL analysis.
Development-tool dependencies are distinct from the deployed standalone console; the
current unfixed `braces` advisory affects tooling, not its runtime image. Do not treat
runtime scanning as a guarantee of absence of unknown vulnerabilities.
