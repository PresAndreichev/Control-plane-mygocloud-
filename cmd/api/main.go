package main

import (
	"context"
	"control-plane/internal/config"
	"control-plane/internal/db"
	"control-plane/internal/executor/docker"
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

	ctx := context.Background()

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

	appSvc := service.NewApplicationService(appRepo)
	depSvc := service.NewDeploymentService(appRepo, depRepo, exec)
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

	slog.Info("server started", "addr", cfg.ServerAddr, "docker", "connected")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)

	}
	slog.Info("server exited")
}
