package queue

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func TestServiceEnqueueAppliesDefaultsAndStoresTask(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	payload := json.RawMessage(`{"email": "saralgupta@gmail.com"}`)
	want := task.Task{
		ID:      "task-1",
		Type:    "send_email",
		Payload: payload,
	}

	got, err := svc.Enqueue(context.Background(), want)

	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	if got.ID != want.ID {
		t.Fatalf("got ID %q, want ID %q", got.ID, want.ID)
	}

	if got.Type != want.Type {
		t.Fatalf("got type %q, want type %q", got.Type, want.Type)
	}

	if string(got.Payload) != string(want.Payload) {
		t.Fatalf("got payload %q, want payload %q", got.Payload, want.Payload)
	}

	if got.Status != task.StatusPending {
		t.Fatalf("got status %q, want status %q", got.Status, task.StatusPending)
	}

	if got.MaxAttempts != 3 {
		t.Fatalf("got maxAttempts %d, want maxAttempts %d", got.MaxAttempts, 3)
	}

	if got.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt not set")
	}

	if got.RunAt.IsZero() {
		t.Fatalf("RunAt not set")
	}
	if got.UpdatedAt.IsZero() {
		t.Fatalf("UpdatedAt not set")
	}
	stored, err := store.GetTask(context.Background(), got.ID)

	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if stored.ID != got.ID {
		t.Fatalf("stored ID %q, want %q", stored.ID, got.ID)
	}

	if stored.Status != got.Status {
		t.Fatalf("stored Status %q, want %q", stored.Status, got.Status)
	}

	if stored.MaxAttempts != got.MaxAttempts {
		t.Fatalf("stored MaxAttempts %d, want %d", stored.MaxAttempts, got.MaxAttempts)
	}

}

func TestServiceEnqueueRejectsMissingID(t *testing.T) {
	store := storage.NewMemoryStore()

	svc := NewService(store)

	task := task.Task{Type: "send_email"}

	_, err := svc.Enqueue(context.Background(), task)

	if !errors.Is(err, ErrTaskIDRequired) {
		t.Fatalf("EnqueueError() error = %v, want %v", err, ErrTaskIDRequired)
	}

}

func TestServiceEnqueueRejectsMissingType(t *testing.T) {
	store := storage.NewMemoryStore()

	svc := NewService(store)

	task := task.Task{ID: "task-1"}

	_, err := svc.Enqueue(context.Background(), task)

	if !errors.Is(err, ErrTaskTypeRequired) {
		t.Fatalf("EnqueueError() error = %v, want %v", err, ErrTaskTypeRequired)
	}

}

func TestServiceEnqueuePreservesExplicitValues(t *testing.T) {

	runAt := time.Now().UTC().Add(10 * time.Minute)
	createdAt := time.Now().UTC().Add(-1 * time.Hour)

	want := task.Task{ID: "task-1", Type: "send_email", Status: task.StatusRunning, MaxAttempts: 7, RunAt: runAt, CreatedAt: createdAt}

	store := storage.NewMemoryStore()
	svc := NewService(store)

	got, err := svc.Enqueue(context.Background(), want)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	if got.Status != task.StatusRunning {
		t.Fatalf("got Status %q, want %q", got.Status, task.StatusRunning)
	}

	if got.MaxAttempts != 7 {
		t.Fatalf("got MaxAttempts %d, want %d", got.MaxAttempts, 7)
	}

	if !got.RunAt.Equal(runAt) {
		t.Fatalf("got RunAt %v, want %v", got.RunAt, runAt)
	}

	if !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("got CreatedAt %v, want %v", got.CreatedAt, createdAt)
	}

	if got.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt was not set")
	}

}

func TestServiceEnqueueReturnsStoreError(t *testing.T) {

	want := task.Task{ID: "task-1", Type: "send_email", Status: task.StatusRunning, MaxAttempts: 7}

	store := storage.NewMemoryStore()
	svc := NewService(store)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.Enqueue(ctx, want)
	if !errors.Is(err, context.Canceled) {

		t.Fatalf("Enqueue() error = %v, want %v", err, context.Canceled)
	}

}

