# Architecture Overview

Agent Usage Tracker is a Go CLI application that tracks AI coding agent usage. This document provides a high-level overview of the system architecture.

## System Components

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                            │
│                    (cmd/root.go)                            │
│  - syncCmd    - statsCmd    - usageCmd    - infoCmd       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Config Layer                            │
│              (internal/config/config.go)                    │
│  - TOML config loading                                     │
│  - Agent enablement                                        │
│  - Database path resolution                                │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Tracker Layer                            │
│             (internal/tracker/*.go)                         │
│  - SQLiteTracker: Database operations                     │
│  - Codex Parser: Parse Codex session files                │
│  - Claude Parser: Parse Claude session files               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Database Layer                          │
│               (internal/tracker/db.go)                     │
│  - SQLite database (modernc.org/sqlite)                    │
│  - Sessions, Messages, ToolCalls, Metadata tables         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      UI Layer                              │
│              (internal/ui/display.go)                      │
│  - Terminal output with ANSI colors                        │
│  - Formatted statistics display                           │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

### Sync Command Flow

```
User runs: ./agent-usage sync codex
         │
         ▼
┌─────────────────────────────────────────┐
│ cmd/root.go: syncCmd.Run()              │
│ - Read agent name from args             │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Find session files                      │
│ - Walk ~/.codex/sessions directory      │
│ - Find all *.jsonl files                │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Parse each session file                  │
│ - CodexParser.ParseCodexSession()       │
│ - Extract: ID, model, tokens, etc.      │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Store in database                       │
│ - SQLiteTracker.TrackSession()          │
│ - Insert into sessions table            │
│ - Insert messages, tool_calls          │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Update last sync time                   │
│ - SetLastSyncTime()                    │
│ - Store in metadata table               │
└─────────────────────────────────────────┘
```

### Stats Command Flow

```
User runs: ./agent-usage stats day
         │
         ▼
┌─────────────────────────────────────────┐
│ Check autosync config                   │
│ - If autosync=true, call runSyncAll()  │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Query database                          │
│ - SQLiteTracker.GetUsageStatsAll()     │
│ - GetAggregatedStatsAll()              │
│ - GetPerAgentStats()                   │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Get last sync time                     │
│ - GetLastSyncTime() for each agent     │
│ - Use most recent timestamp            │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│ Render output                           │
│ - ui.DisplayAllStats()                │
│ - Format with colors                   │
└─────────────────────────────────────────┘
```

## Key Interfaces

### Tracker Interface

```go
type Tracker interface {
    StartSession(agent Agent) (*Session, error)
    EndSession(session *Session) error
    GetUsage(agent Agent) (*UsageStats, error)
}
```

### SQLiteTracker

Implements the Tracker interface with additional methods:

```go
type SQLiteTracker struct {
    db    *DB
    debug bool
}

// Key methods
func NewSQLiteTracker(dbPath string) (*SQLiteTracker, error)
func (t *SQLiteTracker) TrackSession(ctx context.Context, session *CodexSession) error
func (t *SQLiteTracker) TrackClaudeSession(ctx context.Context, session *ClaudeSession) error
func (t *SQLiteTracker) GetUsageStats(ctx context.Context, agent Agent, period Period) (*UsageStatsData, error)
func (t *SQLiteTracker) GetUsageStatsAll(ctx context.Context, period Period) (*UsageStatsData, error)
func (t *SQLiteTracker) SetLastSyncTime(ctx context.Context, agent string, timestamp int64) error
```

## Configuration

The system uses Viper for configuration management:

1. Check for `--config` flag
2. Fall back to `~/.agent-usage/config.toml`
3. Error if no config found

## Error Handling

- Commands use `os.Exit(1)` for fatal errors
- Database errors are wrapped with `fmt.Errorf`
- UI displays errors with `ui.Error()` function (red text)

## Debug Mode

When `--debug` flag is provided:
- SQL queries are printed to stdout
- Raw session data is displayed
- Time filters show start/end timestamps
