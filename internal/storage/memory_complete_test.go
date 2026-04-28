package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func TestMemoryStoreCompleteTaskMarksTaskCompleted(t *testing.T) {
	store := NewMemoryStore()

	now := time.Now().UTC().Add(-1 * time.Hour)
	lockedBy := "worker-1"
	lockedUntil := now.Add(30 * time.Second)

	err := store.CreateTask(context.Background(), task.Task{
		ID:          "task-1",
		Type:        "send_email",
		Status:      task.StatusRunning,
		Attempts:    1,
		MaxAttempts: 3,
		RunAt:       now.Add(-1 * time.Minute),
		LockedBy:    &lockedBy,
		LockedUntil: &lockedUntil,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	err = store.CompleteTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("CompleteTask() error = %v", err)
	}

	got, err := store.GetTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.Status != task.StatusCompleted {
		t.Fatalf("got Status %q, want %q", got.Status, task.StatusCompleted)
	}

	if got.LockedBy != nil {
		t.Fatalf("LockedBy = %v, want nil", got.LockedBy)
	}

	if got.LockedUntil != nil {
		t.Fatalf("LockedUntil = %v, want nil", got.LockedUntil)
	}

	if got.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt was not changed from %v", now)
	}
}

func TestMemoryStoreCompleteTaskNotFound(t *testing.T) {
	store := NewMemoryStore()

	err := store.CompleteTask(context.Background(), "missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("CompleteTask() error = %v, want %v", err, ErrTaskNotFound)
	}
}

func TestMemoryStoreCompleteTaskWithCancelledContext(t *testing.T) {
	store := NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.CompleteTask(ctx, "task-1")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("CompleteTask() error = %v, want %v", err, context.Canceled)
	}
}
