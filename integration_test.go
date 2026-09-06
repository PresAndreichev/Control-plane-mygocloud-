package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"control-plane/internal/config"
	"control-plane/internal/domain"
	lockmem "control-plane/internal/lock/memory"
	"control-plane/internal/queue/channel"
	"control-plane/internal/repository/memory"
	"control-plane/internal/server"
	"control-plane/internal/service"
	"control-plane/internal/worker"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopExecutor is a fake executor for integration tests.
// It returns successful immediately without touching Docker.
type noopExecutor struct{}

func (n *noopExecutor) Deploy(ctx context.Context, appID, depID, image, version string) (string, error) {
	return "noop-container-" + depID[:8], nil
}

func (n *noopExecutor) Status(ctx context.Context, containerID string) (string, error) {
	return "successful", nil
}

func (n *noopExecutor) Logs(ctx context.Context, containerID string, tail int) (string, error) {
	return "noop logs", nil
}

func (n *noopExecutor) Stop(ctx context.Context, containerID string) error {
	return nil
}

func setupIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()

	// In-memory repositories
	appRepo := memory.NewApplicationRepository()
	depRepo := memory.NewDeploymentRepository()
	userRepo := memory.NewUserRepository()

	// Queue + Worker (fast poll for tests)
	q := channel.New(100)
	exec := &noopExecutor{}
	locker := lockmem.New()
	w := worker.NewWithPoll(q, appRepo, depRepo, exec, locker, 2, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	require.NoError(t, w.Start(ctx))
	t.Cleanup(func() {
		_ = q.Close()
		w.Stop()
	})

	// Services
	appSvc := service.NewApplicationService(appRepo)
	depSvc := service.NewDeploymentService(appRepo, depRepo, q, exec)
	userSvc := service.NewUserService(userRepo)

	cfg := config.TestConfig()
	srv := server.New(cfg, &server.Dependencies{
		AppService:  appSvc,
		DepService:  depSvc,
		UserService: userSvc,
	})

	return httptest.NewServer(srv.Router())
}

func countStatus(deps []domain.Deployment, status string) int {
	c := 0
	for _, d := range deps {
		if d.Status == status {
			c++
		}
	}
	return c
}

func findLatest(deps []domain.Deployment) *domain.Deployment {
	var latest *domain.Deployment
	for i := range deps {
		if latest == nil || deps[i].CreatedAt.After(latest.CreatedAt) {
			latest = &deps[i]
		}
	}
	return latest
}

func TestIntegration_FullDeploymentLifecycle(t *testing.T) {
	ts := setupIntegrationServer(t)
	defer ts.Close()
	client := ts.Client()

	// 1. Create a user
	userBody, _ := json.Marshal(map[string]string{"email": "dev@example.com", "name": "Developer"})
	resp, err := client.Post(ts.URL+"/api/v1/users", "application/json", bytes.NewReader(userBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var user domain.User
	json.NewDecoder(resp.Body).Decode(&user)
	resp.Body.Close()
	require.NotEqual(t, uuid.Nil, user.ID)

	// 2. Create an application (with docker_image for Phase 3)
	appBody, _ := json.Marshal(map[string]interface{}{
		"name":         "api-gateway",
		"description":  "Edge proxy",
		"docker_image": "nginx",
		"owner_id":     user.ID,
	})
	resp, err = client.Post(ts.URL+"/api/v1/applications", "application/json", bytes.NewReader(appBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var app domain.Application
	json.NewDecoder(resp.Body).Decode(&app)
	resp.Body.Close()
	require.NotEqual(t, uuid.Nil, app.ID)

	// 3. Deploy v1.0.0
	deployBody, _ := json.Marshal(map[string]string{"version": "1.0.0"})
	resp, err = client.Post(ts.URL+"/api/v1/applications/"+app.ID.String()+"/deploy", "application/json", bytes.NewReader(deployBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	var dep1 domain.Deployment
	json.NewDecoder(resp.Body).Decode(&dep1)
	resp.Body.Close()
	assert.Equal(t, "pending", dep1.Status)

	// 4. Deploy v1.1.0
	deployBody, _ = json.Marshal(map[string]string{"version": "1.1.0"})
	resp, err = client.Post(ts.URL+"/api/v1/applications/"+app.ID.String()+"/deploy", "application/json", bytes.NewReader(deployBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	var dep2 domain.Deployment
	json.NewDecoder(resp.Body).Decode(&dep2)
	resp.Body.Close()
	assert.Equal(t, "pending", dep2.Status)

	// 5. Wait for deployments to complete (noop executor is instant)
	time.Sleep(100 * time.Millisecond)

	// 6. List deployments
	resp, err = client.Get(ts.URL + "/api/v1/applications/" + app.ID.String() + "/deployments")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var deps []domain.Deployment
	json.NewDecoder(resp.Body).Decode(&deps)
	resp.Body.Close()
	require.Len(t, deps, 2)

	// The newest deployment should be successful; the older one was stopped by the newer deploy.
	require.Len(t, deps, 2)
	successfulCount := countStatus(deps, "successful")
	otherCount := countStatus(deps, "stopped") + countStatus(deps, "cancelled")
	assert.Equal(t, 1, successfulCount, "expected 1 successful deployment")
	assert.Equal(t, 1, otherCount, "expected 1 terminal non-successful deployment")
	// 7. Rollback
	resp, err = client.Post(ts.URL+"/api/v1/applications/"+app.ID.String()+"/rollback", "application/json", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)

	var rollbackDep domain.Deployment
	json.NewDecoder(resp.Body).Decode(&rollbackDep)
	resp.Body.Close()
	assert.Equal(t, "1.1.0", rollbackDep.Version)
	assert.Equal(t, "pending", rollbackDep.Status)

	// Wait for rollback to complete
	time.Sleep(100 * time.Millisecond)

	// 8. Verify rollback completed
	resp, err = client.Get(ts.URL + "/api/v1/applications/" + app.ID.String() + "/deployments")
	require.NoError(t, err)

	var finalDeps []domain.Deployment
	json.NewDecoder(resp.Body).Decode(&finalDeps)
	resp.Body.Close()

	// Then the assertions...
	require.Len(t, finalDeps, 3)
	successfulCount = countStatus(finalDeps, "successful")
	otherCount = countStatus(finalDeps, "stopped") + countStatus(finalDeps, "cancelled")
	assert.Equal(t, 1, successfulCount, "expected 1 successful deployment after rollback")
	assert.Equal(t, 2, otherCount, "expected 2 terminal non-successful deployments after rollback")

	latest := findLatest(finalDeps)
	require.NotNil(t, latest)
	assert.Equal(t, "successful", latest.Status)
	assert.Equal(t, "1.1.0", latest.Version)
}

func TestIntegration_Deploy_AppNotFound(t *testing.T) {
	ts := setupIntegrationServer(t)
	defer ts.Close()

	deployBody, _ := json.Marshal(map[string]string{"version": "1.0.0"})
	resp, err := http.Post(ts.URL+"/api/v1/applications/"+uuid.New().String()+"/deploy", "application/json", bytes.NewReader(deployBody))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
