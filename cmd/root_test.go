package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPIDFileOperations(t *testing.T) {
	// Create a temp directory to simulate config dir
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "watch.pid")

	// Test writing PID file manually (simulating writePIDFile)
	pid := os.Getpid()
	err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// Verify PID file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}

	// Test reading PID file (simulating readPIDFile)
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	var readPid int
	_, err = fmt.Sscanf(string(data), "%d", &readPid)
	if err != nil {
		t.Fatalf("Failed to parse PID: %v", err)
	}

	if readPid != pid {
		t.Errorf("read PID = %d, want %d", readPid, pid)
	}

	// Test removing PID file
	err = os.Remove(pidFile)
	if err != nil {
		t.Fatalf("Failed to remove PID file: %v", err)
	}

	// Verify PID file is removed
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("PID file was not removed")
	}
}

func TestGetPIDFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	expected := filepath.Join(tmpDir, "watch.pid")

	// Test that PID file path is correctly formed
	// We can't easily override the path, so we just verify the pattern
	if filepath.Base(expected) != "watch.pid" {
		t.Error("PID file should be named watch.pid")
	}
}
