#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE7_SMOKE_PORT:-18097}"
NATS_PORT="${SENTINEL_PHASE7_NATS_PORT:-14227}"
NATS_MONITOR_PORT="${SENTINEL_PHASE7_NATS_MONITOR_PORT:-18227}"
NATS_CONTAINER="sentinel-phase7-nats-smoke"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase7-smoke-gateway"
SIM_BINARY="/tmp/sentinel-phase7-smoke-sim"

cleanup() {
  if [[ -n "${GATEWAY_PID}" ]] && kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    kill "${GATEWAY_PID}" 2>/dev/null || true
    wait "${GATEWAY_PID}" 2>/dev/null || true
  fi
  docker rm -f "${NATS_CONTAINER}" >/dev/null 2>&1 || true
  rm -f "${GATEWAY_BINARY}" "${SIM_BINARY}"
}
trap cleanup EXIT

cd "${ROOT_DIR}"

echo "Starting Phase 7 dependencies..."
docker compose up -d postgres >/dev/null
docker compose up -d --build ml >/dev/null

docker rm -f "${NATS_CONTAINER}" >/dev/null 2>&1 || true
docker run -d --rm   --name "${NATS_CONTAINER}"   -p "${NATS_PORT}:4222"   -p "${NATS_MONITOR_PORT}:8222"   nats:2.10-alpine -js -m 8222 >/dev/null

for _ in {1..30}; do
  docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1 && break
  sleep 1
done

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:8090/health" >/tmp/sentinel-phase7-ml-health.json 2>/dev/null && break
  sleep 1
done
curl -fsS "http://127.0.0.1:8090/health" >/tmp/sentinel-phase7-ml-health.json

python3 - <<'PY'
import json
health=json.load(open("/tmp/sentinel-phase7-ml-health.json"))
assert health["status"] == "ok"
assert health["model_version"] == "sentinel-behaviour-iforest-v1"
assert health["model_kind"] == "IsolationForest"
PY

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${NATS_MONITOR_PORT}/healthz" >/dev/null 2>&1 && break
  sleep 1
done

echo "Building isolated Phase 7 gateway and simulator..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)
(
  cd simulator
  go build -o "${SIM_BINARY}" ./cmd/sentinel-sim
)

SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" SENTINEL_BEHAVIOUR_ENABLED=true SENTINEL_ML_URL="http://127.0.0.1:8090" NATS_URL="nats://127.0.0.1:${NATS_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase7-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/tmp/sentinel-phase7-ready.json 2>/dev/null && break
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase7-gateway.log >&2
    exit 1
  fi
  sleep 1
done

BASE="http://127.0.0.1:${GATEWAY_PORT}/api/v1"

echo "Submitting baseline-like behaviour..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario behaviour-normal   -delay 0s >/tmp/sentinel-phase7-normal.log

echo "Submitting anomalous cross-zone behaviour..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario behaviour-anomaly   -delay 0s >/tmp/sentinel-phase7-anomaly.log

wait_for_event() {
  local scenario="$1"
  local event_id=""
  for _ in {1..30}; do
    event_id="$(
      docker compose exec -T postgres psql -U sentinel -d sentinel -Atc         "SELECT id FROM events WHERE labels->>'scenario'='${scenario}' ORDER BY created_at DESC LIMIT 1;"
    )"
    [[ -n "${event_id}" ]] && break
    sleep 1
  done
  [[ -n "${event_id}" ]] || {
    echo "Event for scenario ${scenario} was not persisted." >&2
    return 1
  }
  printf '%s' "${event_id}"
}

NORMAL_EVENT_ID="$(wait_for_event behaviour-normal)"
ANOMALY_EVENT_ID="$(wait_for_event behaviour-anomaly)"

for event_id in "${NORMAL_EVENT_ID}" "${ANOMALY_EVENT_ID}"; do
  for _ in {1..30}; do
    count="$(
      docker compose exec -T postgres psql -U sentinel -d sentinel -Atc         "SELECT COUNT(*) FROM behavioural_scores WHERE event_id='${event_id}';"
    )"
    [[ "${count}" -gt 0 ]] && break
    sleep 1
  done
  [[ "${count:-0}" -gt 0 ]] || {
    echo "Behavioural score for ${event_id} was not persisted." >&2
    exit 1
  }
done

curl -fsS "${BASE}/events/${NORMAL_EVENT_ID}/behaviour"   >/tmp/sentinel-phase7-normal-score.json
curl -fsS "${BASE}/events/${ANOMALY_EVENT_ID}/behaviour"   >/tmp/sentinel-phase7-anomaly-score.json
curl -fsS "${BASE}/behaviour/scores?limit=20"   >/tmp/sentinel-phase7-scores.json
curl -fsS "${BASE}/behaviour/metrics"   >/tmp/sentinel-phase7-metrics.json

python3 - <<PY
import json

normal=json.load(open("/tmp/sentinel-phase7-normal-score.json"))
anomaly=json.load(open("/tmp/sentinel-phase7-anomaly-score.json"))
scores=json.load(open("/tmp/sentinel-phase7-scores.json"))
metrics=json.load(open("/tmp/sentinel-phase7-metrics.json"))

assert normal["event_id"] == "${NORMAL_EVENT_ID}"
assert normal["model_kind"] == "IsolationForest"
assert normal["anomalous"] is False, normal
assert normal["anomaly_score"] < normal["threshold"], normal

assert anomaly["event_id"] == "${ANOMALY_EVENT_ID}"
assert anomaly["entity_id"] == "svc-behaviour-anomaly"
assert anomaly["model_version"] == "sentinel-behaviour-iforest-v1"
assert anomaly["anomalous"] is True, anomaly
assert anomaly["anomaly_score"] >= anomaly["threshold"], anomaly
assert anomaly["severity"] in {"medium", "high"}
assert len(anomaly["explanations"]) >= 1
features={item["feature"] for item in anomaly["explanations"]}
assert {"cross_zone", "auth_failure", "service_account", "ot_activity"} & features

assert scores["count"] >= 2
assert metrics["total_scores"] >= 2
assert metrics["anomalous_scores"] >= 1
assert metrics["average_score"] >= 0
PY

echo "Running Python model tests and evaluation..."
docker compose run --rm ml python -m pytest -q
docker compose run --rm ml python -m app.evaluation   >/tmp/sentinel-phase7-evaluation.json

python3 - <<'PY'
import json
evaluation=json.load(open("/tmp/sentinel-phase7-evaluation.json"))
assert evaluation["normal_count"] == 120
assert evaluation["anomaly_count"] == 30
assert evaluation["recall"] >= 0.90
assert evaluation["false_positive_rate"] <= 0.15
PY

echo "Phase 7 behavioural-analytics smoke test passed."
echo "  Python ML service health: verified"
echo "  deterministic Isolation Forest model: verified"
echo "  baseline-like activity: verified"
echo "  anomalous cross-zone behaviour: verified"
echo "  durable behavioural scores: verified"
echo "  analyst-visible explanations: verified"
echo "  score and metrics APIs: verified"
echo "  synthetic model evaluation: verified"
