package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saral-gupta7/dispatch-queue/internal/queue"
	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
)

func TestHealthReturnsOK(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestCreateTaskEnqueuesTask(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString(`{"id":"task-1","type":"send_email","payload":{"email":"user@example.com"}}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got taskResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != "task-1" {
		t.Fatalf("ID = %q, want %q", got.ID, "task-1")
	}

	if got.Type != "send_email" {
		t.Fatalf("Type = %q, want %q", got.Type, "send_email")
	}

	if got.Status != task.StatusPending {
		t.Fatalf("Status = %q, want %q", got.Status, task.StatusPending)
	}
}

func TestCreateTaskGeneratesID(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString(`{"type":"send_email","payload":{"email":"user@example.com"}}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var got taskResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID == "" {
		t.Fatal("ID was empty")
	}
}

func TestCreateTaskRejectsMissingType(t *testing.T) {
	handler := newTestHandler()

	body := bytes.NewBufferString(`{"id":"task-1","payload":{"email":"user@example.com"}}`)
	req := httptest.NewRequest(http.MethodPost, "/tasks", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGetTaskReturnsTask(t *testing.T) {
	store := storage.NewMemoryStore()
	svc := queue.NewService(store)
	handler := NewHandler(svc)

	_, err := svc.Enqueue(httptest.NewRequest(http.MethodPost, "/", nil).Context(), task.Task{
		ID:   "task-1",
		Type: "send_email",
	})
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/tasks/task-1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var got taskResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.ID != "task-1" {
		t.Fatalf("ID = %q, want %q", got.ID, "task-1")
	}
}

func TestGetTaskReturnsNotFound(t *testing.T) {
	handler := newTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/tasks/missing", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func newTestHandler() http.Handler {
	store := storage.NewMemoryStore()
	svc := queue.NewService(store)
	return NewHandler(svc)
}
