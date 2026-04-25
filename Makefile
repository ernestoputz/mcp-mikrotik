BINARY    := mcp-server
IMAGE     := mcp-mikrotik
TAG       := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build run run-http test lint tidy docker-build docker-run docker-stop help

## Compile binary to bin/mcp-server (CGO_ENABLED=0)
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/$(BINARY) ./cmd/server

## Run locally via stdio (requires .env)
run: build
	@set -a; source .env; set +a; MCP_TRANSPORT=stdio ./bin/$(BINARY)

## Run locally via HTTP on :8080 (requires .env)
run-http: build
	@set -a; source .env; set +a; MCP_TRANSPORT=http HTTP_HOST=0.0.0.0 ./bin/$(BINARY)

## Run tests
test:
	go test ./... -race -count=1

## Lint
lint:
	golangci-lint run ./...

## Tidy modules
tidy:
	go mod tidy

## Build Docker image
docker-build:
	docker build -t $(IMAGE):$(TAG) -t $(IMAGE):latest -t $(IMAGE):local .

## Start container via Docker Compose (HTTP transport on :8080)
docker-run:
	@if [ ! -f .env ]; then echo "ERROR: .env not found — copy .env.example to .env and fill it in"; exit 1; fi
	docker compose up --build -d

## Stop and remove containers
docker-stop:
	docker compose down

## Show help
help:
	@grep -E '^##' Makefile | sed 's/## /  /'
