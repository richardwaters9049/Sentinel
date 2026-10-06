#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE1_SMOKE_PORT:-18081}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase1-smoke-gateway"
EVENT_ID="evt_smoke_$(date +%s)_$RANDOM"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

cleanup() {
  if [[ -n "${GATEWAY_PID}" ]] && kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    kill "${GATEWAY_PID}" 2>/dev/null || true
    wait "${GATEWAY_PID}" 2>/dev/null || true
  fi

  rm -f "${GATEWAY_BINARY}"
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

echo "Waiting for NATS..."
for _ in {1..30}; do
  if curl -fsS http://127.0.0.1:8222/healthz >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if ! curl -fsS http://127.0.0.1:8222/healthz >/dev/null 2>&1; then
  echo "NATS did not become ready." >&2
  exit 1
fi

echo "Building gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

echo "Starting gateway on port ${GATEWAY_PORT}..."
SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase1-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase1-gateway.log >&2
    exit 1
  fi
  sleep 1
done

echo "Submitting telemetry event..."
HTTP_CODE="$(
  curl -sS     -o /tmp/sentinel-phase1-ingest.json     -w "%{http_code}"     -X POST     -H "Content-Type: application/json"     --data "{
      \"event_id\": \"${EVENT_ID}\",
      \"timestamp\": \"${TIMESTAMP}\",
      \"source\": {
        \"type\": \"identity\",
        \"vendor\": \"sentinel-smoke\",
        \"collector\": \"phase1-smoke\"
      },
      \"asset\": {
        \"id\": \"asset-phase1-smoke\",
        \"hostname\": \"phase1-smoke-host\",
        \"zone\": \"lab\"
      },
      \"actor\": {
        \"id\": \"user-phase1-smoke\",
        \"type\": \"human\",
        \"name\": \"Phase One Smoke User\"
      },
      \"event\": {
        \"category\": \"authentication\",
        \"action\": \"login\",
        \"outcome\": \"success\"
      },
      \"labels\": {
        \"environment\": \"lab\",
        \"scenario\": \"phase1-smoke\"
      }
    }"     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry"
)"

if [[ "${HTTP_CODE}" != "202" ]]; then
  cat /tmp/sentinel-phase1-ingest.json >&2
  echo "Expected HTTP 202, got ${HTTP_CODE}." >&2
  exit 1
fi

if ! grep -q "${EVENT_ID}" /tmp/sentinel-phase1-ingest.json; then
  cat /tmp/sentinel-phase1-ingest.json >&2
  echo "Ingestion response did not contain event id ${EVENT_ID}." >&2
  exit 1
fi

echo "Waiting for durable persistence..."
FOUND="false"
for _ in {1..30}; do
  curl -fsS     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/events?asset_id=asset-phase1-smoke&limit=50"     >/tmp/sentinel-phase1-events.json

  if grep -q "${EVENT_ID}" /tmp/sentinel-phase1-events.json; then
    FOUND="true"
    break
  fi
  sleep 1
done

if [[ "${FOUND}" != "true" ]]; then
  cat /tmp/sentinel-phase1-events.json >&2
  echo "Accepted event was not persisted within the smoke-test window." >&2
  exit 1
fi

echo "Checking invalid telemetry rejection..."
INVALID_CODE="$(
  curl -sS     -o /tmp/sentinel-phase1-invalid.json     -w "%{http_code}"     -X POST     -H "Content-Type: application/json"     --data '{"source":{"type":"identity","collector":"smoke"},"event":{"category":"authentication","action":"login"}}'     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry"
)"

if [[ "${INVALID_CODE}" != "400" ]]; then
  cat /tmp/sentinel-phase1-invalid.json >&2
  echo "Expected invalid telemetry to return HTTP 400, got ${INVALID_CODE}." >&2
  exit 1
fi

echo "Phase 1 telemetry smoke test passed: ${EVENT_ID}"
