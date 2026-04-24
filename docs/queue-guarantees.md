# Queue Guarantees

This project implements a durable distributed task queue in Go.

## Core Guarantee

The queue provides **at-least-once execution**.

That means once a task is accepted, the system will try to execute it one or more
times until it either succeeds or reaches its retry limit.

## Why Not Exactly Once?

Exactly-once execution is not realistic once workers perform external side effects.

For example, a worker may successfully send an email but crash before marking the
task as complete. The system cannot always know whether the email was sent, so it may
retry the task.

Because of this, task handlers should be designed to be **idempotent**.

## Durability

Accepted tasks must be stored durably so they are not lost when the API server or
worker process crashes.

In this project, durable task storage will be handled by PostgreSQL.

## Leasing

Workers claim tasks using a temporary lease.

If a worker crashes or takes too long, the lease expires and another worker can claim
the task later.

This prevents tasks from getting stuck forever.

## Retries And Backoff

Failed tasks are retried with backoff.

Backoff means the system waits before retrying, instead of retrying immediately in a
tight loop.

## Dead-Letter Tasks

If a task fails too many times, it moves to a dead-letter state.

Dead-letter tasks are preserved for debugging instead of being silently deleted.

## Summary

This queue is designed for:

- durable task storage
- at-least-once execution
- safe worker retries
- lease-based task claiming
- dead-letter handling
- idempotent task handlers
