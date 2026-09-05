package service

import (
	"context"
	"testing"

	"control-plane/internal/domain"
	"control-plane/internal/queue"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type mockAppRepo struct {
	mock.Mock
}

func (m *mockAppRepo) Create(ctx context.Context, app *domain.Application) error {
	args := m.Called(ctx, app)
	return args.Error(0)
}

func (m *mockAppRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Application, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Application), args.Error(1)
}

func (m *mockAppRepo) List(ctx context.Context) ([]domain.Application, error) {
	args := m.Called(ctx)
	return args.Get(0).([]domain.Application), args.Error(1)
}

func (m *mockAppRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Application, error) {
	args := m.Called(ctx, ownerID)
	return args.Get(0).([]domain.Application), args.Error(1)
}

type mockDepRepo struct {
	mock.Mock
}

func (m *mockDepRepo) Create(ctx context.Context, dep *domain.Deployment) error {
	args := m.Called(ctx, dep)
	return args.Error(0)
}

func (m *mockDepRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Deployment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deployment), args.Error(1)
}

func (m *mockDepRepo) ListByApplication(ctx context.Context, appID uuid.UUID) ([]domain.Deployment, error) {
	args := m.Called(ctx, appID)
	return args.Get(0).([]domain.Deployment), args.Error(1)
}

func (m *mockDepRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status, message string) error {
	args := m.Called(ctx, id, status, message)
	return args.Error(0)
}

func (m *mockDepRepo) UpdateContainerID(ctx context.Context, id uuid.UUID, containerID string) error {
	args := m.Called(ctx, id, containerID)
	return args.Error(0)
}

func (m *mockDepRepo) GetLastSuccessful(ctx context.Context, appID uuid.UUID) (*domain.Deployment, error) {
	args := m.Called(ctx, appID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Deployment), args.Error(1)
}

type mockExecutor struct {
	mock.Mock
}

func (m *mockExecutor) Deploy(ctx context.Context, appID, depID, image, version string) (string, error) {
	args := m.Called(ctx, appID, depID, image, version)
	return args.String(0), args.Error(1)
}

func (m *mockExecutor) Status(ctx context.Context, containerID string) (string, error) {
	args := m.Called(ctx, containerID)
	return args.String(0), args.Error(1)
}

func (m *mockExecutor) Logs(ctx context.Context, containerID string, tail int) (string, error) {
	args := m.Called(ctx, containerID, tail)
	return args.String(0), args.Error(1)
}

func (m *mockExecutor) Stop(ctx context.Context, containerID string) error {
	args := m.Called(ctx, containerID)
	return args.Error(0)
}

type mockQueue struct {
	mock.Mock
}

func (m *mockQueue) Publish(ctx context.Context, job queue.DeploymentJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *mockQueue) Consume(ctx context.Context) (<-chan queue.DeploymentJob, error) {
	args := m.Called(ctx)
	return args.Get(0).(<-chan queue.DeploymentJob), args.Error(1)
}

func (m *mockQueue) Close() error {
	args := m.Called()
	return args.Error(0)
}

// --- Tests ---

func TestDeploymentService_Deploy_Success(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	appID := uuid.New()
	version := "1.2.3"

	appRepo.On("GetByID", mock.Anything, appID).Return(&domain.Application{
		ID: appID, Name: "app", DockerImage: "nginx",
	}, nil)
	depRepo.On("Create", mock.Anything, mock.MatchedBy(func(d *domain.Deployment) bool {
		return d.ApplicationID == appID && d.Version == version && d.Status == "pending"
	})).Return(nil)
	q.On("Publish", mock.Anything, mock.MatchedBy(func(job queue.DeploymentJob) bool {
		return job.ApplicationID == appID.String() && job.Version == version && job.Image == "nginx"
	})).Return(nil)

	dep, err := svc.Deploy(context.Background(), appID, version)

	assert.NoError(t, err)
	assert.Equal(t, "pending", dep.Status)
	assert.Equal(t, version, dep.Version)
	appRepo.AssertExpectations(t)
	depRepo.AssertExpectations(t)
	q.AssertExpectations(t)
}

