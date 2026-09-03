package memory

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentRepository_Create(t *testing.T) {
	repo := NewDeploymentRepository()
	dep := &domain.Deployment{
		ID:            uuid.New(),
		ApplicationID: uuid.New(),
		Version:       "1.0.0",
		Status:        "pending",
	}

	err := repo.Create(context.Background(), dep)
	require.NoError(t, err)
	assert.NotZero(t, dep.CreatedAt)
}

func TestDeploymentRepository_GetByID(t *testing.T) {
	repo := NewDeploymentRepository()
	dep := &domain.Deployment{
		ID:            uuid.New(),
		ApplicationID: uuid.New(),
		Version:       "1.0.0",
	}
	repo.Create(context.Background(), dep)

	found, err := repo.GetByID(context.Background(), dep.ID)
	require.NoError(t, err)
	assert.Equal(t, dep.Version, found.Version)

	_, err = repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeploymentRepository_ListByApplication(t *testing.T) {
	repo := NewDeploymentRepository()
	appID := uuid.New()

	deps, err := repo.ListByApplication(context.Background(), appID)
	require.NoError(t, err)
	assert.Empty(t, deps)

	dep1 := &domain.Deployment{ID: uuid.New(), ApplicationID: appID, Version: "1.0.0"}
	dep2 := &domain.Deployment{ID: uuid.New(), ApplicationID: appID, Version: "1.1.0"}
	dep3 := &domain.Deployment{ID: uuid.New(), ApplicationID: uuid.New(), Version: "2.0.0"}
	repo.Create(context.Background(), dep1)
	repo.Create(context.Background(), dep2)
	repo.Create(context.Background(), dep3)

	deps, err = repo.ListByApplication(context.Background(), appID)
	require.NoError(t, err)
	assert.Len(t, deps, 2)
}

func TestDeploymentRepository_UpdateStatus(t *testing.T) {
	repo := NewDeploymentRepository()
	dep := &domain.Deployment{
		ID:      uuid.New(),
		Version: "1.0.0",
		Status:  "pending",
	}
	repo.Create(context.Background(), dep)

	err := repo.UpdateStatus(context.Background(), dep.ID, "running", "in progress")
	require.NoError(t, err)

	updated, err := repo.GetByID(context.Background(), dep.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", updated.Status)
	assert.Equal(t, "in progress", updated.Message)
	assert.True(t, updated.UpdatedAt.After(dep.CreatedAt))

	// Test completed_at is set for terminal statuses
	err = repo.UpdateStatus(context.Background(), dep.ID, "successful", "done")
	require.NoError(t, err)
	updated, _ = repo.GetByID(context.Background(), dep.ID)
	assert.NotNil(t, updated.CompletedAt)

	// Non-existent ID
	err = repo.UpdateStatus(context.Background(), uuid.New(), "running", "")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeploymentRepository_GetLastSuccessful(t *testing.T) {
	repo := NewDeploymentRepository()
	appID := uuid.New()

	_, err := repo.GetLastSuccessful(context.Background(), appID)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// Create successful deployment
	dep1 := &domain.Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		Version:       "1.0.0",
		Status:        "successful",
	}
	repo.Create(context.Background(), dep1)
	repo.UpdateStatus(context.Background(), dep1.ID, "successful", "") // set completed_at

	// Create failed deployment (newer)
	time.Sleep(10 * time.Millisecond)
	dep2 := &domain.Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		Version:       "1.1.0",
		Status:        "failed",
	}
	repo.Create(context.Background(), dep2)

	last, err := repo.GetLastSuccessful(context.Background(), appID)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", last.Version)
}
