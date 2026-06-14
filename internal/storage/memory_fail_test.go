package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func TestMemoryStoreFailTaskSchedulesRetry(t *testing.T) {
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
		RunAt:       now,
		LockedBy:    &lockedBy,
		LockedUntil: &lockedUntil,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	err = store.FailTask(context.Background(), "task-1", "smtp timeout", 5*time.Second)
	if err != nil {
		t.Fatalf("FailTask() error = %v", err)
	}

	got, err := store.GetTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.Status != task.StatusPending {
		t.Fatalf("got Status %q, want %q", got.Status, task.StatusPending)
	}

	if got.LastError == nil || *got.LastError != "smtp timeout" {
		t.Fatalf("LastError = %v, want smtp timeout", got.LastError)
	}

	if got.LockedBy != nil {
		t.Fatalf("LockedBy = %v, want nil", got.LockedBy)
	}

	if got.LockedUntil != nil {
		t.Fatalf("LockedUntil = %v, want nil", got.LockedUntil)
	}

	if !got.RunAt.After(now) {
		t.Fatalf("RunAt = %v, want after %v", got.RunAt, now)
	}

	if got.UpdatedAt.Equal(now) {
		t.Fatalf("UpdatedAt was not changed from %v", now)
	}
}

func TestMemoryStoreFailTaskMovesExhaustedTaskToDead(t *testing.T) {
	store := NewMemoryStore()

	now := time.Now().UTC().Add(-1 * time.Hour)
	lockedBy := "worker-1"
	lockedUntil := now.Add(30 * time.Second)

	err := store.CreateTask(context.Background(), task.Task{
		ID:          "task-1",
		Type:        "send_email",
		Status:      task.StatusRunning,
		Attempts:    3,
		MaxAttempts: 3,
		RunAt:       now,
		LockedBy:    &lockedBy,
		LockedUntil: &lockedUntil,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	err = store.FailTask(context.Background(), "task-1", "permanent failure", 5*time.Second)
	if err != nil {
		t.Fatalf("FailTask() error = %v", err)
	}

	got, err := store.GetTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.Status != task.StatusDead {
		t.Fatalf("got Status %q, want %q", got.Status, task.StatusDead)
	}

	if got.LastError == nil || *got.LastError != "permanent failure" {
		t.Fatalf("LastError = %v, want permanent failure", got.LastError)
	}

	if got.LockedBy != nil {
		t.Fatalf("LockedBy = %v, want nil", got.LockedBy)
	}

	if got.LockedUntil != nil {
		t.Fatalf("LockedUntil = %v, want nil", got.LockedUntil)
	}
}

func TestMemoryStoreFailTaskNotFound(t *testing.T) {
	store := NewMemoryStore()

	err := store.FailTask(context.Background(), "missing-task", "failed", 5*time.Second)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("FailTask() error = %v, want %v", err, ErrTaskNotFound)
	}
}

func TestMemoryStoreFailTaskWithCancelledContext(t *testing.T) {
	store := NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.FailTask(ctx, "task-1", "failed", 5*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("FailTask() error = %v, want %v", err, context.Canceled)
	}
}
