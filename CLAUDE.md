# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Agent Usage Tracker is a Go CLI tool to track AI coding agent usage (Codex, Claude Code). It uses a standard Go project layout with Cobra for CLI commands and Viper for configuration.

## Build & Run

```bash
go build -o agent-usage .           # Build binary
./agent-usage --help                # Show help
./agent-usage info                  # Show loaded config
./agent-usage -c /path/to/config.toml info  # Use custom config
```

## Architecture

- **cmd/root.go**: Cobra root command with `--config` flag. Loads config in PersistentPreRunE before subcommands execute. Add new subcommands here.
- **internal/config/config.go**: Config loading. First checks `--config` flag, falls back to `~/.agent-usage/config.toml`. Returns error if no config found.
- **internal/tracker/tracker.go**: Placeholder types (Agent, Session, Tracker interface, UsageStats). Tracking implementation not yet built.

## Configuration

Config file format (TOML):
```toml
[agents]
codex = true
claude_code = true

# Auto-sync before showing stats/usage (default: false)
autosync = true
```

Default config path: `~/.agent-usage/config.toml`

## Commands

- `./agent-usage sync <agent>` - Sync sessions from agent directory (codex, claude)
- `./agent-usage sync all` - Sync all enabled agents at once
- `./agent-usage stats [period]` - Show combined stats (auto-syncs if autosync=true)
- `./agent-usage usage <agent> [period]` - Show per-agent stats (auto-syncs if autosync=true)

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

- **Last Sync Time**: Stats and usage output display the last sync timestamp. Shows "Never synced" if no sync has occurred.
- **Auto-sync**: When `autosync=true` in config, the `stats` and `usage` commands automatically sync before showing data.
- **Sync All**: Use `sync all` to sync all enabled agents in one command.
