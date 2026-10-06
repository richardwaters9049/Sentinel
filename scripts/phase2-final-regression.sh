#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

echo "=== Phase 2 final regression ==="

echo
echo "[1/6] Static checks, unit tests, vet, and race detector"
make check

echo
echo "[2/6] Core dependency resilience"
./scripts/phase1-resilience.sh

echo
echo "[3/6] Temporal detection vertical slice"
./scripts/phase2-smoke.sh

echo
echo "[4/6] Analyst workflow and runtime rule controls"
./scripts/phase2-workflow-smoke.sh

echo
echo "[5/6] Multi-detection catalogue"
./scripts/phase2-catalogue-smoke.sh

echo
echo "[6/6] Detection quality metrics and contextual evidence"
./scripts/phase2-quality-smoke.sh

echo
echo "Phase 2 final regression passed."
echo "  unit/race checks: verified"
echo "  NATS outage/recovery: verified"
echo "  PostgreSQL buffering/redelivery: verified"
echo "  temporal detection: verified"
echo "  analyst workflow/audit: verified"
echo "  rule enable/disable: verified"
echo "  multi-detection catalogue: verified"
echo "  contextual evidence: verified"
echo "  detection quality metrics: verified"
