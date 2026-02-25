# CLI Commands Reference

This document provides detailed reference for all CLI commands.

## Command Overview

| Command | Description |
|---------|-------------|
| `sync` | Sync sessions from agent directories |
| `stats` | Show combined usage statistics |
| `usage` | Show per-agent usage statistics |
| `info` | Display loaded configuration |

## Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Path to config file | `~/.agent-usage/config.toml` |

## sync

Sync sessions from agent directory into the database.

### Usage

```bash
agent-usage sync <agent>
agent-usage sync all
```

### Arguments

| Argument | Description |
|----------|-------------|
| `agent` | Agent name: `codex`, `claude`, or `all` |

### Description

The sync command:
1. Scans the agent's session directory for JSONL files
2. Parses each session file
3. Stores sessions in the SQLite database
4. Updates the last sync timestamp

### Session Directories

- Codex: `~/.codex/sessions/`
- Claude: `~/.claude/projects/`

### Examples

```bash
# Sync only Codex sessions
./agent-usage sync codex

# Sync only Claude sessions
./agent-usage sync claude

# Sync all enabled agents
./agent-usage sync all
```

### Output

```
Found 15 session files
Tracked: session-abc123 (model: o3)
Tracked: session-def456 (model: o3)
...

Sync complete: 12 new sessions tracked, 3 skipped
```

## stats

Show combined usage statistics for all agents.

### Usage

```bash
agent-usage stats [period]
```

### Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `period` | Time period: `day`, `week`, `month` | `day` |

### Description

Shows aggregated statistics across all enabled agents. If `autosync=true` in config, automatically syncs before displaying stats.

### Options

- `day` - Last 24 hours
- `week` - Last 7 days
- `month` - Last 30 days

### Examples

```bash
# Today's stats
./agent-usage stats

# This week's stats
./agent-usage stats week

# Last 30 days
./agent-usage stats month
```

### Output Fields

- **Per-Agent Breakdown**: Sessions, time, tokens per agent
- **Summary**: Total sessions, time, tokens, unique projects
- **Last Sync**: Timestamp of last sync (or "Never synced")
- **Top Models**: Most used models by session count
- **Recent Sessions**: Last N sessions with details

## usage

Show usage statistics for a specific agent.

### Usage

```bash
agent-usage usage <agent> [period]
agent-usage usage <agent> [period] [flags]
```

### Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `agent` | Agent name: `codex`, `claude` | Required |
| `period` | Time period: `day`, `week`, `month` | `day` |

### Flags

| Flag | Short | Description |
|------|------|-------------|
| `--debug` | `-d` | Show debug output |

### Description

Shows detailed statistics for a single agent. If `autosync=true` in config, automatically syncs before displaying stats.

### Examples

```bash
# Codex stats for today
./agent-usage usage codex

# Claude stats for this week
./agent-usage usage claude week

# With debug output
./agent-usage usage codex day --debug
./agent-usage usage claude -d
```

### Output Fields

- **Last Session**: Most recent session details
- **Summary**: Sessions, time, tokens, messages
- **Last Sync**: Timestamp of last sync
- **Daily/Weekly Summary**: Breakdown by day/week (for week/month periods)
- **Top Models**: Most used models

## info

Display loaded configuration.

### Usage

```bash
agent-usage info
```

### Description

Shows which agents are enabled and the config file path being used.

### Example Output

```
Config loaded:
  Codex: true
  Claude: true
```

## Debug Mode

Use the `--debug` or `-d` flag with `usage` command to see:

- SQL queries being executed
- Raw session data returned
- Time filters applied (start/end timestamps)

### Example

```bash
./agent-usage usage codex day --debug
```

### Debug Output

```
[DEBUG] Time Filter:
  Period: day
  Start:  2026-02-25 14:30:45 (timestamp: 1737825045)
  End:    2026-02-26 14:30:45 (timestamp: 1737911445)
  Agent:  codex

[DEBUG] Sessions Data (5 sessions):
  1. ID: session-abc
     Model: o3, Project: /Users/user/project
     Started: 2026-02-26 10:00:00
     Ended: 2026-02-26 10:30:00, Duration: 30m0s
     Tokens: 100.0K (in: 80.0K, out: 20.0K, cached: 0)
  ...
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Error (invalid arguments, file not found, database error) |

## Configuration Precedence

1. Command-line flag: `--config /path/to/config.toml`
2. Default location: `~/.agent-usage/config.toml`
3. Error if no config found
