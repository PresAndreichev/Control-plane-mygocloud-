.PHONY: build build-cli build-worker test test-unit test-integration test-cli test-cli-integration run run-cli run-worker docker-up docker-down coverage lint deps install-cli

# API
build:
	@mkdir -p bin
	go build -o bin/api ./cmd/api
	@chmod +x bin/api 


run: build
	./bin/api

# CLI
build-cli:
	@mkdir -p bin
	go build -o bin/mygocloud ./cmd/mygocloud

install-cli: build-cli
	@mkdir -p ~/go/bin 2>/dev/null || true
	cp bin/mygocloud ~/go/bin/mygocloud 2>/dev/null || cp bin/mygocloud /usr/local/bin/mygocloud 2>/dev/null || echo "Install manually: cp bin/mygocloud /usr/local/bin/"

run-cli: build-cli
	./bin/mygocloud

# Worker
build-worker:
	@mkdir -p bin
	go build -o bin/worker ./cmd/worker

run-worker: build-worker
	./bin/worker

# Tests
test: test-unit test-integration test-cli test-cli-integration

test-unit:
	go test -v -race -count=1 ./internal/... ./cli/config/... ./cli/printer/... ./cli/client/...

test-integration:
	go test -v -race -count=1 -run Integration ./...

test-cli:
	go test -v -race -count=1 ./cli/commands/...

test-cli-integration:
	go test -v -race -count=1 -run TestCLI ./cli/...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down -v

lint:
	golangci-lint run ./...

deps:
	go mod download
	go mod tidy 
