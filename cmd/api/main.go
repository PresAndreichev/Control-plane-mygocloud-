package main

import (
	"context"
	"control-plane/internal/config"
	"control-plane/internal/db"
	"control-plane/internal/executor/noop"
	"control-plane/internal/queue/rabbitmq"
	"control-plane/internal/repository/postgres"
	"control-plane/internal/server"
	"control-plane/internal/service"
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

	appRepo := postgres.NewApplicationRepository(pool)
	depRepo := postgres.NewDeploymentRepository(pool)
	userRepo := postgres.NewUserRepository(pool)

	// Phase 5: API publishes to RabbitMQ; workers consume and talk to Docker.
	q, err := rabbitmq.New(cfg.RabbitMQ.URL, "deployment_jobs")
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer q.Close()

	// API no longer needs Docker directly; use a noop executor for the interface.
	exec := noop.New()

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

	slog.Info("server started", "addr", cfg.ServerAddr, "queue", "rabbitmq")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited")
}
