# Agent Usage Tracker

A Go CLI tool to track AI coding agent usage (Codex, Claude Code). Monitor your agent usage statistics, sync sessions, and analyze productivity metrics.

## Features

- **Multi-agent Support**: Track usage for Codex and Claude Code agents
- **Session Tracking**: Automatically parses and stores session data from agent log files
- **Usage Statistics**: View daily, weekly, and monthly usage stats
- **Auto-sync**: Automatically sync sessions before viewing stats (configurable)
- **Cost Tracking**: Estimated costs based on token usage
- **Project-level Insights**: See which projects use the most agent time

## Installation

```bash
# Clone and build
git clone https://github.com/ari/agent-usage.git
cd agent-usage
make build

# Or use Go directly
go build -o agent-usage .

# Install globally
make install
# or
go install .
```

## Build Commands

```bash
make build        # Build to build/agent-usage
make build/osx    # Build for macOS
make build/linux  # Build for Linux
make build/windows # Build for Windows
make test         # Run tests
make clean        # Clean build artifacts
```

## Quick Start

1. Create a config file at `~/.agent-usage/config.toml`:

```toml
[agents]
codex = true
claude_code = true

# Optional: auto-sync before showing stats
autosync = false
```

2. Run your first sync:

```bash
./agent-usage sync all        # Sync all enabled agents
./agent-usage sync codex       # Sync only Codex
./agent-usage sync claude      # Sync only Claude
```

3. View usage statistics:

```bash
./agent-usage stats            # Combined stats for all agents
./agent-usage usage codex     # Codex-specific stats
./agent-usage usage claude    # Claude-specific stats
```

## Commands

| Command | Description |
|---------|-------------|
| `./agent-usage sync <agent>` | Sync sessions from agent directory |
| `./agent-usage sync all` | Sync all enabled agents |
| `./agent-usage stats [period]` | Show combined usage stats |
| `./agent-usage usage <agent> [period]` | Show per-agent stats |
| `./agent-usage info` | Show loaded configuration |
| `./agent-usage --help` | Show help |

### Period Options

For `stats` and `usage` commands, specify a time period:
- `day` - Last 24 hours (default)
- `week` - Last 7 days
- `month` - Last 30 days

## Configuration

Config file location: `~/.agent-usage/config.toml`

```toml
[agents]
codex = true           # Enable Codex tracking
claude_code = true     # Enable Claude tracking

# Database path (optional, defaults to ~/.agent-usage/usage.db)
database = ""

# Auto-sync before showing stats (default: false)
autosync = false
```

### Custom Config Path

Use `-c` or `--config` flag to specify a custom config:

```bash
./agent-usage -c /path/to/config.toml stats
```

## Output Examples

### Stats Command

```
Combined Usage Statistics - Day
============================================================

Per-Agent Breakdown
  Agent        Sessions        Time   Tokens (in/out/cached)
------------------------------------------------------------
  Claude             15      2.5h        1.2M/450K/200K
  Codex              12      1.8h        800K/300K/0
------------------------------------------------------------
  Total              27      4.3h        2.0M/750K/200K

Summary
  Total Sessions:      27
  Total Session Time:  4.3h
  Total Tokens:        2.0M (in: 2.0M, out: 750K, cached: 200K)
  Unique Projects:     8
  Last Sync:           2026-02-26 10:30

Top Models (by session count)
  1. claude-sonnet-4-20250514 - 10 sessions
  2. claude-3-opus - 5 sessions
  3. o3 - 8 sessions
```

### Usage Command

```
Claude Usage Statistics - Week
============================================================

Last Session
  ID:         claude-session-abc123
  Start:      2026-02-25 14:30
  Project:    /Users/user/project
  Model:      claude-sonnet-4-20250514
  Provider:   anthropic
  End:        2026-02-25 15:45
  Duration:   1.2h
  Tokens:     450K (in: 300K, out: 150K, cached: 50K)

Summary
  Total Sessions:     15
  Total Session Time: 2.5h
  Total Tokens:       1.2M (in: 800K, out: 400K, cached: 200K)
  Total Messages:     450
  Last Sync:          2026-02-26 10:30

Top Models (by session count)
  1. claude-sonnet-4-20250514 - 10 sessions
  2. claude-3-opus - 5 sessions
```

## Debug Mode

Use `--debug` or `-d` flag to see detailed debug output:

```bash
./agent-usage usage codex day --debug
```

Debug output shows:
- SQL queries being executed
- Raw session data
- Time filters being applied (start/end timestamps)

## Database

The tool uses SQLite to store session data:

- **Location**: `~/.agent-usage/usage.db` (or custom path)
- **Tables**:
  - `sessions` - Individual agent sessions
  - `messages` - Messages within sessions
  - `tool_calls` - Tool calls made during sessions
  - `metadata` - Key-value store for sync timestamps

## How It Works

1. **Session Discovery**: The tool walks through agent session directories to find JSONL log files
2. **Parsing**: Each session file is parsed to extract:
   - Session ID, project path, model
   - Start/end timestamps
   - Token usage (input, output, cached)
   - Cost estimation
3. **Storage**: Parsed sessions are stored in SQLite database
4. **Analysis**: Stats queries aggregate data by time period

### Session File Locations

- **Codex**: `~/.codex/sessions/*.jsonl`
- **Claude**: `~/.claude/projects/**/*.jsonl`

## Development

```bash
# Run tests
go test ./...

# Build
go build -o agent-usage .

# Run with verbose output
go run . stats -d
```

## License

MIT License
