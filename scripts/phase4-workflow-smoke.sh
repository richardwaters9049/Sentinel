#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE4_GATEWAY_PORT:-18094}"
CONSOLE_PORT="${SENTINEL_PHASE4_CONSOLE_PORT:-13004}"
GATEWAY_PID=""
CONSOLE_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase4-smoke-gateway"
RUN_ID="$(date +%s)_$RANDOM"
EVENT_ID="evt_phase4_console_${RUN_ID}"
ACTOR_ID="svc-phase4-${RUN_ID}"
ASSET_ID="asset-phase4-${RUN_ID}"
DETECTION_ID=""
DETECTION_ORIGINAL=""
DETECTION_TOGGLED=0

restore_detection() {
  if [[ "${DETECTION_TOGGLED}" == "1" && -n "${DETECTION_ID}" && -n "${DETECTION_ORIGINAL}" ]]; then
    curl -fsS       -X PATCH       -H "Content-Type: application/json"       -H "X-Sentinel-Actor: phase4-regression"       --data "{\"enabled\":${DETECTION_ORIGINAL}}"       "http://127.0.0.1:${CONSOLE_PORT}/api/sentinel/api/v1/detections/${DETECTION_ID}"       >/dev/null 2>&1 || true
  fi
}

cleanup() {
  restore_detection
  if [[ -n "${CONSOLE_PID}" ]] && kill -0 "${CONSOLE_PID}" 2>/dev/null; then
    kill "${CONSOLE_PID}" 2>/dev/null || true
    wait "${CONSOLE_PID}" 2>/dev/null || true
  fi
  if [[ -n "${GATEWAY_PID}" ]] && kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    kill "${GATEWAY_PID}" 2>/dev/null || true
    wait "${GATEWAY_PID}" 2>/dev/null || true
  fi
  rm -f "${GATEWAY_BINARY}"
}

trap cleanup EXIT
cd "${ROOT_DIR}"

echo "Starting Phase 4 dependencies..."
docker compose up -d postgres nats >/dev/null

for _ in {1..30}; do
  docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1 && break
  sleep 1
done
docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1 || {
  echo "PostgreSQL did not become ready." >&2
  exit 1
}

for _ in {1..30}; do
  curl -fsS http://127.0.0.1:8222/healthz >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS http://127.0.0.1:8222/healthz >/dev/null 2>&1 || {
  echo "NATS did not become ready." >&2
  exit 1
}

echo "Building and starting isolated gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)
SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase4-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1 && break
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase4-gateway.log >&2
    exit 1
  fi
  sleep 1
done

echo "Starting production-built analyst console..."
(
  cd apps/console
  SENTINEL_CONSOLE_AUTH_MODE=development SENTINEL_GATEWAY_URL="http://127.0.0.1:${GATEWAY_PORT}"     bun run start -- -p "${CONSOLE_PORT}"
) >/tmp/sentinel-phase4-console.log 2>&1 &
CONSOLE_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${CONSOLE_PORT}/" >/dev/null 2>&1 && break
  if ! kill -0 "${CONSOLE_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase4-console.log >&2
    exit 1
  fi
  sleep 1
done

BASE="http://127.0.0.1:${CONSOLE_PORT}/api/sentinel/api/v1"

echo "Verifying primary analyst routes..."
for route in / /environment /findings /hunts /investigations /detections; do
  code="$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:${CONSOLE_PORT}${route}")"
  [[ "${code}" == "200" ]] || {
    echo "Route ${route} returned ${code}." >&2
    exit 1
  }
done

TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo "Submitting synthetic service-account telemetry..."
curl -fsS   -X POST   -H "Content-Type: application/json"   --data "{
    \"event_id\": \"${EVENT_ID}\",
    \"timestamp\": \"${TIMESTAMP}\",
    \"source\": {
      \"type\": \"identity\",
      \"vendor\": \"sentinel-phase4\",
      \"collector\": \"phase4-console-smoke\"
    },
    \"asset\": {
      \"id\": \"${ASSET_ID}\",
      \"hostname\": \"${ASSET_ID}\",
      \"zone\": \"corporate\"
    },
    \"actor\": {
      \"id\": \"${ACTOR_ID}\",
      \"type\": \"service_account\",
      \"name\": \"Phase Four Service\"
    },
    \"event\": {
      \"category\": \"authentication\",
      \"action\": \"login\",
      \"outcome\": \"success\"
    },
    \"network\": {
      \"source_ip\": \"10.104.0.20\",
      \"destination_ip\": \"10.104.0.30\",
      \"destination_port\": 22,
      \"destination_zone\": \"dmz\",
      \"protocol\": \"tcp\"
    },
    \"labels\": {
      \"environment\": \"lab\",
      \"scenario\": \"phase4-console\"
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null

echo "Waiting for detection finding..."
FINDING_ID=""
for _ in {1..30}; do
  FINDING_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-AUTH-002' AND evidence->>'terminal_event_id'='${EVENT_ID}' LIMIT 1;"
  )"
  [[ -n "${FINDING_ID}" ]] && break
  sleep 1
done
[[ -n "${FINDING_ID}" ]] || {
  echo "Expected DET-AUTH-002 finding was not created." >&2
  exit 1
}

