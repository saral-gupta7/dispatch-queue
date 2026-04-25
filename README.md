# Dispatch Queue

Dispatch Queue is a distributed task queue built in Go. The project is focused
on durable background job execution, clear task lifecycle modeling, and
production-style backend structure.

## Goals

- Accept background tasks from an API process.
- Store tasks durably.
- Let worker processes claim and execute tasks safely.
- Support at-least-once execution, retries, leases, and dead-letter handling.
- Keep queue, storage, worker, and task domain code cleanly separated.

## Current Status

Implemented so far:

- Production-oriented Go project structure.
- Queue guarantees documented in `docs/queue-guarantees.md`.
- Task domain model with lifecycle statuses.
- Task helper methods for status validation, terminal states, and retry checks.
- Focused tests for task model behavior.
- Storage interface and in-memory store implementation.

Planned next:

- Queue service layer.
- PostgreSQL-backed durable storage.
- Task claiming with row-level locking.
- Worker loop with graceful shutdown.
- Retry backoff and dead-letter transitions.
- HTTP API for submitting and inspecting tasks.

## Project Structure

```text
cmd/
  api/       API server entry point
  worker/    worker process entry point
internal/
  task/      task domain model and lifecycle helpers
  storage/   storage interfaces and implementations
  queue/     queue orchestration logic
  worker/    worker execution logic
  config/    configuration loading
  logger/    logging setup
docs/        design notes
migrations/  database migrations
```

## Development

Run tests:

```bash
make test
```

Format code:

```bash
make fmt
```

Build binaries:

```bash
make build
```

Run local placeholders:

```bash
make run-api
make run-worker
```

## Delivery Semantics

Dispatch Queue is designed around durable **at-least-once execution**.

Task handlers should be idempotent because a worker may complete an external
side effect and crash before recording success. In that case, the task may be
retried by another worker.
