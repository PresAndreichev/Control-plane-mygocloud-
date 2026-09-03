package commands

import (
	"bytes"
	"os"
	"testing"

	"control-plane/cli/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := config.Init("")
	require.NoError(t, err)

	cmd := newLoginCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--endpoint", "http://test.example.com:9090"})

	err = cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "http://test.example.com:9090")

	// Verify it was saved
	cfg := config.Get()
	assert.Equal(t, "http://test.example.com:9090", cfg.Endpoint)
}
