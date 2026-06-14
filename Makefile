.PHONY: fmt vet test build run-api run-api-env run-worker run-worker-env compose-build compose-up compose-down compose-logs migrate-up

APP_NAME := dispatch-queue
GOCACHE ?= /tmp/go-build-cache

fmt:
	gofmt -w ./cmd ./internal

vet:
	GOCACHE=$(GOCACHE) go vet ./...

test:
	GOCACHE=$(GOCACHE) go test ./...

build:
	GOCACHE=$(GOCACHE) go build -o bin/api ./cmd/api
	GOCACHE=$(GOCACHE) go build -o bin/worker ./cmd/worker

run-api:
	GOCACHE=$(GOCACHE) go run ./cmd/api

run-api-env:
	set -a; source .env; set +a; GOCACHE=$(GOCACHE) go run ./cmd/api

migrate-up:
	docker compose exec -T postgres psql -U dispatch -d dispatch_queue < migrations/001_create_tasks.sql

run-worker:
	GOCACHE=$(GOCACHE) go run ./cmd/worker

run-worker-env:
	set -a; source .env; set +a; GOCACHE=$(GOCACHE) go run ./cmd/worker

compose-build:
	docker compose build api worker

compose-up:
	docker compose up -d --build postgres migrate api worker

compose-down:
	docker compose down

compose-logs:
	docker compose logs --no-color --tail=120 api worker postgres migrate
