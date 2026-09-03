package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"control-plane/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) Create(ctx context.Context, email, name string) (*domain.User, error) {
	args := m.Called(ctx, email, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func TestUserHandler_Create(t *testing.T) {
	svc := new(mockUserService)
	h := NewUserHandler(svc)
	r := chi.NewRouter()
	h.Routes(r)

	user := &domain.User{
		ID:    uuid.New(),
		Email: "test@example.com",
		Name:  "Test",
	}
	svc.On("Create", mock.Anything, "test@example.com", "Test").Return(user, nil)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com", "name": "Test"})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp domain.User
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, user.Email, resp.Email)
	svc.AssertExpectations(t)
}

func TestUserHandler_Create_InvalidBody(t *testing.T) {
	svc := new(mockUserService)
	h := NewUserHandler(svc)
	r := chi.NewRouter()
	h.Routes(r)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader([]byte("invalid")))
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_Get(t *testing.T) {
	svc := new(mockUserService)
	h := NewUserHandler(svc)
	r := chi.NewRouter()
	h.Routes(r)

	userID := uuid.New()
	user := &domain.User{ID: userID, Email: "test@example.com"}
	svc.On("GetByID", mock.Anything, userID).Return(user, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/"+userID.String(), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp domain.User
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, user.Email, resp.Email)
	svc.AssertExpectations(t)
}

func TestUserHandler_Get_InvalidUUID(t *testing.T) {
	svc := new(mockUserService)
	h := NewUserHandler(svc)
	r := chi.NewRouter()
	h.Routes(r)

	req := httptest.NewRequest(http.MethodGet, "/users/bad-uuid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
