package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"control-plane/cli/config"
	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAppTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/applications" && r.Method == http.MethodPost:
			var req struct {
				Name        string    `json:"name"`
				Description string    `json:"description"`
				OwnerID     uuid.UUID `json:"owner_id"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(domain.Application{
				ID:      uuid.New(),
				Name:    req.Name,
				OwnerID: req.OwnerID,
			})

		case r.URL.Path == "/api/v1/applications" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode([]domain.Application{
				{ID: uuid.New(), Name: "app1"},
				{ID: uuid.New(), Name: "app2"},
			})

		case r.Method == http.MethodGet && len(r.URL.Path) > len("/api/v1/applications/"):
			json.NewEncoder(w).Encode(domain.Application{
				ID:   uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				Name: "test-app",
			})

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications/a1b2c3d4-e5f6-7890-abcd-ef1234567890/deploy":
			json.NewEncoder(w).Encode(domain.Deployment{
				ID:      uuid.New(),
				Version: "1.0.0",
				Status:  "pending",
			})

		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/a1b2c3d4-e5f6-7890-abcd-ef1234567890/deployments":
			json.NewEncoder(w).Encode([]domain.Deployment{
				{ID: uuid.New(), Version: "1.0.0", Status: "successful"},
			})

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/applications/a1b2c3d4-e5f6-7890-abcd-ef1234567890/rollback":
			json.NewEncoder(w).Encode(domain.Deployment{
				ID:      uuid.New(),
				Version: "1.0.0",
				Status:  "pending",
				Message: "Rollback to version 1.0.0",
			})
		}
	}))
}

func setupTestConfig(t *testing.T, endpoint string) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)
	config.Init("")
	config.Save(&config.Config{Endpoint: endpoint})
}

func TestAppCreateCommand(t *testing.T) {
	srv := setupAppTestServer(t)
	defer srv.Close()
	setupTestConfig(t, srv.URL)

	cmd := newAppCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"create", "--name", "my-app", "--owner-id", uuid.New().String()})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "my-app")
}

func TestAppListCommand(t *testing.T) {
	srv := setupAppTestServer(t)
	defer srv.Close()
	setupTestConfig(t, srv.URL)

	cmd := newAppCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "app1")
	assert.Contains(t, buf.String(), "app2")
}

func TestAppDeployCommand(t *testing.T) {
	srv := setupAppTestServer(t)
	defer srv.Close()
	setupTestConfig(t, srv.URL)

	cmd := newAppCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"deploy", "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "--version", "1.0.0"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "pending")
}

func TestAppRollbackCommand(t *testing.T) {
	srv := setupAppTestServer(t)
	defer srv.Close()
	setupTestConfig(t, srv.URL)

	cmd := newAppCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"rollback", "a1b2c3d4-e5f6-7890-abcd-ef1234567890"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Rollback")
}
