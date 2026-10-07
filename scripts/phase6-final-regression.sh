#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "=== Phase 6 final regression ==="

echo
echo "[1/3] Phase 5 regression baseline"
./scripts/phase5-final-regression.sh

echo
echo "[2/3] Analyst console lint and production build"
make console-lint
make console-build

echo
echo "[3/3] Intelligence and enrichment workflow"
./scripts/phase6-intelligence-smoke.sh

echo
echo "Phase 6 final regression passed."
echo "  Phase 5 defensive platform baseline: verified"
echo "  frontend lint/build: verified"
echo "  local threat-intel catalogue: verified"
echo "  IP/domain/SHA-256 enrichment: verified"
echo "  provenance/confidence model: verified"
echo "  multi-source disagreement: verified"
echo "  source expiry/state controls: verified"
echo "  cache invalidation: verified"
echo "  finding/investigation enrichment: verified"
