package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

// PostgresStore stores tasks in PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a PostgreSQL-backed store.
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)

	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)

	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

// Close releases database connections held by the store.
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// CreateTask stores a new task in PostgreSQL.
func (s *PostgresStore) CreateTask(ctx context.Context, t task.Task) error {
	query := `
		INSERT INTO tasks (
			id,
			type,
			payload,
			status,
			attempts,
			max_attempts,
			last_error,
			run_at,
			locked_by,
			locked_until,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := s.pool.Exec(
		ctx,
		query,
		t.ID,
		t.Type,
		t.Payload,
		t.Status,
		t.Attempts,
		t.MaxAttempts,
		t.LastError,
		t.RunAt,
		t.LockedBy,
		t.LockedUntil,
		t.CreatedAt,
		t.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}
	return nil
}

// GetTask returns a task by ID from PostgreSQL.
func (s *PostgresStore) GetTask(ctx context.Context, id string) (task.Task, error) {
	query := `
		SELECT
			id,
			type,
			payload,
			status,
			attempts,
			max_attempts,
			last_error,
			run_at,
			locked_by,
			locked_until,
			created_at,
			updated_at
		FROM tasks
		WHERE id = $1
	`

	var t task.Task
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&t.ID,
		&t.Type,
		&t.Payload,
		&t.Status,
		&t.Attempts,
		&t.MaxAttempts,
		&t.LastError,
		&t.RunAt,
		&t.LockedBy,
		&t.LockedUntil,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return task.Task{}, ErrTaskNotFound
		}
		return task.Task{}, fmt.Errorf("get task: %w", err)
	}

	return t, nil
}

// ClaimNextTask claims one pending task for a worker.
func (s *PostgresStore) ClaimNextTask(ctx context.Context, workerID string, leaseDuration time.Duration) (task.Task, error) {
	now := time.Now().UTC()
	lockedUntil := now.Add(leaseDuration)
	query := `
		WITH candidate AS (
			SELECT id
			FROM tasks
			WHERE status = 'pending'
				AND run_at <= $1
			ORDER BY run_at, created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE tasks
		SET
			status = 'running',
			attempts = attempts + 1,
			locked_by = $2,
			locked_until = $3,
			updated_at = $1
		FROM candidate
		WHERE tasks.id = candidate.id
		RETURNING
			tasks.id,
			tasks.type,
			tasks.payload,
			tasks.status,
			tasks.attempts,
			tasks.max_attempts,
			tasks.last_error,
			tasks.run_at,
			tasks.locked_by,
			tasks.locked_until,
			tasks.created_at,
			tasks.updated_at
	`

	var t task.Task
	err := s.pool.QueryRow(ctx, query, now, workerID, lockedUntil).Scan(
		&t.ID,
		&t.Type,
		&t.Payload,
		&t.Status,
		&t.Attempts,
		&t.MaxAttempts,
		&t.LastError,
		&t.RunAt,
		&t.LockedBy,
		&t.LockedUntil,
		&t.CreatedAt,
		&t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return task.Task{}, ErrNoTaskAvailable
		}
		return task.Task{}, fmt.Errorf("claim next task: %w", err)
	}

	return t, nil
}

// CompleteTask marks a task as completed in PostgreSQL.
func (s *PostgresStore) CompleteTask(ctx context.Context, taskID string) error {
	now := time.Now().UTC()

	query := `
		UPDATE tasks
		SET
			status = 'completed',
			locked_by = NULL,
			locked_until = NULL,
			updated_at = $1
		WHERE id = $2
	`

	tag, err := s.pool.Exec(ctx, query, now, taskID)
	if err != nil {
		return fmt.Errorf("complete task: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}

// FailTask records a task failure in PostgreSQL.
//
// If the task still has retry attempts left, it becomes pending again and is
// scheduled for a future run. If it has no attempts left, it becomes dead.
func (s *PostgresStore) FailTask(ctx context.Context, taskID string, message string, retryDelay time.Duration) error {
	now := time.Now().UTC()
	nextRunAt := now.Add(retryDelay)

	query := `
		UPDATE tasks
		SET
				status = CASE
						WHEN attempts < max_attempts THEN 'pending'
						ELSE 'dead'
				END,
				last_error = $1,
				run_at = CASE
						WHEN attempts < max_attempts THEN $2
						ELSE run_at
				END,
				locked_by = NULL,
				locked_until = NULL,
				updated_at = $3
		WHERE id = $4
        `

	tag, err := s.pool.Exec(ctx, query, message, nextRunAt, now, taskID)
	if err != nil {
		return fmt.Errorf("fail task: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrTaskNotFound
	}

	return nil
}
