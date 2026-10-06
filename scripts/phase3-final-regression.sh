#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "=== Phase 3 final regression ==="

echo
echo "[1/5] Static checks, unit tests, vet, and race detector"
make check

echo
echo "[2/5] Phase 2 regression baseline"
./scripts/phase2-final-regression.sh

echo
echo "[3/5] Saved hunts and investigation vertical slice"
./scripts/phase3-smoke.sh

echo
echo "[4/5] Analyst pivots, hunt history, and investigation enrichment"
./scripts/phase3-pivots-smoke.sh

echo
echo "[5/5] Hunt versioning, richer operators, and workflow metrics"
./scripts/phase3-maturity-smoke.sh

echo
echo "Phase 3 final regression passed."
echo "  unit/race checks: verified"
echo "  Phase 2 regression baseline: verified"
echo "  saved hunts: verified"
echo "  hunt execution/history: verified"
echo "  asset/identity pivots: verified"
echo "  investigations/timeline/audit: verified"
echo "  hunt-run attachment: verified"
echo "  owner/priority updates: verified"
echo "  hunt versioning: verified"
echo "  richer hunt operators: verified"
echo "  hunt/investigation metrics: verified"
