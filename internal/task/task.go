package task

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusDead      Status = "dead"
)

// locked by is a pointer so that it can be null as well for unclaimed tasks
type Task struct {
	ID          string
	Type        string
	Payload     json.RawMessage
	Status      Status
	Attempts    int
	MaxAttempts int
	LastError   *string
	RunAt       time.Time
	LockedBy    *string
	LockedUntil *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
