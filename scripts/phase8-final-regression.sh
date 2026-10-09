#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
docker compose up -d postgres
export SENTINEL_TEST_DATABASE_URL="${SENTINEL_TEST_DATABASE_URL:-postgres://sentinel:sentinel_dev_only@127.0.0.1:55432/sentinel?sslmode=disable}"
(cd services/gateway && go test -count=1 -race ./... && go vet ./...)
(cd simulator && go test -count=1 -race ./... && go vet ./...)
(cd apps/console && bun run lint && bun test && bunx tsc --noEmit && bun run build)
python3 scripts/phase8-access-smoke.py
python3 scripts/phase8-console-session-smoke.py
echo "Phase Eight regression passed: roles/sessions, signed collectors/replay, durable audit, retention and bounded rate limits."