func TestDeploymentService_Deploy_AppNotFound(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	appID := uuid.New()
	appRepo.On("GetByID", mock.Anything, appID).Return(nil, domain.ErrNotFound)

	dep, err := svc.Deploy(context.Background(), appID, "1.0.0")

	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, dep)
}

func TestDeploymentService_Deploy_InvalidVersion(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	dep, err := svc.Deploy(context.Background(), uuid.New(), "")

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	assert.Nil(t, dep)
}

func TestDeploymentService_Deploy_QueuePublishFails(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	appID := uuid.New()
	version := "1.0.0"

	appRepo.On("GetByID", mock.Anything, appID).Return(&domain.Application{
		ID: appID, DockerImage: "nginx",
	}, nil)
	depRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	q.On("Publish", mock.Anything, mock.Anything).Return(assert.AnError)
	depRepo.On("UpdateStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), "failed", mock.Anything).Return(nil)

	dep, err := svc.Deploy(context.Background(), appID, version)

	assert.Error(t, err)
	assert.Nil(t, dep)
	depRepo.AssertExpectations(t)
}

func TestDeploymentService_Rollback_Success(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	appID := uuid.New()
	lastDep := &domain.Deployment{ID: uuid.New(), ApplicationID: appID, Version: "1.1.0", Status: "successful"}

	appRepo.On("GetByID", mock.Anything, appID).Return(&domain.Application{ID: appID, DockerImage: "nginx"}, nil)
	depRepo.On("GetLastSuccessful", mock.Anything, appID).Return(lastDep, nil)
	depRepo.On("Create", mock.Anything, mock.MatchedBy(func(d *domain.Deployment) bool {
		return d.Version == "1.1.0" && d.Status == "pending" && d.Message == "Rollback to version 1.1.0"
	})).Return(nil)
	q.On("Publish", mock.Anything, mock.MatchedBy(func(job queue.DeploymentJob) bool {
		return job.ApplicationID == appID.String() && job.Version == "1.1.0"
	})).Return(nil)

	dep, err := svc.Rollback(context.Background(), appID)

	assert.NoError(t, err)
	assert.Equal(t, "1.1.0", dep.Version)
	assert.Equal(t, "pending", dep.Status)
	appRepo.AssertExpectations(t)
	depRepo.AssertExpectations(t)
	q.AssertExpectations(t)
}

func TestDeploymentService_Rollback_NoSuccessfulDeployment(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	appID := uuid.New()
	appRepo.On("GetByID", mock.Anything, appID).Return(&domain.Application{ID: appID}, nil)
	depRepo.On("GetLastSuccessful", mock.Anything, appID).Return(nil, domain.ErrNotFound)

	dep, err := svc.Rollback(context.Background(), appID)

	assert.ErrorIs(t, err, domain.ErrRollback)
	assert.Nil(t, dep)
}

func TestDeploymentService_ListByApplication(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	appID := uuid.New()
	expected := []domain.Deployment{
		{ID: uuid.New(), ApplicationID: appID, Version: "1.0.0"},
	}
	depRepo.On("ListByApplication", mock.Anything, appID).Return(expected, nil)

	deps, err := svc.ListByApplication(context.Background(), appID)

	assert.NoError(t, err)
	assert.Len(t, deps, 1)
	depRepo.AssertExpectations(t)
}

func TestDeploymentService_GetLogs(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	exec.On("Logs", mock.Anything, "container-123", 50).Return("log line 1\nlog line 2", nil)

	logs, err := svc.GetLogs(context.Background(), "container-123", 50)

	assert.NoError(t, err)
	assert.Contains(t, logs, "log line 1")
	exec.AssertExpectations(t)
}

func TestDeploymentService_GetLogs_EmptyContainerID(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	q := new(mockQueue)
	exec := new(mockExecutor)
	svc := NewDeploymentService(appRepo, depRepo, q, exec)

	_, err := svc.GetLogs(context.Background(), "", 50)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}
