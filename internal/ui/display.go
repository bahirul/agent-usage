package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ari/agent-usage/internal/tracker"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorMagenta = "\033[35m"
	ColorBold   = "\033[1m"
)

// FormatDuration formats seconds into a human-readable duration
func FormatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	d := time.Duration(seconds) * time.Second
	h := d.Hours()
	if h >= 1 {
		return fmt.Sprintf("%.1fh", h)
	}
	m := d.Minutes()
	return fmt.Sprintf("%.1fm", m)
}

// FormatTokens formats token count with K/M suffix
func FormatTokens(tokens int64) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	}
	if tokens >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(tokens)/1_000)
	}
	return fmt.Sprintf("%d", tokens)
}

// FormatCost formats cost with $ prefix
func FormatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}

// FormatDateTime formats a Unix timestamp into a human-readable datetime
func FormatDateTime(timestamp int64) string {
	if timestamp == 0 {
		return "-"
	}
	t := time.Unix(timestamp, 0)
	return t.Format("2006-01-02 15:04")
}

// DisplayUsageStats displays the usage statistics with formatting
func DisplayUsageStats(agent string, period tracker.Period, stats *tracker.UsageStatsData) {
	// Print period header
	periodStr := string(period)
	if period == "" {
		periodStr = "day"
	}
	fmt.Printf("\n%s Usage Statistics - %s\n", strings.Title(agent), strings.Title(periodStr))
	fmt.Println(strings.Repeat("=", 60))

	// Last Session
	fmt.Printf("\n%s%sLast Session%s\n", ColorBold, ColorBlue, ColorReset)
	if stats.LastSession != nil {
		fmt.Printf("  ID:         %s\n", stats.LastSession.ExternalID)
		fmt.Printf("  Start:      %s\n", FormatDateTime(stats.LastSession.StartedAt))
		fmt.Printf("  Project:    %s\n", stats.LastSession.ProjectPath)
		fmt.Printf("  Model:      %s\n", stats.LastSession.Model)
		fmt.Printf("  Provider:   %s\n", stats.LastSession.Provider)
		if stats.LastSession.EndedAt != nil {
			duration := *stats.LastSession.EndedAt - stats.LastSession.StartedAt
			fmt.Printf("  End:        %s\n", FormatDateTime(*stats.LastSession.EndedAt))
			fmt.Printf("  Duration:   %s\n", FormatDuration(duration))
		}
		fmt.Printf("  Tokens:     %s (in: %s, out: %s, cached: %s)\n",
			FormatTokens(stats.LastSession.TotalTokens),
			FormatTokens(stats.LastSession.InputTokens),
			FormatTokens(stats.LastSession.OutputTokens),
			FormatTokens(stats.LastSession.CachedTokens))
	} else {
		fmt.Printf("  %sNo sessions in this period%s\n", ColorYellow, ColorReset)
	}

	// Summary Stats
	fmt.Printf("\n%s%sSummary%s\n", ColorBold, ColorMagenta, ColorReset)
	fmt.Printf("  Total Sessions:     %d\n", stats.SessionCount)
	fmt.Printf("  Total Session Time: %s\n", FormatDuration(stats.TotalSessionTime))
	fmt.Printf("  Total Tokens:       %s (in: %s, out: %s, cached: %s)\n",
		FormatTokens(stats.TotalTokens),
		FormatTokens(stats.TotalInputTokens),
		FormatTokens(stats.TotalOutputTokens),
		FormatTokens(stats.TotalCachedTokens))
	fmt.Printf("  Total Messages:     %d\n", stats.TotalMessages)

	// Last Sync Time
	fmt.Printf("  Last Sync:         ")
	if stats.LastSyncTime > 0 {
		fmt.Printf("%s\n", FormatDateTime(stats.LastSyncTime))
	} else {
		fmt.Printf("%sNever synced%s\n", ColorYellow, ColorReset)
	}

	// Daily Summary (for weekly period)
	if len(stats.DailySummaries) > 0 {
		fmt.Printf("\n%s%sDaily Summary (last 7 days)%s\n", ColorBold, ColorCyan, ColorReset)
		fmt.Printf("  %-12s %10s %12s %12s\n", "Date", "Sessions", "Duration", "Tokens")
		fmt.Printf("  %s\n", strings.Repeat("-", 52))
		for _, d := range stats.DailySummaries {
			fmt.Printf("  %-12s %10d %12s %12s\n",
				d.Date,
				d.SessionCount,
				FormatDuration(d.TotalTime),
				FormatTokens(d.TotalTokens))
		}
	}

	// Weekly Summary (for monthly period)
	if len(stats.WeeklySummaries) > 0 {
		fmt.Printf("\n%s%sWeekly Summary (last 30 days)%s\n", ColorBold, ColorCyan, ColorReset)
		fmt.Printf("  %-12s %10s %12s %12s\n", "Week", "Sessions", "Duration", "Tokens")
		fmt.Printf("  %s\n", strings.Repeat("-", 52))
		for _, w := range stats.WeeklySummaries {
			fmt.Printf("  %-12s %10d %12s %12s\n",
				w.WeekStart,
				w.SessionCount,
				FormatDuration(w.TotalTime),
				FormatTokens(w.TotalTokens))
		}
	}

	// Top Models
	fmt.Printf("\n%s%sTop Models (by session count)%s\n", ColorBold, ColorGreen, ColorReset)
	if len(stats.TopModels) > 0 {
		for i, m := range stats.TopModels {
			fmt.Printf("  %d. %s - %d sessions\n", i+1, m.Model, m.SessionCount)
		}
	} else {
		fmt.Printf("  %sNo data%s\n", ColorYellow, ColorReset)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

// Error displays an error message
func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%sError: %s%s\n", ColorRed, msg, ColorReset)
}

// DisplayAllStats displays combined usage statistics for all agents
func DisplayAllStats(period tracker.Period, stats *tracker.UsageStatsData, perAgent []tracker.PerAgentStats) {
	// Print period header
	periodStr := string(period)
	if period == "" {
		periodStr = "day"
	}
	fmt.Printf("\n%sCombined Usage Statistics - %s%s\n", ColorBold, strings.Title(periodStr), ColorReset)
	fmt.Println(strings.Repeat("=", 60))

	// Per-agent breakdown
	fmt.Printf("\n%s%sPer-Agent Breakdown%s\n", ColorBold, ColorBlue, ColorReset)
	fmt.Printf("  %-12s %10s %12s %20s\n", "Agent", "Sessions", "Time", "Tokens (in/out/cached)")
	fmt.Printf("  %s\n", strings.Repeat("-", 58))

	var totalSessions int64
	var totalTime int64
	var totalTokens int64
	var totalInputTokens int64
	var totalOutputTokens int64
	var totalCachedTokens int64

	for _, p := range perAgent {
		source := p.Source
		if source == "codex" {
			source = "Codex"
		} else if source == "claude" {
			source = "Claude"
		}
		fmt.Printf("  %-12s %10d %12s %s\n",
			source,
			p.SessionCount,
			FormatDuration(p.TotalTime),
			fmt.Sprintf("%s/%s/%s", FormatTokens(p.TotalInputTokens), FormatTokens(p.TotalOutputTokens), FormatTokens(p.TotalCachedTokens)))
		totalSessions += p.SessionCount
		totalTime += p.TotalTime
		totalTokens += p.TotalTokens
		totalInputTokens += p.TotalInputTokens
		totalOutputTokens += p.TotalOutputTokens
		totalCachedTokens += p.TotalCachedTokens
	}

	// Combined totals
	fmt.Printf("  %s\n", strings.Repeat("-", 58))
	fmt.Printf("  %-12s %10d %12s %s\n",
		"Total",
		totalSessions,
		FormatDuration(totalTime),
		fmt.Sprintf("%s/%s/%s", FormatTokens(totalInputTokens), FormatTokens(totalOutputTokens), FormatTokens(totalCachedTokens)))

	// Summary Stats
	fmt.Printf("\n%s%sSummary%s\n", ColorBold, ColorMagenta, ColorReset)
	fmt.Printf("  Total Sessions:      %d\n", stats.SessionCount)
	fmt.Printf("  Total Session Time:  %s\n", FormatDuration(stats.TotalSessionTime))
	fmt.Printf("  Total Tokens:        %s (in: %s, out: %s, cached: %s)\n",
		FormatTokens(stats.TotalTokens),
		FormatTokens(stats.TotalInputTokens),
		FormatTokens(stats.TotalOutputTokens),
		FormatTokens(stats.TotalCachedTokens))
	fmt.Printf("  Unique Projects:     %d\n", stats.UniqueProjects)

	// Last Sync Time
	fmt.Printf("  Last Sync:          ")
	if stats.LastSyncTime > 0 {
		fmt.Printf("%s\n", FormatDateTime(stats.LastSyncTime))
	} else {
		fmt.Printf("%sNever synced%s\n", ColorYellow, ColorReset)
	}

	// Top Models
	fmt.Printf("\n%s%sTop Models (by session count)%s\n", ColorBold, ColorGreen, ColorReset)
	if len(stats.TopModels) > 0 {
		for i, m := range stats.TopModels {
			fmt.Printf("  %d. %s - %d sessions\n", i+1, m.Model, m.SessionCount)
		}
	} else {
		fmt.Printf("  %sNo data%s\n", ColorYellow, ColorReset)
	}

	// Recent Sessions
	fmt.Printf("\n%s%sLast %d Sessions%s\n", ColorBold, ColorCyan, len(stats.RecentSessions), ColorReset)
	if len(stats.RecentSessions) > 0 {
		for i, s := range stats.RecentSessions {
			// Format source name
			source := s.Source
			if source == "codex" {
				source = "Codex"
			} else if source == "claude" {
				source = "Claude"
			}

			// Format model
			model := s.Model
			if model == "" {
				model = "(unknown)"
			}

			// Get project name from path
			project := s.ProjectPath
			if project != "" {
				// Extract just the folder name
				if idx := strings.LastIndex(project, "/"); idx >= 0 {
					project = project[idx+1:]
				}
			} else {
				project = "(no project)"
			}

			// Format time
			startTime := time.Unix(s.StartedAt, 0)
			timeStr := startTime.Format("Jan 02 15:04")

			// Duration
			duration := "-"
			if s.EndedAt != nil {
				durationSec := *s.EndedAt - s.StartedAt
				duration = FormatDuration(durationSec)
			}

			// Tokens
			tokens := FormatTokens(s.TotalTokens)
			cachedTokens := FormatTokens(s.CachedTokens)

			fmt.Printf("  %d. %s %s | %s | %s | %s | %s (cached: %s)\n",
				i+1, timeStr, source, model, project, duration, tokens, cachedTokens)
		}
	} else {
		fmt.Printf("  %sNo data%s\n", ColorYellow, ColorReset)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}
