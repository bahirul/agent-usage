package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Agents    AgentsConfig `mapstructure:"agents"`
	Database  string       `mapstructure:"database"`
	AutoSync  bool         `mapstructure:"autosync"`
}

// AgentsConfig contains the enabled agents
type AgentsConfig struct {
	Codex      bool `mapstructure:"codex"`
	ClaudeCode bool `mapstructure:"claude"`
}

// LoadConfig loads configuration from the specified path or default location
func LoadConfig(configPath string) (*Config, error) {
	viperInstance := viper.New()

	// If custom config path provided, use it directly
	if configPath != "" {
		viperInstance.SetConfigFile(configPath)
		if err := viperInstance.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	} else {
		// Try default location: ~/.agent-usage/config.toml
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		defaultPath := filepath.Join(homeDir, ".agent-usage", "config.toml")
		viperInstance.SetConfigFile(defaultPath)

		if err := viperInstance.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config at default location %s: %w", defaultPath, err)
		}
	}

	var cfg Config
	if err := viperInstance.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// GetDatabasePath returns the database path, using default if not specified
func (c *Config) GetDatabasePath() string {
	if c.Database != "" {
		return c.Database
	}
	// Default: ~/.agent-usage/usage.db
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.agent-usage/usage.db"
	}
	return filepath.Join(homeDir, ".agent-usage", "usage.db")
}
