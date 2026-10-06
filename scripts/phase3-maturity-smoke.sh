#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE3_MATURITY_PORT:-18089}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase3-maturity-gateway"
RUN_ID="$(date +%s)_$RANDOM"
ASSET_ID="asset-phase3-maturity-${RUN_ID}"
IDENTITY_ID="svc-phase3-maturity-${RUN_ID}"
PROCESS_EVENT_ID="evt_phase3_maturity_process_${RUN_ID}"
LOGIN_EVENT_ID="evt_phase3_maturity_login_${RUN_ID}"

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

for _ in {1..30}; do
  curl -fsS http://127.0.0.1:8222/healthz >/dev/null 2>&1 && break
  sleep 1
done

echo "Building Phase 3 maturity gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase3-maturity.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  if curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase3-maturity.log >&2
    exit 1
  fi
  sleep 1
done

NOW="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

post_event() {
  local event_id="$1"
  local category="$2"
  local action="$3"
  local port="$4"

  curl -fsS     -X POST     -H "Content-Type: application/json"     --data "{
      \"event_id\": \"${event_id}\",
      \"timestamp\": \"${NOW}\",
      \"source\": {
        \"type\": \"endpoint\",
        \"vendor\": \"sentinel-phase3-maturity\",
        \"collector\": \"maturity-smoke\"
      },
      \"asset\": {
        \"id\": \"${ASSET_ID}\",
        \"hostname\": \"${ASSET_ID}\",
        \"zone\": \"corporate\"
      },
      \"actor\": {
        \"id\": \"${IDENTITY_ID}\",
        \"type\": \"service_account\",
        \"name\": \"Phase Three Maturity Service\"
      },
      \"event\": {
        \"category\": \"${category}\",
        \"action\": \"${action}\",
        \"outcome\": \"success\"
      },
      \"network\": {
        \"source_ip\": \"10.105.0.20\",
        \"destination_ip\": \"10.105.0.30\",
        \"destination_port\": ${port},
        \"protocol\": \"tcp\"
      },
      \"labels\": {
        \"environment\": \"lab\",
        \"scenario\": \"phase3-maturity\"
      }
    }"     "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null
}

echo "Submitting telemetry for richer hunt operators..."
post_event "${PROCESS_EVENT_ID}" "process" "process_start" 8443
post_event "${LOGIN_EVENT_ID}" "authentication" "login" 22

for _ in {1..30}; do
  COUNT="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT COUNT(*) FROM events WHERE id IN ('${PROCESS_EVENT_ID}','${LOGIN_EVENT_ID}');"
  )"
  [[ "${COUNT}" == "2" ]] && break
  sleep 1
done

echo "Creating version-one saved hunt..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-maturity-hunter"   --data "{
    \"name\": \"Phase 3 maturity hunt ${RUN_ID}\",
    \"description\": \"Initial broad identity hunt.\",
    \"hypothesis\": \"The synthetic service identity has recent successful activity.\",
    \"query\": {
      \"identity_id\": \"${IDENTITY_ID}\",
      \"outcomes\": [\"success\"],
      \"last_minutes\": 60,
      \"limit\": 100
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts"   >/tmp/sentinel-phase3-maturity-hunt-v1.json

HUNT_ID="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-maturity-hunt-v1.json"))["id"])')"

echo "Editing hunt to version two with multi-field operators..."
curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-maturity-hunter"   --data "{
    \"name\": \"Phase 3 maturity hunt ${RUN_ID}\",
    \"description\": \"Narrowed to SSH-style successful activity in the lab.\",
    \"hypothesis\": \"The synthetic service identity may be authenticating interactively over SSH.\",
    \"query\": {
      \"categories\": [\"authentication\", \"process\"],
      \"outcomes\": [\"success\"],
      \"identity_id\": \"${IDENTITY_ID}\",
      \"source_zones\": [\"corporate\"],
      \"destination_ports\": [22],
      \"labels\": {
        \"environment\": \"lab\",
        \"scenario\": \"phase3-maturity\"
      },
      \"last_minutes\": 60,
      \"limit\": 100
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts/${HUNT_ID}"   >/tmp/sentinel-phase3-maturity-hunt-v2.json

VERSION="$(python3 -c 'import json; print(json.load(open("/tmp/sentinel-phase3-maturity-hunt-v2.json"))["version"])')"
[[ "${VERSION}" == "2" ]] || {
  cat /tmp/sentinel-phase3-maturity-hunt-v2.json >&2
  echo "Expected current hunt version 2." >&2
  exit 1
}

echo "Verifying append-only hunt version history..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts/${HUNT_ID}/versions"   >/tmp/sentinel-phase3-maturity-versions.json

python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase3-maturity-versions.json"))
versions=[item["version"] for item in d["versions"]]
assert versions[:2] == [2,1], d
PY

echo "Executing version-two hunt..."
curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-maturity-hunter"   --data '{"override":{}}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts/${HUNT_ID}/run"   >/tmp/sentinel-phase3-maturity-run.json

python3 - <<PY
import json
d=json.load(open("/tmp/sentinel-phase3-maturity-run.json"))
ids={event["id"] for event in d["events"]}
assert "${LOGIN_EVENT_ID}" in ids, d
assert "${PROCESS_EVENT_ID}" not in ids, d
assert d["result_count"] == 1, d
PY

echo "Creating investigation for metrics..."
FINDING_ID=""
for _ in {1..30}; do
  FINDING_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-AUTH-002' AND evidence->>'terminal_event_id'='${LOGIN_EVENT_ID}' LIMIT 1;"
  )"
  [[ -n "${FINDING_ID}" ]] && break
  sleep 1
done

curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase3-maturity-analyst"   --data "{
    \"title\": \"Phase 3 maturity investigation ${RUN_ID}\",
    \"priority\": \"high\",
    \"finding_ids\": [\"${FINDING_ID}\"],
    \"event_ids\": [\"${LOGIN_EVENT_ID}\"]
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations"   >/tmp/sentinel-phase3-maturity-investigation.json

echo "Checking hunt metrics..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/hunts/metrics"   >/tmp/sentinel-phase3-maturity-hunt-metrics.json

python3 - <<PY
import json
d=json.load(open("/tmp/sentinel-phase3-maturity-hunt-metrics.json"))
m=next(item for item in d["hunts"] if item["hunt_id"]=="${HUNT_ID}")
assert m["version"] == 2, m
assert m["run_count"] >= 1, m
assert m["total_result_count"] >= 1, m
assert m["last_run_at"], m
PY

echo "Checking investigation metrics..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/investigations/metrics"   >/tmp/sentinel-phase3-maturity-investigation-metrics.json

python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase3-maturity-investigation-metrics.json"))
assert d["total"] >= 1, d
assert d["by_status"].get("open",0) >= 1, d
assert d["by_priority"].get("high",0) >= 1, d
PY

echo "Phase 3 hunt-maturity smoke test passed."
echo "  saved-hunt editing: verified"
echo "  append-only hunt versions: verified"
echo "  multi-value operators: verified"
echo "  destination-port filter: verified"
echo "  label filter: verified"
echo "  relative-time filter: verified"
echo "  hunt metrics: verified"
echo "  investigation metrics: verified"
