package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_MissingFile(t *testing.T) {
	// Use a non-existent path
	tmpDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	nonExistentPath := filepath.Join(tmpDir, "non-existent.toml")
	
	// Should NOT return error, but use defaults
	cfg, err := LoadConfig(nonExistentPath)
	if err != nil {
		t.Fatalf("LoadConfig failed for missing file: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected config, got nil")
	}

	// Verify defaults
	if !cfg.Agents.Codex {
		t.Error("Expected default Codex=true")
	}
	if !cfg.Agents.ClaudeCode {
		t.Error("Expected default Claude=true")
	}
}
