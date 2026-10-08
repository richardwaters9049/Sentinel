#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${SENTINEL_API_URL:-http://127.0.0.1:8080}"
ACTOR="${SENTINEL_ACTOR:-phase7-evaluator}"

cd "${ROOT_DIR}"

settings_file="$(mktemp)"
evaluation_file="$(mktemp)"
payload_file="$(mktemp)"
cleanup() {
  rm -f "${settings_file}" "${evaluation_file}" "${payload_file}"
}
trap cleanup EXIT

curl -fsS "${BASE_URL}/api/v1/behaviour/settings" >"${settings_file}"

threshold="$(python3 - "${settings_file}" <<'PY'
import json, sys
print(json.load(open(sys.argv[1]))["anomaly_threshold"])
PY
)"

SENTINEL_BEHAVIOUR_THRESHOLD="${threshold}"   docker compose run --rm   -e SENTINEL_BEHAVIOUR_THRESHOLD="${threshold}"   ml python -m app.evaluation >"${evaluation_file}"

python3 - "${evaluation_file}" "${payload_file}" <<'PY'
import json, sys

source = json.load(open(sys.argv[1]))
payload = {
    "model_version": source["model_version"],
    "dataset_name": source["dataset_name"],
    "threshold": source["threshold"],
    "normal_count": source["normal_count"],
    "anomaly_count": source["anomaly_count"],
    "true_positive": source["true_positive"],
    "false_positive": source["false_positive"],
    "true_negative": source["true_negative"],
    "false_negative": source["false_negative"],
    "precision": source["precision"],
    "recall": source["recall"],
    "false_positive_rate": source["false_positive_rate"],
}
json.dump(payload, open(sys.argv[2], "w"))
PY

curl -fsS   -X POST   -H "Content-Type: application/json"   -H "X-Sentinel-Actor: ${ACTOR}"   --data-binary "@${payload_file}"   "${BASE_URL}/api/v1/behaviour/evaluations"

printf "\n"
