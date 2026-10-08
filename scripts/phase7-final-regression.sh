#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

for dependency in go bun docker curl python3; do
  command -v "${dependency}" >/dev/null || { echo "Missing dependency: ${dependency}" >&2; exit 1; }
done
python3 scripts/tests/phase7-cleanup-test.py
docker info >/dev/null
# Earlier resilience gates cycle this local broker and expect exclusive workers.
docker compose up -d nats >/dev/null
for _ in {1..30}; do
  curl -fsS http://127.0.0.1:8222/healthz >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS http://127.0.0.1:8222/connz | python3 -c '
import json, sys
connections = json.load(sys.stdin)["connections"]
if any(connection.get("name") == "sentinel-gateway" for connection in connections):
    sys.exit("Stop the local development gateway before this gate: inherited smoke tests require exclusive NATS workers.")
'

echo "=== Phase 7 final regression ==="
echo "[1/4] Complete Phase 6 regression baseline (includes local dependency recovery)"
./scripts/phase6-final-regression.sh

echo "[2/4] Required PostgreSQL integration and migration rerun with race detection"
docker compose up -d postgres >/dev/null
for _ in {1..30}; do
  docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1 && break
  sleep 1
done
docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null
(
  cd services/gateway
  SENTINEL_TEST_DATABASE_URL="${SENTINEL_TEST_DATABASE_URL:-postgres://sentinel:sentinel_dev_only@127.0.0.1:55432/sentinel?sslmode=disable}" \
    go test -count=1 -race -v ./internal/database -run '^TestBehaviourMonitorIntegration$'
)

echo "[3/4] Behavioural console API contracts"
(
  cd apps/console
  bun test lib/sentinel/client-monitor.test.ts lib/sentinel/client-catalogue.test.ts
)

echo "[4/4] Behavioural runtime, governance, model evaluation and catalogue limitations"
./scripts/phase7-behaviour-smoke.sh

echo "Phase 7 final regression passed."
echo "  Phase 1–6 baseline, lint and production build: verified"
echo "  PostgreSQL migrations, entity isolation and monitoring: verified"
echo "  Behavioural API clients, runtime and synthetic evaluation: verified"
echo "  Catalogue limitations: BA-001/003/004 remain below threshold on all 12 changed cases"
echo "  Scope: simulated defensive analytics; completion does not establish production detection accuracy"
