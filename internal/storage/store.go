package storage

import (
	"context"

	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

type Store interface {
	CreateTask(ctx context.Context, t task.Task) error
	GetTask(ctx context.Context, id string) (task.Task, error)
}
