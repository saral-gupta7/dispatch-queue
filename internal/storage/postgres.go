package storage

import (
	"context"
	"fmt"

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
