# CLI Commands Reference

This document provides detailed reference for all CLI commands.

## Command Overview

| Command | Description |
|---------|-------------|
| `stats` | Show combined usage statistics |
| `usage` | Show per-agent usage statistics |
| `info` | Display loaded configuration and status |

## Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--config` | `-c` | Path to config file | `~/.agent-usage/config.toml` |

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

Shows aggregated statistics across all enabled agents. This command **automatically syncs** all enabled agents before displaying the results.

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
|------|-------|-------------|
| `--debug` | `-d` | Show debug output |

### Description

Shows detailed statistics for a single agent. This command **automatically syncs** the specified agent before displaying the results.

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

Display loaded configuration and status.

### Usage

```bash
agent-usage info
```

### Description

Shows:
- Configuration (agents)
- Last sync time per agent

### Example Output

```
=== Configuration ===
  Agents:
    Codex: true
    Claude: true

=== Last Sync ===
  Codex: 2026-02-26 05:50:52
  Claude: 2026-02-26 05:50:52
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
     Tokens: 100.0K (in: 80.0K, out: 20.0K, cache: 0/0)
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
