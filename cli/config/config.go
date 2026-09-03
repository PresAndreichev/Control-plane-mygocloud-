package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const DefaultEndpoint = "http://localhost:8080"

type Config struct {
	Endpoint string `mapstructure:"endpoint"`
	Token    string `mapstructure:"token"`
}

var (
	cfgFile string
	v       *viper.Viper
)

func Init(configPath string) error {
	v = viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if configPath != "" {
		// Ensure parent directory exists
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		v.SetConfigFile(configPath)
		cfgFile = configPath
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir := filepath.Join(home, ".mygocloud")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		v.AddConfigPath(dir)
		cfgFile = filepath.Join(dir, "config.yaml")
	}

	v.SetDefault("endpoint", DefaultEndpoint)

	if err := v.ReadInConfig(); err != nil {
		// Handle both viper's wrapped error and raw os errors
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func Save(c *Config) error {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	v.Set("endpoint", endpoint)
	v.Set("token", c.Token)

	if cfgFile != "" {
		return v.WriteConfigAs(cfgFile)
	}
	return v.WriteConfig()
}

func Get() *Config {
	endpoint := v.GetString("endpoint")
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Config{
		Endpoint: endpoint,
		Token:    v.GetString("token"),
	}
}

func CfgFile() string {
	return cfgFile
}
