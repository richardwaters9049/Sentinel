#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE3_PIVOT_PORT:-18088}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase3-pivot-gateway"
RUN_ID="$(date +%s)_$RANDOM"
ASSET_ID="asset-phase3-pivot-${RUN_ID}"
IDENTITY_ID="svc-phase3-pivot-${RUN_ID}"
LOGIN_EVENT_ID="evt_phase3_pivot_login_${RUN_ID}"
CONTEXT_EVENT_ID="evt_phase3_pivot_context_${RUN_ID}"

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

echo "Building Phase 3 pivot gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase3-pivot.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1 && break
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase3-pivot.log >&2
    exit 1
  fi
  sleep 1
done

BASE_EPOCH="$(date -u +%s)"
CONTEXT_TIME="$(date -u -r "$((BASE_EPOCH - 20))" +"%Y-%m-%dT%H:%M:%SZ")"
LOGIN_TIME="$(date -u -r "${BASE_EPOCH}" +"%Y-%m-%dT%H:%M:%SZ")"

post_event() {
  local event_id="$1"
  local timestamp="$2"
  local category="$3"
  local action="$4"
  local outcome="$5"

  curl -fsS     -X POST     -H "Content-Type: application/json"     --data "{
      \"event_id\": \"${event_id}\",
      \"timestamp\": \"${timestamp}\",
      \"source\": {
        \"type\": \"endpoint\",
        \"vendor\": \"sentinel-phase3-pivot\",
        \"collector\": \"pivot-smoke\"
      },
      \"asset\": {
        \"id\": \"${ASSET_ID}\",
        \"hostname\": \"${ASSET_ID}\",
        \"zone\": \"corporate\"
      },
      \"actor\": {
        \"id\": \"${IDENTITY_ID}\",
        \"type\": \"service_account\",
        \"name\": \"Pivot Service\"
      },
      \"event\": {
        \"category\": \"${category}\",
        \"action\": \"${action}\",
        \"outcome\": \"${outcome}\"
      },
      \"network\": {
        \"source_ip\": \"10.104.0.20\",
        \"destination_ip\": \"10.104.0.30\",
        \"destination_port\": 22,
        \"protocol\": \"tcp\"
      },
      \"labels\": {
        \"environment\": \"lab\",
        \"scenario\": \"phase3-pivot\"
      }
    }"     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null
}

echo "Submitting related telemetry..."
post_event "${CONTEXT_EVENT_ID}" "${CONTEXT_TIME}" "process" "process_start" "success"
post_event "${LOGIN_EVENT_ID}" "${LOGIN_TIME}" "authentication" "login" "success"

FINDING_ID=""
for _ in {1..30}; do
  FINDING_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-AUTH-002' AND evidence->>'terminal_event_id'='${LOGIN_EVENT_ID}' LIMIT 1;"
  )"
  [[ -n "${FINDING_ID}" ]] && break
  sleep 1
done
[[ -n "${FINDING_ID}" ]] || {
  echo "Expected DET-AUTH-002 finding was not created." >&2
  exit 1
}

echo "Creating and running identity hunt..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-pivot-hunter"   --data "{
    \"name\": \"Identity pivot hunt ${RUN_ID}\",
    \"hypothesis\": \"The service identity has related endpoint activity.\",
    \"query\": {
      \"identity_id\": \"${IDENTITY_ID}\",
      \"limit\": 50
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts" >/tmp/sentinel-phase3-pivot-hunt.json

HUNT_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-pivot-hunt.json"))["id"])')"

curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-pivot-hunter"   --data '{"override":{}}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts/${HUNT_ID}/run" >/tmp/sentinel-phase3-pivot-run.json

RUN_RECORD_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-pivot-run.json"))["run_id"])')"
RESULT_COUNT="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-pivot-run.json"))["result_count"])')"
if [[ "${RESULT_COUNT}" -lt "2" ]]; then
  cat /tmp/sentinel-phase3-pivot-run.json >&2
  echo "Expected hunt to return both related events." >&2
  exit 1
fi

echo "Verifying hunt-run history..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts/${HUNT_ID}/runs?limit=10"   >/tmp/sentinel-phase3-pivot-runs.json
if ! grep -q "\"id\":${RUN_RECORD_ID}" /tmp/sentinel-phase3-pivot-runs.json; then
  cat /tmp/sentinel-phase3-pivot-runs.json >&2
  echo "Hunt-run history did not include the executed run." >&2
  exit 1
fi

echo "Verifying asset pivot..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/assets/${ASSET_ID}/pivot?limit=25"   >/tmp/sentinel-phase3-asset-pivot.json
python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase3-asset-pivot.json"))
assert len(d["recent_events"]) >= 2, d
assert any(f["detection_id"] == "DET-AUTH-002" for f in d["findings"]), d
PY

echo "Verifying identity pivot..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/identities/${IDENTITY_ID}/pivot?limit=25"   >/tmp/sentinel-phase3-identity-pivot.json
python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase3-identity-pivot.json"))
assert len(d["recent_events"]) >= 2, d
assert any(f["detection_id"] == "DET-AUTH-002" for f in d["findings"]), d
PY

echo "Creating investigation with finding only..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-pivot-analyst"   --data "{
    \"title\": \"Pivot investigation ${RUN_ID}\",
    \"description\": \"Validate attaching a durable hunt run to an existing investigation.\",
    \"priority\": \"medium\",
    \"finding_ids\": [\"${FINDING_ID}\"]
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations" >/tmp/sentinel-phase3-pivot-investigation.json

INVESTIGATION_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-pivot-investigation.json"))["id"])')"

echo "Attaching exact hunt-run results..."
curl -fsS   -X POST   -H "X-Sentinel-Actor: phase3-pivot-analyst"   -H "X-Request-ID: req-phase3-${RUN_ID}-attach"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations/${INVESTIGATION_ID}/hunt-runs/${RUN_RECORD_ID}"   >/tmp/sentinel-phase3-pivot-attached.json

python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase3-pivot-attached.json"))
ids={event["id"] for event in d["events"]}
assert len(ids) >= 2, d
PY

echo "Updating investigation ownership and priority..."
curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-pivot-analyst"   -H "X-Request-ID: req-phase3-${RUN_ID}-metadata"   --data '{"owner_id":"senior-analyst","priority":"critical"}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations/${INVESTIGATION_ID}"   >/tmp/sentinel-phase3-pivot-metadata.json

python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase3-pivot-metadata.json"))
assert d["owner_id"] == "senior-analyst", d
assert d["priority"] == "critical", d
actions=[item["action"] for item in d["audit"]]
assert "investigation.hunt_run_attached" in actions, actions
assert "investigation.metadata_changed" in actions, actions
PY

RUN_EVENT_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM hunt_run_events WHERE hunt_run_id=${RUN_RECORD_ID};"
)"
if [[ "${RUN_EVENT_COUNT}" -lt "2" ]]; then
  echo "Expected exact hunt-run event membership to be persisted." >&2
  exit 1
fi

echo "Phase 3 pivot/enrichment smoke test passed."
echo "  hunt-run history: verified"
echo "  exact hunt-run membership: verified"
echo "  asset pivot: verified"
echo "  identity pivot: verified"
echo "  hunt-run attachment: verified"
echo "  investigation ownership: verified"
echo "  investigation priority: verified"
echo "  metadata audit trail: verified"
