package main

import (
	"context"
	"control-plane/internal/config"
	"control-plane/internal/db"
	"control-plane/internal/executor/docker"
	"control-plane/internal/queue/channel"
	"control-plane/internal/repository/postgres"
	"control-plane/internal/server"
	"control-plane/internal/service"
	"control-plane/internal/worker"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

	appRepo := postgres.NewApplicationRepository(pool)
	depRepo := postgres.NewDeploymentRepository(pool)
	userRepo := postgres.NewUserRepository(pool)

	// In-memory queue for Phase 4. Swap for RabbitMQ in Phase 5.
	q := channel.New(100)

	// Deployment workers: consume queue and talk to Docker.
	// numWorkers can be increased; for true distribution, run workers as separate processes.
	deploymentWorker := worker.New(q, appRepo, depRepo, exec, 3)
	if err := deploymentWorker.Start(ctx); err != nil {
		log.Fatalf("failed to start deployment worker: %v", err)
	}

	appSvc := service.NewApplicationService(appRepo)
	depSvc := service.NewDeploymentService(appRepo, depRepo, q, exec)
	userSvc := service.NewUserService(userRepo)

	srv := server.New(cfg, &server.Dependencies{
		AppService:  appSvc,
		DepService:  depSvc,
		UserService: userSvc,
	})

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	slog.Info("server started", "addr", cfg.ServerAddr, "docker", "connected", "workers", 3)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// Signal workers to stop accepting new jobs.
	cancel()

	// Close the queue so blocked consumers wake up.
	_ = q.Close()

	// Graceful HTTP shutdown.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	// Wait for in-flight deployments to finish.
	deploymentWorker.Stop()

	slog.Info("server exited")
}
