package tracker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetRecentSessions(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Insert test sessions
	now := time.Now().Unix()
	sessions := []SessionRow{
		{
			ExternalID:   "session-1",
			Source:       "claude",
			ProjectPath:  "/test/project1",
			Model:        "claude-3-5-sonnet",
			Provider:     "anthropic",
			StartedAt:    now - 3600, // 1 hour ago
			InputTokens:  1000,
			OutputTokens: 500,
			TotalTokens:  1500,
		},
		{
			ExternalID:   "session-2",
			Source:       "claude",
			ProjectPath:  "/test/project2",
			Model:        "claude-3-5-sonnet",
			Provider:     "anthropic",
			StartedAt:    now - 7200, // 2 hours ago
			InputTokens: 2000,
			OutputTokens: 800,
			TotalTokens:  2800,
		},
		{
			ExternalID:   "session-3",
			Source:       "codex",
			ProjectPath:  "/test/project3",
			Model:        "gpt-5.3-codex",
			Provider:     "openai",
			StartedAt:    now - 10800, // 3 hours ago
			InputTokens: 3000,
			OutputTokens: 1200,
			TotalTokens:  4200,
		},
	}

	for _, s := range sessions {
		_, err := db.InsertSession(ctx, &s)
		if err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}
	}

	// Test GetRecentSessions with limit 2
	recent, err := db.GetRecentSessions(ctx, 2)
	if err != nil {
		t.Fatalf("GetRecentSessions() error = %v", err)
	}

	if len(recent) != 2 {
		t.Errorf("len(recent) = %d; want 2", len(recent))
	}

	// Should be ordered by started_at DESC (most recent first)
	if recent[0].ExternalID != "session-1" {
		t.Errorf("recent[0].ExternalID = %s; want session-1", recent[0].ExternalID)
	}
	if recent[1].ExternalID != "session-2" {
		t.Errorf("recent[1].ExternalID = %s; want session-2", recent[1].ExternalID)
	}
}

func TestGetRecentSessionsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	recent, err := db.GetRecentSessions(ctx, 5)
	if err != nil {
		t.Fatalf("GetRecentSessions() error = %v", err)
	}

	if len(recent) != 0 {
		t.Errorf("len(recent) = %d; want 0", len(recent))
	}
}

func TestGetTopModelsExcludesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	now := time.Now().Unix()

	// Insert sessions with different models
	sessions := []SessionRow{
		{ExternalID: "s1", Source: "claude", Model: "claude-3-5-sonnet", StartedAt: now - 3600},
		{ExternalID: "s2", Source: "claude", Model: "claude-3-5-sonnet", StartedAt: now - 3600},
		{ExternalID: "s3", Source: "claude", Model: "", StartedAt: now - 3600},                      // Empty model
		{ExternalID: "s4", Source: "claude", Model: "", StartedAt: now - 3600},                      // Empty model
		{ExternalID: "s5", Source: "claude", Model: "", StartedAt: now - 3600},                      // Empty model
		{ExternalID: "s6", Source: "claude", Model: "claude-haiku-4-5", StartedAt: now - 3600},
	}

	for _, s := range sessions {
		_, err := db.InsertSession(ctx, &s)
		if err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}
	}

	// Get top models - should exclude empty models
	topModels, err := db.GetTopModels(ctx, "claude", now-86400, 10)
	if err != nil {
		t.Fatalf("GetTopModels() error = %v", err)
	}

	// Should only have 2 models (not the 3 empty ones)
	if len(topModels) != 2 {
		t.Errorf("len(topModels) = %d; want 2 (excluding empty models)", len(topModels))
	}

	// Check that empty model is not in results
	for _, m := range topModels {
		if m.Model == "" {
			t.Error("Empty model should be excluded from top models")
		}
	}

	// Verify counts
	modelCounts := make(map[string]int64)
	for _, m := range topModels {
		modelCounts[m.Model] = m.SessionCount
	}

	if modelCounts["claude-3-5-sonnet"] != 2 {
		t.Errorf("claude-3-5-sonnet count = %d; want 2", modelCounts["claude-3-5-sonnet"])
	}
	if modelCounts["claude-haiku-4-5"] != 1 {
		t.Errorf("claude-haiku-4-5 count = %d; want 1", modelCounts["claude-haiku-4-5"])
	}
}

func TestGetTopModelsAllExcludesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	now := time.Now().Unix()

	// Insert sessions with empty and non-empty models from different sources
	sessions := []SessionRow{
		{ExternalID: "s1", Source: "claude", Model: "claude-3-5-sonnet", StartedAt: now - 3600},
		{ExternalID: "s2", Source: "claude", Model: "", StartedAt: now - 3600},
		{ExternalID: "s3", Source: "codex", Model: "gpt-5.3-codex", StartedAt: now - 3600},
		{ExternalID: "s4", Source: "codex", Model: "", StartedAt: now - 3600},
	}

	for _, s := range sessions {
		_, err := db.InsertSession(ctx, &s)
		if err != nil {
			t.Fatalf("Failed to insert session: %v", err)
		}
	}

	// Get top models across all sources
	topModels, err := db.GetTopModelsAll(ctx, now-86400, 10)
	if err != nil {
		t.Fatalf("GetTopModelsAll() error = %v", err)
	}

	// Should exclude empty models
	for _, m := range topModels {
		if m.Model == "" {
			t.Error("Empty model should be excluded from GetTopModelsAll")
		}
	}
}

func TestParseClaudeSessionTokenTotal(t *testing.T) {
	// Create a temporary session file with tokens
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "test-claude-session.jsonl")

	// Session with known token counts
	sessionContent := `{"type":"session_meta","timestamp":"2026-02-24T22:55:00Z","session_id":"test-claude-123","cwd":"/test/project","model":"claude-3-5-sonnet-20241022"}
{"type":"assistant","timestamp":"2026-02-24T22:55:01Z","message":{"model":"claude-3-5-sonnet-20241022","usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":200,"cache_read_input_tokens":300}}}
{"type":"assistant","timestamp":"2026-02-24T22:55:02Z","message":{"model":"claude-3-5-sonnet-20241022","usage":{"input_tokens":2000,"output_tokens":1000,"cache_creation_input_tokens":100,"cache_read_input_tokens":150}}}
`

	if err := os.WriteFile(sessionFile, []byte(sessionContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parsed, err := ParseClaudeSession(sessionFile)
	if err != nil {
		t.Fatalf("ParseClaudeSession() error = %v", err)
	}

	// Input tokens: 1000 + 2000 = 3000
	if parsed.Tokens.Input != 3000 {
		t.Errorf("Tokens.Input = %d; want 3000", parsed.Tokens.Input)
	}

	// Output tokens: 500 + 1000 = 1500
	if parsed.Tokens.Output != 1500 {
		t.Errorf("Tokens.Output = %d; want 1500", parsed.Tokens.Output)
	}

	// Cached tokens: (200+300) + (100+150) = 750
	if parsed.Tokens.Cached != 750 {
		t.Errorf("Tokens.Cached = %d; want 750", parsed.Tokens.Cached)
	}

	// Total should be Input + Output + Cached
	expectedTotal := 3000 + 1500 + 750
	if parsed.Tokens.Total != expectedTotal {
		t.Errorf("Tokens.Total = %d; want %d (Input + Output + Cached)", parsed.Tokens.Total, expectedTotal)
	}
}
