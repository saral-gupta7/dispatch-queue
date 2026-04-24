.PHONY: fmt vet test build run-api run-worker

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

run-worker:
	GOCACHE=$(GOCACHE) go run ./cmd/worker
