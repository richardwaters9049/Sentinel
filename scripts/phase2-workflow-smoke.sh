#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE2_WORKFLOW_PORT:-18084}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase2-workflow-gateway"
RUN_ID="$(date +%s)_$RANDOM"
SOURCE_IP="10.88.0.44"

cleanup() {
  docker compose start postgres nats >/dev/null 2>&1 || true
  docker compose exec -T postgres psql -U sentinel -d sentinel -c     "UPDATE detections SET enabled=TRUE WHERE id='DET-AUTH-001';" >/dev/null 2>&1 || true

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

echo "Building Phase 2 workflow gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

echo "Starting gateway on port ${GATEWAY_PORT}..."
SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase2-workflow.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase2-workflow.log >&2
    exit 1
  fi
  sleep 1
done

docker compose exec -T postgres psql -U sentinel -d sentinel -c   "UPDATE detections SET enabled=TRUE WHERE id='DET-AUTH-001';" >/dev/null

post_login() {
  local event_id="$1"
  local identity_id="$2"
  local asset_id="$3"
  local timestamp="$4"
  local outcome="$5"

  curl -fsS     -X POST     -H "Content-Type: application/json"     --data "{
      \"event_id\": \"${event_id}\",
      \"timestamp\": \"${timestamp}\",
      \"source\": {
        \"type\": \"identity\",
        \"vendor\": \"sentinel-phase2-workflow\",
        \"collector\": \"phase2-workflow\"
      },
      \"asset\": {
        \"id\": \"${asset_id}\",
        \"hostname\": \"${asset_id}\",
        \"zone\": \"ot\"
      },
      \"actor\": {
        \"id\": \"${identity_id}\",
        \"type\": \"human\",
        \"name\": \"Phase Two Workflow Operator\"
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
        \"scenario\": \"phase2-workflow\"
      }
    }"     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null
}

emit_auth_burst() {
  local suffix="$1"
  local identity_id="user-phase2-workflow-${RUN_ID}-${suffix}"
  local asset_id="asset-phase2-workflow-${RUN_ID}-${suffix}"
  local base_epoch
  base_epoch="$(date -u +%s)"

  for index in 0 1 2 3; do
    local event_time
    event_time="$(date -u -r "$((base_epoch + index * 10))" +"%Y-%m-%dT%H:%M:%SZ")"
    post_login       "evt_phase2_workflow_fail_${RUN_ID}_${suffix}_${index}"       "${identity_id}"       "${asset_id}"       "${event_time}"       "failure"
  done

  local success_time
  success_time="$(date -u -r "$((base_epoch + 45))" +"%Y-%m-%dT%H:%M:%SZ")"
  local success_id="evt_phase2_workflow_success_${RUN_ID}_${suffix}"

  post_login     "${success_id}"     "${identity_id}"     "${asset_id}"     "${success_time}"     "success"

  printf '%s' "${success_id}"
}

echo "Creating an analyst-workflow finding..."
SUCCESS_ID="$(emit_auth_burst enabled)"

FINDING_ID=""
for _ in {1..30}; do
  FINDING_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE evidence->>'success_event_id'='${SUCCESS_ID}' LIMIT 1;"
  )"
  if [[ -n "${FINDING_ID}" ]]; then
    break
  fi
  sleep 1
done

if [[ -z "${FINDING_ID}" ]]; then
  echo "Expected workflow finding was not created." >&2
  exit 1
fi

echo "Checking finding detail evidence..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings/${FINDING_ID}"   >/tmp/sentinel-phase2-detail.json

EVENT_COUNT="$(python3 -c 'import json; d=json.load(open("/tmp/sentinel-phase2-detail.json")); print(len(d["events"]))')"
if [[ "${EVENT_COUNT}" != "5" ]]; then
  cat /tmp/sentinel-phase2-detail.json >&2
  echo "Expected five evidence events, found ${EVENT_COUNT}." >&2
  exit 1
fi

echo "Checking invalid workflow transition..."
INVALID_CODE="$(
  curl -sS     -o /tmp/sentinel-phase2-invalid-transition.json     -w "%{http_code}"     -X PATCH     -H "Content-Type: application/json"     -H "X-Sentinel-Actor: analyst-workflow-test"     --data '{"status":"confirmed"}'     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings/${FINDING_ID}/status"
)"
if [[ "${INVALID_CODE}" != "409" ]]; then
  cat /tmp/sentinel-phase2-invalid-transition.json >&2
  echo "Expected invalid transition to return HTTP 409, got ${INVALID_CODE}." >&2
  exit 1
