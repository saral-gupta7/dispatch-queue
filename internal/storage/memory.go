package storage

import (
	"context"
	"sync"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

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

// ClaimNextTask claims one pending task for a worker.
func (s *MemoryStore) ClaimNextTask(ctx context.Context, workerID string, leaseDuration time.Duration) (task.Task, error) {
	if err := ctx.Err(); err != nil {
		return task.Task{}, err
	}

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, t := range s.tasks {
		if t.Status != task.StatusPending {
			continue
		}

		if t.RunAt.After(now) {
			continue
		}

		lockedBy := workerID
		lockedUntil := now.Add(leaseDuration)

		t.Status = task.StatusRunning
		t.LockedBy = &lockedBy
		t.LockedUntil = &lockedUntil
		t.Attempts++
		t.UpdatedAt = now

		s.tasks[id] = t

		return t, nil
	}

	return task.Task{}, ErrNoTaskAvailable
}

// CompleteTask marks a task as completed.
func (s *MemoryStore) CompleteTask(ctx context.Context, taskID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[taskID]

	if !ok {
		return ErrTaskNotFound
	}
	t.Status = task.StatusCompleted
	t.LockedBy = nil
	t.LockedUntil = nil
	t.UpdatedAt = now

	s.tasks[taskID] = t
	return nil
}
