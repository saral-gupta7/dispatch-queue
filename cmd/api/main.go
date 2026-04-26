package main

import (
	"context"
	"log"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/config"
	"github.com/saral-gupta7/dispatch-queue/internal/storage"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)

	}

	store, err := storage.NewPostgresStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}

	defer store.Close()
	log.Println("api connected to postgres")
}
