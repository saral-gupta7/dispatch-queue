package queue

import (
	"context"
	"errors"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

// Queue validation errors returned by Service methods.
var (
	ErrTaskIDRequired       = errors.New("task id is required")
	ErrTaskTypeRequired     = errors.New("task type is required")
	ErrWorkerIDRequired     = errors.New("worker id is required")
	ErrLeaseDurationInvalid = errors.New("lease duration must be positive")
)

// Service coordinates queue operations using a storage backend.
type Service struct {
	store storage.Store
}

// NewService creates a Service backed by the provided store.
func NewService(store storage.Store) *Service {
	return &Service{store: store}
}

// Enqueue validates, normalizes, and stores a task.
func (s *Service) Enqueue(ctx context.Context, t task.Task) (task.Task, error) {
	if t.ID == "" {
		return task.Task{}, ErrTaskIDRequired
	}

	if t.Type == "" {
		return task.Task{}, ErrTaskTypeRequired
	}

	now := time.Now().UTC()

	if t.Status == "" {
		t.Status = task.StatusPending
	}

	if t.MaxAttempts <= 0 {
		t.MaxAttempts = 3
	}

	if t.RunAt.IsZero() {
		t.RunAt = now
	}

	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}

	t.UpdatedAt = now

	err := s.store.CreateTask(ctx, t)

	if err != nil {
		return task.Task{}, err
	}

	return t, nil
}

// GetTask returns a task by ID.
func (s *Service) GetTask(ctx context.Context, id string) (task.Task, error) {
	if id == "" {
		return task.Task{}, ErrTaskIDRequired
	}

	return s.store.GetTask(ctx, id)
}

// ClaimNextTask claims one available task for a worker.
func (s *Service) ClaimNextTask(ctx context.Context, workerID string, leaseDuration time.Duration) (task.Task, error) {
	if workerID == "" {
		return task.Task{}, ErrWorkerIDRequired
	}

	if leaseDuration <= 0 {
		return task.Task{}, ErrLeaseDurationInvalid
	}

	return s.store.ClaimNextTask(ctx, workerID, leaseDuration)
}
