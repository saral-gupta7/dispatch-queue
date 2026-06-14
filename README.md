# Dispatch Queue

Dispatch Queue is a distributed task queue built in Go. The project is focused
on durable background job execution, clear task lifecycle modeling, and
production-style backend structure.

## V1 Goal

- Accept background tasks from an API process.
- Store tasks durably in PostgreSQL.
- Let worker processes claim and execute tasks safely.
- Support at-least-once execution, retries, leases, and dead-letter handling.
- Keep task, storage, queue, worker, and API code cleanly separated.

## Current Status

Implemented so far:

- Production-oriented Go project structure.
- Queue guarantees documented in `docs/queue-guarantees.md`.
- Task domain model with lifecycle statuses.
- Task helper methods for status validation, terminal states, and retry checks.
- Storage interface with create, lookup, claim, complete, and fail operations.
- In-memory store implementation for tests and learning.
- PostgreSQL store using `pgxpool`.
- Docker Compose PostgreSQL service.
- Initial `tasks` table migration with queue-oriented indexes.
- Queue service layer for enqueue, lookup, claim, complete, and fail workflows.
- Expired running lease reclaiming in memory and PostgreSQL stores.
- Worker package with handler registration, polling, completion, failure, and graceful shutdown support.
- `cmd/worker` wired to configuration, PostgreSQL, queue service, and a sample `send_email` handler.
- HTTP API package with `POST /tasks`, `GET /tasks/{id}`, and `GET /health`.
- `cmd/api` wired to configuration, PostgreSQL, queue service, HTTP serving, and graceful shutdown.
- Unit tests for task, config, API, queue, worker, and memory storage behavior.
- PostgreSQL integration tests for create, lookup, claim, complete, and fail workflows.

Still being wired:

- Production-style retry policy tuning beyond the current fixed retry delay.
- Structured logging and metrics.
- CI workflow and production deployment notes.

Important current limitation:

- The sample worker handler logs `send_email` tasks; it does not send real email.
- Docker Compose publishes API on `localhost:18080` and PostgreSQL on
  `localhost:55432` to avoid common local port conflicts.

## Project Structure

```text
cmd/
  api/       API process entry point
  worker/    worker process entry point
internal/
  api/       HTTP handlers for task submission, lookup, and health
  task/      task domain model and lifecycle helpers
  storage/   storage interface plus memory and PostgreSQL implementations
  queue/     queue orchestration service
  worker/    worker execution loop and handler registry
  config/    configuration loading
  logger/    reserved for logging setup
docs/        design notes
migrations/  database migrations
```

## Current Verification

Run unit tests:

```bash
make test
```

Run vet:

```bash
make vet
```

Run PostgreSQL locally and apply the migration:

```bash
docker compose up -d postgres
make migrate-up
```

Run PostgreSQL integration tests:

```bash
RUN_POSTGRES_TESTS=1 GOCACHE=/tmp/go-build-cache go test ./internal/storage
```

Build binaries:

```bash
make build
```

Run the API:

```bash
make run-api-env
```

Run the worker in another shell:

```bash
make run-worker-env
```

Submit a task:

```bash
curl -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"type":"send_email","payload":{"email":"user@example.com"}}'
```

Inspect a task:

```bash
curl http://localhost:8080/tasks/<task-id>
```

Run the full Docker Compose stack:

```bash
make compose-up
```

In the Compose stack, the API is published on `localhost:18080`:

```bash
curl http://localhost:18080/health

curl -X POST http://localhost:18080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"id":"smoke-1","type":"send_email","payload":{"email":"user@example.com"}}'

curl http://localhost:18080/tasks/smoke-1
```

Stop the Compose stack:

```bash
make compose-down
```

## Remaining V1 Wiring

1. Add richer retry policy tests around fixed delay versus future exponential backoff.
2. Add structured logging and basic metrics hooks.
3. Add request/response documentation for the HTTP API.
4. Add CI for tests, vet, and container build validation.
5. Add production deployment notes.

## Delivery Semantics

Dispatch Queue is designed around durable **at-least-once execution**.

Task handlers should be idempotent because a worker may complete an external
side effect and crash before recording success. In that case, the task may be
retried by another worker.
