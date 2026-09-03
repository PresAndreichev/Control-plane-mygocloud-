package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"control-plane/cli/config"
	"control-plane/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockServer(t *testing.T) (*httptest.Server, *http.ServeMux) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	return srv, mux
}

func TestNew_EmptyEndpoint(t *testing.T) {
	_, err := New(&config.Config{Endpoint: ""})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no API endpoint configured")
}

func TestClient_CreateUser(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var req struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "dev@example.com", req.Email)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(domain.User{
			ID:    uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			Email: req.Email,
			Name:  req.Name,
		})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	user, err := c.CreateUser(context.Background(), "dev@example.com", "Dev")
	require.NoError(t, err)
	assert.Equal(t, "dev@example.com", user.Email)
}

func TestClient_GetUser(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	userID := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	mux.HandleFunc("/api/v1/users/"+userID.String(), func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode(domain.User{ID: userID, Email: "test@example.com"})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	user, err := c.GetUser(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)
}

func TestClient_CreateApplication(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	ownerID := uuid.New()
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var req struct {
			Name        string    `json:"name"`
			Description string    `json:"description"`
			DockerImage string    `json:"docker_image"`
			OwnerID     uuid.UUID `json:"owner_id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "nginx", req.DockerImage)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(domain.Application{
			ID:          uuid.New(),
			Name:        req.Name,
			DockerImage: req.DockerImage,
			OwnerID:     req.OwnerID,
		})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	app, err := c.CreateApplication(context.Background(), "api-gateway", "desc", "nginx", ownerID)
	require.NoError(t, err)
	assert.Equal(t, "api-gateway", app.Name)
	assert.Equal(t, "nginx", app.DockerImage)
}

func TestClient_Deploy(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	appID := uuid.New()
	mux.HandleFunc("/api/v1/applications/"+appID.String()+"/deploy", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(domain.Deployment{
			ID:            uuid.New(),
			ApplicationID: appID,
			Version:       "1.0.0",
			Status:        "pending",
		})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	dep, err := c.Deploy(context.Background(), appID, "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "pending", dep.Status)
}

func TestClient_ListDeployments(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	appID := uuid.New()
	mux.HandleFunc("/api/v1/applications/"+appID.String()+"/deployments", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode([]domain.Deployment{
			{ID: uuid.New(), ApplicationID: appID, Version: "1.0.0", Status: "successful"},
		})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	deps, err := c.ListDeployments(context.Background(), appID)
	require.NoError(t, err)
	assert.Len(t, deps, 1)
}

func TestClient_Rollback(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	appID := uuid.New()
	mux.HandleFunc("/api/v1/applications/"+appID.String()+"/rollback", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		json.NewEncoder(w).Encode(domain.Deployment{
			ID:            uuid.New(),
			ApplicationID: appID,
			Version:       "1.0.0",
			Status:        "pending",
			Message:       "Rollback to version 1.0.0",
		})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	dep, err := c.Rollback(context.Background(), appID)
	require.NoError(t, err)
	assert.Equal(t, "pending", dep.Status)
	assert.Contains(t, dep.Message, "Rollback")
}

func TestClient_GetLogs(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	mux.HandleFunc("/api/v1/deployments/logs", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		containerID := r.URL.Query().Get("container_id")
		assert.Equal(t, "abc123", containerID)

		json.NewEncoder(w).Encode(map[string]string{"logs": "log line 1\nlog line 2"})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	logs, err := c.GetLogs(context.Background(), "abc123", 50)
	require.NoError(t, err)
	assert.Contains(t, logs, "log line 1")
}

func TestClient_APIError(t *testing.T) {
	srv, mux := setupMockServer(t)
	defer srv.Close()

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  409,
			"code":    "conflict",
			"message": "email already exists",
		})
	})

	c, err := New(&config.Config{Endpoint: srv.URL})
	require.NoError(t, err)

	_, err = c.CreateUser(context.Background(), "dup@example.com", "Dup")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "409")
	assert.Contains(t, err.Error(), "conflict")
}

func TestClient_NetworkError(t *testing.T) {
	c, err := New(&config.Config{Endpoint: "http://localhost:59999"})
	require.NoError(t, err)

	_, err = c.ListApplications(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}
