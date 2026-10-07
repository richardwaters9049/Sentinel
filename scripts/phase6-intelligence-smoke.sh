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
RUN_ID="$(date +%s)_$RANDOM"
SOURCE_REQUEST_ID="phase6-disable-review-source-${RUN_ID}"
REVIEW_SOURCE_DISABLED=0

cleanup() {
  if [[ "${REVIEW_SOURCE_DISABLED}" == "1" && -n "${GATEWAY_PID}" ]] &&
    kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    curl -fsS       -X PATCH       -H "Content-Type: application/json"       -H "X-Sentinel-Actor: phase6-regression-cleanup"       --data '{"enabled":true}'       "http://127.0.0.1:${GATEWAY_PORT}/api/v1/intelligence/sources/intel-local-review"       >/dev/null 2>&1 || true
  fi
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

latest_event_id() {
  local scenario="$1"
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT id FROM events WHERE labels->>'scenario'='${scenario}' ORDER BY created_at DESC LIMIT 1;"
}

wait_for_event() {
  local scenario="$1"
  local event_id=""
  for _ in {1..30}; do
    event_id="$(latest_event_id "${scenario}")"
    [[ -n "${event_id}" ]] && break
    sleep 1
  done
  [[ -n "${event_id}" ]] || {
    echo "Event for scenario ${scenario} was not persisted." >&2
    return 1
  }
  printf '%s' "${event_id}"
}

wait_for_matches() {
  local event_id="$1"
  local minimum="$2"
  local count="0"
  for _ in {1..30}; do
    count="$(
      docker compose exec -T postgres psql -U sentinel -d sentinel -Atc         "SELECT COUNT(*) FROM event_enrichments WHERE event_id='${event_id}';"
    )"
    [[ "${count}" -ge "${minimum}" ]] && break
    sleep 1
  done
  [[ "${count}" -ge "${minimum}" ]] || {
    echo "Expected at least ${minimum} enrichment matches for ${event_id}; got ${count}." >&2
    return 1
  }
}

echo "Verifying multi-source and validity-aware intelligence fixtures..."
curl -fsS "${BASE}/intelligence/indicators?type=ip&value=198.51.100.66"   >/tmp/sentinel-phase6-indicators.json
curl -fsS "${BASE}/intelligence/indicators?type=ip&value=192.0.2.44"   >/tmp/sentinel-phase6-expired.json
curl -fsS "${BASE}/intelligence/indicators?type=domain&value=telemetry-sync.example"   >/tmp/sentinel-phase6-domain-indicators.json
curl -fsS "${BASE}/intelligence/indicators?type=sha256&value=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"   >/tmp/sentinel-phase6-hash-indicators.json

python3 - <<'PY'
import json

ip_data=json.load(open("/tmp/sentinel-phase6-indicators.json"))
assert ip_data["count"] == 2, ip_data
by_source={item["source_id"]: item for item in ip_data["indicators"]}
assert set(by_source) == {"intel-local-northstar", "intel-local-review"}
assert by_source["intel-local-northstar"]["source_confidence"] == 85
assert by_source["intel-local-northstar"]["confidence"] == 95
assert by_source["intel-local-review"]["source_confidence"] == 70
assert by_source["intel-local-review"]["confidence"] == 60
assert all(item["provenance"]["external_dependency"] is False for item in ip_data["indicators"])

expired=json.load(open("/tmp/sentinel-phase6-expired.json"))
assert expired["count"] == 0, expired

domains=json.load(open("/tmp/sentinel-phase6-domain-indicators.json"))
assert domains["count"] == 2, domains
assert {item["source_id"] for item in domains["indicators"]} == {
    "intel-local-northstar",
    "intel-local-review",
}

hashes=json.load(open("/tmp/sentinel-phase6-hash-indicators.json"))
assert hashes["count"] == 1, hashes
assert hashes["indicators"][0]["id"] == "ioc-local-sha256-001"
PY

echo "Submitting synthetic multi-source IP IOC telemetry..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario intel-ioc-match   -delay 0s >/tmp/sentinel-phase6-ip.log

IP_EVENT_ID="$(wait_for_event intel-ioc-match)"
wait_for_matches "${IP_EVENT_ID}" 3

curl -fsS "${BASE}/events/${IP_EVENT_ID}/enrichments"   >/tmp/sentinel-phase6-ip-enrichments.json

