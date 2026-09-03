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

// --- Mocks ---

type mockAppService struct {
	mock.Mock
}

func (m *mockAppService) Create(ctx context.Context, name, description, dockerImage string, ownerID uuid.UUID) (*domain.Application, error) {
	args := m.Called(ctx, name, description, dockerImage, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Application), args.Error(1)
}

func (m *mockAppService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Application), args.Error(1)
}

func (m *mockAppService) List(ctx context.Context) ([]domain.Application, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Application), args.Error(1)
}

func (m *mockAppService) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Application, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).([]domain.Application), args.Error(1)
}

type mockDepService struct {
	mock.Mock
}

func (m *mockDepService) Deploy(ctx context.Context, appID uuid.UUID, version string) (*domain.Deployment, error) {
	args := m.Called(ctx, appID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deployment), args.Error(1)
}

func (m *mockDepService) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	args := m.Called(ctx, appID)
	return args.Get(0).([]domain.Deployment), args.Error(1)
}

func (m *mockDepService) Rollback(ctx context.Context, appID uuid.UUID) (*domain.Deployment, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deployment), args.Error(1)
}

func (m *mockDepService) GetLogs(ctx context.Context, containerID string, tail int) (string, error) {
	args := m.Called(ctx, containerID, tail)
	return args.String(0), args.Error(1)
}

// --- Helpers ---

func setupHandler(appSvc domain.ApplicationService, depSvc domain.DeploymentService) (http.Handler, *ApplicationHandler) {
	h := NewApplicationHandler(appSvc, depSvc)
	r := chi.NewRouter()
	h.Routes(r)
	return r, h
}

// --- Tests ---

func TestApplicationHandler_Deploy(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	appID := uuid.New()
	depID := uuid.New()

	depSvc.On("Deploy", mock.Anything, appID, "2.0.0").Return(&domain.Deployment{
		ID:            depID,
		ApplicationID: appID,
		Version:       "2.0.0",
		Status:        "pending",
	}, nil)

	body, _ := json.Marshal(map[string]string{"version": "2.0.0"})
	req := httptest.NewRequest(http.MethodPost, "/applications/"+appID.String()+"/deploy", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp domain.Deployment
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "pending", resp.Status)
	assert.Equal(t, "2.0.0", resp.Version)

	depSvc.AssertExpectations(t)
}

func TestApplicationHandler_Deploy_InvalidUUID(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	req := httptest.NewRequest(http.MethodPost, "/applications/bad-uuid/deploy", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApplicationHandler_ListDeployments(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	appID := uuid.New()
	deps := []domain.Deployment{
		{ID: uuid.New(), ApplicationID: appID, Version: "1.0.0", Status: "successful"},
	}

	depSvc.On("ListByApplication", mock.Anything, appID).Return(deps, nil)

	req := httptest.NewRequest(http.MethodGet, "/applications/"+appID.String()+"/deployments", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []domain.Deployment
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Len(t, resp, 1)
}

func TestApplicationHandler_Rollback(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	appID := uuid.New()
	dep := &domain.Deployment{
		ID:            uuid.New(),
		ApplicationID: appID,
		Version:       "1.1.0",
		Status:        "pending",
		Message:       "Rollback to version 1.1.0",
	}

	depSvc.On("Rollback", mock.Anything, appID).Return(dep, nil)

	req := httptest.NewRequest(http.MethodPost, "/applications/"+appID.String()+"/rollback", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)

	var resp domain.Deployment
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "pending", resp.Status)
	assert.Contains(t, resp.Message, "Rollback")
}

func TestApplicationHandler_Create(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	ownerID := uuid.New()
	app := &domain.Application{
		ID:          uuid.New(),
		Name:        "my-app",
		DockerImage: "nginx",
		OwnerID:     ownerID,
	}

	appSvc.On("Create", mock.Anything, "my-app", "desc", "nginx", ownerID).Return(app, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"name":         "my-app",
		"description":  "desc",
		"docker_image": "nginx",
		"owner_id":     ownerID,
	})
	req := httptest.NewRequest(http.MethodPost, "/applications", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestApplicationHandler_Get(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	appID := uuid.New()
	app := &domain.Application{
		ID:   appID,
		Name: "test-app",
	}

	appSvc.On("GetByID", mock.Anything, appID).Return(app, nil)

	req := httptest.NewRequest(http.MethodGet, "/applications/"+appID.String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestApplicationHandler_List(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	apps := []domain.Application{
		{ID: uuid.New(), Name: "app1"},
		{ID: uuid.New(), Name: "app2"},
	}

	appSvc.On("List", mock.Anything).Return(apps, nil)

	req := httptest.NewRequest(http.MethodGet, "/applications", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestApplicationHandler_ListByOwner(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	ownerID := uuid.New()
	apps := []domain.Application{
		{ID: uuid.New(), Name: "app1", OwnerID: ownerID},
		{ID: uuid.New(), Name: "app2", OwnerID: ownerID},
	}

	appSvc.On("ListByOwner", mock.Anything, ownerID).Return(apps, nil)

	req := httptest.NewRequest(http.MethodGet, "/applications?owner_id="+ownerID.String(), nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []domain.Application
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Len(t, resp, 2)
}

func TestApplicationHandler_ListByOwner_InvalidUUID(t *testing.T) {
	appSvc := new(mockAppService)
	depSvc := new(mockDepService)
	router, _ := setupHandler(appSvc, depSvc)

	req := httptest.NewRequest(http.MethodGet, "/applications?owner_id=bad-uuid", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
