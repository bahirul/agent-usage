# AGENTS.md

This document provides guidance on adding support for new AI coding agents.

## Supported Agents

| Agent | Config Key | Session Parser | Log Location |
|-------|------------|----------------|---------------|
| Codex | `codex` | `codex_parser.go` | `~/.codex/sessions/*.jsonl` |
| Claude Code | `claude` | `claude_parser.go` | `~/.claude/projects/**/*.jsonl` |

## Adding a New Agent

To add support for a new agent (e.g., Cursor, Windsurf, etc.):

### 1. Define Agent Constant

Add to `internal/tracker/tracker.go`:

```go
const (
    AgentCodex      Agent = "codex"
    AgentClaudeCode Agent = "claude"
    AgentCursor     Agent = "cursor"  // New agent
)
```

### 2. Add Config Option

Update `internal/config/config.go`:

```go
type AgentsConfig struct {
    Codex      bool `mapstructure:"codex"`
    ClaudeCode bool `mapstructure:"claude"`
    Cursor     bool `mapstructure:"cursor"`  // New agent
}
```

### 3. Create Session Parser

Create `internal/tracker/cursor_parser.go`:

```go
func ParseCursorSession(path string) (*CursorSession, error) {
    // Parse the agent's session log format
}
```

Implement these types:
- `CursorSession` - represents a single session
- Parse function that reads JSONL files
- Extract: session ID, project path, model, timestamps, token usage

### 4. Add Sync Logic

Update `cmd/root.go` `runSync()` function to handle the new agent:

```go
case "cursor":
    sessionsDir = tracker.GetCursorSessionsDir()
    parseFunc = func(path string) (interface{}, error) {
        return tracker.ParseCursorSession(path)
    }
```

### 5. Update Display

Update `internal/ui/display.go` to handle the new agent name in output.

## Session Parser Requirements

Each parser must extract:

| Field | Type | Description |
|-------|------|-------------|
| ExternalID | string | Unique session identifier |
| ProjectPath | string | Working directory |
| Model | string | Model name used |
| StartedAt | int64 | Unix timestamp |
| EndedAt | *int64 | Unix timestamp (nil if active) |
| InputTokens | int | Input token count |
| OutputTokens | int | Output token count |
| CacheCreationTokens | int | Cache creation tokens |
| CacheReadTokens | int | Cache read tokens |

## Build & Run

```bash
make build                          # Build binary to build/agent-usage
./build/agent-usage --help          # Show help
./build/agent-usage info            # Show loaded config and status
./build/agent-usage -c /path/to/config.toml info  # Use custom config

# Or use Go directly
go build -o agent-usage .
```

## Architecture

- **cmd/root.go**: Cobra root command with `--config` flag. Loads config in PersistentPreRunE before subcommands execute. Add new subcommands here.
- **internal/config/config.go**: Config loading. First checks `--config` flag, falls back to `~/.agent-usage/config.toml`. Returns error if no config found.
- **internal/tracker/tracker.go**: Agent types, Session, Tracker interface, UsageStats.

## Configuration

Config file format (TOML):
```toml
[agents]
codex = true
claude = true

[sync]
autosync = true
sync_interval = 5
```

Default config path: `~/.agent-usage/config.toml`

## Commands

- `./agent-usage stats [period]` - Show combined stats (automatically syncs all agents)
- `./agent-usage usage <agent> [period]` - Show per-agent stats (automatically syncs the agent)
- `./agent-usage info` - Show configuration and status

## Testing

```bash
go test ./...              # Run all tests
go test ./internal/ui/...  # Run UI tests
go test ./internal/tracker/...  # Run tracker tests
go test -v ./...           # Run with verbose output
go test -cover ./...       # Run with coverage
```

## Debug Mode

Use the `--debug` or `-d` flag with the `usage` command to show debug output:
```bash
./agent-usage usage codex daily --debug
```

Debug output shows:
- SQL queries being executed
- Raw data returned from database
- Time filters being applied (start/end timestamps)

## Features

- **Last Sync Time**: Stats and usage output display the last sync timestamp with seconds precision. Shows "Never synced" if no sync has occurred.
- **Info Command**: Shows configuration and last sync time per agent.
