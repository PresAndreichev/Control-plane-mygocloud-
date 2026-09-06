package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any env vars that might interfere
	os.Unsetenv("SERVER_ADDR")
	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")
	os.Unsetenv("DB_SSLMODE")
	os.Unsetenv("RABBITMQ_URL")

	cfg := Load()

	assert.Equal(t, ":8080", cfg.ServerAddr)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, 5432, cfg.DB.Port)
	assert.Equal(t, "controlplane", cfg.DB.User)
	assert.Equal(t, "controlplane", cfg.DB.Password)
	assert.Equal(t, "controlplane", cfg.DB.Database)
	assert.Equal(t, "disable", cfg.DB.SSLMode)
	assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.RabbitMQ.URL)
}

func TestLoad_CustomEnv(t *testing.T) {
	os.Setenv("SERVER_ADDR", ":9090")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "admin")
	os.Setenv("DB_PASSWORD", "secret")
	os.Setenv("DB_NAME", "prod")
	os.Setenv("DB_SSLMODE", "require")
	os.Setenv("RABBITMQ_URL", "amqp://admin:secret@mq.example.com:5672/")
	defer func() {
		os.Unsetenv("SERVER_ADDR")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")
		os.Unsetenv("RABBITMQ_URL")
	}()

	cfg := Load()

	assert.Equal(t, ":9090", cfg.ServerAddr)
	assert.Equal(t, "db.example.com", cfg.DB.Host)
	assert.Equal(t, 5433, cfg.DB.Port)
	assert.Equal(t, "admin", cfg.DB.User)
	assert.Equal(t, "secret", cfg.DB.Password)
	assert.Equal(t, "prod", cfg.DB.Database)
	assert.Equal(t, "require", cfg.DB.SSLMode)
	assert.Equal(t, "amqp://admin:secret@mq.example.com:5672/", cfg.RabbitMQ.URL)
}

func TestDBConfig_DSN(t *testing.T) {
	cfg := DBConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "user",
		Password: "pass",
		Database: "db",
		SSLMode:  "disable",
	}
	expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
	assert.Equal(t, expected, cfg.DSN())
}

func TestTestConfig(t *testing.T) {
	cfg := TestConfig()
	assert.Equal(t, "127.0.0.1:0", cfg.ServerAddr)
	assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.RabbitMQ.URL)
}
