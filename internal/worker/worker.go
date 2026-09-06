package worker

import (
	"context"
	"control-plane/internal/domain"
	"control-plane/internal/executor"
	"control-plane/internal/lock"
	"control-plane/internal/queue"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DeploymentWorker consumes deployment jobs from a queue and executes them
// via the Docker executor. Application-level distributed locking prevents
// concurrent deploys/rollbacks for the same application across multiple
// worker processes.
type DeploymentWorker struct {
	queue         queue.Queue
	appRepo       domain.ApplicationRepository
	depRepo       domain.DeploymentRepository
	executor      executor.Executor
	locker        lock.Locker
	pollIntervals time.Duration
	numWorkers    int
	wg            sync.WaitGroup
}

// New creates a DeploymentWorker with the given dependencies.
func New(q queue.Queue, appRepo domain.ApplicationRepository, depRepo domain.DeploymentRepository, executor executor.Executor, locker lock.Locker, numWorkers int) *DeploymentWorker {
	return &DeploymentWorker{
		queue:         q,
		appRepo:       appRepo,
		depRepo:       depRepo,
		executor:      executor,
		locker:        locker,
		pollIntervals: 2 * time.Second,
		numWorkers:    numWorkers,
	}
}

// NewWithPoll is a test helper that overrides the status-poll interval.
func NewWithPoll(q queue.Queue, appRepo domain.ApplicationRepository, depRepo domain.DeploymentRepository, executor executor.Executor, locker lock.Locker, numWorkers int, pollIntervals time.Duration) *DeploymentWorker {
	w := New(q, appRepo, depRepo, executor, locker, numWorkers)
	w.pollIntervals = pollIntervals
	return w
}

func (w *DeploymentWorker) Start(ctx context.Context) error {
	jobs, err := w.queue.Consume(ctx)
	if err != nil {
		return fmt.Errorf("failed to consume from queue: %w", err)
	}
	slog.Info("deployment worker started", "workers", w.numWorkers)

	for i := 0; i < w.numWorkers; i++ {
		w.wg.Add(1)
		go w.loop(ctx, jobs, i)
	}
	return nil
}

func (w *DeploymentWorker) Stop() {
	slog.Info("stopping deployment worker...")
	w.wg.Wait()
	slog.Info("deployment worker stopped")
}

func (w *DeploymentWorker) loop(ctx context.Context, jobs <-chan queue.DeploymentJob, workerID int) {
	defer w.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			w.processJob(ctx, job, workerID)
		}
	}
}

