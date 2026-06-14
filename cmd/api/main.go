package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/api"
	"github.com/saral-gupta7/dispatch-queue/internal/config"
	"github.com/saral-gupta7/dispatch-queue/internal/queue"
	"github.com/saral-gupta7/dispatch-queue/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	startupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	store, err := storage.NewPostgresStore(startupCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}

	defer store.Close()

	svc := queue.NewService(store)

	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           api.NewHandler(svc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown api: %v", err)
		}
	}()

	log.Printf("api listening on %s", cfg.APIAddr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve api: %v", err)
	}

	log.Println("api stopped")
}
