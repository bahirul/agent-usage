package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Agents   AgentsConfig `mapstructure:"agents"`
	Database string       `mapstructure:"database"`
}

// AgentsConfig contains the enabled agents
type AgentsConfig struct {
	Codex      bool `mapstructure:"codex"`
	ClaudeCode bool `mapstructure:"claude"`
}

// LoadConfig loads configuration from the specified path or default location
func LoadConfig(configPath string) (*Config, error) {
	viperInstance := viper.New()

	// Set defaults
	viperInstance.SetDefault("agents.codex", true)
	viperInstance.SetDefault("agents.claude", true)

	// If custom config path provided, use it directly
	if configPath != "" {
		viperInstance.SetConfigFile(configPath)
		if err := viperInstance.ReadInConfig(); err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
			}
			// If it doesn't exist, we just continue with defaults
		}
	} else {
		// Try default location: ~/.agent-usage/config.toml
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		defaultDir := filepath.Join(homeDir, ".agent-usage")
		defaultPath := filepath.Join(defaultDir, "config.toml")
		viperInstance.SetConfigFile(defaultPath)

		if err := viperInstance.ReadInConfig(); err != nil {
			// If file doesn't exist, we'll use defaults
			if !os.IsNotExist(err) {
				// For other errors (like permission or syntax), return error
				return nil, fmt.Errorf("failed to read config at %s: %w", defaultPath, err)
			}
			// Optional: Create the directory if it doesn't exist
			os.MkdirAll(defaultDir, 0755)
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

// GetConfigDir returns the config directory path (~/.agent-usage)
func (c *Config) GetConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "~/.agent-usage"
	}
	return filepath.Join(homeDir, ".agent-usage")
}