python3 - <<PY
import json
data=json.load(open("/tmp/sentinel-phase6-ip-enrichments.json"))
assert data["event_id"] == "${IP_EVENT_ID}"
assert data["count"] == 3, data

destination=[
    item for item in data["enrichments"]
    if item["event_field"] == "network.destination_ip"
]
assert len(destination) == 2, destination
assert {item["source_id"] for item in destination} == {
    "intel-local-northstar",
    "intel-local-review",
}
assert {
    item["context"]["classification"] for item in destination
} == {"malicious", "suspicious"}

primary=next(
    item for item in destination
    if item["source_id"] == "intel-local-northstar"
)
assert primary["effective_confidence"] == 81

internal=next(
    item for item in data["enrichments"]
    if item["observed_value"] == "10.10.0.25"
)
assert internal["event_field"] == "network.source_ip"
assert internal["context"]["classification"] == "suspicious"
PY


echo "Verifying audited source disable and cache invalidation..."
curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase6-regression"   -H "X-Request-ID: ${SOURCE_REQUEST_ID}"   --data '{"enabled":false}'   "${BASE}/intelligence/sources/intel-local-review"   >/tmp/sentinel-phase6-source-disabled.json
REVIEW_SOURCE_DISABLED=1

python3 - <<'PY'
import json
source=json.load(open("/tmp/sentinel-phase6-source-disabled.json"))
assert source["id"] == "intel-local-review"
assert source["active"] is False
PY

"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario intel-ioc-match   -delay 0s >/tmp/sentinel-phase6-cache-invalidation.log

CACHE_EVENT_ID="$(wait_for_event intel-ioc-match)"
if [[ "${CACHE_EVENT_ID}" == "${IP_EVENT_ID}" ]]; then
  for _ in {1..30}; do
    CACHE_EVENT_ID="$(latest_event_id intel-ioc-match)"
    [[ -n "${CACHE_EVENT_ID}" && "${CACHE_EVENT_ID}" != "${IP_EVENT_ID}" ]] && break
    sleep 1
  done
fi
[[ "${CACHE_EVENT_ID}" != "${IP_EVENT_ID}" ]] || {
  echo "Second IOC-match telemetry event was not persisted." >&2
  exit 1
}

wait_for_matches "${CACHE_EVENT_ID}" 2
curl -fsS "${BASE}/events/${CACHE_EVENT_ID}/enrichments"   >/tmp/sentinel-phase6-cache-enrichments.json

python3 - <<PY
import json
data=json.load(open("/tmp/sentinel-phase6-cache-enrichments.json"))
destination=[
    item for item in data["enrichments"]
    if item["event_field"] == "network.destination_ip"
]
assert len(destination) == 1, destination
assert destination[0]["source_id"] == "intel-local-northstar"
PY

AUDIT_COUNT="$(
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*)
     FROM audit_events
     WHERE action='intelligence_source.enabled_changed'
       AND resource_type='intelligence_source'
       AND resource_id='intel-local-review'
       AND request_id='${SOURCE_REQUEST_ID}';"
)"
[[ "${AUDIT_COUNT}" -eq 1 ]] || {
  echo "Expected one audited intelligence source state change; got ${AUDIT_COUNT}." >&2
  exit 1
}

curl -fsS   -X PATCH   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase6-regression"   --data '{"enabled":true}'   "${BASE}/intelligence/sources/intel-local-review" >/dev/null
REVIEW_SOURCE_DISABLED=0

echo "Submitting synthetic domain IOC telemetry..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario intel-domain-match   -delay 0s >/tmp/sentinel-phase6-domain.log

DOMAIN_EVENT_ID="$(wait_for_event intel-domain-match)"
wait_for_matches "${DOMAIN_EVENT_ID}" 2
curl -fsS "${BASE}/events/${DOMAIN_EVENT_ID}/enrichments"   >/tmp/sentinel-phase6-domain-enrichments.json

python3 - <<PY
import json
data=json.load(open("/tmp/sentinel-phase6-domain-enrichments.json"))
assert data["count"] == 2, data
assert all(item["indicator_type"] == "domain" for item in data["enrichments"])
assert all(item["event_field"] == "labels.dns.query" for item in data["enrichments"])
assert all(item["observed_value"] == "telemetry-sync.example" for item in data["enrichments"])
PY

echo "Submitting synthetic SHA-256 IOC telemetry..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario intel-sha256-match   -delay 0s >/tmp/sentinel-phase6-hash.log

