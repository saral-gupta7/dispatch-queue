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
