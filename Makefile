.PHONY: help dev-up dev-down dev-logs gateway-run gateway-test gateway-vet simulator-run simulator-test test fmt check smoke smoke-phase1 resilience-phase1 smoke-phase2 workflow-phase2 catalogue-phase2 quality-phase2 final-phase2 smoke-phase3 pivots-phase3 maturity-phase3 final-phase3

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
