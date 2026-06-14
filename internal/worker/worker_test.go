package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/queue"
	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func TestNewRejectsInvalidConfig(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := queue.NewService(store)

	tests := []struct {
		name    string
		queue   Queue
		config  Config
		wantErr error
	}{
		{
			name:    "missing queue",
			queue:   nil,
			config:  validConfig(),
			wantErr: ErrQueueRequired,
		},
		{
			name:    "missing worker id",
			queue:   svc,
			config:  Config{LeaseDuration: time.Second, PollInterval: time.Second},
			wantErr: ErrWorkerIDRequired,
		},
		{
			name:    "invalid lease duration",
			queue:   svc,
			config:  Config{WorkerID: "worker-1", PollInterval: time.Second},
			wantErr: ErrLeaseDurationInvalid,
		},
		{
			name:    "invalid poll interval",
			queue:   svc,
			config:  Config{WorkerID: "worker-1", LeaseDuration: time.Second},
			wantErr: ErrPollIntervalInvalid,
		},
		{
			name:    "invalid retry delay",
			queue:   svc,
			config:  Config{WorkerID: "worker-1", LeaseDuration: time.Second, PollInterval: time.Second, RetryDelay: -1 * time.Second},
			wantErr: ErrRetryDelayInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.queue, tt.config)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("New() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegisterHandlerValidatesInput(t *testing.T) {
	w := newTestWorker(t)

	if err := w.RegisterHandler("", func(context.Context, task.Task) error { return nil }); !errors.Is(err, ErrTaskTypeRequired) {
		t.Fatalf("RegisterHandler() error = %v, want %v", err, ErrTaskTypeRequired)
	}

	if err := w.RegisterHandler("send_email", nil); !errors.Is(err, ErrTaskHandlerRequired) {
		t.Fatalf("RegisterHandler() error = %v, want %v", err, ErrTaskHandlerRequired)
	}

	if err := w.RegisterHandler("send_email", func(context.Context, task.Task) error { return nil }); err != nil {
		t.Fatalf("RegisterHandler() error = %v", err)
	}

	if err := w.RegisterHandler("send_email", func(context.Context, task.Task) error { return nil }); !errors.Is(err, ErrHandlerAlreadyExists) {
		t.Fatalf("RegisterHandler() error = %v, want %v", err, ErrHandlerAlreadyExists)
	}
}

func TestProcessOneCompletesSuccessfulTask(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := queue.NewService(store)
	w := newTestWorkerWithQueue(t, svc)

	if err := w.RegisterHandler("send_email", func(context.Context, task.Task) error { return nil }); err != nil {
		t.Fatalf("RegisterHandler() error = %v", err)
	}

	created := enqueueTask(t, svc, task.Task{
		ID:   "task-1",
		Type: "send_email",
	})

	processed, err := w.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}

	if !processed {
		t.Fatal("ProcessOne() processed = false, want true")
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.Status != task.StatusCompleted {
		t.Fatalf("got Status %q, want %q", got.Status, task.StatusCompleted)
	}
}

func TestProcessOneFailsHandlerError(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := queue.NewService(store)
	w := newTestWorkerWithQueue(t, svc)

	if err := w.RegisterHandler("send_email", func(context.Context, task.Task) error {
		return errors.New("smtp timeout")
	}); err != nil {
		t.Fatalf("RegisterHandler() error = %v", err)
	}

	created := enqueueTask(t, svc, task.Task{
		ID:          "task-1",
		Type:        "send_email",
		MaxAttempts: 3,
	})

	processed, err := w.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}

	if !processed {
		t.Fatal("ProcessOne() processed = false, want true")
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.Status != task.StatusPending {
		t.Fatalf("got Status %q, want %q", got.Status, task.StatusPending)
	}

	if got.LastError == nil || *got.LastError != "smtp timeout" {
		t.Fatalf("LastError = %v, want smtp timeout", got.LastError)
	}
}

func TestProcessOneFailsMissingHandler(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := queue.NewService(store)
	w := newTestWorkerWithQueue(t, svc)

	created := enqueueTask(t, svc, task.Task{
		ID:          "task-1",
		Type:        "send_email",
		MaxAttempts: 3,
	})

	processed, err := w.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}

	if !processed {
		t.Fatal("ProcessOne() processed = false, want true")
	}

	got, err := svc.GetTask(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if got.Status != task.StatusPending {
		t.Fatalf("got Status %q, want %q", got.Status, task.StatusPending)
	}

	if got.LastError == nil || *got.LastError == "" {
		t.Fatal("LastError was not recorded")
	}
}

func TestProcessOneReturnsFalseWhenNoTaskAvailable(t *testing.T) {
	w := newTestWorker(t)

	processed, err := w.ProcessOne(context.Background())
	if err != nil {
		t.Fatalf("ProcessOne() error = %v", err)
	}

	if processed {
		t.Fatal("ProcessOne() processed = true, want false")
	}
}

func newTestWorker(t *testing.T) *Worker {
	t.Helper()

	store := storage.NewMemoryStore()
	svc := queue.NewService(store)
	return newTestWorkerWithQueue(t, svc)
}

func newTestWorkerWithQueue(t *testing.T, svc *queue.Service) *Worker {
	t.Helper()

	w, err := New(svc, validConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return w
}

func validConfig() Config {
	return Config{
		WorkerID:      "worker-1",
		LeaseDuration: 30 * time.Second,
		PollInterval:  10 * time.Millisecond,
		RetryDelay:    5 * time.Second,
	}
}

func enqueueTask(t *testing.T, svc *queue.Service, taskToCreate task.Task) task.Task {
	t.Helper()

	created, err := svc.Enqueue(context.Background(), taskToCreate)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	return created
}
