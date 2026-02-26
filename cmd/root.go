package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ari/agent-usage/internal/config"
	"github.com/ari/agent-usage/internal/tracker"
	"github.com/ari/agent-usage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfgPath string
	cfg     *config.Config
	debug   bool
)

var rootCmd = &cobra.Command{
	Use:   "agent-usage",
	Short: "Track AI coding agent usage",
	Long:  `A CLI tool to track usage of AI-powered coding agents (Codex, Claude).`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for help command
		if cmd.Name() == "help" {
			return nil
		}
		var err error
		cfg, err = config.LoadConfig(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show loaded configuration and status",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("=== Configuration ===")
		fmt.Printf("  Agents:\n")
		fmt.Printf("    Codex: %v\n", cfg.Agents.Codex)
		fmt.Printf("    Claude: %v\n", cfg.Agents.ClaudeCode)

		// Get last sync times from database
		dbPath := cfg.GetDatabasePath()
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			fmt.Println("\n=== Last Sync ===")
			fmt.Println("  No database found")
			return
		}

		db, err := tracker.NewSQLiteTracker(dbPath)
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
			return
		}
		defer db.Close()

		ctx := context.Background()
		fmt.Println("\n=== Last Sync ===")
		if cfg.Agents.Codex {
			syncTime, _ := db.GetLastSyncTime(ctx, "codex")
			if syncTime > 0 {
				fmt.Printf("  Codex: %s\n", time.Unix(syncTime, 0).Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  Codex: Never synced\n")
			}
		}
		if cfg.Agents.ClaudeCode {
			syncTime, _ := db.GetLastSyncTime(ctx, "claude")
			if syncTime > 0 {
				fmt.Printf("  Claude: %s\n", time.Unix(syncTime, 0).Format("2006-01-02 15:04:05"))
			} else {
				fmt.Printf("  Claude: Never synced\n")
			}
		}
	},
}

var usageCmd = &cobra.Command{
	Use:   "usage <agent> [period]",
	Short: "Show usage statistics for an agent",
	Long:  "Show usage statistics for Codex or Claude. Period can be day, week, or month (default: day)",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		agentName := args[0]
		period := tracker.PeriodDay
		if len(args) > 1 {
			switch args[1] {
			case "day":
				period = tracker.PeriodDay
			case "week":
				period = tracker.PeriodWeek
			case "month":
				period = tracker.PeriodMonth
			default:
				fmt.Printf("Invalid period: %s. Use day, week, or month\n", args[1])
				os.Exit(1)
			}
		}

		// Validate agent name
		var agent tracker.Agent
		switch agentName {
		case "codex":
			agent = tracker.AgentCodex
		case "claude":
			agent = tracker.AgentClaudeCode
		default:
			fmt.Printf("Invalid agent: %s. Use codex or claude\n", agentName)
			os.Exit(1)
		}

		// Run sync for the selected agent
		runSync(agentName)

		// Get database path
		dbPath := cfg.GetDatabasePath()

		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// Database doesn't exist yet, show empty stats
			ui.DisplayUsageStats(agentName, period, &tracker.UsageStatsData{
				TopModels: []tracker.ModelUsage{},
			})
			return
		}

		// Open database with debug mode
		db, err := tracker.NewSQLiteTracker(dbPath)
		if err != nil {
			ui.Error(fmt.Sprintf("Error opening database: %v", err))
			os.Exit(1)
		}
		defer db.Close()

		// Enable debug mode on tracker if flag is set
		db.SetDebug(debug)

		// Calculate time filter for debug output
		if debug {
			now := time.Now()
			var startTime time.Time
			switch period {
			case tracker.PeriodDay:
				startTime = now.AddDate(0, 0, -1)
			case tracker.PeriodWeek:
				startTime = now.AddDate(0, 0, -7)
			case tracker.PeriodMonth:
				startTime = now.AddDate(0, 0, -30)
			default:
				startTime = now.AddDate(0, 0, -1)
			}
			fmt.Printf("\n%s[DEBUG] Time Filter:%s\n", ui.ColorBlue, ui.ColorReset)
			fmt.Printf("  Period: %s\n", period)
			fmt.Printf("  Start:  %s (timestamp: %d)\n", startTime.Format("2006-01-02 15:04:05"), startTime.Unix())
			fmt.Printf("  End:    %s (timestamp: %d)\n", now.Format("2006-01-02 15:04:05"), now.Unix())
			fmt.Printf("  Agent:  %s\n\n", agent)
		}

		// Get usage stats
		ctx := context.Background()
		stats, err := db.GetUsageStats(ctx, agent, period)
		if err != nil {
			ui.Error(fmt.Sprintf("Error getting usage stats: %v", err))
			os.Exit(1)
		}

		// Show debug output for raw session data
		if debug {
			sessions, err := db.GetSessionsInPeriod(ctx, agent, period)
			if err != nil {
				ui.Error(fmt.Sprintf("Error getting sessions: %v", err))
				os.Exit(1)
			}

			fmt.Printf("%s[DEBUG] Sessions Data (%d sessions):%s\n", ui.ColorBlue, len(sessions), ui.ColorReset)
			for i, s := range sessions {
				fmt.Printf("  %d. ID: %s\n", i+1, s.ExternalID)
				fmt.Printf("     Model: %s, Project: %s\n", s.Model, s.ProjectPath)
				fmt.Printf("     Started: %s\n", ui.FormatDateTime(s.StartedAt))
				if s.EndedAt != nil {
					duration := *s.EndedAt - s.StartedAt
					fmt.Printf("     Ended: %s, Duration: %s\n", ui.FormatDateTime(*s.EndedAt), ui.FormatDuration(duration))
				} else {
					fmt.Printf("     Ended: (active)\n")
				}
				fmt.Printf("     Tokens: %s (in: %s, out: %s, cache: %s/%s)\n",
					ui.FormatTokens(s.TotalTokens),
					ui.FormatTokens(s.InputTokens),
					ui.FormatTokens(s.OutputTokens),
					ui.FormatTokens(s.CacheCreationTokens),
					ui.FormatTokens(s.CacheReadTokens))
			}
			fmt.Println()
		}

		// Display stats
		ui.DisplayUsageStats(agentName, period, stats)
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats [period]",
	Short: "Show combined usage stats for all agents",
	Long:  "Show usage statistics for all agents combined. Period can be day, week, or month (default: day)",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		period := tracker.PeriodDay
		if len(args) > 0 {
			switch args[0] {
			case "day":
				period = tracker.PeriodDay
			case "week":
				period = tracker.PeriodWeek
			case "month":
				period = tracker.PeriodMonth
			default:
				fmt.Printf("Invalid period: %s. Use day, week, or month\n", args[0])
				os.Exit(1)
			}
		}

		// Run sync for all enabled agents
		runSyncAll()

		// Get database path
		dbPath := cfg.GetDatabasePath()

		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// Database doesn't exist yet, show empty stats
			ui.DisplayAllStats(period, &tracker.UsageStatsData{
				TopModels: []tracker.ModelUsage{},
			}, []tracker.PerAgentStats{})
			return
		}

		// Open database
		db, err := tracker.NewSQLiteTracker(dbPath)
		if err != nil {
			ui.Error(fmt.Sprintf("Error opening database: %v", err))
			os.Exit(1)
		}
		defer db.Close()

		// Get usage stats
		ctx := context.Background()
		stats, err := db.GetUsageStatsAll(ctx, period)
		if err != nil {
			ui.Error(fmt.Sprintf("Error getting usage stats: %v", err))
			os.Exit(1)
		}

		// Get per-agent stats
		perAgent, err := db.GetPerAgentStats(ctx, period)
		if err != nil {
			ui.Error(fmt.Sprintf("Error getting per-agent stats: %v", err))
			os.Exit(1)
		}

		// Display stats
		ui.DisplayAllStats(period, stats, perAgent)
	},
}

