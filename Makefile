.PHONY: help dev-up dev-down dev-logs gateway-run gateway-test gateway-vet test fmt check

help:
	@echo "Sentinel development commands"
	@echo "  make dev-up        Start PostgreSQL and NATS"
	@echo "  make dev-down      Stop local infrastructure"
	@echo "  make dev-logs      Follow local infrastructure logs"
	@echo "  make gateway-run   Run the Go gateway"
	@echo "  make gateway-test  Run gateway tests"
	@echo "  make gateway-vet   Run go vet for the gateway"
	@echo "  make test          Run all current tests"
	@echo "  make fmt           Format Go code"
	@echo "  make check         Format check, vet, and test"

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

test: gateway-test

fmt:
	cd services/gateway && gofmt -w $$(find . -name '*.go' -type f)

check:
	cd services/gateway && test -z "$$(gofmt -l .)"
	$(MAKE) gateway-vet
	$(MAKE) test
