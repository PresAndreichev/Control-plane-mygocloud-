package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServerAddr string
	DB         DBConfig
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8080"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     port,
			User:     getEnv("DB_USER", "controlplane"),
			Password: getEnv("DB_PASSWORD", "controlplane"),
			Database: getEnv("DB_NAME", "controlplane"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func TestConfig() *Config {
	return &Config{
		ServerAddr: "127.0.0.1:0",
		DB:         DBConfig{},
	}
}
