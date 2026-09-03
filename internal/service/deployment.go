package service

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"control-plane/internal/domain"
	"control-plane/internal/executor"

	"github.com/google/uuid"
)

type deploymentService struct {
	appRepo      domain.ApplicationRepository
	depRepo      domain.DeploymentRepository
	executor     executor.Executor
	pollInterval time.Duration
	appLocks     map[uuid.UUID]*sync.Mutex
	locksMu      sync.Mutex
}

func NewDeploymentService(appRepo domain.ApplicationRepository, depRepo domain.DeploymentRepository, exec executor.Executor) domain.DeploymentService {
	return &deploymentService{
		appRepo:      appRepo,
		depRepo:      depRepo,
		executor:     exec,
		pollInterval: 2 * time.Second,
		appLocks:     make(map[uuid.UUID]*sync.Mutex),
	}
}

func NewDeploymentServiceWithPoll(appRepo domain.ApplicationRepository, depRepo domain.DeploymentRepository, exec executor.Executor, poll time.Duration) domain.DeploymentService {
	return &deploymentService{
		appRepo:      appRepo,
		depRepo:      depRepo,
		executor:     exec,
		pollInterval: poll,
		appLocks:     make(map[uuid.UUID]*sync.Mutex),
	}
}

func (s *deploymentService) getAppLock(appID uuid.UUID) *sync.Mutex {
	s.locksMu.Lock()
	defer s.locksMu.Unlock()
	if s.appLocks[appID] == nil {
		s.appLocks[appID] = &sync.Mutex{}
	}
	return s.appLocks[appID]
}

func (s *deploymentService) Deploy(ctx context.Context, appID uuid.UUID, version string) (*domain.Deployment, error) {
	if version == "" {
		return nil, domain.ErrInvalidInput
	}

	if _, err := s.appRepo.GetByID(ctx, appID); err != nil {
		return nil, err
	}

	dep := &domain.Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		Version:       version,
		Status:        "pending",
		Message:       "Queued for deployment",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.depRepo.Create(ctx, dep); err != nil {
		return nil, err
	}

	slog.Info("deployment queued", "dep_id", dep.ID, "app_id", appID, "version", version)
	go s.runDeployment(dep.ID, appID, version)

	return dep, nil
}

