package memory

import (
	"context"
	"testing"

	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationRepository_Create(t *testing.T) {
	repo := NewApplicationRepository()
	app := &domain.Application{
		ID:          uuid.New(),
		Name:        "test-app",
		Description: "desc",
		OwnerID:     uuid.New(),
	}

	err := repo.Create(context.Background(), app)
	require.NoError(t, err)
	assert.NotZero(t, app.CreatedAt)
	assert.NotZero(t, app.UpdatedAt)
}

func TestApplicationRepository_GetByID(t *testing.T) {
	repo := NewApplicationRepository()
	app := &domain.Application{
		ID:   uuid.New(),
		Name: "test-app",
	}
	err := repo.Create(context.Background(), app)
	require.NoError(t, err)

	found, err := repo.GetByID(context.Background(), app.ID)
	require.NoError(t, err)
	assert.Equal(t, app.Name, found.Name)

	_, err = repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestApplicationRepository_List(t *testing.T) {
	repo := NewApplicationRepository()

	apps, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, apps)

	app1 := &domain.Application{ID: uuid.New(), Name: "app1"}
	app2 := &domain.Application{ID: uuid.New(), Name: "app2"}
	repo.Create(context.Background(), app1)
	repo.Create(context.Background(), app2)

	apps, err = repo.List(context.Background())
	require.NoError(t, err)
	assert.Len(t, apps, 2)
}
