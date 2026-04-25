package storage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func TestMemoryStoreCreateAndGetTask(t *testing.T) {
	store := NewMemoryStore()

	now := time.Now().UTC()
	payload := json.RawMessage(`{"email": "saralgupta@gmail.com"}`)

	want := task.Task{
		ID:          "task-1",
		Type:        "send_email",
		Payload:     payload,
		Status:      task.StatusPending,
		Attempts:    0,
		MaxAttempts: 3,
		RunAt:       now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := store.CreateTask(context.Background(), want)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	got, err := store.GetTask(context.Background(), "task-1")

	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.ID != want.ID {
		t.Fatalf("got ID %q, want %q", got.ID, want.ID)
	}

	if got.Type != want.Type {
		t.Fatalf("got Type %q, want %q", got.Type, want.Type)
	}

	if string(got.Payload) != string(want.Payload) {
		t.Fatalf("got Payload %s, want %s", got.Payload, want.Payload)
	}

	if got.Status != want.Status {
		t.Fatalf("got Status %q, want %q", got.Status, want.Status)
	}

	if got.Attempts != want.Attempts {
		t.Fatalf("got Attempts %d, want %d", got.Attempts, want.Attempts)
	}

	if got.MaxAttempts != want.MaxAttempts {
		t.Fatalf("got MaxAttempts %d, want %d", got.MaxAttempts, want.MaxAttempts)
	}
}
func TestMemoryStoreGetTaskNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.GetTask(context.Background(), "missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetTask() error = %v, want = %v", err, ErrTaskNotFound)
	}

}
func TestMemoryStoreCreateTaskWithCancelledContext(t *testing.T) {
	store := NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.CreateTask(ctx, task.Task{ID: "task-1"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateTask() error = %v, want %v", err, context.Canceled)
	}
}
func TestMemoryStoreGetTaskWithCancelledContext(t *testing.T) {
	store := NewMemoryStore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.GetTask(ctx, "task-1")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetTask() error = %v, want %v", err, context.Canceled)
	}
}
