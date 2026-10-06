#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE3_SMOKE_PORT:-18087}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase3-smoke-gateway"
RUN_ID="$(date +%s)_$RANDOM"
EVENT_ID="evt_phase3_service_${RUN_ID}"
ACTOR_ID="svc-phase3-${RUN_ID}"
ASSET_ID="asset-phase3-${RUN_ID}"

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

echo "Building Phase 3 gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase3-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1 && break
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase3-gateway.log >&2
    exit 1
  fi
  sleep 1
done

TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

echo "Submitting huntable service-account telemetry..."
curl -fsS   -X POST   -H "Content-Type: application/json"   --data "{
    \"event_id\": \"${EVENT_ID}\",
    \"timestamp\": \"${TIMESTAMP}\",
    \"source\": {
      \"type\": \"identity\",
      \"vendor\": \"sentinel-phase3\",
      \"collector\": \"phase3-smoke\"
    },
    \"asset\": {
      \"id\": \"${ASSET_ID}\",
      \"hostname\": \"${ASSET_ID}\",
      \"zone\": \"corporate\"
    },
    \"actor\": {
      \"id\": \"${ACTOR_ID}\",
      \"type\": \"service_account\",
      \"name\": \"Phase Three Service\"
    },
    \"event\": {
      \"category\": \"authentication\",
      \"action\": \"login\",
      \"outcome\": \"success\"
    },
    \"network\": {
      \"source_ip\": \"10.103.0.20\",
      \"destination_ip\": \"10.103.0.30\",
      \"destination_port\": 22,
      \"protocol\": \"tcp\"
    },
    \"labels\": {
      \"environment\": \"lab\",
      \"scenario\": \"phase3-hunt\"
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null

echo "Waiting for finding..."
FINDING_ID=""
for _ in {1..30}; do
  FINDING_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-AUTH-002' AND evidence->>'terminal_event_id'='${EVENT_ID}' LIMIT 1;"
  )"
  [[ -n "${FINDING_ID}" ]] && break
  sleep 1
done

if [[ -z "${FINDING_ID}" ]]; then
  echo "Expected DET-AUTH-002 finding was not created." >&2
  exit 1
fi

echo "Creating saved hunt..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-hunter"   --data "{
    \"name\": \"Phase 3 service-account hunt\",
    \"description\": \"Find successful authentication activity for the synthetic service identity.\",
    \"hypothesis\": \"A service identity may be used interactively from a corporate asset.\",
    \"query\": {
      \"category\": \"authentication\",
      \"action\": \"login\",
      \"outcome\": \"success\",
      \"identity_id\": \"${ACTOR_ID}\",
      \"limit\": 50
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts"   >/tmp/sentinel-phase3-hunt.json

HUNT_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-hunt.json"))["id"])')"
if [[ -z "${HUNT_ID}" ]]; then
  cat /tmp/sentinel-phase3-hunt.json >&2
  echo "Saved hunt was not created." >&2
  exit 1
fi

echo "Executing saved hunt..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-hunter"   --data '{"override":{}}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts/${HUNT_ID}/run"   >/tmp/sentinel-phase3-run.json

if ! grep -q "${EVENT_ID}" /tmp/sentinel-phase3-run.json; then
  cat /tmp/sentinel-phase3-run.json >&2
  echo "Hunt did not return the target event." >&2
  exit 1
fi

RUN_COUNT="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-run.json"))["result_count"])')"
if [[ "${RUN_COUNT}" -lt "1" ]]; then
  echo "Expected hunt to return at least one event." >&2
  exit 1
fi

echo "Creating investigation from finding and event..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-analyst"   -H "X-Request-ID: req-phase3-${RUN_ID}-create"   --data "{
    \"title\": \"Investigate service-account login ${RUN_ID}\",
    \"description\": \"Investigation created from Phase 3 hunt output and its related finding.\",
    \"priority\": \"high\",
    \"owner_id\": \"phase3-analyst\",
    \"finding_ids\": [\"${FINDING_ID}\"],
    \"event_ids\": [\"${EVENT_ID}\"]
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations"   >/tmp/sentinel-phase3-investigation.json

INVESTIGATION_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-investigation.json"))["id"])')"
if [[ -z "${INVESTIGATION_ID}" ]]; then
  cat /tmp/sentinel-phase3-investigation.json >&2
  echo "Investigation was not created." >&2
  exit 1
fi

echo "Adding analyst note..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-analyst"   -H "X-Request-ID: req-phase3-${RUN_ID}-note"   --data '{"body":"Service-account activity confirmed in saved-hunt results; validating whether usage was approved."}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations/${INVESTIGATION_ID}/notes"   >/tmp/sentinel-phase3-note.json

echo "Moving investigation into active analysis..."
curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-analyst"   -H "X-Request-ID: req-phase3-${RUN_ID}-status"   --data '{"status":"investigating"}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations/${INVESTIGATION_ID}/status"   >/tmp/sentinel-phase3-status.json

echo "Verifying investigation timeline..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations/${INVESTIGATION_ID}"   >/tmp/sentinel-phase3-detail.json

python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase3-detail.json"))
assert d["status"] == "investigating", d
assert len(d["findings"]) == 1, d
assert len(d["events"]) == 1, d
assert len(d["notes"]) == 1, d
types={item["type"] for item in d["timeline"]}
assert {"finding","event","note","audit"}.issubset(types), d["timeline"]
PY

HUNT_RUN_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM hunt_runs WHERE hunt_id='${HUNT_ID}' AND actor_id='phase3-hunter';"
)"
if [[ "${HUNT_RUN_COUNT}" -lt "1" ]]; then
  echo "Hunt execution history was not recorded." >&2
  exit 1
fi

echo "Phase 3 threat-hunting smoke test passed."
echo "  saved hunt: verified"
echo "  hypothesis/query persistence: verified"
echo "  hunt execution: verified"
echo "  hunt run history: verified"
echo "  investigation creation: verified"
echo "  finding/event pivots: verified"
echo "  analyst notes: verified"
echo "  investigation workflow: verified"
echo "  unified timeline: verified"
