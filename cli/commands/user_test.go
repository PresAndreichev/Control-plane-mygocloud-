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

func TestUserCreateCommand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/users", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var req struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		json.NewEncoder(w).Encode(domain.User{
			ID:    uuid.New(),
			Email: req.Email,
			Name:  req.Name,
		})
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)
	config.Init("")
	config.Save(&config.Config{Endpoint: srv.URL})

	cmd := newUserCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"create", "--email", "dev@example.com", "--name", "Developer"})

	err := cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "dev@example.com")
}

func TestUserGetCommand_InvalidUUID(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)
	config.Init("")

	cmd := newUserCommand()
	cmd.SetArgs([]string{"get", "bad-uuid"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid UUID")
}
