package tracker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
}

// claudeEntry represents a JSONL entry in a Claude session file
type claudeEntry struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	SessionID string          `json:"sessionId"`
	Cwd       string          `json:"cwd"`
	Message   *claudeMessage `json:"message"`
	System    *claudeSystem  `json:"system"`
}

// claudeMessage represents a message in a Claude session
type claudeMessage struct {
	Model string        `json:"model"`
	Role  string        `json:"role"`
	Usage *claudeUsage  `json:"usage"`
}

// claudeUsage represents token usage for a Claude message
type claudeUsage struct {
	InputTokens            int `json:"input_tokens"`
	OutputTokens           int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens   int `json:"cache_read_input_tokens"`
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
			// Extract session metadata from first entry
			session.ID = entry.SessionID
			session.ProjectPath = entry.Cwd
		}
		lastTimestamp = ts

		// Handle different entry types
		switch entry.Type {
		case "assistant":
			if entry.Message != nil {
				// Get model from message
				if entry.Message.Model != "" {
					session.Model = entry.Message.Model
				}
				// Accumulate tokens from usage
				if entry.Message.Usage != nil {
					session.Tokens.Input += entry.Message.Usage.InputTokens
					session.Tokens.Output += entry.Message.Usage.OutputTokens
					session.Tokens.Cached += entry.Message.Usage.CacheCreationInputTokens + entry.Message.Usage.CacheReadInputTokens
				}
			}

		case "system":
			// System entries can contain turn_duration and other metadata
			// For now, we just track them but don't extract additional data
		}
	}

	session.StartedAt = firstTimestamp
	session.EndedAt = &lastTimestamp
	// Total = Input + Output + Cached
	session.Tokens.Total = session.Tokens.Input + session.Tokens.Output + session.Tokens.Cached

	// Calculate cost using Anthropic pricing
	// Current pricing (as of 2025): $3/million input, $15/million output
	// Cache pricing: $3.75/million for cache creation, $0.30/million for cache read
	session.Cost = calculateClaudeCost(session.Tokens)

	return session, nil
}

// calculateClaudeCost calculates the cost for a Claude session
func calculateClaudeCost(tokens TokenUsage) float64 {
	// Input tokens (excluding cached)
	inputTokens := tokens.Input
	// Cached tokens have different pricing
	cacheCreationTokens := tokens.Cached / 2       // Approximate split
	cacheReadTokens := tokens.Cached / 2            // Approximate split
	outputTokens := tokens.Output

	// Anthropic pricing (approximate, can be updated)
	inputCost := float64(inputTokens) * 3.0 / 1_000_000
	cacheCreationCost := float64(cacheCreationTokens) * 3.75 / 1_000_000
	cacheReadCost := float64(cacheReadTokens) * 0.30 / 1_000_000
	outputCost := float64(outputTokens) * 15.0 / 1_000_000

	return inputCost + cacheCreationCost + cacheReadCost + outputCost
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
