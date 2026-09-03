package memory

import (
	"context"
	"sync"
	"time"

	"control-plane/internal/domain"

	"github.com/google/uuid"
)

type DeploymentRepository struct {
	mu   sync.RWMutex
	data map[uuid.UUID]domain.Deployment
}

func NewDeploymentRepository() *DeploymentRepository {
	return &DeploymentRepository{
		data: make(map[uuid.UUID]domain.Deployment),
	}
}

func (r *DeploymentRepository) Create(ctx context.Context, dep *domain.Deployment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dep.CreatedAt = time.Now()
	dep.UpdatedAt = dep.CreatedAt
	r.data[dep.ID] = *dep
	return nil
}

func (r *DeploymentRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dep, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &dep, nil
}

func (r *DeploymentRepository) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var deps []domain.Deployment
	for _, d := range r.data {
		if d.ApplicationID == appID {
			deps = append(deps, d)
		}
	}
	return deps, nil
}

func (r *DeploymentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dep, ok := r.data[id]
	if !ok {
		return domain.ErrNotFound
	}
	dep.Status = status
	dep.Message = message
	dep.UpdatedAt = time.Now()
	if status == "successful" || status == "failed" || status == "rolled_back" {
		now := time.Now()
		dep.CompletedAt = &now
	}
	r.data[id] = dep
	return nil
}

func (r *DeploymentRepository) GetLastSuccessful(ctx context.Context, appID uuid.UUID) (*domain.Deployment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var latest *domain.Deployment
	for _, d := range r.data {
		if d.ApplicationID == appID && d.Status == "successful" {
			if latest == nil || d.CreatedAt.After(latest.CreatedAt) {
				latest = &d
			}
		}
	}
	if latest == nil {
		return nil, domain.ErrNotFound
	}
	// Return a copy
	d := *latest
	return &d, nil
}

func (r *DeploymentRepository) UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	dep, ok := r.data[id]
	if !ok {
		return domain.ErrNotFound
	}
	dep.ContainerID = containerID
	dep.UpdatedAt = time.Now()
	r.data[id] = dep
	return nil
}
