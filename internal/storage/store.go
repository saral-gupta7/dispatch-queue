package storage

import (
	"context"
	"errors"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

var (
	ErrTaskNotFound    = errors.New("task not found")
	ErrNoTaskAvailable = errors.New("no task available")
)

type Store interface {
	CreateTask(ctx context.Context, t task.Task) error
	GetTask(ctx context.Context, id string) (task.Task, error)
	ClaimNextTask(ctx context.Context, workerID string, leaseDuration time.Duration) (task.Task, error)
	CompleteTask(ctx context.Context, taskID string) error
}
