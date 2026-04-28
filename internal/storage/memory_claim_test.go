package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func TestMemoryStoreClaimNextTaskClaimsReadyPendingTask(t *testing.T) {
	store := NewMemoryStore()

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "task-1",
		Type:        "send_email",
		Status:      task.StatusPending,
		Attempts:    0,
		MaxAttempts: 3,
		RunAt:       now.Add(-1 * time.Minute),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	claimed, err := store.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if err != nil {
		t.Fatalf("ClaimNextTask() error = %v", err)
	}

	if claimed.ID != "task-1" {
		t.Fatalf("claimed ID %q, want %q", claimed.ID, "task-1")
	}

	if claimed.Status != task.StatusRunning {
		t.Fatalf("claimed Status %q, want %q", claimed.Status, task.StatusRunning)
	}

	if claimed.Attempts != 1 {
		t.Fatalf("claimed Attempts %d, want %d", claimed.Attempts, 1)
	}

	if claimed.LockedBy == nil || *claimed.LockedBy != "worker-1" {
		t.Fatalf("claimed LockedBy %v, want worker-1", claimed.LockedBy)
	}

	if claimed.LockedUntil == nil {
		t.Fatal("claimed LockedUntil was nil")
	}

	if !claimed.LockedUntil.After(now) {
		t.Fatalf("claimed LockedUntil %v should be after %v", claimed.LockedUntil, now)
	}

	stored, err := store.GetTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if stored.Status != task.StatusRunning {
		t.Fatalf("stored Status %q, want %q", stored.Status, task.StatusRunning)
	}

	if stored.LockedBy == nil || *stored.LockedBy != "worker-1" {
		t.Fatalf("stored LockedBy %v, want worker-1", stored.LockedBy)
	}
}

func TestMemoryStoreClaimNextTaskSkipsFutureTasks(t *testing.T) {
	store := NewMemoryStore()

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "future-task",
		Type:        "send_email",
		Status:      task.StatusPending,
		MaxAttempts: 3,
		RunAt:       now.Add(10 * time.Minute),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	_, err = store.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if !errors.Is(err, ErrNoTaskAvailable) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, ErrNoTaskAvailable)
	}
}

func TestMemoryStoreClaimNextTaskSkipsNonPendingTasks(t *testing.T) {
	store := NewMemoryStore()

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "running-task",
		Type:        "send_email",
		Status:      task.StatusRunning,
		MaxAttempts: 3,
		RunAt:       now.Add(-1 * time.Minute),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	_, err = store.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if !errors.Is(err, ErrNoTaskAvailable) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, ErrNoTaskAvailable)
	}
}

func TestMemoryStoreClaimNextTaskReturnsNoTaskAvailable(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if !errors.Is(err, ErrNoTaskAvailable) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, ErrNoTaskAvailable)
	}
}

func TestMemoryStoreClaimNextTaskWithCancelledContext(t *testing.T) {
	store := NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.ClaimNextTask(ctx, "worker-1", 30*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, context.Canceled)
	}
}