func (s *deploymentService) runDeployment(depID uuid.UUID, appID uuid.UUID, version string) {
	// PANIC RECOVERY — log and mark failed
	defer func() {
		if r := recover(); r != nil {
			slog.Error("deployment goroutine panicked", "dep_id", depID, "panic", r, "stack", string(debug.Stack()))
			ctx := context.Background()
			_ = s.depRepo.UpdateStatus(ctx, depID, "failed", "Internal error: deployment goroutine panicked")
		}
	}()

	lock := s.getAppLock(appID)
	lock.Lock()
	defer lock.Unlock()

	ctx := context.Background()

	// Am I still the latest deployment for this app?
	deps, err := s.depRepo.ListByApplication(ctx, appID)
	if err != nil {
		slog.Error("failed to list deployments for stale check", "dep_id", depID, "error", err)
	}

	var latestID uuid.UUID
	var latestTime time.Time
	for _, d := range deps {
		if d.CreatedAt.After(latestTime) || latestTime.IsZero() {
			latestTime = d.CreatedAt
			latestID = d.ID
		}
	}
	if latestID != depID && latestID != uuid.Nil {
		slog.Info("cancelling stale deployment", "dep_id", depID, "latest", latestID)
		if err := s.depRepo.UpdateStatus(ctx, depID, "cancelled", "Superseded by newer deployment"); err != nil {
			slog.Error("failed to mark deployment cancelled", "dep_id", depID, "error", err)
		}
		return
	}

	// Stop any previous containers for this app
	for _, d := range deps {
		if d.ID != depID && d.ContainerID != "" {
			slog.Info("stopping old container", "dep_id", d.ID, "container", d.ContainerID)
			if err := s.executor.Stop(ctx, d.ContainerID); err != nil {
				slog.Warn("failed to stop old container", "container", d.ContainerID, "error", err)
			}
			if err := s.depRepo.UpdateStatus(ctx, d.ID, "stopped", "Replaced by newer deployment"); err != nil {
				slog.Error("failed to mark old deployment stopped", "dep_id", d.ID, "error", err)
			}
		}
	}

	// Get app image
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		slog.Error("app not found for deployment", "dep_id", depID, "error", err)
		if updErr := s.depRepo.UpdateStatus(ctx, depID, "failed", "application not found: "+err.Error()); updErr != nil {
			slog.Error("failed to mark deployment failed", "dep_id", depID, "error", updErr)
		}
		return
	}

	image := app.DockerImage
	if image == "" {
		image = "nginx"
	}

	slog.Info("starting docker deploy", "dep_id", depID, "image", image, "version", version)
	if err := s.depRepo.UpdateStatus(ctx, depID, "running", "Pulling image "+image+":"+version); err != nil {
		slog.Error("failed to mark deployment running", "dep_id", depID, "error", err)
	}

	containerID, err := s.executor.Deploy(ctx, appID.String(), depID.String(), image, version)
	if err != nil {
		slog.Error("docker deploy failed", "dep_id", depID, "error", err)
		if updErr := s.depRepo.UpdateStatus(ctx, depID, "failed", "container error: "+err.Error()); updErr != nil {
			slog.Error("failed to mark deployment failed", "dep_id", depID, "error", updErr)
		}
		return
	}

	slog.Info("container started", "dep_id", depID, "container_id", containerID)
	if err := s.depRepo.UpdateContainerID(ctx, depID, containerID); err != nil {
		slog.Error("failed to save container_id", "error", err)
	}

	// Poll container status — simple sleep loop with deadline (more reliable than ticker)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		statusCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		status, err := s.executor.Status(statusCtx, containerID)
		cancel()

		if err != nil {
			slog.Error("status check failed", "dep_id", depID, "error", err)
			if updErr := s.depRepo.UpdateStatus(ctx, depID, "failed", "status check error: "+err.Error()); updErr != nil {
				slog.Error("failed to mark deployment failed", "dep_id", depID, "error", updErr)
			}
			return
		}

		slog.Info("container status check", "dep_id", depID, "status", status)

		switch status {
		case "successful":
			if err := s.depRepo.UpdateStatus(ctx, depID, "successful", "Container exited successfully"); err != nil {
				slog.Error("failed to mark deployment successful", "dep_id", depID, "error", err)
			} else {
				slog.Info("deployment successful", "dep_id", depID)
			}
			return
		case "failed":
			if err := s.depRepo.UpdateStatus(ctx, depID, "failed", "Container exited with error"); err != nil {
				slog.Error("failed to mark deployment failed", "dep_id", depID, "error", err)
			} else {
				slog.Info("deployment failed", "dep_id", depID)
			}
			return
		case "running":
			if err := s.depRepo.UpdateStatus(ctx, depID, "successful", "Container running successfully"); err != nil {
				slog.Error("failed to mark deployment successful", "dep_id", depID, "error", err)
			} else {
				slog.Info("deployment successful (daemon)", "dep_id", depID)
			}
			return
		default:
			slog.Info("container in intermediate state, continuing poll", "dep_id", depID, "status", status)
		}

		time.Sleep(s.pollInterval)
	}

	// Deadline exceeded
	slog.Error("deployment timed out", "dep_id", depID)
	if err := s.depRepo.UpdateStatus(ctx, depID, "failed", "Deployment timed out after 30s"); err != nil {
		slog.Error("failed to mark deployment timed out", "dep_id", depID, "error", err)
	}
}

func (s *deploymentService) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	return s.depRepo.ListByApplication(ctx, appID)
}

func (s *deploymentService) Rollback(ctx context.Context, appID uuid.UUID) (*domain.Deployment, error) {
	if _, err := s.appRepo.GetByID(ctx, appID); err != nil {
		return nil, err
	}

	last, err := s.depRepo.GetLastSuccessful(ctx, appID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrRollback
		}
		return nil, err
	}

	deps, _ := s.depRepo.ListByApplication(ctx, appID)
	for _, d := range deps {
		if d.ContainerID != "" && d.Status != "stopped" && d.Status != "failed" && d.Status != "cancelled" {
			if stopErr := s.executor.Stop(ctx, d.ContainerID); stopErr != nil {
				slog.Warn("failed to stop container during rollback", "container", d.ContainerID, "error", stopErr)
			}
			if updErr := s.depRepo.UpdateStatus(ctx, d.ID, "stopped", "Rolled back"); updErr != nil {
				slog.Error("failed to mark deployment stopped during rollback", "dep_id", d.ID, "error", updErr)
			}
		}
	}

	dep := &domain.Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		Version:       last.Version,
		Status:        "pending",
		Message:       "Rollback to version " + last.Version,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.depRepo.Create(ctx, dep); err != nil {
		return nil, err
	}

	go s.runDeployment(dep.ID, appID, last.Version)

	return dep, nil
}

func (s *deploymentService) GetLogs(ctx context.Context, containerID string, tail int) (string, error) {
	if containerID == "" {
		return "", domain.ErrInvalidInput
	}
	return s.executor.Logs(ctx, containerID, tail)
}
