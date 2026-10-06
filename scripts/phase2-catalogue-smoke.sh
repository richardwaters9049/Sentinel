#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE2_CATALOGUE_PORT:-18085}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase2-catalogue-gateway"
RUN_ID="$(date +%s)_$RANDOM"

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

echo "Building catalogue-test gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

echo "Starting gateway on port ${GATEWAY_PORT}..."
SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase2-catalogue.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase2-catalogue.log >&2
    exit 1
  fi
  sleep 1
done

TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
SERVICE_EVENT_ID="evt_service_catalogue_${RUN_ID}"
ITOT_EVENT_ID="evt_itot_catalogue_${RUN_ID}"

echo "Submitting service-account interactive login..."
curl -fsS   -X POST   -H "Content-Type: application/json"   --data "{
    \"event_id\": \"${SERVICE_EVENT_ID}\",
    \"timestamp\": \"${TIMESTAMP}\",
    \"source\": {
      \"type\": \"identity\",
      \"vendor\": \"sentinel-phase2-catalogue\",
      \"collector\": \"catalogue-smoke\"
    },
    \"asset\": {
      \"id\": \"asset-service-${RUN_ID}\",
      \"hostname\": \"application-server-${RUN_ID}\",
      \"zone\": \"corporate\"
    },
    \"actor\": {
      \"id\": \"svc-backup-${RUN_ID}\",
      \"type\": \"service_account\",
      \"name\": \"Backup Service\"
    },
    \"event\": {
      \"category\": \"authentication\",
      \"action\": \"login\",
      \"outcome\": \"success\"
    },
    \"network\": {
      \"source_ip\": \"10.66.0.20\",
      \"destination_ip\": \"10.66.0.30\",
      \"destination_port\": 22,
      \"protocol\": \"tcp\"
    },
    \"labels\": {
      \"environment\": \"lab\",
      \"scenario\": \"service-account-login\"
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null

echo "Submitting corporate-to-OT connection..."
curl -fsS   -X POST   -H "Content-Type: application/json"   --data "{
    \"event_id\": \"${ITOT_EVENT_ID}\",
    \"timestamp\": \"${TIMESTAMP}\",
    \"source\": {
      \"type\": \"network\",
      \"vendor\": \"sentinel-phase2-catalogue\",
      \"collector\": \"catalogue-smoke\"
    },
    \"asset\": {
      \"id\": \"asset-corporate-${RUN_ID}\",
      \"hostname\": \"employee-workstation-${RUN_ID}\",
      \"zone\": \"corporate\"
    },
    \"event\": {
      \"category\": \"network\",
      \"action\": \"connection\",
      \"outcome\": \"success\"
    },
    \"network\": {
      \"source_ip\": \"10.10.0.20\",
      \"destination_ip\": \"10.30.0.10\",
      \"destination_port\": 502,
      \"destination_zone\": \"ot\",
      \"protocol\": \"tcp\"
    },
    \"labels\": {
      \"environment\": \"lab\",
      \"scenario\": \"it-to-ot-connection\"
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null

echo "Waiting for new findings..."
SERVICE_FOUND="false"
ITOT_FOUND="false"

for _ in {1..30}; do
  curl -fsS     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings?status=new&limit=200"     >/tmp/sentinel-phase2-catalogue-findings.json

  if grep -q "${SERVICE_EVENT_ID}" /tmp/sentinel-phase2-catalogue-findings.json; then
    SERVICE_FOUND="true"
  fi
  if grep -q "${ITOT_EVENT_ID}" /tmp/sentinel-phase2-catalogue-findings.json; then
    ITOT_FOUND="true"
  fi

  if [[ "${SERVICE_FOUND}" == "true" && "${ITOT_FOUND}" == "true" ]]; then
    break
  fi
  sleep 1
done

if [[ "${SERVICE_FOUND}" != "true" ]]; then
  cat /tmp/sentinel-phase2-catalogue-findings.json >&2
  echo "DET-AUTH-002 finding was not created." >&2
  exit 1
fi

if [[ "${ITOT_FOUND}" != "true" ]]; then
  cat /tmp/sentinel-phase2-catalogue-findings.json >&2
  echo "DET-NET-001 finding was not created." >&2
  exit 1
fi

SERVICE_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM findings WHERE detection_id='DET-AUTH-002' AND evidence->>'terminal_event_id'='${SERVICE_EVENT_ID}';"
)"
ITOT_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM findings WHERE detection_id='DET-NET-001' AND evidence->>'terminal_event_id'='${ITOT_EVENT_ID}';"
)"

if [[ "${SERVICE_COUNT}" != "1" ]]; then
  echo "Expected one DET-AUTH-002 finding, found ${SERVICE_COUNT}." >&2
  exit 1
fi

if [[ "${ITOT_COUNT}" != "1" ]]; then
  echo "Expected one DET-NET-001 finding, found ${ITOT_COUNT}." >&2
  exit 1
fi

DETECTION_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM detections WHERE id IN ('DET-AUTH-001','DET-AUTH-002','DET-NET-001') AND enabled=TRUE;"
)"
if [[ "${DETECTION_COUNT}" != "3" ]]; then
  echo "Expected all three Phase 2 detections to be enabled, found ${DETECTION_COUNT}." >&2
  exit 1
fi

echo "Phase 2 detection catalogue smoke test passed."
echo "  DET-AUTH-001: registered"
echo "  DET-AUTH-002: service-account login finding verified"
echo "  DET-NET-001: corporate-to-OT connection finding verified"
