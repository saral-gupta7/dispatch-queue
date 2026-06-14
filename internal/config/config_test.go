package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("API_ADDR", "")
	t.Setenv("WORKER_ID", "")
	t.Setenv("WORKER_POLL_INTERVAL", "")
	t.Setenv("WORKER_LEASE_DURATION", "")
	t.Setenv("WORKER_RETRY_DELAY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}

	if cfg.APIAddr != defaultAPIAddr {
		t.Fatalf("APIAddr = %q, want %q", cfg.APIAddr, defaultAPIAddr)
	}

	if cfg.WorkerID != defaultWorkerID {
		t.Fatalf("WorkerID = %q, want %q", cfg.WorkerID, defaultWorkerID)
	}

	if cfg.WorkerPollInterval != defaultWorkerPollInterval {
		t.Fatalf("WorkerPollInterval = %v, want %v", cfg.WorkerPollInterval, defaultWorkerPollInterval)
	}

	if cfg.WorkerLeaseDuration != defaultWorkerLeaseDuration {
		t.Fatalf("WorkerLeaseDuration = %v, want %v", cfg.WorkerLeaseDuration, defaultWorkerLeaseDuration)
	}

	if cfg.WorkerRetryDelay != defaultWorkerRetryDelay {
		t.Fatalf("WorkerRetryDelay = %v, want %v", cfg.WorkerRetryDelay, defaultWorkerRetryDelay)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	wantDatabaseURL := "postgres://user:pass@db:5432/app?sslmode=disable"

	t.Setenv("DATABASE_URL", wantDatabaseURL)
	t.Setenv("API_ADDR", ":9090")
	t.Setenv("WORKER_ID", "worker-test")
	t.Setenv("WORKER_POLL_INTERVAL", "250ms")
	t.Setenv("WORKER_LEASE_DURATION", "45s")
	t.Setenv("WORKER_RETRY_DELAY", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != wantDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, wantDatabaseURL)
	}

	if cfg.APIAddr != ":9090" {
		t.Fatalf("APIAddr = %q, want %q", cfg.APIAddr, ":9090")
	}

	if cfg.WorkerID != "worker-test" {
		t.Fatalf("WorkerID = %q, want %q", cfg.WorkerID, "worker-test")
	}

	if cfg.WorkerPollInterval != 250*time.Millisecond {
		t.Fatalf("WorkerPollInterval = %v, want %v", cfg.WorkerPollInterval, 250*time.Millisecond)
	}

	if cfg.WorkerLeaseDuration != 45*time.Second {
		t.Fatalf("WorkerLeaseDuration = %v, want %v", cfg.WorkerLeaseDuration, 45*time.Second)
	}

	if cfg.WorkerRetryDelay != 2*time.Second {
		t.Fatalf("WorkerRetryDelay = %v, want %v", cfg.WorkerRetryDelay, 2*time.Second)
	}
}

func TestLoadRejectsInvalidDatabaseURLScheme(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://user:pass@localhost:5432/app")

	_, err := Load()
	if !errors.Is(err, ErrInvalidDatabaseURLScheme) {
		t.Fatalf("Load() error = %v, want %v", err, ErrInvalidDatabaseURLScheme)
	}
}

func TestLoadRejectsDatabaseURLWithoutHost(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres:///dispatch_queue")

	_, err := Load()
	if !errors.Is(err, ErrDatabaseURLHostRequired) {
		t.Fatalf("Load() error = %v, want %v", err, ErrDatabaseURLHostRequired)
	}
}

func TestLoadRejectsInvalidWorkerDuration(t *testing.T) {
	t.Setenv("WORKER_POLL_INTERVAL", "soon")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
