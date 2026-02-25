# Configuration Guide

This document explains all configuration options for Agent Usage Tracker.

## Config File Location

The tool looks for configuration in this order:

1. **Custom path** via `--config` or `-c` flag
2. **Default location**: `~/.agent-usage/config.toml`
3. **Error** if no config found

## Config File Format

```toml
# Agent Usage Tracker Configuration

[agents]
codex = true
claude_code = true

# Database configuration
database = ""

# Auto-sync behavior
autosync = false
```

## Configuration Options

### [agents]

Controls which agents to track.

| Key | Type | Description | Default |
|-----|------|-------------|---------|
| `codex` | boolean | Enable Codex tracking | `false` |
| `claude_code` | boolean | Enable Claude tracking | `false` |

At least one agent must be enabled.

### database

Custom database file path.

| Type | Description |
|------|-------------|
| string | Full path to SQLite database file |

**Default**: `~/.agent-usage/usage.db`

Examples:
```toml
# Default location
database = ""

# Custom location
database = "/Users/name/data/agent-usage.db"

# In project directory
database = "./data/usage.db"
```

### autosync

Automatically sync sessions before showing stats or usage.

| Type | Description | Default |
|------|-------------|---------|
| boolean | Auto-sync before displaying stats | `false` |

When enabled:
- `stats` command syncs all enabled agents first
- `usage` command syncs the specified agent first

Example:
```toml
autosync = true
```

## Environment Variables

Currently not supported. Use config file or `--config` flag.

## Example Configurations

### Minimal Setup

```toml
[agents]
codex = true
```

### Full Setup

```toml
[agents]
codex = true
claude_code = true

# Custom database location
database = "/Users/developer/data/agent-usage.db"

# Auto-sync before showing stats
autosync = true
```

### Development Setup

```toml
[agents]
codex = true
claude_code = true

# Use local database for development
database = "./dev.db"

# Enable auto-sync during development
autosync = true
```

## Config Loading Process

The config is loaded in `cmd/root.go` using Viper:

```go
func LoadConfig(configPath string) (*Config, error) {
    viperInstance := viper.New()

    // If custom config path provided, use it directly
    if configPath != "" {
        viperInstance.SetConfigFile(configPath)
        if err := viperInstance.ReadInConfig(); err != nil {
            return nil, fmt.Errorf("failed to read config file: %w", err)
        }
    } else {
        // Try default location: ~/.agent-usage/config.toml
        homeDir, err := os.UserHomeDir()
        defaultPath := filepath.Join(homeDir, ".agent-usage", "config.toml")
        viperInstance.SetConfigFile(defaultPath)

        if err := viperInstance.ReadInConfig(); err != nil {
            return nil, fmt.Errorf("failed to read config: %w", err)
        }
    }

    var cfg Config
    if err := viperInstance.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    return &cfg, nil
}
```

## Config Struct

```go
type Config struct {
    Agents    AgentsConfig `mapstructure:"agents"`
    Database  string       `mapstructure:"database"`
    AutoSync  bool         `mapstructure:"autosync"`
}

type AgentsConfig struct {
    Codex      bool `mapstructure:"codex"`
    ClaudeCode bool `mapstructure:"claude"`
}
```

Note: The TOML key `claude` maps to `ClaudeCode` in the struct.

## Troubleshooting

### "failed to load config" error

Make sure your config file exists at `~/.agent-usage/config.toml` or use `--config` flag:

```bash
./agent-usage -c /path/to/config.toml stats
```

### Config file not found

Create the config file:

```bash
mkdir -p ~/.agent-usage
touch ~/.agent-usage/config.toml
```

### Invalid configuration

Check TOML syntax. Common issues:
- Missing section brackets `[agents]`
- Boolean values must be `true` or `false` (lowercase)
- No trailing commas

## Auto-Sync Behavior

When `autosync = true`:

1. **stats command**:
   - Before querying stats, calls `runSyncAll()`
   - Syncs all agents enabled in config
   - Then displays combined stats

2. **usage command**:
   - Before querying stats for an agent, calls `runSync(agentName)`
   - Syncs only the specified agent
   - Then displays agent-specific stats

This ensures you always see the latest data without manual syncing.

## Database Path Resolution

The database path is resolved in this order:

1. Use `database` value from config if provided
2. Default to `~/.agent-usage/usage.db`

```go
func (c *Config) GetDatabasePath() string {
    if c.Database != "" {
        return c.Database
    }
    // Default: ~/.agent-usage/usage.db
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".agent-usage", "usage.db")
}
```
