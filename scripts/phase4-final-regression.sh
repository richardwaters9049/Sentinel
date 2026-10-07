#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "=== Phase 4 final regression ==="

echo
echo "[1/4] Backend static, unit, vet, and race checks"
make check

echo
echo "[2/4] Phase 3 regression baseline"
./scripts/phase3-final-regression.sh

echo
echo "[3/4] Analyst console lint and production build"
make console-lint
make console-build

echo
echo "[4/4] End-to-end analyst console workflow"
./scripts/phase4-workflow-smoke.sh

echo
echo "Phase 4 final regression passed."
echo "  backend regression baseline: verified"
echo "  frontend lint/build: verified"
echo "  live overview: verified"
echo "  environment/findings/hunts/investigations: verified"
echo "  detection management: verified"
echo "  end-to-end analyst workflow: verified"
