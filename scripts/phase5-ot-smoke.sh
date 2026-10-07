#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GATEWAY_PORT="${SENTINEL_PHASE5_SMOKE_PORT:-18095}"
NATS_PORT="${SENTINEL_PHASE5_NATS_PORT:-14225}"
NATS_MONITOR_PORT="${SENTINEL_PHASE5_NATS_MONITOR_PORT:-18225}"
NATS_CONTAINER="sentinel-phase5-nats-smoke"
GATEWAY_PID=""
GATEWAY_BINARY="/tmp/sentinel-phase5-smoke-gateway"
SIM_BINARY="/tmp/sentinel-phase5-smoke-sim"

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

echo "Starting Sentinel dependencies..."
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
  echo "NATS did not become ready." >&2
  exit 1
}

echo "Building isolated Phase 5 gateway and simulator..."
(
  cd services/gateway
  go build -o "${GATEWAY_BINARY}" ./cmd/gateway
)
(
  cd simulator
  go build -o "${SIM_BINARY}" ./cmd/sentinel-sim
)

SENTINEL_HTTP_ADDR=":${GATEWAY_PORT}" NATS_URL="nats://127.0.0.1:${NATS_PORT}" "${GATEWAY_BINARY}" >/tmp/sentinel-phase5-gateway.log 2>&1 &
GATEWAY_PID=$!

for _ in {1..30}; do
  curl -fsS "http://127.0.0.1:${GATEWAY_PORT}/ready" >/dev/null 2>&1 && break
  if ! kill -0 "${GATEWAY_PID}" 2>/dev/null; then
    cat /tmp/sentinel-phase5-gateway.log >&2
    exit 1
  fi
  sleep 1
done

BASE="http://127.0.0.1:${GATEWAY_PORT}/api/v1"

echo "Verifying OT detection catalogue..."
curl -fsS "${BASE}/detections" >/tmp/sentinel-phase5-detections.json
python3 - <<'PY'
import json
data=json.load(open("/tmp/sentinel-phase5-detections.json"))
by_id={item["id"]: item for item in data["detections"]}
assert "DET-OT-001" in by_id, by_id.keys()
assert "DET-OT-002" in by_id, by_id.keys()
assert "DET-OT-003" in by_id, by_id.keys()
assert by_id["DET-OT-001"]["enabled"] is True
assert by_id["DET-OT-002"]["enabled"] is True
assert by_id["DET-OT-003"]["enabled"] is True
assert any(m.get("technique_id") == "T0836" for m in by_id["DET-OT-001"]["mitre"])
assert any(m.get("technique_id") == "T1692.001" for m in by_id["DET-OT-002"]["mitre"])
assert any(m.get("technique_id") == "T0836" for m in by_id["DET-OT-003"]["mitre"])
PY

count_findings() {
  local detection_id="$1"
  docker compose exec -T postgres psql -U sentinel -d sentinel -Atc     "SELECT COUNT(*) FROM findings WHERE detection_id='${detection_id}';"
}

PARAM_BEFORE="$(count_findings DET-OT-001)"
COMMAND_BEFORE="$(count_findings DET-OT-002)"
SEQUENCE_BEFORE="$(count_findings DET-OT-003)"

echo "Emitting normal OT HMI telemetry..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario ot-hmi-read-baseline   -delay 0s >/tmp/sentinel-phase5-baseline.log

sleep 1
[[ "$(count_findings DET-OT-001)" == "${PARAM_BEFORE}" ]] || {
  echo "Baseline OT read unexpectedly created DET-OT-001." >&2
  exit 1
}
[[ "$(count_findings DET-OT-002)" == "${COMMAND_BEFORE}" ]] || {
  echo "Baseline OT read unexpectedly created DET-OT-002." >&2
  exit 1
}
[[ "$(count_findings DET-OT-003)" == "${SEQUENCE_BEFORE}" ]] || {
  echo "Baseline OT read unexpectedly created DET-OT-003." >&2
  exit 1
}

echo "Emitting historian and sensor normal-behaviour fixtures..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario ot-historian-read-baseline   -delay 0s >/tmp/sentinel-phase5-historian.log
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario ot-sensor-telemetry-baseline   -delay 0s >/tmp/sentinel-phase5-sensor.log
sleep 1
[[ "$(count_findings DET-OT-001)" == "${PARAM_BEFORE}" ]] || {
  echo "Normal historian/sensor activity unexpectedly created DET-OT-001." >&2
  exit 1
}
[[ "$(count_findings DET-OT-002)" == "${COMMAND_BEFORE}" ]] || {
  echo "Normal historian/sensor activity unexpectedly created DET-OT-002." >&2
  exit 1
}
[[ "$(count_findings DET-OT-003)" == "${SEQUENCE_BEFORE}" ]] || {
  echo "Normal historian/sensor activity unexpectedly created DET-OT-003." >&2
  exit 1
}

echo "Emitting standalone controller mode-change telemetry..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario ot-controller-mode-change   -delay 0s >/tmp/sentinel-phase5-mode.log
sleep 1
[[ "$(count_findings DET-OT-003)" == "${SEQUENCE_BEFORE}" ]] || {
  echo "Standalone mode change unexpectedly created DET-OT-003." >&2
  exit 1
}

echo "Emitting synthetic controller parameter-change telemetry..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario ot-plc-parameter-change   -delay 0s >/tmp/sentinel-phase5-parameter.log

