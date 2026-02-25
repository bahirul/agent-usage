package tracker

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"hello", []string{"hello"}},
		{"hello\nworld", []string{"hello", "world"}},
		{"line1\nline2\nline3", []string{"line1", "line2", "line3"}},
		{"a\nb\n", []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitLines(%q) = %v; want %v", tt.input, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitLines(%q)[%d] = %s; want %s", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestExtractMessageContent(t *testing.T) {
	tests := []struct {
		name     string
		input    []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		}
		expected string
	}{
		{
			name:     "empty input",
			input:    []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			}{},
			expected: "",
		},
		{
			name: "output_text only",
			input: []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			}{
				{Type: "output_text", Text: "Hello world"},
			},
			expected: "Hello world",
		},
		{
			name: "multiple content types",
			input: []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			}{
				{Type: "output_text", Text: "First "},
				{Type: "input_text", Text: "Second"},
			},
			expected: "First Second",
		},
		{
			name: "ignores other types",
			input: []struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			}{
				{Type: "image", Text: "should be ignored"},
				{Type: "output_text", Text: "visible"},
			},
			expected: "visible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMessageContent(tt.input)
			if result != tt.expected {
				t.Errorf("extractMessageContent() = %s; want %s", result, tt.expected)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	session := &CodexSession{
		Messages: []CodexMessage{
			{Role: "developer", Content: "Hello"},           // 5 chars
			{Role: "user", Content: "World"},                 // 5 chars
			{Role: "assistant", Content: "Response text"},    // 13 chars
		},
	}

	estimateTokens(session)

	// Input: "Hello" + "World" = 10 chars -> ~2.5 tokens, floored to 2
	// Output: "Response text" = 13 chars -> ~3.25 tokens, floored to 3
	if session.Tokens.Input != 2 {
		t.Errorf("Tokens.Input = %d; want 2", session.Tokens.Input)
	}
	if session.Tokens.Output != 3 {
		t.Errorf("Tokens.Output = %d; want 3", session.Tokens.Output)
	}
	if session.Tokens.Total != 5 {
		t.Errorf("Tokens.Total = %d; want 5", session.Tokens.Total)
	}

	// Cost: $3/1M input, $15/1M output
	// 2 * 3 / 1e6 + 3 * 15 / 1e6 = 6e-6 + 45e-6 = 51e-6 = $0.000051
	expectedCost := float64(2)*3/1_000_000 + float64(3)*15/1_000_000
	if session.Cost != expectedCost {
		t.Errorf("Cost = %v; want %v", session.Cost, expectedCost)
	}
}

func TestParseCodexSession(t *testing.T) {
	// Create a temporary session file
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "test-session.jsonl")

	sessionContent := `{"type":"session_meta","timestamp":"2026-02-24T22:55:00Z","payload":{"id":"test-123","cwd":"/test/project","model_provider":"openai","originator":"claude-3-5-sonnet"}}
{"type":"response_item","timestamp":"2026-02-24T22:55:05Z","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hello"}]}}
{"type":"response_item","timestamp":"2026-02-24T22:56:00Z","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Hi"}]}}
`
	if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parsed, err := ParseCodexSession(sessionFile)
	if err != nil {
		t.Fatalf("ParseCodexSession() error = %v", err)
	}

	if parsed.ID != "test-123" {
		t.Errorf("ID = %s; want test-123", parsed.ID)
	}
	if parsed.ProjectPath != "/test/project" {
		t.Errorf("ProjectPath = %s; want /test/project", parsed.ProjectPath)
	}
	if parsed.Model != "claude-3-5-sonnet" {
		t.Errorf("Model = %s; want claude-3-5-sonnet", parsed.Model)
	}
	if parsed.Provider != "openai" {
		t.Errorf("Provider = %s; want openai", parsed.Provider)
	}

	if len(parsed.Messages) != 2 {
		t.Errorf("len(Messages) = %d; want 2", len(parsed.Messages))
	}

	// Check started/ended timestamps
	expectedStart := time.Date(2026, 2, 24, 22, 55, 0, 0, time.UTC)
	if !parsed.StartedAt.Equal(expectedStart) {
		t.Errorf("StartedAt = %v; want %v", parsed.StartedAt, expectedStart)
	}
	if parsed.EndedAt == nil {
		t.Error("EndedAt should not be nil")
	} else {
		expectedEnd := time.Date(2026, 2, 24, 22, 56, 0, 0, time.UTC)
		if !parsed.EndedAt.Equal(expectedEnd) {
			t.Errorf("EndedAt = %v; want %v", *parsed.EndedAt, expectedEnd)
		}
	}
}

func TestParseCodexSessionEmptyFile(t *testing.T) {
	// Create a temporary empty session file
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "empty-session.jsonl")

	if err := os.WriteFile(sessionFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	session, err := ParseCodexSession(sessionFile)
	if err != nil {
		t.Fatalf("ParseCodexSession() error = %v", err)
	}

	// Empty session should have zero timestamps
	if !session.StartedAt.IsZero() {
		t.Errorf("StartedAt should be zero for empty file")
	}
	if session.EndedAt != nil && !session.EndedAt.IsZero() {
		t.Errorf("EndedAt should be zero or nil for empty file")
	}
}
