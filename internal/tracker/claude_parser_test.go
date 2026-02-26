package tracker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseClaudeSession_Accurate(t *testing.T) {
	// Create a temporary JSONL file with known token counts
	content := `{"type":"assistant","timestamp":"2026-02-26T10:00:00Z","sessionId":"sess-123","cwd":"/path/to/project","message":{"model":"claude-3-5-sonnet","role":"assistant","usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":200,"cache_read_input_tokens":100}}}`
	
	tmpDir, err := os.MkdirTemp("", "claude-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, "session.jsonl")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	session, err := ParseClaudeSession(tmpFile)
	if err != nil {
		t.Fatalf("ParseClaudeSession failed: %v", err)
	}

	// Verify tokens
	if session.Tokens.Input != 1000 {
		t.Errorf("Expected input tokens 1000, got %d", session.Tokens.Input)
	}
	if session.Tokens.Output != 500 {
		t.Errorf("Expected output tokens 500, got %d", session.Tokens.Output)
	}
	if session.Tokens.CacheCreation != 200 {
		t.Errorf("Expected cache creation tokens 200, got %d", session.Tokens.CacheCreation)
	}
	if session.Tokens.CacheRead != 100 {
		t.Errorf("Expected cache read tokens 100, got %d", session.Tokens.CacheRead)
	}
	
	// Expected cost calculation:
	// input: 1000 * 3 / 1,000,000 = 0.003
	// output: 500 * 15 / 1,000,000 = 0.0075
	// creation: 200 * 3.75 / 1,000,000 = 0.00075
	// read: 100 * 0.30 / 1,000,000 = 0.00003
	// total = 0.003 + 0.0075 + 0.00075 + 0.00003 = 0.01128
	expectedCost := 0.01128
	if session.Cost != expectedCost {
		t.Errorf("Expected cost %f, got %f", expectedCost, session.Cost)
	}
}

func TestParseClaudeSession_Messages(t *testing.T) {
	content := `{"type":"user","timestamp":"2026-02-26T10:00:00Z","sessionId":"sess-123","cwd":"/path","input":"Hello Claude"}
{"type":"assistant","timestamp":"2026-02-26T10:00:01Z","sessionId":"sess-123","cwd":"/path","message":{"model":"claude-3","role":"assistant","content":[{"type":"text","text":"Hello Human"}],"usage":{"input_tokens":10,"output_tokens":10}}}`
	
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "session.jsonl")
	os.WriteFile(tmpFile, []byte(content), 0644)

	session, err := ParseClaudeSession(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if len(session.Messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(session.Messages))
	}

	if session.Messages[0].Role != "user" || session.Messages[0].Content != "Hello Claude" {
		t.Errorf("First message incorrect: %+v", session.Messages[0])
	}

	if session.Messages[1].Role != "assistant" || session.Messages[1].Content != "Hello Human" {
		t.Errorf("Second message incorrect: %+v", session.Messages[1])
	}
}
