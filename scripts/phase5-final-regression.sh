#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "=== Phase 5 final regression ==="

echo
echo "[1/2] Phase 4 regression baseline"
./scripts/phase4-final-regression.sh

echo
echo "[2/2] OT simulation, normal-behaviour, and correlation workflow"
./scripts/phase5-ot-smoke.sh

echo
echo "Phase 5 final regression passed."
echo "  Phase 4 analyst-console baseline: verified"
echo "  OT simulator catalogue: verified"
echo "  normal OT fixtures: verified"
echo "  OT findings: verified"
echo "  temporal OT correlation: verified"
echo "  ATT&CK for ICS context: verified"
echo "  safety annotations: verified"
echo "  frontend OT explainability: lint/build verified"
