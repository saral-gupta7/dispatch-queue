package storage

import (
	"context"
	"errors"
	"sync"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

// ErrTaskNotFound is returned when a task does not exist in storage.
var ErrTaskNotFound = errors.New("task not found")

// MemoryStore stores tasks in memory.
//
// It is useful for tests and local learning, but it is not durable. If the
// process exits, all tasks stored here are lost.
type MemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]task.Task
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[string]task.Task),
	}
}

// CreateTask stores a new task in memory.
func (s *MemoryStore) CreateTask(ctx context.Context, t task.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[t.ID] = t
	return nil
}

// GetTask returns a task by ID.
func (s *MemoryStore) GetTask(ctx context.Context, id string) (task.Task, error) {
	if err := ctx.Err(); err != nil {
		return task.Task{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]

	if !ok {
		return task.Task{}, ErrTaskNotFound
	}
	return t, nil
}
