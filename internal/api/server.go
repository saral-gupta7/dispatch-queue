package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/queue"
	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

// Queue is the queue behavior the API needs.
type Queue interface {
	Enqueue(ctx context.Context, t task.Task) (task.Task, error)
	GetTask(ctx context.Context, id string) (task.Task, error)
}

type server struct {
	queue Queue
}

type createTaskRequest struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	MaxAttempts int             `json:"max_attempts"`
	RunAt       *time.Time      `json:"run_at"`
}

type taskResponse struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      task.Status     `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	LastError   *string         `json:"last_error,omitempty"`
	RunAt       time.Time       `json:"run_at"`
	LockedBy    *string         `json:"locked_by,omitempty"`
	LockedUntil *time.Time      `json:"locked_until,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// NewHandler creates the HTTP API handler.
func NewHandler(queue Queue) http.Handler {
	s := &server{queue: queue}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/tasks/", s.handleTaskByID)

	return mux
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req createTaskRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	id := req.ID
	if id == "" {
		generatedID, err := generateID()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "generate task id")
			return
		}
		id = generatedID
	}

	payload := req.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}

	t := task.Task{
		ID:          id,
		Type:        req.Type,
		Payload:     payload,
		MaxAttempts: req.MaxAttempts,
	}

	if req.RunAt != nil {
		t.RunAt = req.RunAt.UTC()
	}

	created, err := s.queue.Enqueue(r.Context(), t)
	if err != nil {
		switch {
		case errors.Is(err, queue.ErrTaskIDRequired), errors.Is(err, queue.ErrTaskTypeRequired):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "enqueue task")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toTaskResponse(created))
}

func (s *server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/tasks/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	t, err := s.queue.GetTask(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, queue.ErrTaskIDRequired):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "get task")
		}
		return
	}

	writeJSON(w, http.StatusOK, toTaskResponse(t))
}

func toTaskResponse(t task.Task) taskResponse {
	return taskResponse{
		ID:          t.ID,
		Type:        t.Type,
		Payload:     t.Payload,
		Status:      t.Status,
		Attempts:    t.Attempts,
		MaxAttempts: t.MaxAttempts,
		LastError:   t.LastError,
		RunAt:       t.RunAt,
		LockedBy:    t.LockedBy,
		LockedUntil: t.LockedUntil,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func generateID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
