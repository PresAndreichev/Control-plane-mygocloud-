package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit_DefaultPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	err := Init("")
	require.NoError(t, err)
	assert.Contains(t, CfgFile(), ".mygocloud/config.yaml")
}

func TestInit_CustomPath(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "myconfig.yaml")

	err := Init(customPath)
	require.NoError(t, err)
	assert.Equal(t, customPath, CfgFile())
}

func TestGet_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := Init(configPath)
	require.NoError(t, err)

	cfg := Get()
	assert.Equal(t, DefaultEndpoint, cfg.Endpoint)
	assert.Empty(t, cfg.Token)
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// First init
	err := Init(configPath)
	require.NoError(t, err)

	// Save custom values
	err = Save(&Config{
		Endpoint: "http://api.example.com",
		Token:    "secret-token",
	})
	require.NoError(t, err)

	// Verify file was written
	_, err = os.Stat(configPath)
	require.NoError(t, err, "config file should exist on disk")

	// Re-init to reload from disk
	err = Init(configPath)
	require.NoError(t, err)

	cfg := Get()
	assert.Equal(t, "http://api.example.com", cfg.Endpoint)
	assert.Equal(t, "secret-token", cfg.Token)
}

func TestGet_EmptyEndpointFallsBack(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a config file with explicitly empty endpoint
	badConfig := []byte("endpoint: \"\"\ntoken: \"\"\n")
	err := os.WriteFile(configPath, badConfig, 0644)
	require.NoError(t, err)

	// Init loads the file
	err = Init(configPath)
	require.NoError(t, err)

	cfg := Get()
	assert.Equal(t, DefaultEndpoint, cfg.Endpoint)
}

func TestSave_EmptyEndpointUsesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := Init(configPath)
	require.NoError(t, err)

	err = Save(&Config{Endpoint: ""})
	require.NoError(t, err)

	cfg := Get()
	assert.Equal(t, DefaultEndpoint, cfg.Endpoint)
}
