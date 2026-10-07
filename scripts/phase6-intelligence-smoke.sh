#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE6_SMOKE_PORT:-18096}"
NATS_PORT="${SENTINEL_PHASE6_NATS_PORT:-14226}"
NATS_MONITOR_PORT="${SENTINEL_PHASE6_NATS_MONITOR_PORT:-18226}"
NATS_CONTAINER="sentinel-phase6-nats-smoke"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase6-smoke-gateway"
SIM_BINARY="/tmp/sentinel-phase6-smoke-sim"

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

echo "Starting Phase 6 dependencies..."
docker compose up -d postgres >/dev/null

docker rm -f "${NATS_CONTAINER}" >/dev/null 2>&1 || true
docker run -d --rm   --name "${NATS_CONTAINER}"   -p "${NATS_PORT}:4222"   -p "${NATS_MONITOR_PORT}:8222"   nats:2.10-alpine -js -m 8222 >/dev/null

for _ in {1..30}; do
  docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1 && break
  sleep 1
done
docker compose exec -T postgres pg_isready -U sentinel -d sentinel >/dev/null 2>&1 || {
  echo "PostgreSQL did not become ready." >&2
  exit 1
}

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${NATS_MONITOR_PORT}/healthz" >/dev/null 2>&1 && break
  sleep 1
done
curl -fsS "http://127.0.0.1:${NATS_MONITOR_PORT}/healthz" >/dev/null 2>&1 || {
  echo "Phase 6 NATS did not become ready." >&2
  exit 1
}

echo "Building isolated Phase 6 gateway and simulator..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)
(
  cd simulator
  go build -o "${SIM_BINARY}" ./cmd/sentinel-sim
)

SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" NATS_URL="nats://127.0.0.1:${NATS_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase6-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1 && break
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase6-gateway.log >&2
    exit 1
  fi
  sleep 1
done

BASE="http://127.0.0.1:${GATEWAY_PORT}/api/v1"

echo "Verifying local threat-intelligence fixtures..."
curl -fsS "${BASE}/intelligence/indicators?type=ip&value=198.51.100.66"   >/tmp/sentinel-phase6-indicators.json
python3 - <<'PY'
import json
data=json.load(open("/tmp/sentinel-phase6-indicators.json"))
assert data["count"] == 1, data
indicator=data["indicators"][0]
assert indicator["id"] == "ioc-local-ip-001"
assert indicator["source_id"] == "intel-local-northstar"
assert indicator["source_type"] == "local_fixture"
assert indicator["source_confidence"] == 85
assert indicator["confidence"] == 95
assert indicator["provenance"]["external_dependency"] is False
assert "synthetic" in indicator["tags"]
PY

echo "Submitting synthetic IOC-match telemetry..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario intel-ioc-match   -delay 0s >/tmp/sentinel-phase6-sim.log

EVENT_ID=""
for _ in {1..30}; do
  EVENT_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM events WHERE labels->>'scenario'='intel-ioc-match' ORDER BY created_at DESC LIMIT 1;"
  )"
  [[ -n "${EVENT_ID}" ]] && break
  sleep 1
done
[[ -n "${EVENT_ID}" ]] || {
  echo "IOC-match telemetry event was not persisted." >&2
  exit 1
}

echo "Waiting for enrichment match..."
for _ in {1..30}; do
  MATCH_COUNT="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT COUNT(*) FROM event_enrichments WHERE event_id='${EVENT_ID}';"
  )"
  [[ "${MATCH_COUNT}" -gt 0 ]] && break
  sleep 1
done
[[ "${MATCH_COUNT:-0}" -gt 0 ]] || {
  echo "Expected IOC enrichment was not persisted." >&2
  exit 1
}

echo "Checking event enrichment confidence and provenance..."
curl -fsS "${BASE}/events/${EVENT_ID}/enrichments"   >/tmp/sentinel-phase6-enrichments.json
python3 - <<PY
import json
data=json.load(open("/tmp/sentinel-phase6-enrichments.json"))
assert data["event_id"] == "${EVENT_ID}"
assert data["count"] == 1, data
match=data["enrichments"][0]
assert match["indicator_id"] == "ioc-local-ip-001"
assert match["event_field"] == "network.destination_ip"
assert match["observed_value"] == "198.51.100.66"
assert match["source_confidence"] == 85
assert match["indicator_confidence"] == 95
assert match["effective_confidence"] == 81
assert match["source_type"] == "local_fixture"
assert match["context"]["classification"] == "malicious"
assert match["provenance"]["origin"] == "Sentinel repository"
PY

echo "Checking Phase 6 intelligence metrics..."
curl -fsS "${BASE}/intelligence/metrics" >/tmp/sentinel-phase6-metrics.json
python3 - <<'PY'
import json
metrics=json.load(open("/tmp/sentinel-phase6-metrics.json"))
assert metrics["active_sources"] >= 1
assert metrics["active_indicators"] >= 3
assert metrics["enriched_events"] >= 1
assert metrics["total_matches"] >= 1
assert metrics["high_confidence_hits"] >= 1
PY

echo "Phase 6 intelligence/enrichment smoke test passed."
echo "  local threat-intel fixtures: verified"
echo "  IOC lookup API: verified"
echo "  telemetry IOC matching: verified"
echo "  event enrichment persistence: verified"
echo "  source/indicator confidence model: verified"
echo "  provenance preservation: verified"
echo "  intelligence metrics: verified"
