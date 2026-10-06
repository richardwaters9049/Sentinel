#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_SMOKE_PORT:-18080}"
GATEWAY_PID=""

cleanup() {
  if [[ -n "${GATEWAY_PID}" ]] && kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    kill "${GATEWAY_PID}" 2>/dev/null || true
    wait "${GATEWAY_PID}" 2>/dev/null || true
  fi
}

trap cleanup EXIT

cd "${ROOT_DIR}"

echo "Starting Sentinel dependencies..."
docker compose up -d postgres nats

echo "Waiting for PostgreSQL..."
for _ in {1..30}; do
  if docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1; then
  echo "PostgreSQL did not become ready." >&2
  exit 1
fi

echo "Starting gateway on port ${GATEWAY_PORT}..."
(
  cd services/gateway
  SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" go run ./cmd/gateway
) >/tmp/sentinel-phase0-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/health" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase0-gateway.log >&2
    exit 1
  fi
  sleep 1
done

echo "Checking health..."
curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/health"

echo
echo "Checking dependency readiness..."
curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready"

echo
echo "Checking schema migration..."
TABLE_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM pg_tables WHERE schemaname='public' AND tablename IN ('assets','identities','events','detections','findings','finding_events','audit_events','schema_migrations');"
)"

if [[ "${TABLE_COUNT}" != "8" ]]; then
  echo "Expected 8 Sentinel tables, found ${TABLE_COUNT}." >&2
  exit 1
fi

echo "Phase 0 smoke test passed."