fi

echo "Walking finding through analyst workflow..."
for target in triaged investigating confirmed contained closed; do
  curl -fsS     -X PATCH     -H "Content-Type: application/json"     -H "X-Sentinel-Actor: analyst-workflow-test"     -H "X-Request-ID: req-${RUN_ID}-${target}"     --data "{\"status\":\"${target}\"}"     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings/${FINDING_ID}/status"     >/tmp/sentinel-phase2-status.json

  if ! grep -q "\"status\":\"${target}\"" /tmp/sentinel-phase2-status.json; then
    cat /tmp/sentinel-phase2-status.json >&2
    echo "Finding did not transition to ${target}." >&2
    exit 1
  fi
done

curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings/${FINDING_ID}"   >/tmp/sentinel-phase2-final-detail.json

AUDIT_COUNT="$(python3 -c 'import json; d=json.load(open("/tmp/sentinel-phase2-final-detail.json")); print(len(d["audit"]))')"
if [[ "${AUDIT_COUNT}" != "5" ]]; then
  cat /tmp/sentinel-phase2-final-detail.json >&2
  echo "Expected five finding audit records, found ${AUDIT_COUNT}." >&2
  exit 1
fi

echo "Checking detection metadata..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/detections"   >/tmp/sentinel-phase2-detections.json

if ! grep -q '"id":"DET-AUTH-001"' /tmp/sentinel-phase2-detections.json; then
  cat /tmp/sentinel-phase2-detections.json >&2
  echo "DET-AUTH-001 was not returned by the detection API." >&2
  exit 1
fi

echo "Disabling DET-AUTH-001 through the audited API..."
curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: analyst-workflow-test"   -H "X-Request-ID: req-${RUN_ID}-disable"   --data '{"enabled":false}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/detections/DET-AUTH-001"   >/tmp/sentinel-phase2-disable.json

if ! grep -q '"enabled":false' /tmp/sentinel-phase2-disable.json; then
  cat /tmp/sentinel-phase2-disable.json >&2
  echo "Detection was not disabled." >&2
  exit 1
fi

DISABLED_SUCCESS_ID="$(emit_auth_burst disabled)"
sleep 2

DISABLED_FINDING_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM findings WHERE evidence->>'success_event_id'='${DISABLED_SUCCESS_ID}';"
)"
if [[ "${DISABLED_FINDING_COUNT}" != "0" ]]; then
  echo "Disabled detection still generated a finding." >&2
  exit 1
fi

echo "Re-enabling DET-AUTH-001..."
curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: analyst-workflow-test"   -H "X-Request-ID: req-${RUN_ID}-enable"   --data '{"enabled":true}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/detections/DET-AUTH-001"   >/tmp/sentinel-phase2-enable.json

REENABLED_SUCCESS_ID="$(emit_auth_burst reenabled)"
REENABLED_FOUND="false"
for _ in {1..30}; do
  COUNT="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT COUNT(*) FROM findings WHERE evidence->>'success_event_id'='${REENABLED_SUCCESS_ID}';"
  )"
  if [[ "${COUNT}" == "1" ]]; then
    REENABLED_FOUND="true"
    break
  fi
  sleep 1
done

if [[ "${REENABLED_FOUND}" != "true" ]]; then
  echo "Re-enabled detection did not generate a finding." >&2
  exit 1
fi

DETECTION_AUDIT_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM audit_events WHERE resource_type='detection' AND resource_id='DET-AUTH-001' AND actor_id='analyst-workflow-test';"
)"
if [[ "${DETECTION_AUDIT_COUNT}" -lt "2" ]]; then
  echo "Expected audited detection state changes." >&2
  exit 1
fi

echo "Phase 2 analyst workflow smoke test passed."
echo "  finding detail: verified"
echo "  workflow transitions: verified"
echo "  finding audit trail: verified"
echo "  detection disable: verified"
echo "  disabled rule suppression: verified"
echo "  detection re-enable: verified"
echo "  detection audit trail: verified"
