package memory

import (
	"context"
	"testing"

	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_Create(t *testing.T) {
	repo := NewUserRepository()
	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test",
	}

	err := repo.Create(context.Background(), user)
	require.NoError(t, err)
	assert.NotZero(t, user.CreatedAt)
}

func TestUserRepository_GetByID(t *testing.T) {
	repo := NewUserRepository()
	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test",
	}
	repo.Create(context.Background(), user)

	found, err := repo.GetByID(context.Background(), user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.Email, found.Email)

	_, err = repo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	repo := NewUserRepository()
	user := &domain.User{
		ID:    uuid.New(),
		Email: "findme@example.com",
		Name:  "Find Me",
	}
	repo.Create(context.Background(), user)

	found, err := repo.GetByEmail(context.Background(), "findme@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.Name, found.Name)

	_, err = repo.GetByEmail(context.Background(), "missing@example.com")
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
