# agent-usage

A Go CLI tool to track AI coding agent usage (Codex, Claude Code). Monitor your agent usage statistics, sync sessions, and analyze productivity metrics.

## Features

- **Multi-agent Support**: Track usage for Codex and Claude Code agents
- **Session Tracking**: Automatically parses and stores session data from agent log files
- **Usage Statistics**: View daily, weekly, and monthly usage stats
- **Auto-sync**: Automatically sync sessions before viewing stats
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
claude = true
```

2. View usage statistics (syncs automatically):

```bash
./agent-usage stats            # Combined stats for all agents
./agent-usage usage codex     # Codex-specific stats
./agent-usage usage claude    # Claude-specific stats
```

## Commands

| Command | Description |
|---------|-------------|
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
# Database path (optional, defaults to ~/.agent-usage/usage.db)
database = ""

[agents]
codex = true           # Enable Codex tracking
claude_code = true     # Enable Claude tracking
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
  Agent          Sessions         Time Tokens (in/out/crea/read)   Messages
  --------------------------------------------------------------------------
  Claude               31        12.2h 1.8M/246.4K/0/0       4434
  Codex                 2         6.0m 1.3M/17.6K/1.2M/0         12
  --------------------------------------------------------------------------
  Total                33        12.3h 3.1M/264.0K/1.2M/0       4446

Summary
  Total Sessions:      33
  Total Session Time:  12.3h
  Total Tokens:        129.2M (in: 3.1M, out: 264.0K, cache: 1.2M/0)
  Total Messages:      4446
  Unique Projects:     7
  Last Sync:          2026-02-26 12:30:12

Top Models (by session count)
  1. MiniMax-M2.5 - 27 sessions
  2. gpt-5.2-codex - 2 sessions
  3. claude-sonnet-4-6 - 1 sessions

Last 5 Sessions
  1. Feb 26 11:56 Codex | gpt-5.2-codex | my-project | 6.0m | 1.3M (cache: 1.2M/0, msgs: 7)
  2. Feb 26 11:56 Codex | gpt-5.2-codex | my-project | 1s | 2.6K (cache: 0/0, msgs: 5)
  3. Feb 26 05:54 Claude | MiniMax-M2.5 | agent-usage | 7.7m | 1.6M (cache: 0/0, msgs: 111)
  4. Feb 26 05:08 Claude | MiniMax-M2.5 | api-service | 7.0m | 739.5K (cache: 0/0, msgs: 309)
  5. Feb 26 04:51 Claude | MiniMax-M2.5 | web-app | 1.9m | 1.3M (cache: 0/0, msgs: 66)

============================================================
```

### Usage Command

```
Claude Usage Statistics - Day
============================================================

Last Session
  ID:         16667d34-9b1a-4490-a9a4-35921f53fd56
  Start:      2026-02-26 05:54:37
  Project:    /Users/developer/agent-usage
  Model:      MiniMax-M2.5
  Provider:   anthropic
  End:        2026-02-26 06:02:18
  Duration:   7.7m
  Tokens:     1.6M (in: 34.2K, out: 5.5K, cache: 0/0)
  Messages:   111

Summary
  Total Sessions:     31
  Total Session Time: 12.2h
  Total Tokens:       127.9M (in: 1.8M, out: 246.4K, cache: 0/0)
  Total Messages:     4434
  Last Sync:         2026-02-26 12:30:16

Top Models (by session count)
  1. MiniMax-M2.5 - 27 sessions
  2. claude-sonnet-4-6 - 1 sessions
  3. claude-opus-4-6 - 1 sessions

============================================================
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
