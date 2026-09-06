package worker

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/domain"
	"control-plane/internal/lock/memory"
	"control-plane/internal/queue"
	"control-plane/internal/queue/channel"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks (self-contained for this package) ---

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

// --- Tests ---

func TestWorker_Deploy_Success(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	exec := new(mockExecutor)
	q := channel.New(10)
	locker := memory.New()

	w := NewWithPoll(q, appRepo, depRepo, exec, locker, 1, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	require.NoError(t, w.Start(ctx))

	// CRITICAL: close queue first so consumers wake up, then stop workers, then cancel context.
	t.Cleanup(func() {
		_ = q.Close()
		w.Stop()
		cancel()
	})

	appID := uuid.New()
	depID := uuid.New()

	// Image is empty so worker falls back to app record.
	appRepo.On("GetByID", mock.Anything, appID).Return(&domain.Application{
		ID: appID, DockerImage: "nginx",
	}, nil)
	depRepo.On("ListByApplication", mock.Anything, appID).Return([]domain.Deployment{}, nil)
	depRepo.On("UpdateStatus", mock.Anything, depID, "running", mock.Anything).Return(nil).Once()
	depRepo.On("UpdateContainerID", mock.Anything, depID, "container-abc").Return(nil).Once()
	exec.On("Deploy", mock.Anything, appID.String(), depID.String(), "nginx", "1.0.0").Return("container-abc", nil)
	exec.On("Status", mock.Anything, "container-abc").Return("pending", nil).Once()
	exec.On("Status", mock.Anything, "container-abc").Return("successful", nil).Once()
	depRepo.On("UpdateStatus", mock.Anything, depID, "successful", mock.Anything).Return(nil).Once()

	err := q.Publish(ctx, queue.DeploymentJob{
		Type:          queue.JobTypeDeploy,
		DeploymentID:  depID.String(),
		ApplicationID: appID.String(),
		Image:         "", // forces lookup
		Version:       "1.0.0",
	})
	require.NoError(t, err)

	time.Sleep(150 * time.Millisecond)

	appRepo.AssertExpectations(t)
	depRepo.AssertExpectations(t)
	exec.AssertExpectations(t)
}

func TestWorker_Deploy_StaleDeployment(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	exec := new(mockExecutor)
	q := channel.New(10)
	locker := memory.New()

	w := NewWithPoll(q, appRepo, depRepo, exec, locker, 1, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	require.NoError(t, w.Start(ctx))

	t.Cleanup(func() {
		_ = q.Close()
		w.Stop()
		cancel()
	})

	appID := uuid.New()
	oldDepID := uuid.New()
	newDepID := uuid.New()

	depRepo.On("ListByApplication", mock.Anything, appID).Return([]domain.Deployment{
		{ID: oldDepID, ApplicationID: appID, CreatedAt: time.Now().Add(-time.Hour)},
		{ID: newDepID, ApplicationID: appID, CreatedAt: time.Now()},
	}, nil)
	depRepo.On("UpdateStatus", mock.Anything, oldDepID, "cancelled", mock.Anything).Return(nil).Once()

	err := q.Publish(ctx, queue.DeploymentJob{
		Type:          queue.JobTypeDeploy,
		DeploymentID:  oldDepID.String(),
		ApplicationID: appID.String(),
		Image:         "nginx",
		Version:       "1.0.0",
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	depRepo.AssertExpectations(t)
	exec.AssertNotCalled(t, "Deploy", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestWorker_Deploy_DockerFail(t *testing.T) {
	appRepo := new(mockAppRepo)
	depRepo := new(mockDepRepo)
	exec := new(mockExecutor)
	q := channel.New(10)
	locker := memory.New()

	w := NewWithPoll(q, appRepo, depRepo, exec, locker, 1, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())

	require.NoError(t, w.Start(ctx))

	t.Cleanup(func() {
		_ = q.Close()
		w.Stop()
		cancel()
	})

	appID := uuid.New()
	depID := uuid.New()

	appRepo.On("GetByID", mock.Anything, appID).Return(&domain.Application{
		ID: appID, DockerImage: "nginx",
	}, nil)
	depRepo.On("ListByApplication", mock.Anything, appID).Return([]domain.Deployment{}, nil)
	depRepo.On("UpdateStatus", mock.Anything, depID, "running", mock.Anything).Return(nil).Once()
	exec.On("Deploy", mock.Anything, appID.String(), depID.String(), "nginx", "1.0.0").Return("", assert.AnError)
	depRepo.On("UpdateStatus", mock.Anything, depID, "failed", mock.Anything).Return(nil).Once()

	err := q.Publish(ctx, queue.DeploymentJob{
		Type:          queue.JobTypeDeploy,
		DeploymentID:  depID.String(),
		ApplicationID: appID.String(),
		Image:         "",
		Version:       "1.0.0",
	})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	appRepo.AssertExpectations(t)
	depRepo.AssertExpectations(t)
	exec.AssertExpectations(t)
}
