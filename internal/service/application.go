package service

import (
	"control-plane/internal/domain"

	"github.com/google/uuid"

	"context"

	"time"
)

type applicationService struct {
	repo domain.ApplicationRepository
}

func NewApplicationService(repo domain.ApplicationRepository) domain.ApplicationService {
	return &applicationService{repo: repo}
}

func (s *applicationService) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Application, error) {
	return s.repo.ListByOwner(ctx, ownerID)
}

func (s *applicationService) Create(ctx context.Context, name, description, dockerImage string, ownerID uuid.UUID) (*domain.Application, error) {
	if name == "" {
		return nil, domain.ErrInvalidInput
	}

	app := &domain.Application{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		DockerImage: dockerImage,
		OwnerID:     ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *applicationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *applicationService) List(ctx context.Context) ([]domain.Application, error) {
	return s.repo.List(ctx)
}
