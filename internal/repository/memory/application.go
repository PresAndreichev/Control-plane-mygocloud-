package memory

import (
	"context"
	"control-plane/internal/domain"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ApplicationRepository struct {
	mu   sync.RWMutex
	data map[uuid.UUID]domain.Application
}

func NewApplicationRepository() *ApplicationRepository {
	return &ApplicationRepository{
		data: make(map[uuid.UUID]domain.Application),
	}
}

func (r *ApplicationRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var apps []domain.Application
	for _, a := range r.data {
		if a.OwnerID == ownerID {
			apps = append(apps, a)
		}
	}
	return apps, nil
}

func (r *ApplicationRepository) Create(ctx context.Context, app *domain.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	app.CreatedAt = time.Now()
	app.UpdatedAt = app.CreatedAt
	r.data[app.ID] = *app
	return nil
}

func (r *ApplicationRepository) List(ctx context.Context) ([]domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var apps []domain.Application
	for _, a := range r.data {
		apps = append(apps, a)
	}
	return apps, nil
}

func (r *ApplicationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	app, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &app, nil

}
