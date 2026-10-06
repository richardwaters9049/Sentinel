#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE1_RESILIENCE_PORT:-18083}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase1-resilience-gateway"
RUN_ID="$(date +%s)_$RANDOM"

cleanup() {
  docker compose start postgres nats >/dev/null 2>&1 || true

  if [[ -n "${GATEWAY_PID}" ]] && kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    kill "${GATEWAY_PID}" 2>/dev/null || true
    wait "${GATEWAY_PID}" 2>/dev/null || true
  fi

  rm -f "${GATEWAY_BINARY}"
}

trap cleanup EXIT

cd "${ROOT_DIR}"
docker compose up -d postgres nats >/dev/null

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

echo "Building resilience-test gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

echo "Starting resilience-test gateway on port ${GATEWAY_PORT}..."
SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase1-resilience-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase1-resilience-gateway.log >&2
    exit 1
  fi
  sleep 1
done

post_event() {
  local event_id="$1"
  local asset_id="$2"
  local collector="$3"
  local output_file="$4"
  local timestamp
  timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  curl -sS     -o "${output_file}"     -w "%{http_code}"     -X POST     -H "Content-Type: application/json"     --data "{
      \"event_id\": \"${event_id}\",
      \"timestamp\": \"${timestamp}\",
      \"source\": {
        \"type\": \"identity\",
        \"vendor\": \"sentinel-resilience\",
        \"collector\": \"${collector}\"
      },
      \"asset\": {
        \"id\": \"${asset_id}\",
        \"hostname\": \"${asset_id}\",
        \"zone\": \"lab\"
      },
      \"event\": {
        \"category\": \"authentication\",
        \"action\": \"login\",
        \"outcome\": \"success\"
      },
      \"labels\": {
        \"environment\": \"lab\",
        \"scenario\": \"phase1-resilience\"
      }
    }"     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry"
}

echo "Checking duplicate idempotency..."
DUPLICATE_ID="evt_duplicate_${RUN_ID}"
DUPLICATE_ASSET="asset-duplicate-${RUN_ID}"

CODE="$(post_event "${DUPLICATE_ID}" "${DUPLICATE_ASSET}" "duplicate-test" /tmp/sentinel-resilience-dup1.json)"
[[ "${CODE}" == "202" ]] || {
  cat /tmp/sentinel-resilience-dup1.json >&2
  exit 1
}

CODE="$(post_event "${DUPLICATE_ID}" "${DUPLICATE_ASSET}" "duplicate-test" /tmp/sentinel-resilience-dup2.json)"
[[ "${CODE}" == "202" ]] || {
  cat /tmp/sentinel-resilience-dup2.json >&2
  exit 1
}

sleep 1

DUPLICATE_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM events WHERE id='${DUPLICATE_ID}';"
)"

if [[ "${DUPLICATE_COUNT}" != "1" ]]; then
  echo "Expected one stored duplicate-test event, found ${DUPLICATE_COUNT}." >&2
  exit 1
fi

echo "Checking NATS outage behaviour..."
docker compose stop nats >/dev/null
sleep 1

READY_CODE=""
for _ in {1..10}; do
  READY_CODE="$(
    curl -sS       -o /tmp/sentinel-resilience-nats-ready.json       -w "%{http_code}"       "http://127.0.0.1:${GATEWAY_PORT}/ready" 2>/dev/null || true
  )"

  if [[ "${READY_CODE}" == "503" ]]; then
    break
  fi

  sleep 1
done

[[ "${READY_CODE}" == "503" ]] || {
  echo "Expected readiness 503 with NATS down, got ${READY_CODE:-no response}." >&2
  exit 1
}

NATS_FAILURE_ID="evt_nats_down_${RUN_ID}"
NATS_CODE="$(post_event "${NATS_FAILURE_ID}" "asset-nats-down-${RUN_ID}" "nats-down-test" /tmp/sentinel-resilience-nats-post.json)"
[[ "${NATS_CODE}" == "503" ]] || {
  cat /tmp/sentinel-resilience-nats-post.json >&2
  echo "Expected ingestion 503 with NATS down, got ${NATS_CODE}." >&2
  exit 1
}

docker compose start nats >/dev/null

NATS_RECOVERED="false"
for _ in {1..20}; do
  CODE="$(
    curl -sS       -o /tmp/sentinel-resilience-nats-up.json       -w "%{http_code}"       "http://127.0.0.1:${GATEWAY_PORT}/ready" || true
  )"

  if [[ "${CODE}" == "200" ]]; then
    NATS_RECOVERED="true"
    break
  fi

  sleep 1
done

[[ "${NATS_RECOVERED}" == "true" ]] || {
  echo "Gateway did not recover its NATS connection." >&2
  exit 1
}

echo "Checking PostgreSQL outage and JetStream redelivery..."
docker compose stop postgres >/dev/null
sleep 1

READY_CODE=""
for _ in {1..10}; do
  READY_CODE="$(
    curl -sS       -o /tmp/sentinel-resilience-postgres-ready.json       -w "%{http_code}"       "http://127.0.0.1:${GATEWAY_PORT}/ready" 2>/dev/null || true
  )"

  if [[ "${READY_CODE}" == "503" ]]; then
    break
  fi

  sleep 1
done

[[ "${READY_CODE}" == "503" ]] || {
  echo "Expected readiness 503 with PostgreSQL down, got ${READY_CODE:-no response}." >&2
  exit 1
}

RECOVERY_ID="evt_postgres_recovery_${RUN_ID}"
RECOVERY_ASSET="asset-postgres-recovery-${RUN_ID}"
POSTGRES_CODE="$(post_event "${RECOVERY_ID}" "${RECOVERY_ASSET}" "postgres-recovery-test" /tmp/sentinel-resilience-postgres-post.json)"
[[ "${POSTGRES_CODE}" == "202" ]] || {
  cat /tmp/sentinel-resilience-postgres-post.json >&2
  echo "Expected ingestion 202 while PostgreSQL was down, got ${POSTGRES_CODE}." >&2
  exit 1
}

docker compose start postgres >/dev/null

POSTGRES_RECOVERED="false"
for _ in {1..30}; do
  CODE="$(
    curl -sS       -o /tmp/sentinel-resilience-postgres-up.json       -w "%{http_code}"       "http://127.0.0.1:${GATEWAY_PORT}/ready" || true
  )"

  if [[ "${CODE}" == "200" ]]; then
    POSTGRES_RECOVERED="true"
    break
  fi

  sleep 1
done

[[ "${POSTGRES_RECOVERED}" == "true" ]] || {
  echo "Gateway did not recover its PostgreSQL connection." >&2
  exit 1
}

PERSISTED="false"
for _ in {1..30}; do
  RESPONSE="$(
    curl -fsS       "http://127.0.0.1:${GATEWAY_PORT}/api/v1/events?asset_id=${RECOVERY_ASSET}&limit=20"       2>/dev/null || true
  )"

  if echo "${RESPONSE}" | grep -q "${RECOVERY_ID}"; then
    PERSISTED="true"
    break
  fi

  sleep 1
done

[[ "${PERSISTED}" == "true" ]] || {
  echo "Queued event was not persisted after PostgreSQL recovery." >&2
  exit 1
}

echo "Phase 1 resilience test passed."
echo "  duplicate idempotency: verified"
echo "  NATS failure/recovery: verified"
echo "  PostgreSQL buffering/redelivery: verified"
