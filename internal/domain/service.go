package domain

import (
	"context"

	"github.com/google/uuid"
)

type ApplicationService interface {
	Create(ctx context.Context, name, description, dockerImage string, ownerID uuid.UUID) (*Application, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Application, error)
	List(ctx context.Context) ([]Application, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]Application, error)
}

type DeploymentService interface {
	Deploy(ctx context.Context, appID uuid.UUID, version string) (*Deployment, error)
	ListByApplication(ctx context.Context, appID uuid.UUID) ([]Deployment, error)
	Rollback(ctx context.Context, appID uuid.UUID) (*Deployment, error)
	GetLogs(ctx context.Context, containerID string, tail int) (string, error)
}

type UserService interface {
	Create(ctx context.Context, email, name string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
}
