package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"control-plane/internal/domain"
	"control-plane/internal/executor"
	"control-plane/internal/queue"

	"github.com/google/uuid"
)

type deploymentService struct {
	appRepo  domain.ApplicationRepository
	depRepo  domain.DeploymentRepository
	queue    queue.Queue
	executor executor.Executor
}

// NewDeploymentService creates a deployment service that enqueues jobs
// for asynchronous execution by a DeploymentWorker.
func NewDeploymentService(appRepo domain.ApplicationRepository, depRepo domain.DeploymentRepository, q queue.Queue, exec executor.Executor) domain.DeploymentService {
	return &deploymentService{
		appRepo:  appRepo,
		depRepo:  depRepo,
		queue:    q,
		executor: exec,
	}
}

func (s *deploymentService) Deploy(ctx context.Context, appID uuid.UUID, version string) (*domain.Deployment, error) {
	if version == "" {
		return nil, domain.ErrInvalidInput
	}

	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
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

	job := queue.DeploymentJob{
		Type:          queue.JobTypeDeploy,
		DeploymentID:  dep.ID.String(),
		ApplicationID: appID.String(),
		Image:         app.DockerImage,
		Version:       version,
	}
	if err := s.queue.Publish(ctx, job); err != nil {
		slog.Error("failed to publish deployment job", "dep_id", dep.ID, "error", err)
		if updErr := s.depRepo.UpdateStatus(ctx, dep.ID, "failed", "Failed to queue deployment: "+err.Error()); updErr != nil {
			slog.Error("failed to mark deployment failed after queue error", "dep_id", dep.ID, "error", updErr)
		}
		return nil, fmt.Errorf("failed to queue deployment: %w", err)
	}

	slog.Info("deployment queued", "dep_id", dep.ID, "app_id", appID, "version", version)
	return dep, nil
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

	job := queue.DeploymentJob{
		Type:          queue.JobTypeRollback,
		DeploymentID:  dep.ID.String(),
		ApplicationID: appID.String(),
		Image:         "", // Worker will resolve from application record
		Version:       last.Version,
	}
	if err := s.queue.Publish(ctx, job); err != nil {
		slog.Error("failed to publish rollback job", "dep_id", dep.ID, "error", err)
		if updErr := s.depRepo.UpdateStatus(ctx, dep.ID, "failed", "Failed to queue rollback: "+err.Error()); updErr != nil {
			slog.Error("failed to mark rollback failed after queue error", "dep_id", dep.ID, "error", updErr)
		}
		return nil, fmt.Errorf("failed to queue rollback: %w", err)
	}

	slog.Info("rollback queued", "dep_id", dep.ID, "app_id", appID, "version", last.Version)
	return dep, nil
}

func (s *deploymentService) GetLogs(ctx context.Context, containerID string, tail int) (string, error) {
	if containerID == "" {
		return "", domain.ErrInvalidInput
	}
	return s.executor.Logs(ctx, containerID, tail)
}
