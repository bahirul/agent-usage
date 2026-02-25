# Session Parsing

Agent Usage Tracker parses session log files from Codex and Claude Code agents. This document details how each agent's session files are parsed.

## Session File Locations

| Agent | Default Directory | File Pattern |
|-------|------------------|--------------|
| Codex | `~/.codex/sessions/` | `*.jsonl` |
| Claude | `~/.claude/projects/` | `**/*.jsonl` |

## Session File Format

Both agents use JSONL (JSON Lines) format - one JSON object per line.

## Codex Session Parsing

### File Structure

A Codex session file contains various entry types:

```json
{"type": "session_meta", "timestamp": "2026-02-26T10:30:00Z", "payload": {...}}
{"type": "turn_context", "timestamp": "2026-02-26T10:30:01Z", "payload": {...}}
{"type": "response_item", "timestamp": "2026-02-26T10:30:02Z", "payload": {...}}
{"type": "event_msg", "timestamp": "2026-02-26T10:30:03Z", "payload": {...}}
```

### Entry Types

#### session_meta

Contains session metadata:

```json
{
  "type": "session_meta",
  "payload": {
    "id": "session-abc123",
    "cwd": "/Users/user/project",
    "model_provider": "openai",
    "originator": "o3",
    "timestamp": "2026-02-26T10:30:00Z"
  }
}
```

Extracted fields:
- `id` → Session ID
- `cwd` → Project path
- `model_provider` → Provider
- `originator` → Model name

#### turn_context

Contains the actual model used for the session (overrides `originator` from session_meta):

```json
{
  "type": "turn_context",
  "payload": {
    "model": "o3",
    ...
  }
}
```

#### response_item

Contains messages exchanged:

```json
{
  "type": "response_item",
  "payload": {
    "type": "message",
    "role": "assistant",
    "content": [
      {"type": "output_text", "text": "Hello!"},
      {"type": "input_text", "text": "Hi"}
    ]
  }
}
```

Extracted fields:
- Role (user/assistant/developer)
- Content (text from output_text and input_text types)
- Stored in messages table

#### event_msg

Contains tool use events:

```json
{
  "type": "event_msg",
  "payload": {
    "type": "tool_use",
    "name": "read_file",
    "input": {"path": "main.go"}
  }
}
```

Extracted fields:
- Tool name
- Arguments (JSON)
- Stored in tool_calls table

### Token Estimation

Codex session files don't contain explicit token counts. The parser estimates tokens using character counts:

```go
func estimateTokens(session *CodexSession) {
    var inputChars, outputChars int

    for _, msg := range session.Messages {
        if msg.Role == "developer" || msg.Role == "user" {
            inputChars += len(msg.Content)
        } else {
            outputChars += len(msg.Content)
        }
    }

    // Rough estimate: 4 chars per token
    session.Tokens.Input = inputChars / 4
    session.Tokens.Output = outputChars / 4
    session.Tokens.Total = session.Tokens.Input + session.Tokens.Output

    // Cost: $3/million input, $15/million output
    session.Cost = float64(session.Tokens.Input)*3/1_000_000 +
                   float64(session.Tokens.Output)*15/1_000_000
}
```

## Claude Session Parsing

### File Structure

Claude session files have a different structure:

```json
{"type": "assistant", "timestamp": "2026-02-26T10:30:00Z", "sessionId": "abc123", "cwd": "/Users/user/project", "message": {...}}
{"type": "system", "timestamp": "2026-02-26T10:30:01Z", "sessionId": "abc123", "system": {...}}
```

### Entry Types

#### assistant

Contains message data with token usage:

```json
{
  "type": "assistant",
  "timestamp": "2026-02-26T10:30:00Z",
  "sessionId": "claude-session-xyz",
  "cwd": "/Users/user/project",
  "message": {
    "model": "claude-sonnet-4-20250514",
    "role": "assistant",
    "usage": {
      "input_tokens": 1000,
      "output_tokens": 500,
      "cache_creation_input_tokens": 100,
      "cache_read_input_tokens": 200
    }
  }
}
```

Extracted fields:
- `sessionId` → Session ID
- `cwd` → Project path
- `message.model` → Model name
- `message.usage` → Token counts (input, output, cached)
- Timestamps from entry

#### system

Contains session metadata and duration:

```json
{
  "type": "system",
  "timestamp": "2026-02-26T10:30:00Z",
  ...
}
```

### Token Tracking

Claude sessions include explicit token usage:

```go
session.Tokens.Input += entry.Message.Usage.InputTokens
session.Tokens.Output += entry.Message.Usage.OutputTokens
session.Tokens.Cached += entry.Message.Usage.CacheCreationInputTokens +
                         entry.Message.Usage.CacheReadInputTokens
session.Tokens.Total = session.Tokens.Input + session.Tokens.Output + session.Tokens.Cached
```

### Cost Calculation

Uses Anthropic pricing:

```go
func calculateClaudeCost(tokens TokenUsage) float64 {
    inputTokens := tokens.Input
    cacheCreationTokens := tokens.Cached / 2
    cacheReadTokens := tokens.Cached / 2
    outputTokens := tokens.Output

    inputCost := float64(inputTokens) * 3.0 / 1_000_000
    cacheCreationCost := float64(cacheCreationTokens) * 3.75 / 1_000_000
    cacheReadCost := float64(cacheReadTokens) * 0.30 / 1_000_000
    outputCost := float64(outputTokens) * 15.0 / 1_000_000

    return inputCost + cacheCreationCost + cacheReadCost + outputCost
}
```

## Parsing Flow

### Step 1: Read File

```go
data, err := os.ReadFile(path)
if err != nil {
    return nil, fmt.Errorf("failed to read file: %w", err)
}
```

### Step 2: Split into Lines

```go
lines := splitLines(string(data))
for _, line := range lines {
    // Parse each line as JSON
}
```

### Step 3: Parse Each Entry

For each JSON line:
1. Unmarshal into appropriate struct
2. Extract timestamp
3. Extract type-specific fields
4. Accumulate totals (tokens, messages, tool calls)

### Step 4: Finalize Session

- Set start time to first entry timestamp
- Set end time to last entry timestamp
- Calculate total tokens
- Calculate cost

## Error Handling

The parser is resilient to malformed entries:

```go
var entry jsonlEntry
if err := json.Unmarshal([]byte(line), &entry); err != nil {
    continue  // Skip malformed lines
}
```

This allows parsing to continue even if individual entries are corrupted.

## Duplicate Detection

Before storing, sessions are checked for duplicates:

```go
existing, err := t.db.GetSessionByExternalID(ctx, session.ID)
if existing != nil {
    return fmt.Errorf("session %s already tracked", session.ID)
}
```

This prevents duplicate entries when syncing multiple times.
