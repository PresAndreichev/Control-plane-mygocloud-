package main

import (
	"context"
	"control-plane/internal/config"
	"control-plane/internal/db"
	"control-plane/internal/executor/docker"
	"control-plane/internal/lock/postgres"
	"control-plane/internal/queue/rabbitmq"
	repo "control-plane/internal/repository/postgres"
	"control-plane/internal/worker"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	exec, err := docker.New()
	if err != nil {
		log.Fatalf("failed to connect to docker: %v", err)
	}

	appRepo := repo.NewApplicationRepository(pool)
	depRepo := repo.NewDeploymentRepository(pool)

	q, err := rabbitmq.New(cfg.RabbitMQ.URL, "deployment_jobs")
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer q.Close()

	// Distributed application-level locking via PostgreSQL advisory locks.
	locker := postgres.New(pool)

	deploymentWorker := worker.New(q, appRepo, depRepo, exec, locker, 3)
	if err := deploymentWorker.Start(ctx); err != nil {
		log.Fatalf("failed to start deployment worker: %v", err)
	}

	slog.Info("worker started", "rabbitmq", cfg.RabbitMQ.URL, "docker", "connected", "workers", 3)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down worker...")

	cancel()
	deploymentWorker.Stop()

	slog.Info("worker exited")
}
