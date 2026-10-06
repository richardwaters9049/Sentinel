#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE2_SMOKE_PORT:-18082}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase2-smoke-gateway"
RUN_ID="$(date +%s)_$RANDOM"
IDENTITY_ID="user-phase2-${RUN_ID}"
ASSET_ID="asset-phase2-${RUN_ID}"
SOURCE_IP="10.77.0.44"

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

echo "Building Phase 2 gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

echo "Starting gateway on port ${GATEWAY_PORT}..."
SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase2-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase2-gateway.log >&2
    exit 1
  fi
  sleep 1
done

post_login() {
  local event_id="$1"
  local timestamp="$2"
  local outcome="$3"

  curl -fsS     -X POST     -H "Content-Type: application/json"     --data "{
      \"event_id\": \"${event_id}\",
      \"timestamp\": \"${timestamp}\",
      \"source\": {
        \"type\": \"identity\",
        \"vendor\": \"sentinel-phase2-smoke\",
        \"collector\": \"phase2-smoke\"
      },
      \"asset\": {
        \"id\": \"${ASSET_ID}\",
        \"hostname\": \"${ASSET_ID}\",
        \"zone\": \"ot\"
      },
      \"actor\": {
        \"id\": \"${IDENTITY_ID}\",
        \"type\": \"human\",
        \"name\": \"Phase Two Demo Operator\"
      },
      \"event\": {
        \"category\": \"authentication\",
        \"action\": \"login\",
        \"outcome\": \"${outcome}\"
      },
      \"network\": {
        \"source_ip\": \"${SOURCE_IP}\",
        \"destination_ip\": \"10.20.0.15\",
        \"destination_port\": 443,
        \"protocol\": \"tcp\"
      },
      \"labels\": {
        \"environment\": \"lab\",
        \"scenario\": \"phase2-auth-burst\"
      }
    }"     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null
}

BASE_EPOCH="$(date -u +%s)"
echo "Submitting four failed logins..."
for index in 0 1 2 3; do
  EVENT_TIME="$(date -u -r "$((BASE_EPOCH + index * 10))" +"%Y-%m-%dT%H:%M:%SZ")"
  post_login "evt_phase2_fail_${RUN_ID}_${index}" "${EVENT_TIME}" "failure"
done

echo "Submitting successful login..."
SUCCESS_TIME="$(date -u -r "$((BASE_EPOCH + 45))" +"%Y-%m-%dT%H:%M:%SZ")"
SUCCESS_ID="evt_phase2_success_${RUN_ID}"
post_login "${SUCCESS_ID}" "${SUCCESS_TIME}" "success"

echo "Waiting for DET-AUTH-001 finding..."
FOUND="false"
for _ in {1..30}; do
  curl -fsS     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings?status=new&severity=high&limit=100"     >/tmp/sentinel-phase2-findings.json

  if grep -q "${SUCCESS_ID}" /tmp/sentinel-phase2-findings.json; then
    FOUND="true"
    break
  fi
  sleep 1
done

if [[ "${FOUND}" != "true" ]]; then
  cat /tmp/sentinel-phase2-findings.json >&2
  echo "Expected DET-AUTH-001 finding was not created." >&2
  exit 1
fi

if ! grep -q '"detection_id":"DET-AUTH-001"' /tmp/sentinel-phase2-findings.json; then
  cat /tmp/sentinel-phase2-findings.json >&2
  echo "Finding does not reference DET-AUTH-001." >&2
  exit 1
fi

FINDING_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM findings WHERE evidence->>'success_event_id'='${SUCCESS_ID}';"
)"

if [[ "${FINDING_COUNT}" != "1" ]]; then
  echo "Expected exactly one finding for ${SUCCESS_ID}, found ${FINDING_COUNT}." >&2
  exit 1
fi

LINK_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM finding_events fe JOIN findings f ON f.id=fe.finding_id WHERE f.evidence->>'success_event_id'='${SUCCESS_ID}';"
)"

if [[ "${LINK_COUNT}" != "5" ]]; then
  echo "Expected five evidence-event links, found ${LINK_COUNT}." >&2
  exit 1
fi

echo "Phase 2 detection smoke test passed."
echo "  detection: DET-AUTH-001"
echo "  failures correlated: 4"
echo "  success event: ${SUCCESS_ID}"
echo "  evidence links: 5"