HASH_EVENT_ID="$(wait_for_event intel-sha256-match)"
wait_for_matches "${HASH_EVENT_ID}" 1
curl -fsS "${BASE}/events/${HASH_EVENT_ID}/enrichments"   >/tmp/sentinel-phase6-hash-enrichments.json

python3 - <<PY
import json
data=json.load(open("/tmp/sentinel-phase6-hash-enrichments.json"))
assert data["count"] == 1, data
match=data["enrichments"][0]
assert match["indicator_type"] == "sha256"
assert match["event_field"] == "labels.file.sha256"
assert match["observed_value"] == "a"*64
assert match["effective_confidence"] == 77
PY

echo "Verifying enrichment propagation into findings and investigations..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario intel-finding-match   -delay 0s >/tmp/sentinel-phase6-finding.log

FINDING_EVENT_ID="$(wait_for_event intel-finding-match)"
wait_for_matches "${FINDING_EVENT_ID}" 1

FINDING_ID=""
for _ in {1..30}; do
  FINDING_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT f.id
       FROM findings f
       JOIN finding_events fe ON fe.finding_id=f.id
       WHERE f.detection_id='DET-NET-001'
         AND fe.event_id='${FINDING_EVENT_ID}'
       ORDER BY f.created_at DESC
       LIMIT 1;"
  )"
  [[ -n "${FINDING_ID}" ]] && break
  sleep 1
done
[[ -n "${FINDING_ID}" ]] || {
  echo "Expected enriched DET-NET-001 finding was not created." >&2
  exit 1
}

curl -fsS "${BASE}/findings/${FINDING_ID}"   >/tmp/sentinel-phase6-finding-detail.json
curl -fsS "${BASE}/findings/${FINDING_ID}/evidence?context_minutes=5"   >/tmp/sentinel-phase6-finding-evidence.json

python3 - <<PY
import json
detail=json.load(open("/tmp/sentinel-phase6-finding-detail.json"))
evidence=json.load(open("/tmp/sentinel-phase6-finding-evidence.json"))
assert detail["detection_id"] == "DET-NET-001"
assert any(item["event_id"] == "${FINDING_EVENT_ID}" for item in detail["enrichments"])
assert any(item["observed_value"] == "10.10.0.25" for item in detail["enrichments"])
assert any(item["event_id"] == "${FINDING_EVENT_ID}" for item in evidence["enrichments"])
PY

curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: phase6-regression"   --data "{
    \"title\": \"Phase 6 intelligence case\",
    \"description\": \"Verifies IOC context follows evidence into investigations.\",
    \"priority\": \"high\",
    \"owner_id\": \"phase6-regression\",
    \"finding_ids\": [\"${FINDING_ID}\"],
    \"event_ids\": [\"${FINDING_EVENT_ID}\"]
  }"   "${BASE}/investigations" >/tmp/sentinel-phase6-investigation.json

python3 - <<PY
import json
detail=json.load(open("/tmp/sentinel-phase6-investigation.json"))
assert len(detail["enrichments"]) >= 1
assert any(item["event_id"] == "${FINDING_EVENT_ID}" for item in detail["enrichments"])
assert any(item["observed_value"] == "10.10.0.25" for item in detail["enrichments"])
PY

echo "Checking Phase 6 intelligence metrics..."
curl -fsS "${BASE}/intelligence/metrics" >/tmp/sentinel-phase6-metrics.json
python3 - <<'PY'
import json
metrics=json.load(open("/tmp/sentinel-phase6-metrics.json"))
assert metrics["active_sources"] >= 2
assert metrics["active_indicators"] >= 6
assert metrics["enriched_events"] >= 4
assert metrics["total_matches"] >= 7
assert metrics["high_confidence_hits"] >= 1
PY

echo "Phase 6 intelligence/enrichment smoke test passed."
echo "  local multi-source intelligence: verified"
echo "  expired IOC filtering: verified"
echo "  IP enrichment: verified"
echo "  domain enrichment: verified"
echo "  SHA-256 enrichment: verified"
echo "  confidence/provenance disagreement context: verified"
echo "  source enable/disable audit: verified"
echo "  cache invalidation on source change: verified"
echo "  finding enrichment propagation: verified"
echo "  investigation enrichment propagation: verified"
echo "  intelligence metrics: verified"
