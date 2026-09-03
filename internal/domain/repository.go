package domain

import (
	"context"

	"github.com/google/uuid"
)

type ApplicationRepository interface {
	Create(ctx context.Context, app *Application) error
	GetByID(ctx context.Context, id uuid.UUID) (*Application, error)
	List(ctx context.Context) ([]Application, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]Application, error)
}

type DeploymentRepository interface {
	Create(ctx context.Context, dep *Deployment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Deployment, error)
	ListByApplication(ctx context.Context, appID uuid.UUID) ([]Deployment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status, message string) error
	GetLastSuccessful(ctx context.Context, appID uuid.UUID) (*Deployment, error)
	UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}