echo "Verifying overview data APIs and finding evidence..."
curl -fsS "${BASE}/events?limit=20" >/tmp/sentinel-phase4-events.json
curl -fsS "${BASE}/findings?limit=20" >/tmp/sentinel-phase4-findings.json
curl -fsS "${BASE}/findings/${FINDING_ID}" >/tmp/sentinel-phase4-finding.json
curl -fsS "${BASE}/findings/${FINDING_ID}/evidence?context_minutes=5" >/tmp/sentinel-phase4-evidence.json

python3 - <<PY
import json
events=json.load(open("/tmp/sentinel-phase4-events.json"))
findings=json.load(open("/tmp/sentinel-phase4-findings.json"))
detail=json.load(open("/tmp/sentinel-phase4-finding.json"))
evidence=json.load(open("/tmp/sentinel-phase4-evidence.json"))
assert events["count"] >= 1
assert any(item["id"] == "${FINDING_ID}" for item in findings["findings"])
assert detail["detection_id"] == "DET-AUTH-002"
assert any(item["id"] == "${EVENT_ID}" for item in evidence["linked_events"])
PY

echo "Creating and running a hunt through the console proxy..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase4-regression"   --data "{
    \"name\": \"Phase 4 regression hunt ${RUN_ID}\",
    \"description\": \"Console end-to-end hunt.\",
    \"hypothesis\": \"The synthetic service identity should be observable in recent authentication telemetry.\",
    \"query\": {
      \"category\": \"authentication\",
      \"identity_id\": \"${ACTOR_ID}\",
      \"limit\": 20
    }
  }"   "${BASE}/hunts" >/tmp/sentinel-phase4-hunt.json

HUNT_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase4-hunt.json"))["id"])')"
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase4-regression"   --data '{"override":{}}'   "${BASE}/hunts/${HUNT_ID}/run" >/tmp/sentinel-phase4-run.json

RUN_ID_VALUE="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase4-run.json"))["run_id"])')"
python3 - <<PY
import json
run=json.load(open("/tmp/sentinel-phase4-run.json"))
assert run["result_count"] >= 1
assert any(item["id"] == "${EVENT_ID}" for item in run["events"])
PY

echo "Escalating hunt evidence into an investigation..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase4-regression"   --data "{
    \"title\": \"Phase 4 regression case ${RUN_ID}\",
    \"description\": \"End-to-end analyst workflow verification.\",
    \"priority\": \"high\",
    \"owner_id\": \"phase4-regression\",
    \"finding_ids\": [\"${FINDING_ID}\"],
    \"event_ids\": [\"${EVENT_ID}\"]
  }"   "${BASE}/investigations" >/tmp/sentinel-phase4-case.json

INVESTIGATION_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase4-case.json"))["id"])')"

curl -fsS   -X POST   -H "X-Sentinel-Actor: phase4-regression"   "${BASE}/investigations/${INVESTIGATION_ID}/hunt-runs/${RUN_ID_VALUE}"   >/tmp/sentinel-phase4-attached.json

curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase4-regression"   --data '{"body":"Phase Four regression verified the hunt evidence and promoted it into active investigation."}'   "${BASE}/investigations/${INVESTIGATION_ID}/notes"   >/tmp/sentinel-phase4-note.json

curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase4-regression"   --data '{"status":"investigating"}'   "${BASE}/investigations/${INVESTIGATION_ID}/status"   >/tmp/sentinel-phase4-status.json

python3 - <<'PY'
import json
detail=json.load(open("/tmp/sentinel-phase4-status.json"))
assert detail["status"] == "investigating"
assert len(detail["findings"]) >= 1
assert len(detail["events"]) >= 1
assert len(detail["notes"]) >= 1
actions={item["action"] for item in detail["audit"]}
assert "investigation.created" in actions
assert "investigation.hunt_run_attached" in actions
assert "investigation.note_added" in actions
assert "investigation.status_changed" in actions
PY

echo "Verifying audited detection runtime control..."
curl -fsS "${BASE}/detections" >/tmp/sentinel-phase4-detections.json
curl -fsS "${BASE}/detections/metrics" >/tmp/sentinel-phase4-detection-metrics.json

DETECTION_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase4-detections.json"))["detections"][0]["id"])')"
DETECTION_ORIGINAL="$(python3 -c 'import json; print(str(json.load(open("/tmp/sentinel-phase4-detections.json"))["detections"][0]["enabled"]).lower())')"
if [[ "${DETECTION_ORIGINAL}" == "true" ]]; then
  DETECTION_TARGET=false
else
  DETECTION_TARGET=true
fi

curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase4-regression"   --data "{\"enabled\":${DETECTION_TARGET}}"   "${BASE}/detections/${DETECTION_ID}" >/tmp/sentinel-phase4-detection-toggle.json
DETECTION_TOGGLED=1

python3 - <<PY
import json
d=json.load(open("/tmp/sentinel-phase4-detection-toggle.json"))
assert d["id"] == "${DETECTION_ID}"
assert str(d["enabled"]).lower() == "${DETECTION_TARGET}"
PY

restore_detection
DETECTION_TOGGLED=0

echo "Phase 4 analyst-console workflow smoke test passed."
echo "  primary routes: verified"
echo "  overview data: verified"
echo "  finding detail/evidence: verified"
echo "  hunt create/run: verified"
echo "  hunt → investigation pivot: verified"
echo "  investigation notes/status/audit: verified"
echo "  detection metrics/control: verified"