// runSync runs the sync for a given agent
func runSync(agentName string) {
	var sessionsDir string
	var parseFunc func(string) (interface{}, error)
	var trackFunc func(*tracker.SQLiteTracker, context.Context, interface{}) error

	switch agentName {
	case "codex":
		sessionsDir = tracker.GetDefaultSessionsDir()
		parseFunc = func(path string) (interface{}, error) {
			return tracker.ParseCodexSession(path)
		}
		trackFunc = func(t *tracker.SQLiteTracker, ctx context.Context, sess interface{}) error {
			return t.TrackSession(ctx, sess.(*tracker.CodexSession))
		}
	case "claude":
		sessionsDir = tracker.GetClaudeSessionsDir()
		parseFunc = func(path string) (interface{}, error) {
			return tracker.ParseClaudeSession(path)
		}
		trackFunc = func(t *tracker.SQLiteTracker, ctx context.Context, sess interface{}) error {
			return t.TrackClaudeSession(ctx, sess.(*tracker.ClaudeSession))
		}
	default:
		return
	}

	// Get database path
	dbPath := cfg.GetDatabasePath()

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	// Open database
	db, err := tracker.NewSQLiteTracker(dbPath)
	if err != nil {
		return
	}
	defer db.Close()

	// Find all session files recursively
	var sessionFiles []string
	filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".jsonl" {
			sessionFiles = append(sessionFiles, path)
		}
		return nil
	})

	if len(sessionFiles) == 0 {
		return
	}

	// Parse and track each session
	ctx := context.Background()
	tracked := 0
	skipped := 0
	backfilled := 0

	for _, sessionPath := range sessionFiles {
		session, err := parseFunc(sessionPath)
		if err != nil {
			continue
		}

		if err := trackFunc(db, ctx, session); err != nil {
			if errors.Is(err, tracker.ErrSessionBackfilled) {
				backfilled++
				continue
			}
			if errors.Is(err, tracker.ErrSessionAlreadyTracked) {
				skipped++
				continue
			}
			skipped++
			continue
		}

		tracked++
	}

	if tracked > 0 {
		fmt.Printf("[Sync] Synced %d new sessions for %s\n", tracked, agentName)
	}
	if backfilled > 0 {
		fmt.Printf("[Sync] Updated %d existing sessions for %s\n", backfilled, agentName)
	}

	// Save last sync time
	if tracked > 0 || skipped > 0 || backfilled > 0 {
		ctx := context.Background()
		db.SetLastSyncTime(ctx, agentName, time.Now().Unix())
	}
}

// runSyncAll syncs all enabled agents from config
func runSyncAll() {
	if cfg.Agents.Codex {
		runSync("codex")
	}
	if cfg.Agents.ClaudeCode {
		runSync("claude")
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "Path to config file (default: ~/.agent-usage/config.toml)")
	usageCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Show debug output (SQL queries, raw data, time filters)")
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(usageCmd)
	rootCmd.AddCommand(statsCmd)
}
