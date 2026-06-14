package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/saral-gupta7/dispatch-queue/internal/config"
	"github.com/saral-gupta7/dispatch-queue/internal/queue"
	"github.com/saral-gupta7/dispatch-queue/internal/storage"
	"github.com/saral-gupta7/dispatch-queue/internal/task"
	"github.com/saral-gupta7/dispatch-queue/internal/worker"
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

	w, err := worker.New(svc, worker.Config{
		WorkerID:      cfg.WorkerID,
		LeaseDuration: cfg.WorkerLeaseDuration,
		PollInterval:  cfg.WorkerPollInterval,
		RetryDelay:    cfg.WorkerRetryDelay,
	})
	if err != nil {
		log.Fatalf("create worker: %v", err)
	}

	if err := w.RegisterHandler("send_email", logTaskHandler); err != nil {
		log.Fatalf("register handler: %v", err)
	}

	log.Printf("worker %s started", cfg.WorkerID)

	if err := w.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("run worker: %v", err)
	}

	log.Printf("worker %s stopped", cfg.WorkerID)
}

func logTaskHandler(ctx context.Context, t task.Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	log.Printf("handled task id=%s type=%s payload=%s", t.ID, t.Type, string(t.Payload))
	return nil
}