PARAM_ID=""
for _ in {1..30}; do
  PARAM_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-OT-001' ORDER BY created_at DESC LIMIT 1;"
  )"
  PARAM_AFTER="$(count_findings DET-OT-001)"
  [[ "${PARAM_AFTER}" -gt "${PARAM_BEFORE}" ]] && [[ -n "${PARAM_ID}" ]] && break
  sleep 1
done
[[ "$(count_findings DET-OT-001)" -gt "${PARAM_BEFORE}" ]] || {
  echo "DET-OT-001 finding was not created." >&2
  exit 1
}

echo "Emitting synthetic unauthorized-command observation..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario ot-unauthorized-command   -delay 0s >/tmp/sentinel-phase5-command.log

COMMAND_ID=""
for _ in {1..30}; do
  COMMAND_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-OT-002' ORDER BY created_at DESC LIMIT 1;"
  )"
  COMMAND_AFTER="$(count_findings DET-OT-002)"
  [[ "${COMMAND_AFTER}" -gt "${COMMAND_BEFORE}" ]] && [[ -n "${COMMAND_ID}" ]] && break
  sleep 1
done
[[ "$(count_findings DET-OT-002)" -gt "${COMMAND_BEFORE}" ]] || {
  echo "DET-OT-002 finding was not created." >&2
  exit 1
}

echo "Emitting correlated controller-change sequence..."
"${SIM_BINARY}"   -url "http://127.0.0.1:${GATEWAY_PORT}"   -scenario ot-change-sequence   -delay 50ms >/tmp/sentinel-phase5-sequence.log

SEQUENCE_ID=""
for _ in {1..30}; do
  SEQUENCE_ID="$(
    docker compose exec -T postgres psql -U sentinel -d sentinel -Atc       "SELECT id FROM findings WHERE detection_id='DET-OT-003' ORDER BY created_at DESC LIMIT 1;"
  )"
  SEQUENCE_AFTER="$(count_findings DET-OT-003)"
  [[ "${SEQUENCE_AFTER}" -gt "${SEQUENCE_BEFORE}" ]] && [[ -n "${SEQUENCE_ID}" ]] && break
  sleep 1
done
[[ "$(count_findings DET-OT-003)" -gt "${SEQUENCE_BEFORE}" ]] || {
  echo "DET-OT-003 correlated finding was not created." >&2
  exit 1
}

echo "Checking explainable OT evidence..."
curl -fsS "${BASE}/findings/${PARAM_ID}" >/tmp/sentinel-phase5-parameter-finding.json
curl -fsS "${BASE}/findings/${COMMAND_ID}" >/tmp/sentinel-phase5-command-finding.json
curl -fsS "${BASE}/findings/${SEQUENCE_ID}" >/tmp/sentinel-phase5-sequence-finding.json

python3 - <<'PY'
import json
parameter=json.load(open("/tmp/sentinel-phase5-parameter-finding.json"))
command=json.load(open("/tmp/sentinel-phase5-command-finding.json"))
sequence=json.load(open("/tmp/sentinel-phase5-sequence-finding.json"))

assert parameter["detection_id"] == "DET-OT-001"
assert parameter["severity"] == "high"
assert parameter["evidence"]["ot_device_type"] == "plc"
assert parameter["evidence"]["ot_operation"] == "setpoint_change"
assert parameter["evidence"]["safety_impact"] == "potential_process_impact"

assert command["detection_id"] == "DET-OT-002"
assert command["severity"] == "critical"
assert command["evidence"]["ot_authorization"] == "false"
assert command["evidence"]["ot_protocol"] == "modbus-tcp"
assert command["evidence"]["safety_impact"] == "potential_process_impact"

assert sequence["detection_id"] == "DET-OT-003"
assert sequence["severity"] == "critical"
assert sequence["evidence"]["window_seconds"] == 600
assert len(sequence["evidence"]["event_ids"]) >= 2
assert sequence["evidence"]["ot_device_type"] == "plc"
PY

echo "Verifying OT assets and telemetry persistence..."
curl -fsS "${BASE}/events?category=ot&limit=20" >/tmp/sentinel-phase5-events.json
python3 - <<'PY'
import json
data=json.load(open("/tmp/sentinel-phase5-events.json"))
events=data["events"]
assert len(events) >= 8
scenarios={event.get("labels", {}).get("scenario") for event in events}
assert "ot-hmi-read-baseline" in scenarios
assert "ot-historian-read-baseline" in scenarios
assert "ot-sensor-telemetry-baseline" in scenarios
assert "ot-controller-mode-change" in scenarios
assert "ot-plc-parameter-change" in scenarios
assert "ot-change-sequence" in scenarios
assert "ot-unauthorized-command" in scenarios
assert all(event.get("labels", {}).get("ot.simulated") == "true" for event in events)
assets={event.get("asset", {}).get("hostname") for event in events}
assert "plc-sim-02" in assets
assert "sensor-sim-01" in assets
assert "historian-01" in assets
PY

echo "Phase 5 OT simulation smoke test passed."
echo "  normal OT baseline: verified"
echo "  synthetic OT assets: verified"
echo "  OT telemetry persistence: verified"
echo "  DET-OT-001 parameter-change detection: verified"
echo "  DET-OT-002 unauthorized-message detection: verified"
echo "  DET-OT-003 temporal correlation: verified"
echo "  historian/sensor normal fixtures: verified"
echo "  ATT&CK for ICS mappings: verified"
echo "  safety annotations: verified"
