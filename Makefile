.PHONY: fmt vet test build run-api run-api-env run-worker migrate-up

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
