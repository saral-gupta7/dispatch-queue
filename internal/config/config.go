package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	DatabaseURL string
}

var (
	ErrInvalidDatabaseURLScheme = fmt.Errorf("database url scheme must be postgres or postgresql")
	ErrDatabaseURLHostRequired  = fmt.Errorf("database url host is required")
)

const defaultDatabaseURL = "postgres://dispatch:dispatch@localhost:5432/dispatch_queue?sslmode=disable"

func Load() (Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	if err := validateDatabaseURL(databaseURL); err != nil {
		return Config{}, err
	}

	return Config{DatabaseURL: databaseURL}, nil
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
