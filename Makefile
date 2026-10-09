.PHONY: final-phase8 final-phase9 security-gate ml-catalogue-evaluate help dev-up dev-down dev-logs gateway-run gateway-test gateway-vet simulator-run simulator-test console-dev console-lint console-build ml-test ml-evaluate test fmt check smoke smoke-phase1 resilience-phase1 smoke-phase2 workflow-phase2 catalogue-phase2 quality-phase2 final-phase2 smoke-phase3 pivots-phase3 maturity-phase3 final-phase3 workflow-phase4 final-phase4 smoke-phase5 final-phase5 smoke-phase6 final-phase6 smoke-phase7 final-phase7 smoke-phase8 sessions-phase8 provision-lab-auth

help:
	@echo "Sentinel development commands"
	@echo "  make dev-up           Start PostgreSQL and NATS"
	@echo "  make dev-down         Stop local infrastructure"
	@echo "  make dev-logs         Follow local infrastructure logs"
	@echo "  make gateway-run      Run the Go gateway"
	@echo "  make gateway-test     Run gateway tests"
	@echo "  make gateway-vet      Run go vet for the gateway"
	@echo "  make simulator-run    Run the default synthetic telemetry scenario"
	@echo "  make simulator-test   Run simulator tests and vet"
	@echo "  make console-dev      Run the Next.js analyst console"
	@echo "  make console-lint     Lint the analyst console"
	@echo "  make console-build    Build the analyst console"
	@echo "  make ml-test          Run Python behavioural analytics tests"
	@echo "  make ml-evaluate      Evaluate the synthetic behavioural model"
	@echo "  make ml-catalogue-evaluate Evaluate per-profile synthetic coverage"
	@echo "  make record-phase7-evaluation Persist a governed Phase 7 evaluation run"
	@echo "  make test             Run all current tests"
	@echo "  make fmt              Format Go code"
	@echo "  make check            Formatting, vet, tests, and race detector"
	@echo "  make smoke            Run the Phase 0 foundation smoke test"
	@echo "  make smoke-phase1     Run the Phase 1 telemetry smoke test"
	@echo "  make resilience-phase1 Run dependency failure/recovery tests"
	@echo "  make smoke-phase2     Run the temporal detection smoke test"
	@echo "  make workflow-phase2  Run analyst workflow/runtime-control checks"
	@echo "  make catalogue-phase2 Run multi-detection catalogue checks"
	@echo "  make quality-phase2   Run quality metrics/evidence checks"
	@echo "  make final-phase2     Run the complete Phase 2 regression suite"
	@echo "  make smoke-phase3     Run the Phase 3 hunting/investigation vertical slice"
	@echo "  make pivots-phase3    Run Phase 3 pivot/history/investigation enrichment checks"
	@echo "  make maturity-phase3   Run Phase 3 hunt versioning/operator/metrics checks"
	@echo "  make final-phase3      Run the complete Phase 3 regression suite"
	@echo "  make workflow-phase4   Run the Phase 4 analyst-console workflow smoke test"
	@echo "  make final-phase4      Run the complete Phase 4 regression suite"
	@echo "  make smoke-phase5      Run the Phase 5 OT simulation vertical slice"
	@echo "  make final-phase5      Run the complete Phase 5 regression suite"
	@echo "  make smoke-phase6      Run the Phase 6 intelligence/enrichment vertical slice"
	@echo "  make final-phase6      Run the complete Phase 6 regression suite"
	@echo "  make smoke-phase7      Run the Phase 7 behavioural analytics vertical slice"
	@echo "  make final-phase7      Run the complete Phase 7 regression and completion gate"
	@echo "  make smoke-phase8      Run isolated API authentication/RBAC checks"
	@echo "  make sessions-phase8   Build and test interactive console sessions"
	@echo "  make provision-lab-auth Create eight-hour lab credentials in private files"

dev-up:
	docker compose up -d

dev-down:
	docker compose down

dev-logs:
	docker compose logs -f

gateway-run:
	cd services/gateway && go run ./cmd/gateway

gateway-test:
	cd services/gateway && go test ./...

gateway-vet:
	cd services/gateway && go vet ./...

simulator-run:
	cd simulator && go run ./cmd/sentinel-sim -scenario auth-burst

simulator-test:
	cd simulator && go test ./...
	cd simulator && go vet ./...

console-dev:
	cd apps/console && bun run dev

console-lint:
	cd apps/console && bun run lint

console-build:
	cd apps/console && bun run build

test: gateway-test simulator-test

fmt:
	cd services/gateway && gofmt -w $$(find . -name '*.go' -type f)
	cd simulator && gofmt -w $$(find . -name '*.go' -type f)

check:
	cd services/gateway && test -z "$$(gofmt -l .)"
	cd simulator && test -z "$$(gofmt -l .)"
	$(MAKE) gateway-vet
	$(MAKE) test
	cd services/gateway && go test -race ./...

smoke:
	./scripts/phase0-smoke.sh

smoke-phase1:
	./scripts/phase1-smoke.sh

resilience-phase1:
	./scripts/phase1-resilience.sh

smoke-phase2:
	./scripts/phase2-smoke.sh

workflow-phase2:
	./scripts/phase2-workflow-smoke.sh

catalogue-phase2:
	./scripts/phase2-catalogue-smoke.sh

quality-phase2:
	./scripts/phase2-quality-smoke.sh

final-phase2:
	./scripts/phase2-final-regression.sh

smoke-phase3:
	./scripts/phase3-smoke.sh

pivots-phase3:
	./scripts/phase3-pivots-smoke.sh

maturity-phase3:
	./scripts/phase3-maturity-smoke.sh

final-phase3:
	./scripts/phase3-final-regression.sh

workflow-phase4:
	./scripts/phase4-workflow-smoke.sh

final-phase4:
	./scripts/phase4-final-regression.sh

smoke-phase5:
	./scripts/phase5-ot-smoke.sh

final-phase5:
	./scripts/phase5-final-regression.sh

smoke-phase6:
	./scripts/phase6-intelligence-smoke.sh

final-phase6:
	./scripts/phase6-final-regression.sh


smoke-phase7:
	./scripts/phase7-behaviour-smoke.sh


ml-test:
	docker compose build ml
	docker compose run --rm ml python -m pytest -q

ml-evaluate:
	docker compose build ml
	docker compose run --rm ml python -m app.evaluation

record-phase7-evaluation:
	./scripts/phase7-evaluate-record.sh

ml-catalogue-evaluate:
	docker compose build ml
	docker compose run --rm ml python -m app.catalogue_evaluation

final-phase7:
	./scripts/phase7-final-regression.sh

smoke-phase8:
	python3 scripts/phase8-access-smoke.py

sessions-phase8: console-build
	python3 scripts/phase8-console-session-smoke.py

provision-lab-auth:
	python3 scripts/provision-lab-console-auth.py

final-phase8:
	./scripts/phase8-final-regression.sh

final-phase9:
	python3 scripts/phase9-deployment-smoke.py --report /tmp/sentinel-phase9-report.json

security-gate:
	python3 scripts/security-gate.py --artifacts /tmp/sentinel-security-artifacts
