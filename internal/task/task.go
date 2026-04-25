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

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusCompleted, StatusDead, StatusFailed, StatusRunning:
		return true

	default:
		return false
	}
}

func (t Task) IsTerminal() bool {
	return t.Status == StatusCompleted || t.Status == StatusDead
}

func (t Task) CanRetry() bool {
	return t.Attempts < t.MaxAttempts
}
