package config

import (
	"fmt"
	"net/url"
	"os"
	"time"
)

type Config struct {
	DatabaseURL         string
	APIAddr             string
	WorkerID            string
	WorkerPollInterval  time.Duration
	WorkerLeaseDuration time.Duration
	WorkerRetryDelay    time.Duration
}

var (
	ErrInvalidDatabaseURLScheme = fmt.Errorf("database url scheme must be postgres or postgresql")
	ErrDatabaseURLHostRequired  = fmt.Errorf("database url host is required")
	ErrWorkerIDRequired         = fmt.Errorf("worker id is required")
)

const (
	defaultDatabaseURL         = "postgres://dispatch:dispatch@localhost:5432/dispatch_queue?sslmode=disable"
	defaultAPIAddr             = ":8080"
	defaultWorkerID            = "worker-1"
	defaultWorkerPollInterval  = time.Second
	defaultWorkerLeaseDuration = 30 * time.Second
	defaultWorkerRetryDelay    = 5 * time.Second
)

func Load() (Config, error) {
	databaseURL := getEnv("DATABASE_URL", defaultDatabaseURL)

	if err := validateDatabaseURL(databaseURL); err != nil {
		return Config{}, err
	}

	workerID := getEnv("WORKER_ID", defaultWorkerID)
	if workerID == "" {
		return Config{}, ErrWorkerIDRequired
	}

	workerPollInterval, err := durationFromEnv("WORKER_POLL_INTERVAL", defaultWorkerPollInterval)
	if err != nil {
		return Config{}, err
	}

	workerLeaseDuration, err := durationFromEnv("WORKER_LEASE_DURATION", defaultWorkerLeaseDuration)
	if err != nil {
		return Config{}, err
	}

	workerRetryDelay, err := durationFromEnv("WORKER_RETRY_DELAY", defaultWorkerRetryDelay)
	if err != nil {
		return Config{}, err
	}

	return Config{
		DatabaseURL:         databaseURL,
		APIAddr:             getEnv("API_ADDR", defaultAPIAddr),
		WorkerID:            workerID,
		WorkerPollInterval:  workerPollInterval,
		WorkerLeaseDuration: workerLeaseDuration,
		WorkerRetryDelay:    workerRetryDelay,
	}, nil
}

func validateDatabaseURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)

	if err != nil {
		return fmt.Errorf("parse database url: %w", err)
	}

	if parsed.Host == "" {
		return ErrDatabaseURLHostRequired
	}

	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return ErrInvalidDatabaseURLScheme
	}

	return nil
}

func getEnv(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func durationFromEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	return duration, nil
}
