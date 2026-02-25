package main

import (
	"fmt"
	"os"

	"github.com/ari/agent-usage/cmd"
)

// Version and build time set via ldflags
var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	// Show version if requested
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("agent-usage %s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	cmd.Execute()
}
