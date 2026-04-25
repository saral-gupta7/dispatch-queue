package config

import (
	"errors"
	"testing"
)

func TestLoadUsesDefaultDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != defaultDatabaseURL {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, defaultDatabaseURL)
	}
}

func TestLoadUsesDatabaseURLFromEnvironment(t *testing.T) {
	want := "postgres://user:pass@db:5432/app?sslmode=disable"

	t.Setenv("DATABASE_URL", want)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DatabaseURL != want {
		t.Fatalf("DatabaseURL = %q, want %q", cfg.DatabaseURL, want)
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
