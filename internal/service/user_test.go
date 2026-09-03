package service

import (
	"context"
	"testing"

	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserService_Create(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)

	repo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
		return u.Email == "test@example.com" && u.Name == "Test User"
	})).Return(nil)

	user, err := svc.Create(context.Background(), "test@example.com", "Test User")

	assert.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.NotEqual(t, uuid.Nil, user.ID)
	repo.AssertExpectations(t)
}

func TestUserService_Create_InvalidInput(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)

	tests := []struct {
		email string
		name  string
	}{
		{"", "Name"},
		{"email@example.com", ""},
		{"", ""},
	}

	for _, tt := range tests {
		user, err := svc.Create(context.Background(), tt.email, tt.name)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, user)
	}
}

func TestUserService_GetByID(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo)

	userID := uuid.New()
	expected := &domain.User{ID: userID, Email: "test@example.com"}
	repo.On("GetByID", mock.Anything, userID).Return(expected, nil)

	user, err := svc.GetByID(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, expected.Email, user.Email)
	repo.AssertExpectations(t)
}

// mockUserRepo for service tests

type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
