package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"control-plane/cli/client"
	"control-plane/cli/commands"
	cliconfig "control-plane/cli/config"
	"control-plane/internal/config"
	"control-plane/internal/domain"
	"control-plane/internal/repository/memory"
	"control-plane/internal/server"
	"control-plane/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopExecutor for fast, deterministic CLI tests
type noopExecutor struct{}

func (n *noopExecutor) Deploy(ctx context.Context, appID, depID, image, version string) (string, error) {
	return "noop-" + depID[:8], nil
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

func setupIntegrationCLI(t *testing.T) (*httptest.Server, func()) {
	appRepo := memory.NewApplicationRepository()
	depRepo := memory.NewDeploymentRepository()
	userRepo := memory.NewUserRepository()

	appSvc := service.NewApplicationService(appRepo)
	depSvc := service.NewDeploymentServiceWithPoll(appRepo, depRepo, &noopExecutor{}, 10*time.Millisecond)
	userSvc := service.NewUserService(userRepo)

	srv := server.New(&config.Config{
		ServerAddr: "127.0.0.1:0",
	}, &server.Dependencies{
		AppService:  appSvc,
		DepService:  depSvc,
		UserService: userSvc,
	})

	ts := httptest.NewServer(srv.Router())

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	cliconfig.Init("")
	cliconfig.Save(&cliconfig.Config{Endpoint: ts.URL})

	return ts, func() {
		ts.Close()
		os.Setenv("HOME", oldHome)
	}
}

// executeCmd runs a cobra command and returns (stdout, stderr, error)
func executeCmd(args []string) (string, string, error) {
	root := commands.NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func countStatusCLI(deps []domain.Deployment, status string) int {
	c := 0
	for _, d := range deps {
		if d.Status == status {
			c++
		}
	}
	return c
}

func findLatestCLI(deps []domain.Deployment) *domain.Deployment {
	var latest *domain.Deployment
	for i := range deps {
		if latest == nil || deps[i].CreatedAt.After(latest.CreatedAt) {
			latest = &deps[i]
		}
	}
	return latest
}

func TestCLI_FullWorkflow(t *testing.T) {
	_, cleanup := setupIntegrationCLI(t)
	defer cleanup()

	// --- Step 1: Create User ---
	out, errOut, err := executeCmd([]string{"--output", "json", "user", "create", "--email", "cli@example.com", "--name", "CLI User"})
	require.NoError(t, err, "user create failed. stderr: %s", errOut)

	var user domain.User
	err = json.Unmarshal([]byte(out), &user)
	require.NoError(t, err, "failed to parse user JSON. stdout: %s, stderr: %s", out, errOut)
	assert.NotEqual(t, "", user.ID.String())
	t.Logf("Created user: %s", user.ID)

	// --- Step 2: Create Application ---
	out, errOut, err = executeCmd([]string{"--output", "json", "app", "create", "--name", "cli-app", "--description", "From CLI", "--image", "nginx", "--owner-id", user.ID.String()})
	require.NoError(t, err, "app create failed. stderr: %s", errOut)

	var app domain.Application
	err = json.Unmarshal([]byte(out), &app)
	require.NoError(t, err, "failed to parse app JSON. stdout: %s, stderr: %s", out, errOut)
	assert.Equal(t, "cli-app", app.Name)
	t.Logf("Created app: %s", app.ID)

	// --- Step 3: Deploy ---
	out, errOut, err = executeCmd([]string{"--output", "json", "app", "deploy", app.ID.String(), "--version", "1.0.0"})
	require.NoError(t, err, "deploy failed. stderr: %s", errOut)

	var dep domain.Deployment
	err = json.Unmarshal([]byte(out), &dep)
	require.NoError(t, err, "failed to parse deploy JSON. stdout: %s, stderr: %s", out, errOut)
	assert.Equal(t, "pending", dep.Status)
	t.Logf("Created deployment: %s", dep.ID)

	// --- Step 4: List Deployments (immediate) ---
	out, errOut, err = executeCmd([]string{"--output", "json", "app", "deployments", app.ID.String()})
	require.NoError(t, err, "list deployments failed. stderr: %s", errOut)

	var deps []domain.Deployment
	err = json.Unmarshal([]byte(out), &deps)
	require.NoError(t, err, "failed to parse deployments JSON. stdout: %s, stderr: %s", out, errOut)
	require.Len(t, deps, 1)
	// noop executor is fast — may already be successful
	assert.True(t, deps[0].Status == "pending" || deps[0].Status == "successful", "expected pending or successful, got %s", deps[0].Status)

	// --- Step 5: Wait for deployment to complete (noop is fast) ---
	time.Sleep(50 * time.Millisecond)

	out, errOut, err = executeCmd([]string{"--output", "json", "app", "deployments", app.ID.String()})
	require.NoError(t, err, "list deployments (2nd) failed. stderr: %s", errOut)

	err = json.Unmarshal([]byte(out), &deps)
	require.NoError(t, err, "failed to parse deployments JSON (2nd). stdout: %s, stderr: %s", out, errOut)
	assert.Equal(t, "successful", deps[0].Status)
	t.Logf("Deployment completed: %s", deps[0].Status)

	// --- Step 6: Rollback ---
	out, errOut, err = executeCmd([]string{"--output", "json", "app", "rollback", app.ID.String()})
	require.NoError(t, err, "rollback failed. stderr: %s", errOut)

	var rollbackDep domain.Deployment
	err = json.Unmarshal([]byte(out), &rollbackDep)
	require.NoError(t, err, "failed to parse rollback JSON. stdout: %s, stderr: %s", out, errOut)
	assert.Equal(t, "pending", rollbackDep.Status)
	assert.Contains(t, rollbackDep.Message, "Rollback")

	// --- Step 7: Wait for rollback ---
	time.Sleep(50 * time.Millisecond)

	out, errOut, err = executeCmd([]string{"--output", "json", "app", "deployments", app.ID.String()})
	require.NoError(t, err, "list deployments (3rd) failed. stderr: %s", errOut)

	err = json.Unmarshal([]byte(out), &deps)
	require.NoError(t, err, "failed to parse deployments JSON (3rd). stdout: %s, stderr: %s", out, errOut)
	require.Len(t, deps, 2)

	// After rollback: 1 successful (the rollback itself), 1 stopped (the original)
	assert.Equal(t, 1, countStatusCLI(deps, "successful"), "expected 1 successful deployment")
	assert.Equal(t, 1, countStatusCLI(deps, "stopped"), "expected 1 stopped deployment")

	latest := findLatestCLI(deps)
	require.NotNil(t, latest)
	assert.Equal(t, "successful", latest.Status)
	t.Logf("Rollback completed. Total deployments: %d", len(deps))
}

func TestCLI_ClientDirectly(t *testing.T) {
	_, cleanup := setupIntegrationCLI(t)
	defer cleanup()

	c, err := client.New(cliconfig.Get())
	require.NoError(t, err)
	ctx := context.Background()

	user, err := c.CreateUser(ctx, "direct@example.com", "Direct")
	require.NoError(t, err)

	app, err := c.CreateApplication(ctx, "direct-app", "desc", "nginx", user.ID)
	require.NoError(t, err)

	dep, err := c.Deploy(ctx, app.ID, "2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "pending", dep.Status)

	time.Sleep(50 * time.Millisecond)
	deps, err := c.ListDeployments(ctx, app.ID)
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "successful", deps[0].Status)
}
