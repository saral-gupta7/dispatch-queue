package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

var (
	ErrQueueRequired         = errors.New("queue is required")
	ErrWorkerIDRequired      = errors.New("worker id is required")
	ErrLeaseDurationInvalid  = errors.New("lease duration must be positive")
	ErrPollIntervalInvalid   = errors.New("poll interval must be positive")
	ErrRetryDelayInvalid     = errors.New("retry delay cannot be negative")
	ErrTaskTypeRequired      = errors.New("task type is required")
	ErrTaskHandlerRequired   = errors.New("task handler is required")
	ErrHandlerAlreadyExists  = errors.New("handler already exists for task type")
	ErrHandlerMissingForTask = errors.New("handler missing for task type")
)

// Queue is the queue behavior the worker needs.
type Queue interface {
	ClaimNextTask(ctx context.Context, workerID string, leaseDuration time.Duration) (task.Task, error)
	CompleteTask(ctx context.Context, taskID string) error
	FailTask(ctx context.Context, taskID string, message string, retryDelay time.Duration) error
}

// Handler executes a claimed task.
type Handler func(ctx context.Context, t task.Task) error

// Config controls worker polling, leases, and retry scheduling.
type Config struct {
	WorkerID      string
	LeaseDuration time.Duration
	PollInterval  time.Duration
	RetryDelay    time.Duration
}

// Worker claims tasks from a queue and runs registered handlers.
type Worker struct {
	queue    Queue
	config   Config
	handlers map[string]Handler
}

// New creates a Worker with validated configuration.
func New(queue Queue, config Config) (*Worker, error) {
	if queue == nil {
		return nil, ErrQueueRequired
	}

	if config.WorkerID == "" {
		return nil, ErrWorkerIDRequired
	}

	if config.LeaseDuration <= 0 {
		return nil, ErrLeaseDurationInvalid
	}

	if config.PollInterval <= 0 {
		return nil, ErrPollIntervalInvalid
	}

	if config.RetryDelay < 0 {
		return nil, ErrRetryDelayInvalid
	}

	return &Worker{
		queue:    queue,
		config:   config,
		handlers: make(map[string]Handler),
	}, nil
}

// RegisterHandler registers a task handler by task type.
func (w *Worker) RegisterHandler(taskType string, handler Handler) error {
	if taskType == "" {
		return ErrTaskTypeRequired
	}

	if handler == nil {
		return ErrTaskHandlerRequired
	}

	if _, ok := w.handlers[taskType]; ok {
		return ErrHandlerAlreadyExists
	}

	w.handlers[taskType] = handler
	return nil
}

// ProcessOne claims and processes at most one task.
func (w *Worker) ProcessOne(ctx context.Context) (bool, error) {
	t, err := w.queue.ClaimNextTask(ctx, w.config.WorkerID, w.config.LeaseDuration)
	if err != nil {
		if errors.Is(err, storage.ErrNoTaskAvailable) {
			return false, nil
		}
		return false, err
	}

	handler, ok := w.handlers[t.Type]
	if !ok {
		message := fmt.Sprintf("%s: %s", ErrHandlerMissingForTask, t.Type)
		if err := w.queue.FailTask(ctx, t.ID, message, w.config.RetryDelay); err != nil {
			return true, err
		}
		return true, nil
	}

	if err := handler(ctx, t); err != nil {
		if failErr := w.queue.FailTask(ctx, t.ID, err.Error(), w.config.RetryDelay); failErr != nil {
			return true, failErr
		}
		return true, nil
	}

	if err := w.queue.CompleteTask(ctx, t.ID); err != nil {
		return true, err
	}

	return true, nil
}

// Run processes tasks until the context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		processed, err := w.ProcessOne(ctx)
		if err != nil {
			return err
		}

		if processed {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