func TestServiceGetTaskReturnsStoredTask(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	payload := json.RawMessage(`{"email":"user@example.com"}`)

	created, err := svc.Enqueue(context.Background(), task.Task{
		ID:      "task-1",
		Type:    "send_email",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	got, err := svc.GetTask(context.Background(), "task-1")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf("got ID %q, want %q", got.ID, created.ID)
	}

	if got.Type != created.Type {
		t.Fatalf("got Type %q, want %q", got.Type, created.Type)
	}

	if got.Status != created.Status {
		t.Fatalf("got Status %q, want %q", got.Status, created.Status)
	}

	if got.MaxAttempts != created.MaxAttempts {
		t.Fatalf("got MaxAttempts %d, want %d", got.MaxAttempts, created.MaxAttempts)
	}

	if string(got.Payload) != string(created.Payload) {
		t.Fatalf("got Payload %s, want %s", got.Payload, created.Payload)
	}
}

func TestServiceGetTaskRejectsMissingID(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	_, err := svc.GetTask(context.Background(), "")
	if !errors.Is(err, ErrTaskIDRequired) {
		t.Fatalf("GetTask() error = %v, want %v", err, ErrTaskIDRequired)
	}
}

func TestServiceGetTaskReturnsNotFound(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	_, err := svc.GetTask(context.Background(), "missing-task")
	if !errors.Is(err, storage.ErrTaskNotFound) {
		t.Fatalf("GetTask() error = %v, want %v", err, storage.ErrTaskNotFound)
	}
}

func TestServiceClaimNextTaskClaimsAvailableTask(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	now := time.Now().UTC()
	created, err := svc.Enqueue(context.Background(), task.Task{
		ID:          "task-1",
		Type:        "send_email",
		Status:      task.StatusPending,
		RunAt:       now.Add(-1 * time.Minute),
		CreatedAt:   now,
		UpdatedAt:   now,
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	claimed, err := svc.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if err != nil {
		t.Fatalf("ClaimNextTask() error = %v", err)
	}

	if claimed.ID != created.ID {
		t.Fatalf("claimed ID %q, want %q", claimed.ID, created.ID)
	}

	if claimed.Status != task.StatusRunning {
		t.Fatalf("claimed Status %q, want %q", claimed.Status, task.StatusRunning)
	}

	if claimed.LockedBy == nil || *claimed.LockedBy != "worker-1" {
		t.Fatalf("claimed LockedBy %v, want worker-1", claimed.LockedBy)
	}
}

func TestServiceClaimNextTaskRejectsMissingWorkerID(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	_, err := svc.ClaimNextTask(context.Background(), "", 30*time.Second)
	if !errors.Is(err, ErrWorkerIDRequired) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, ErrWorkerIDRequired)
	}
}

func TestServiceClaimNextTaskRejectsInvalidLeaseDuration(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	_, err := svc.ClaimNextTask(context.Background(), "worker-1", 0)
	if !errors.Is(err, ErrLeaseDurationInvalid) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, ErrLeaseDurationInvalid)
	}
}

func TestServiceClaimNextTaskReturnsNoTaskAvailable(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	_, err := svc.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if !errors.Is(err, storage.ErrNoTaskAvailable) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, storage.ErrNoTaskAvailable)
	}
}

func TestServiceCompleteTaskMarksTaskCompleted(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	created, err := svc.Enqueue(context.Background(), task.Task{
		ID:          "task-1",
		Type:        "send_email",
		Status:      task.StatusPending,
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	claimed, err := svc.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if err != nil {
		t.Fatalf("ClaimNextTask() error = %v", err)
	}

	err = svc.CompleteTask(context.Background(), claimed.ID)
	if err != nil {
		t.Fatalf("CompleteTask() error = %v", err)
	}

	got, err := svc.GetTask(context.Background(), created.ID)
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
}

func TestServiceCompleteTaskRejectsMissingID(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	err := svc.CompleteTask(context.Background(), "")
	if !errors.Is(err, ErrTaskIDRequired) {
		t.Fatalf("CompleteTask() error = %v, want %v", err, ErrTaskIDRequired)
	}
}

func TestServiceCompleteTaskReturnsNotFound(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := NewService(store)

	err := svc.CompleteTask(context.Background(), "missing-task")
	if !errors.Is(err, storage.ErrTaskNotFound) {
		t.Fatalf("CompleteTask() error = %v, want %v", err, storage.ErrTaskNotFound)
	}
}
