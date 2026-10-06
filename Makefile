.PHONY: help dev-up dev-down dev-logs gateway-run gateway-test gateway-vet simulator-run simulator-test test fmt check smoke smoke-phase1 resilience-phase1

help:
	@echo "Sentinel development commands"
	@echo "  make dev-up          Start PostgreSQL and NATS"
	@echo "  make dev-down        Stop local infrastructure"
	@echo "  make dev-logs        Follow local infrastructure logs"
	@echo "  make gateway-run     Run the Go gateway"
	@echo "  make gateway-test    Run gateway tests"
	@echo "  make gateway-vet     Run go vet for the gateway"
	@echo "  make simulator-run   Run the default synthetic telemetry scenario"
	@echo "  make simulator-test  Run simulator tests/build checks"
	@echo "  make test            Run all current tests"
	@echo "  make fmt             Format Go code"
	@echo "  make check           Format check, vet, tests, race tests, and simulator build"
	@echo "  make smoke           Run the Phase 0 foundation smoke test"
	@echo "  make smoke-phase1    Run the Phase 1 telemetry vertical-slice smoke test"
	@echo "  make resilience-phase1  Run Phase 1 dependency failure/recovery tests"

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
