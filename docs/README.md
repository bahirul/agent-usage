# Documentation Index

Welcome to the Agent Usage Tracker documentation.

## Quick Links

- [README.md](../README.md) - Quick start guide
- [CLAUDE.md](../CLAUDE.md) - Developer notes

## Technical Documentation

1. [Architecture](architecture.md) - System architecture and component overview
2. [Database Schema](database.md) - Database tables, relationships, and queries
3. [Session Parsing](session-parsing.md) - How agent session files are parsed
4. [CLI Commands](commands.md) - Detailed reference for all commands
5. [Configuration](configuration.md) - Configuration file format and options

## Quick Topics

### Want to...
- **Understand how it works**: Start with [Architecture](architecture.md)
- **Modify the database**: See [Database Schema](database.md)
- **Add support for a new agent**: Read [Session Parsing](session-parsing.md)
- **Customize the CLI**: Check [CLI Commands](commands.md)
- **Change configuration**: See [Configuration](configuration.md)

## Key Concepts

| Concept | Description |
|---------|-------------|
| Session | A single agent interaction (start to end) |
| Sync | Process of importing sessions from agent directories |
| Period | Time range for statistics (day/week/month) |
| Auto-sync | Automatic sync before showing stats |

## Code Structure

```
cmd/root.go           - CLI commands (sync, stats, usage, info)
internal/config/      - Configuration loading
internal/tracker/     - Session parsing and database
  - db.go            - Database operations
  - sqlite.go        - SQLite tracker implementation
  - codex_parser.go  - Codex session parser
  - claude_parser.go - Claude session parser
internal/ui/         - Terminal display
```

## Development

```bash
make build        # Build to build/agent-usage
make test         # Run tests
make test/verbose # Run with verbose output
make test/coverage # Run with coverage
make clean        # Clean build artifacts
make install      # Install to GOBIN
```

### Running

```bash
./build/agent-usage --help
./build/agent-usage --version
./build/agent-usage stats
```
