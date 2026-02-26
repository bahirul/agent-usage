package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ClaudeSession represents a parsed Claude session
type ClaudeSession struct {
	ID          string
	ProjectPath string
	Model       string
	Provider    string // "anthropic"
	StartedAt   time.Time
	EndedAt     *time.Time
	Tokens      TokenUsage
	Cost        float64
	Messages    []ClaudeMessage
}

// ClaudeMessage represents a message in a Claude session
type ClaudeMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// claudeEntry represents a JSONL entry in a Claude session file
type claudeEntry struct {
	Type       string          `json:"type"`
	Timestamp  string          `json:"timestamp"`
	SessionID  string          `json:"sessionId"`
	SessionID2 string          `json:"session_id"`
	Cwd        string          `json:"cwd"`
	Project    string          `json:"project_path"`
	Model      string          `json:"model"`
	Message    *claudeMessage  `json:"message"`
	System     *claudeSystem   `json:"system"`
	Input      string          `json:"input"`
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
}

// claudeMessage represents a message in a Claude session
type claudeMessage struct {
	Model   string          `json:"model"`
	Role    string          `json:"role"`
	Usage   *claudeUsage    `json:"usage"`
	Content json.RawMessage `json:"content"`
}

// claudeUsage represents token usage for a Claude message
type claudeUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// claudeSystem represents a system entry in a Claude session
type claudeSystem struct {
	Type string `json:"type"`
}

// ParseClaudeSession parses a Claude session JSONL file
func ParseClaudeSession(path string) (*ClaudeSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	session := &ClaudeSession{
		Provider: "anthropic",
		Messages: make([]ClaudeMessage, 0),
	}

	lines := splitLines(string(data))
	var firstTimestamp, lastTimestamp time.Time

	for _, line := range lines {
		line = trim(line)
		if line == "" {
			continue
		}

		var entry claudeEntry
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
		if !ts.IsZero() {
			lastTimestamp = ts
		}

		entrySessionID := entry.SessionID
		if entrySessionID == "" {
			entrySessionID = entry.SessionID2
		}
		entryProject := entry.Cwd
		if entryProject == "" {
			entryProject = entry.Project
		}
		if session.ID == "" && entrySessionID != "" {
			session.ID = entrySessionID
		}
		if session.ProjectPath == "" && entryProject != "" {
			session.ProjectPath = entryProject
		}
		if session.Model == "" && entry.Model != "" {
			session.Model = entry.Model
		}

		var messageContent string
		messageRole := ""
		if entry.Message != nil {
			messageContent = extractClaudeMessageContent(entry.Message.Content)
			if entry.Message.Role != "" {
				messageRole = entry.Message.Role
			}
			if entry.Message.Model != "" {
				session.Model = entry.Message.Model
			}
			if entry.Message.Usage != nil {
				session.Tokens.Input += entry.Message.Usage.InputTokens
				session.Tokens.Output += entry.Message.Usage.OutputTokens
				session.Tokens.CacheCreation += entry.Message.Usage.CacheCreationInputTokens
				session.Tokens.CacheRead += entry.Message.Usage.CacheReadInputTokens
			}
		}
		if entry.Message == nil {
			if entry.Content != nil {
				messageContent = extractClaudeMessageContent(entry.Content)
			}
			if entry.Role != "" {
				messageRole = entry.Role
			}
		}

		// Handle different entry types
		switch entry.Type {
		case "user":
			if entry.Input != "" {
				session.Messages = append(session.Messages, ClaudeMessage{
					Role:      "user",
					Content:   entry.Input,
					Timestamp: ts,
				})
				break
			}
			if messageContent != "" {
				role := messageRole
				if role == "" {
					role = "user"
				}
				session.Messages = append(session.Messages, ClaudeMessage{
					Role:      role,
					Content:   messageContent,
					Timestamp: ts,
				})
			}

		case "assistant":
			if messageContent != "" {
				role := messageRole
				if role == "" {
					role = "assistant"
				}
				session.Messages = append(session.Messages, ClaudeMessage{
					Role:      role,
					Content:   messageContent,
					Timestamp: ts,
				})
			}

		case "system":
			// System entries can contain turn_duration and other metadata
			// For now, we just track them but don't extract additional data

		default:
			if messageContent != "" && messageRole != "" {
				session.Messages = append(session.Messages, ClaudeMessage{
					Role:      messageRole,
					Content:   messageContent,
					Timestamp: ts,
				})
			}
		}
	}

	session.StartedAt = firstTimestamp
	session.EndedAt = &lastTimestamp
	// Total = Input + Output + CacheCreation + CacheRead
	session.Tokens.Total = session.Tokens.Input + session.Tokens.Output + session.Tokens.CacheCreation + session.Tokens.CacheRead

	// Calculate cost using Anthropic pricing
	// Current pricing (as of 2025): $3/million input, $15/million output
	// Cache pricing: $3.75/million for cache creation, $0.30/million for cache read
	session.Cost = calculateClaudeCost(session.Tokens)

	return session, nil
}

// calculateClaudeCost calculates the cost for a Claude session
func calculateClaudeCost(tokens TokenUsage) float64 {
	// Anthropic pricing (approximate, can be updated)
	inputCost := float64(tokens.Input) * 3.0 / 1_000_000
	cacheCreationCost := float64(tokens.CacheCreation) * 3.75 / 1_000_000
	cacheReadCost := float64(tokens.CacheRead) * 0.30 / 1_000_000
	outputCost := float64(tokens.Output) * 15.0 / 1_000_000

	return inputCost + cacheCreationCost + cacheReadCost + outputCost
}

func extractClaudeMessageContent(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}

	var contentStr string
	if err := json.Unmarshal(raw, &contentStr); err == nil {
		return contentStr
	}

	var items []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Thinking string `json:"thinking"`
		Name     string `json:"name"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal(raw, &items); err == nil {
		parts := make([]string, 0, len(items))
		for _, item := range items {
			switch item.Type {
			case "text":
				if item.Text != "" {
					parts = append(parts, item.Text)
				}
			case "thinking":
				if item.Thinking != "" {
					parts = append(parts, item.Thinking)
				}
			case "tool_use":
				if item.Name != "" {
					parts = append(parts, "[tool_use:"+item.Name+"]")
				} else {
					parts = append(parts, "[tool_use]")
				}
			case "tool_result":
				if item.Content != "" {
					parts = append(parts, item.Content)
				} else {
					parts = append(parts, "[tool_result]")
				}
			default:
				if item.Text != "" {
					parts = append(parts, item.Text)
				}
				if item.Content != "" {
					parts = append(parts, item.Content)
				}
			}
		}
		return strings.Join(parts, "\n")
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		if text, ok := obj["text"].(string); ok {
			return text
		}
		if content, ok := obj["content"].(string); ok {
			return content
		}
	}

	return ""
}

// GetClaudeSessionsDir returns the default Claude sessions directory
func GetClaudeSessionsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// FindAllClaudeSessions finds all Claude session files recursively
func FindAllClaudeSessions(sessionsDir string) ([]string, error) {
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
		return nil, fmt.Errorf("failed to walk sessions directory: %w", err)
	}

	// Sort by modification time, newest first
	sort.Slice(matches, func(i, j int) bool {
		iInfo, _ := os.Stat(matches[i])
		jInfo, _ := os.Stat(matches[j])
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	return matches, nil
}
