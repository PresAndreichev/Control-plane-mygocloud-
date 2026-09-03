package memory

import (
	"context"
	"control-plane/internal/domain"
	"sync"
	"time"

	"github.com/google/uuid"
)

type UserRepository struct {
	mu    sync.RWMutex
	data  map[uuid.UUID]domain.User
	email map[string]uuid.UUID
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		data:  make(map[uuid.UUID]domain.User),
		email: make(map[string]uuid.UUID),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	user.CreatedAt = time.Now()
	user.UpdatedAt = user.CreatedAt
	r.data[user.ID] = *user
	r.email[user.Email] = user.ID
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.email[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	u := r.data[id]
	return &u, nil
}