func (w *DeploymentWorker) processJob(ctx context.Context, job queue.DeploymentJob, workerID int) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("deployment worker panicked", "worker", workerID, "panic", r, "stack", string(debug.Stack()))
			if depID, err := uuid.Parse(job.DeploymentID); err == nil {
				_ = w.depRepo.UpdateStatus(context.Background(), depID, "failed", "Internal error: worker panicked")
			}
		}
	}()

	depID, err := uuid.Parse(job.DeploymentID)
	if err != nil {
		slog.Error("invalid deployment id in a job", "deployment_id", job.DeploymentID)
		return
	}

	appID, err := uuid.Parse(job.ApplicationID)
	if err != nil {
		slog.Error("invalid application id in a job", "application_id", job.ApplicationID)
		return
	}

	workCtx := context.Background()

	// Distributed lock per application.
	unlock, err := w.locker.Lock(workCtx, appID.String())
	if err != nil {
		slog.Error("failed to acquire app lock", "app_id", appID, "error", err)
		_ = w.depRepo.UpdateStatus(workCtx, depID, "failed", "Failed to acquire lock: "+err.Error())
		return
	}
	defer unlock()

	deps, err := w.depRepo.ListByApplication(workCtx, appID)
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
		if err := w.depRepo.UpdateStatus(workCtx, depID, "cancelled", "Superseded by newer deployment"); err != nil {
			slog.Error("failed to mark deployment cancelled", "dep_id", depID, "error", err)
		}
		return
	}

	for _, d := range deps {
		if d.ID != depID && d.ContainerID != "" {
			slog.Info("stopping old container", "dep_id", d.ID, "container", d.ContainerID)
			if err := w.executor.Stop(workCtx, d.ContainerID); err != nil {
				slog.Warn("failed to stop old container", "container", d.ContainerID, "error", err)
			}
			if err := w.depRepo.UpdateStatus(workCtx, d.ID, "stopped", "replaced by newer deployment"); err != nil {
				slog.Error("failed to mark old deployment stopped", "dep_id", d.ID, "error", err)
			}
		}
	}

	image := job.Image
	if image == "" {
		app, err := w.appRepo.GetByID(workCtx, appID)
		if err != nil {
			slog.Error("failed to get application for deployment", "dep_id", depID, "app_id", appID, "error", err)
			if updErr := w.depRepo.UpdateStatus(workCtx, depID, "failed", "Failed to get application for deployment"); updErr != nil {
				slog.Error("failed to mark deployment failed", "dep_id", depID, "error", updErr)
			}
			return
		}
		image = app.DockerImage
		if image == "" {
			image = "nginx"
		}
	}

	slog.Info("starting docker deploy", "worker", workerID, "dep_id", depID, "image", image, "version", job.Version)

	if err := w.depRepo.UpdateStatus(workCtx, depID, "running", "Pulling image "+image+":"+job.Version); err != nil {
		slog.Error("failed to mark deployment running", "dep_id", depID, "error", err)
	}

	containerID, err := w.executor.Deploy(workCtx, appID.String(), depID.String(), image, job.Version)
	if err != nil {
		slog.Error("docker deploy failed", "dep_id", depID, "error", err)
		if updErr := w.depRepo.UpdateStatus(workCtx, depID, "failed", "container error: "+err.Error()); updErr != nil {
			slog.Error("failed to mark deployment failed", "dep_id", depID, "error", updErr)
		}
		return
	}

	slog.Info("container started", "dep_id", depID, "container_id", containerID)
	if err := w.depRepo.UpdateContainerID(workCtx, depID, containerID); err != nil {
		slog.Error("failed to save container_id", "dep_id", depID, "error", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		statusCtx, cancel := context.WithTimeout(workCtx, 5*time.Second)
		status, err := w.executor.Status(statusCtx, containerID)
		cancel()

		if err != nil {
			slog.Error("status check failed", "dep_id", depID, "error", err)
			if updErr := w.depRepo.UpdateStatus(workCtx, depID, "failed", "status check error: "+err.Error()); updErr != nil {
				slog.Error("failed to mark deployment failed", "dep_id", depID, "error", updErr)
			}
			return
		}
		slog.Info("container status check", "dep_id", depID, "container_id", containerID, "status", status)

		switch status {
		case "successful":
			if err := w.depRepo.UpdateStatus(workCtx, depID, "successful", "Deployment successful"); err != nil {
				slog.Error("failed to mark deployment successful", "dep_id", depID, "error", err)
			} else {
				slog.Info("deployment successful", "dep_id", depID)
			}
			return
		case "failed":
			if err := w.depRepo.UpdateStatus(workCtx, depID, "failed", "Deployment failed"); err != nil {
				slog.Error("failed to mark deployment failed", "dep_id", depID, "error", err)
			} else {
				slog.Info("deployment failed", "dep_id", depID)
			}
			return
		case "running":
			if err := w.depRepo.UpdateStatus(workCtx, depID, "running", "Deployment running"); err != nil {
				slog.Error("failed to mark deployment running", "dep_id", depID, "error", err)
			} else {
				slog.Info("deployment running", "dep_id", depID)
			}
			return
		default:
			slog.Info("container status check", "dep_id", depID, "status", status)
		}
		time.Sleep(w.pollIntervals)
	}

	slog.Error("deployment timed out", "dep_id", depID)
	if err := w.depRepo.UpdateStatus(workCtx, depID, "failed", "Deployment timed out after 30s"); err != nil {
		slog.Error("failed to mark deployment failed", "dep_id", depID, "error", err)
	}
}
