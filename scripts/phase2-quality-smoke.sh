#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE2_QUALITY_PORT:-18086}"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase2-quality-gateway"
RUN_ID="$(date +%s)_$RANDOM"
ASSET_ID="asset-quality-${RUN_ID}"
ACTOR_ID="svc-quality-${RUN_ID}"
CONTEXT_EVENT_ID="evt_quality_context_${RUN_ID}"
DETECTION_EVENT_ID="evt_quality_service_${RUN_ID}"

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

echo "Building quality-test gateway..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)

SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase2-quality.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1 && break
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase2-quality.log >&2
    exit 1
  fi
  sleep 1
done

BASE_EPOCH="$(date -u +%s)"
CONTEXT_TIME="$(date -u -r "$((BASE_EPOCH - 30))" +"%Y-%m-%dT%H:%M:%SZ")"
DETECTION_TIME="$(date -u -r "${BASE_EPOCH}" +"%Y-%m-%dT%H:%M:%SZ")"

echo "Submitting contextual event on the same asset..."
curl -fsS   -X POST   -H "Content-Type: application/json"   --data "{
    \"event_id\": \"${CONTEXT_EVENT_ID}\",
    \"timestamp\": \"${CONTEXT_TIME}\",
    \"source\": {
      \"type\": \"endpoint\",
      \"vendor\": \"sentinel-quality\",
      \"collector\": \"quality-smoke\"
    },
    \"asset\": {
      \"id\": \"${ASSET_ID}\",
      \"hostname\": \"${ASSET_ID}\",
      \"zone\": \"corporate\"
    },
    \"event\": {
      \"category\": \"process\",
      \"action\": \"process_start\",
      \"outcome\": \"success\"
    },
    \"labels\": {
      \"environment\": \"lab\",
      \"scenario\": \"phase2-quality-context\"
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null

echo "Submitting service-account login detection event..."
curl -fsS   -X POST   -H "Content-Type: application/json"   --data "{
    \"event_id\": \"${DETECTION_EVENT_ID}\",
    \"timestamp\": \"${DETECTION_TIME}\",
    \"source\": {
      \"type\": \"identity\",
      \"vendor\": \"sentinel-quality\",
      \"collector\": \"quality-smoke\"
    },
    \"asset\": {
      \"id\": \"${ASSET_ID}\",
      \"hostname\": \"${ASSET_ID}\",
      \"zone\": \"corporate\"
    },
    \"actor\": {
      \"id\": \"${ACTOR_ID}\",
      \"type\": \"service_account\",
      \"name\": \"Quality Test Service\"
    },
    \"event\": {
      \"category\": \"authentication\",
      \"action\": \"login\",
      \"outcome\": \"success\"
    },
    \"network\": {
      \"source_ip\": \"10.99.0.20\",
      \"destination_ip\": \"10.99.0.30\",
      \"destination_port\": 22,
      \"protocol\": \"tcp\"
    },
    \"labels\": {
      \"environment\": \"lab\",
      \"scenario\": \"phase2-quality\"
    }
  }"   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/telemetry" >/dev/null

FINDING_ID=""
for _ in {1..30}; do
  FINDING_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-AUTH-002' AND evidence->>'terminal_event_id'='${DETECTION_EVENT_ID}' LIMIT 1;"
  )"
  [[ -n "${FINDING_ID}" ]] && break
  sleep 1
done

if [[ -z "${FINDING_ID}" ]]; then
  echo "Quality-test finding was not created." >&2
  exit 1
fi

echo "Checking richer evidence retrieval..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings/${FINDING_ID}/evidence?context_minutes=5"   >/tmp/sentinel-phase2-quality-evidence.json

python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase2-quality-evidence.json"))
assert len(d["linked_events"]) == 1, d
assert len(d["context_events"]) >= 1, d
PY

if ! grep -q "${CONTEXT_EVENT_ID}" /tmp/sentinel-phase2-quality-evidence.json; then
  cat /tmp/sentinel-phase2-quality-evidence.json >&2
  echo "Expected contextual event was not returned." >&2
  exit 1
fi

echo "Classifying the finding as false positive..."
curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: quality-test-analyst"   --data '{"status":"triaged"}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings/${FINDING_ID}/status" >/dev/null

curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: quality-test-analyst"   --data '{"status":"false_positive"}'   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/findings/${FINDING_ID}/status" >/dev/null

echo "Checking detection quality metrics..."
curl -fsS   "http://127.0.0.1:${GATEWAY_PORT}/api/v1/detections/metrics"   >/tmp/sentinel-phase2-quality-metrics.json

python3 - <<'PY'
import json
d=json.load(open("/tmp/sentinel-phase2-quality-metrics.json"))
metrics={m["detection_id"]:m for m in d["metrics"]}
m=metrics["DET-AUTH-002"]
assert m["hit_count"] >= 1, m
assert m["false_positive_count"] >= 1, m
assert m["false_positive_rate"] > 0, m
assert m["last_triggered_at"], m
PY

echo "Phase 2 quality/evidence smoke test passed."
echo "  contextual evidence: verified"
echo "  hit counts: verified"
echo "  last-triggered timestamp: verified"
echo "  false-positive count/rate: verified"
