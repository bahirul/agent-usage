package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// CodexSession represents a parsed Codex session
type CodexSession struct {
	ID          string
	ProjectPath string
	Model       string
	Provider    string
	StartedAt   time.Time
	EndedAt     *time.Time
	Tokens      TokenUsage
	Cost        float64
	Messages    []CodexMessage
	ToolCalls   []CodexToolCall
}

// TokenUsage represents token usage for a session
type TokenUsage struct {
	Input     int
	Output    int
	Cached    int
	Reasoning int
	Total     int
}

// CodexMessage represents a message in a Codex session
type CodexMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// CodexToolCall represents a tool call in a Codex session
type CodexToolCall struct {
	ToolName  string
	Arguments string
	Result    string
	Timestamp time.Time
}

// jsonlEntry represents a generic JSONL entry
type jsonlEntry struct {
	Payload    json.RawMessage `json:"payload"`
	Timestamp  string          `json:"timestamp"`
	Type       string          `json:"type"`
}

// responseMessage represents a message in response_item
type responseMessage struct {
	Type string `json:"type"`
	Role string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"content"`
}

// ParseCodexSession parses a Codex session JSONL file
func ParseCodexSession(path string) (*CodexSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	session := &CodexSession{
		Messages:  make([]CodexMessage, 0),
		ToolCalls: make([]CodexToolCall, 0),
	}

	lines := splitLines(string(data))
	var firstTimestamp, lastTimestamp time.Time

	for _, line := range lines {
		line = trim(line)
		if line == "" {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			ts, _ = time.Parse(time.RFC3339, entry.Timestamp)
		}

		if firstTimestamp.IsZero() {
			firstTimestamp = ts
		}
		lastTimestamp = ts

		switch entry.Type {
		case "session_meta":
			// Parse payload directly as a map to extract fields
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err == nil {
				if id, ok := payload["id"].(string); ok {
					session.ID = id
				}
				if cwd, ok := payload["cwd"].(string); ok {
					session.ProjectPath = cwd
				}
				if modelProvider, ok := payload["model_provider"].(string); ok {
					session.Provider = modelProvider
				}
				if originator, ok := payload["originator"].(string); ok {
					session.Model = originator
				}
				if ts, ok := payload["timestamp"].(string); ok {
					startedAt, _ := time.Parse(time.RFC3339Nano, ts)
					if startedAt.IsZero() {
						startedAt, _ = time.Parse(time.RFC3339, ts)
					}
					session.StartedAt = startedAt
				}
			}

		case "turn_context":
			// Parse turn_context to get the actual model name (overrides session_meta.originator)
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err == nil {
				if model, ok := payload["model"].(string); ok {
					session.Model = model
				}
			}

		case "response_item":
			var msg responseMessage
			if err := json.Unmarshal(entry.Payload, &msg); err == nil {
				if msg.Type == "message" {
					content := extractMessageContent(msg.Content)
					if content != "" {
						session.Messages = append(session.Messages, CodexMessage{
							Role:      msg.Role,
							Content:   content,
							Timestamp: ts,
						})
					}
				}
			}

		case "event_msg":
			var event map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &event); err == nil {
				// Check for tool_use events
				if eventType, ok := event["type"].(string); ok && eventType == "tool_use" {
					toolName, _ := event["name"].(string)
					args, _ := json.Marshal(event["input"])
					toolCall := CodexToolCall{
						ToolName:  toolName,
						Arguments: string(args),
						Timestamp: ts,
					}

					// Look for result in following entries (simplified - just store the call)
					session.ToolCalls = append(session.ToolCalls, toolCall)
				}

				// Check for token_count events
				if eventType, ok := event["type"].(string); ok && eventType == "token_count" {
					if info, ok := event["info"].(map[string]interface{}); ok {
						if usage, ok := info["total_token_usage"].(map[string]interface{}); ok {
							if v, ok := usage["input_tokens"].(float64); ok {
								session.Tokens.Input = int(v)
							}
							if v, ok := usage["cached_input_tokens"].(float64); ok {
								session.Tokens.Cached = int(v)
							}
							if v, ok := usage["output_tokens"].(float64); ok {
								session.Tokens.Output = int(v)
							}
							if v, ok := usage["reasoning_output_tokens"].(float64); ok {
								session.Tokens.Reasoning = int(v)
							}
							if v, ok := usage["total_tokens"].(float64); ok {
								session.Tokens.Total = int(v)
							}
						}
					}
				}
			}
		}
	}

	session.StartedAt = firstTimestamp
	session.EndedAt = &lastTimestamp

	// Estimate tokens only if not already set from token_count event
	if session.Tokens.Total == 0 {
		estimateTokens(session)
	} else {
		// Calculate cost using actual token counts
		// $3/million input, $15/million output
		session.Cost = float64(session.Tokens.Input)*3/1_000_000 + float64(session.Tokens.Output)*15/1_000_000
	}

	return session, nil
}

// FindLatestSession finds the most recent session file
func FindLatestSession(sessionsDir string) (string, error) {
	var matches []string

	err := filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".jsonl" {
			matches = append(matches, path)
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk sessions directory: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no session files found in %s", sessionsDir)
	}

	// Sort by modification time, newest first
	sort.Slice(matches, func(i, j int) bool {
		iInfo, _ := os.Stat(matches[i])
		jInfo, _ := os.Stat(matches[j])
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	return matches[0], nil
}

// FindSessionByID finds a session file by ID
func FindSessionByID(sessionsDir, sessionID string) (string, error) {
	var found string

	err := filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".jsonl" {
			base := filepath.Base(path)
			if base == "rollout-"+sessionID+".jsonl" || base == sessionID+".jsonl" {
				found = path
				return filepath.SkipAll
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to search sessions: %w", err)
	}

	if found == "" {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	return found, nil
}

// GetDefaultSessionsDir returns the default Codex sessions directory
func GetDefaultSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codex", "sessions")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trim(s string) string {
	return s
}

func extractMessageContent(content []struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}) string {
	var result string
	for _, c := range content {
		if c.Type == "output_text" || c.Type == "input_text" {
			result += c.Text
		}
	}
	return result
}

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

	// Cost estimation (approximate)
	// $3/input million tokens, $15/output million tokens
	session.Cost = float64(session.Tokens.Input)*3/1_000_000 + float64(session.Tokens.Output)*15/1_000_000
}
