package queue

import (
	"context"
	"errors"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

var (
	ErrTaskIDRequired   = errors.New("task id is required")
	ErrTaskTypeRequired = errors.New("task type is required")
)

type Service struct {
	store storage.Store
}

func NewService(store storage.Store) *Service {
	return &Service{store: store}
}

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

func (s *Service) GetTask(ctx context.Context, id string) (task.Task, error) {

	if id == "" {
		return task.Task{}, ErrTaskIDRequired
	}
	return s.store.GetTask(ctx, id)

}
