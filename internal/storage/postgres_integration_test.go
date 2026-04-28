package storage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/config"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func newTestPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()

	if os.Getenv("RUN_POSTGRES_TESTS") != "1" {
		t.Skip("set RUN_POSTGRES_TESTS=1 to run Postgres integration tests")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	store, err := NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	t.Cleanup(store.Close)

	_, err = store.pool.Exec(ctx, "TRUNCATE TABLE tasks")
	if err != nil {
		t.Fatalf("truncate tasks: %v", err)
	}

	return store
}

func TestPostgresStoreCreateAndGetTask(t *testing.T) {
	store := newTestPostgresStore(t)

	now := time.Now().UTC()
	payload := json.RawMessage(`{"email":"user@example.com"}`)

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

	got, err := store.GetTask(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.ID != want.ID {
		t.Fatalf("got ID %q, want %q", got.ID, want.ID)
	}

	if got.Type != want.Type {
		t.Fatalf("got Type %q, want %q", got.Type, want.Type)
	}

	if !jsonEqual(t, got.Payload, want.Payload) {
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

func jsonEqual(t *testing.T, got json.RawMessage, want json.RawMessage) bool {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("unmarshal got payload: %v", err)
	}

	var wantValue any
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("unmarshal want payload: %v", err)
	}

	return reflect.DeepEqual(gotValue, wantValue)
}

func TestPostgresStoreGetTaskNotFound(t *testing.T) {
	store := newTestPostgresStore(t)

	_, err := store.GetTask(context.Background(), "missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("GetTask() error = %v, want %v", err, ErrTaskNotFound)
	}
}

func TestPostgresStoreClaimNextTaskClaimsReadyPendingTask(t *testing.T) {
	store := newTestPostgresStore(t)

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "claim-ready-task",
		Type:        "send_email",
		Payload:     json.RawMessage(`{"email":"user@example.com"}`),
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

	if claimed.ID != "claim-ready-task" {
		t.Fatalf("claimed ID %q, want %q", claimed.ID, "claim-ready-task")
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

	stored, err := store.GetTask(context.Background(), claimed.ID)
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

func TestPostgresStoreClaimNextTaskSkipsFutureTasks(t *testing.T) {
	store := newTestPostgresStore(t)

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "future-task",
		Type:        "send_email",
		Payload:     json.RawMessage(`{"email":"user@example.com"}`),
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

func TestPostgresStoreClaimNextTaskSkipsNonPendingTasks(t *testing.T) {
	store := newTestPostgresStore(t)

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "running-task",
		Type:        "send_email",
		Payload:     json.RawMessage(`{"email":"user@example.com"}`),
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

func TestPostgresStoreClaimNextTaskReturnsNoTaskAvailable(t *testing.T) {
	store := newTestPostgresStore(t)

	_, err := store.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if !errors.Is(err, ErrNoTaskAvailable) {
		t.Fatalf("ClaimNextTask() error = %v, want %v", err, ErrNoTaskAvailable)
	}
}

func TestPostgresStoreClaimNextTaskDoesNotDoubleClaim(t *testing.T) {
	store := newTestPostgresStore(t)

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "single-task",
		Type:        "send_email",
		Payload:     json.RawMessage(`{"email":"user@example.com"}`),
		Status:      task.StatusPending,
		MaxAttempts: 3,
		RunAt:       now.Add(-1 * time.Minute),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	first, err := store.ClaimNextTask(context.Background(), "worker-1", 30*time.Second)
	if err != nil {
		t.Fatalf("first ClaimNextTask() error = %v", err)
	}

	if first.LockedBy == nil || *first.LockedBy != "worker-1" {
		t.Fatalf("first LockedBy %v, want worker-1", first.LockedBy)
	}

	_, err = store.ClaimNextTask(context.Background(), "worker-2", 30*time.Second)
	if !errors.Is(err, ErrNoTaskAvailable) {
		t.Fatalf("second ClaimNextTask() error = %v, want %v", err, ErrNoTaskAvailable)
	}

	stored, err := store.GetTask(context.Background(), "single-task")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if stored.LockedBy == nil || *stored.LockedBy != "worker-1" {
		t.Fatalf("stored LockedBy %v, want worker-1", stored.LockedBy)
	}

	if stored.Attempts != 1 {
		t.Fatalf("stored Attempts %d, want %d", stored.Attempts, 1)
	}
}

func TestPostgresStoreCompleteTaskMarksTaskCompleted(t *testing.T) {
	store := newTestPostgresStore(t)

	now := time.Now().UTC()
	err := store.CreateTask(context.Background(), task.Task{
		ID:          "complete-task",
		Type:        "send_email",
		Payload:     json.RawMessage(`{"email":"user@example.com"}`),
		Status:      task.StatusPending,
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

	err = store.CompleteTask(context.Background(), claimed.ID)
	if err != nil {
		t.Fatalf("CompleteTask() error = %v", err)
	}

	got, err := store.GetTask(context.Background(), claimed.ID)
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

func TestPostgresStoreCompleteTaskNotFound(t *testing.T) {
	store := newTestPostgresStore(t)

	err := store.CompleteTask(context.Background(), "missing-task")
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("CompleteTask() error = %v, want %v", err, ErrTaskNotFound)
	}
}
