# Database Schema

Agent Usage Tracker uses SQLite to store session data. This document details the database schema and relationships.

## Database Location

- Default: `~/.agent-usage/usage.db`
- Custom: Configurable via `database` key in config.toml

## Tables

### sessions

The main table storing agent session data.

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PRIMARY KEY | Auto-increment ID |
| external_id | TEXT UNIQUE | Agent's session ID |
| source | TEXT NOT NULL | Agent type: "codex" or "claude" |
| project_path | TEXT | Working directory for the session |
| model | TEXT | AI model used |
| provider | TEXT | Model provider (e.g., "anthropic", "openai") |
| started_at | INTEGER NOT NULL | Unix timestamp of session start |
| ended_at | INTEGER | Unix timestamp of session end |
| input_tokens | INTEGER DEFAULT 0 | Input token count |
| output_tokens | INTEGER DEFAULT 0 | Output token count |
| cached_tokens | INTEGER DEFAULT 0 | Cached token count |
| reasoning_tokens | INTEGER DEFAULT 0 | Reasoning token count |
| total_tokens | INTEGER DEFAULT 0 | Total tokens (input + output + cached) |
| cost | REAL DEFAULT 0 | Estimated cost in USD |

**Indexes:**
- `idx_sessions_external_id` on `external_id` (for fast duplicate checking)

### messages

Stores individual messages within sessions.

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PRIMARY KEY | Auto-increment ID |
| session_id | INTEGER NOT NULL | Foreign key to sessions.id |
| role | TEXT NOT NULL | Message role: "user", "assistant", "system" |
| content | TEXT | Message content |
| timestamp | INTEGER NOT NULL | Unix timestamp |

**Indexes:**
- `idx_messages_session_id` on `session_id` (for fast lookups)

**Foreign Keys:**
- `session_id` references `sessions(id)`

### tool_calls

Stores tool invocations during sessions.

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PRIMARY KEY | Auto-increment ID |
| session_id | INTEGER NOT NULL | Foreign key to sessions.id |
| tool_name | TEXT NOT NULL | Name of the tool invoked |
| arguments | TEXT | JSON arguments passed to tool |
| result | TEXT | Tool execution result |
| timestamp | INTEGER NOT NULL | Unix timestamp |

**Indexes:**
- `idx_tool_calls_session_id` on `session_id` (for fast lookups)

**Foreign Keys:**
- `session_id` references `sessions(id)`

### metadata

Key-value store for application metadata (e.g., last sync times).

| Column | Type | Description |
|--------|------|-------------|
| key | TEXT PRIMARY KEY | Metadata key |
| value | TEXT | Metadata value |
| updated_at | INTEGER | Unix timestamp of last update |

**Keys stored:**
- `last_sync_codex` - Unix timestamp of last Codex sync
- `last_sync_claude` - Unix timestamp of last Claude sync

## Relationships

```
┌──────────────┐       ┌──────────────┐
│   sessions   │       │   messages   │
├──────────────┤       ├──────────────┤
│ id (PK)      │◄──────│ session_id   │
│ external_id  │       │ (FK)         │
│ source       │       │ role         │
│ ...          │       │ content      │
└──────────────┘       │ timestamp    │
       │               └──────────────┘
       │
       │               ┌──────────────┐
       │               │  tool_calls  │
       ├──────────────►│ session_id   │
       │               │ (FK)         │
       │               │ tool_name    │
       │               │ arguments    │
       │               │ result       │
       │               └──────────────┘
       │
       │
       │               ┌──────────────┐
       └──────────────►│   metadata   │
                       │ key (PK)     │
                       │ value        │
                       │ updated_at   │
                       └──────────────┘
```

## Example Queries

### Get all sessions for an agent in the last 24 hours

```sql
SELECT * FROM sessions
WHERE source = 'claude'
AND started_at >= strftime('%s', 'now', '-1 day')
ORDER BY started_at DESC;
```

### Get aggregated stats for a period

```sql
SELECT
    COUNT(*) as session_count,
    SUM(input_tokens) as total_input,
    SUM(output_tokens) as total_output,
    SUM(total_tokens) as total_tokens,
    SUM(cost) as total_cost,
    SUM(CASE WHEN ended_at > started_at THEN ended_at - started_at ELSE 0 END) as total_time
FROM sessions
WHERE source = 'codex'
AND started_at >= 1739000000;
```

### Get top models by usage

```sql
SELECT model, COUNT(*) as session_count
FROM sessions
WHERE started_at >= strftime('%s', 'now', '-7 days')
GROUP BY model
ORDER BY session_count DESC
LIMIT 5;
```

### Get last sync time for an agent

```sql
SELECT value FROM metadata
WHERE key = 'last_sync_claude';
```

## Database Migration

Migrations are handled automatically in `db.go`:

```go
func (db *DB) migrate() error {
    schema := `
    CREATE TABLE IF NOT EXISTS sessions (...);
    CREATE TABLE IF NOT EXISTS messages (...);
    CREATE TABLE IF NOT EXISTS tool_calls (...);
    CREATE TABLE IF NOT EXISTS metadata (...);
    -- indexes
    `
    _, err := db.db.Exec(schema)
    return err
}
```

The `IF NOT EXISTS` clause ensures tables are created only if they don't exist, allowing for seamless upgrades.

## Cost Calculation

### Claude Sessions

Uses Anthropic pricing (as of 2025):

| Token Type | Price per Million |
|------------|-------------------|
| Input | $3.00 |
| Output | $15.00 |
| Cache Creation | $3.75 |
| Cache Read | $0.30 |

```go
func calculateClaudeCost(tokens TokenUsage) float64 {
    inputCost := float64(tokens.Input) * 3.0 / 1_000_000
    cacheCreationCost := float64(tokens.Cached/2) * 3.75 / 1_000_000
    cacheReadCost := float64(tokens.Cached/2) * 0.30 / 1_000_000
    outputCost := float64(tokens.Output) * 15.0 / 1_000_000
    return inputCost + cacheCreationCost + cacheReadCost + outputCost
}
```

### Codex Sessions

Estimated pricing:

| Token Type | Price per Million |
|------------|-------------------|
| Input | $3.00 |
| Output | $15.00 |

```go
func estimateTokens(session *CodexSession) {
    // Rough estimate: 4 chars per token
    session.Tokens.Input = inputChars / 4
    session.Tokens.Output = outputChars / 4
    session.Cost = float64(tokens.Input)*3/1_000_000 + float64(tokens.Output)*15/1_000_000
}
```

Note: Codex token counts are estimated from character counts since they're not explicitly provided in session files.
