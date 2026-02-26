package tracker

import (
	"context"
	"fmt"
	"time"
)

// Period represents the time period for usage stats
type Period string

const (
	PeriodDay   Period = "day"
	PeriodWeek  Period = "week"
	PeriodMonth Period = "month"
)

// UsageStatsData holds all usage statistics for display
type UsageStatsData struct {
	LastSession        *SessionRow
	RecentSessions     []SessionRow
	TopModels          []ModelUsage
	DailySummaries     []DailySummary  // For weekly period
	WeeklySummaries    []WeeklySummary // For monthly period
	TotalSessionTime   int64           // in seconds
	TotalInputTokens   int64
	TotalOutputTokens  int64
	TotalCacheCreation int64
	TotalCacheRead     int64
	TotalTokens        int64
	TotalCost          float64
	TotalMessages      int64
	TotalToolCalls     int64
	UniqueProjects     int64
	SessionCount       int64
	LastSyncTime       int64 // Unix timestamp of last sync for the agent
}

// ModelUsage represents model usage count
type ModelUsage struct {
	Model        string
	SessionCount int64
}

// SQLiteTracker implements the Tracker interface using SQLite
type SQLiteTracker struct {
	db    *DB
	debug bool
}

// SetDebug enables or disables debug mode
func (t *SQLiteTracker) SetDebug(enabled bool) {
	t.debug = enabled
}

// IsDebug returns whether debug mode is enabled
func (t *SQLiteTracker) IsDebug() bool {
	return t.debug
}

// NewSQLiteTracker creates a new SQLite tracker
func NewSQLiteTracker(dbPath string) (*SQLiteTracker, error) {
	db, err := Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &SQLiteTracker{db: db}, nil
}

// Close closes the database connection
func (t *SQLiteTracker) Close() error {
	return t.db.Close()
}

// SetLastSyncTime sets the last sync time for an agent
func (t *SQLiteTracker) SetLastSyncTime(ctx context.Context, agent string, timestamp int64) error {
	return t.db.SetLastSyncTime(ctx, agent, timestamp)
}

// GetLastSyncTime returns the last sync time for an agent (unix timestamp, 0 if never synced)
func (t *SQLiteTracker) GetLastSyncTime(ctx context.Context, agent string) (int64, error) {
	return t.db.GetLastSyncTime(ctx, agent)
}

// StartSession is not used for Codex (uses file-based sessions)
func (t *SQLiteTracker) StartSession(agent Agent) (*Session, error) {
	return nil, fmt.Errorf("StartSession not supported for Codex - use file-based tracking")
}

// EndSession is not used for Codex
func (t *SQLiteTracker) EndSession(session *Session) error {
	return fmt.Errorf("EndSession not supported for Codex - use file-based tracking")
}

// GetUsage returns aggregated usage statistics for an agent
func (t *SQLiteTracker) GetUsage(agent Agent) (*UsageStats, error) {
	ctx := context.Background()
	query := `SELECT COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0)
		FROM sessions WHERE source = ?`

	var totalSessions int
	var totalInput, totalOutput int64

	err := t.db.db.QueryRowContext(ctx, query, string(agent)).Scan(&totalSessions, &totalInput, &totalOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	return &UsageStats{
		Agent:             agent,
		TotalSessions:     totalSessions,
		TotalInputTokens:  int(totalInput),
		TotalOutputTokens: int(totalOutput),
	}, nil
}

// TrackSession stores a parsed Codex session into the database
func (t *SQLiteTracker) TrackSession(ctx context.Context, session *CodexSession) error {
	// Check if session already exists
	existing, err := t.db.GetSessionByExternalID(ctx, session.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing session: %w", err)
	}
	if existing != nil {
		if len(session.Messages) > 0 {
			msgCount, err := t.db.GetMessageCountBySessionID(ctx, existing.ID)
			if err != nil {
				return fmt.Errorf("failed to check message count: %w", err)
			}
			if msgCount == 0 {
				for _, msg := range session.Messages {
					msgRow := &MessageRow{
						SessionID: existing.ID,
						Role:      msg.Role,
						Content:   msg.Content,
						Timestamp: msg.Timestamp.Unix(),
					}
					if _, err := t.db.InsertMessage(ctx, msgRow); err != nil {
						return fmt.Errorf("failed to insert message: %w", err)
					}
				}
				return fmt.Errorf("%w: %s", ErrSessionBackfilled, session.ID)
			}
		}
		return fmt.Errorf("%w: %s", ErrSessionAlreadyTracked, session.ID)
	}

	// Convert started_at to unix timestamp
	startedAt := session.StartedAt.Unix()

	var endedAt *int64
	if session.EndedAt != nil {
		ts := session.EndedAt.Unix()
		endedAt = &ts
	}

	sessionRow := &SessionRow{
		ExternalID:          session.ID,
		Source:              "codex",
		ProjectPath:         session.ProjectPath,
		Model:               session.Model,
		Provider:            session.Provider,
		StartedAt:           startedAt,
		EndedAt:             endedAt,
		InputTokens:         int64(session.Tokens.Input),
		OutputTokens:        int64(session.Tokens.Output),
		CacheCreationTokens: int64(session.Tokens.CacheCreation),
		CacheReadTokens:     int64(session.Tokens.CacheRead),
		ReasoningTokens:     int64(session.Tokens.Reasoning),
		TotalTokens:         int64(session.Tokens.Total),
		Cost:                session.Cost,
	}

	sessionID, err := t.db.InsertSession(ctx, sessionRow)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	// Insert messages
	for _, msg := range session.Messages {
		msgRow := &MessageRow{
			SessionID: sessionID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp.Unix(),
		}
		if _, err := t.db.InsertMessage(ctx, msgRow); err != nil {
			return fmt.Errorf("failed to insert message: %w", err)
		}
	}

	// Insert tool calls
	for _, tc := range session.ToolCalls {
		tcRow := &ToolCallRow{
			SessionID: sessionID,
			ToolName:  tc.ToolName,
			Arguments: tc.Arguments,
			Result:    tc.Result,
			Timestamp: tc.Timestamp.Unix(),
		}
		if _, err := t.db.InsertToolCall(ctx, tcRow); err != nil {
			return fmt.Errorf("failed to insert tool call: %w", err)
		}
	}

	return nil
}

// GetSessions returns all tracked sessions
func (t *SQLiteTracker) GetSessions(ctx context.Context) ([]SessionRow, error) {
	return t.db.GetAllSessions(ctx)
}

// GetMessages returns all messages for a session
func (t *SQLiteTracker) GetMessages(ctx context.Context, sessionID int64) ([]MessageRow, error) {
	return t.db.GetMessagesBySessionID(ctx, sessionID)
}

// GetToolCalls returns all tool calls for a session
func (t *SQLiteTracker) GetToolCalls(ctx context.Context, sessionID int64) ([]ToolCallRow, error) {
	return t.db.GetToolCallsBySessionID(ctx, sessionID)
}

// GetUsageStats returns usage statistics for an agent within a period
func (t *SQLiteTracker) GetUsageStats(ctx context.Context, agent Agent, period Period) (*UsageStatsData, error) {
	// Calculate the time filter
	var startTime time.Time
	now := time.Now()
	switch period {
	case PeriodDay:
		startTime = now.AddDate(0, 0, -1)
	case PeriodWeek:
		startTime = now.AddDate(0, 0, -7) // last 7 days
	case PeriodMonth:
		startTime = now.AddDate(0, 0, -30) // last 30 days
	default:
		startTime = now.AddDate(0, 0, -1) // default to day
	}

	startTimestamp := startTime.Unix()
	source := string(agent)

	// Get last session
	lastSession, err := t.db.GetLastSession(ctx, source, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get last session: %w", err)
	}

	// Get top 3 models
	topModels, err := t.db.GetTopModels(ctx, source, startTimestamp, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get top models: %w", err)
	}

	// Get aggregated stats
	stats, err := t.db.GetAggregatedStats(ctx, source, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated stats: %w", err)
	}

	// Get message count
	msgCount, err := t.db.GetMessageCount(ctx, source, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}

	// Get tool call count
	toolCallCount, err := t.db.GetToolCallCount(ctx, source, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool call count: %w", err)
	}

	// Get unique projects
	uniqueProjects, err := t.db.GetUniqueProjects(ctx, source, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique projects: %w", err)
	}

	// Get daily summaries for weekly period
	var dailySummaries []DailySummary
	var weeklySummaries []WeeklySummary

	if period == PeriodWeek {
		dailySummaries, err = t.db.GetDailySummaries(ctx, source, startTimestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to get daily summaries: %w", err)
		}
	}

	if period == PeriodMonth {
		weeklySummaries, err = t.db.GetWeeklySummaries(ctx, source, startTimestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to get weekly summaries: %w", err)
		}
	}

	// Get last sync time
	lastSyncTime, err := t.db.GetLastSyncTime(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("failed to get last sync time: %w", err)
	}

	return &UsageStatsData{
		LastSession:        lastSession,
		TopModels:          topModels,
		DailySummaries:     dailySummaries,
		WeeklySummaries:    weeklySummaries,
		TotalSessionTime:   stats.TotalSessionTime,
		TotalInputTokens:   stats.TotalInputTokens,
		TotalOutputTokens:  stats.TotalOutputTokens,
		TotalCacheCreation: stats.TotalCacheCreation,
		TotalCacheRead:     stats.TotalCacheRead,
		TotalTokens:        stats.TotalTokens,
		TotalCost:          stats.TotalCost,
		TotalMessages:      msgCount,
		TotalToolCalls:     toolCallCount,
		UniqueProjects:     uniqueProjects,
		SessionCount:       stats.SessionCount,
		LastSyncTime:       lastSyncTime,
	}, nil
}

// GetSessionsInPeriod returns all sessions within a time period for debug
func (t *SQLiteTracker) GetSessionsInPeriod(ctx context.Context, agent Agent, period Period) ([]SessionRow, error) {
	// Calculate the time filter
	var startTime time.Time
	now := time.Now()
	switch period {
	case PeriodDay:
		startTime = now.AddDate(0, 0, -1)
	case PeriodWeek:
		startTime = now.AddDate(0, 0, -7)
	case PeriodMonth:
		startTime = now.AddDate(0, 0, -30)
	default:
		startTime = now.AddDate(0, 0, -1)
	}

	startTimestamp := startTime.Unix()
	source := string(agent)

	return t.db.GetSessionsInPeriod(ctx, source, startTimestamp)
}

// TrackClaudeSession stores a parsed Claude session into the database
func (t *SQLiteTracker) TrackClaudeSession(ctx context.Context, session *ClaudeSession) error {
	// Check if session already exists
	existing, err := t.db.GetSessionByExternalID(ctx, session.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing session: %w", err)
	}
	if existing != nil {
		if len(session.Messages) > 0 {
			msgCount, err := t.db.GetMessageCountBySessionID(ctx, existing.ID)
			if err != nil {
				return fmt.Errorf("failed to check message count: %w", err)
			}
			if msgCount == 0 {
				for _, msg := range session.Messages {
					msgRow := &MessageRow{
						SessionID: existing.ID,
						Role:      msg.Role,
						Content:   msg.Content,
						Timestamp: msg.Timestamp.Unix(),
					}
					if _, err := t.db.InsertMessage(ctx, msgRow); err != nil {
						return fmt.Errorf("failed to insert message: %w", err)
					}
				}
				return fmt.Errorf("%w: %s", ErrSessionBackfilled, session.ID)
			}
		}
		return fmt.Errorf("%w: %s", ErrSessionAlreadyTracked, session.ID)
	}

	// Convert started_at to unix timestamp
	startedAt := session.StartedAt.Unix()

	var endedAt *int64
	if session.EndedAt != nil {
		ts := session.EndedAt.Unix()
		endedAt = &ts
	}

	sessionRow := &SessionRow{
		ExternalID:          session.ID,
		Source:              "claude",
		ProjectPath:         session.ProjectPath,
		Model:               session.Model,
		Provider:            session.Provider,
		StartedAt:           startedAt,
		EndedAt:             endedAt,
		InputTokens:         int64(session.Tokens.Input),
		OutputTokens:        int64(session.Tokens.Output),
		CacheCreationTokens: int64(session.Tokens.CacheCreation),
		CacheReadTokens:     int64(session.Tokens.CacheRead),
		ReasoningTokens:     int64(session.Tokens.Reasoning),
		TotalTokens:         int64(session.Tokens.Total),
		Cost:                session.Cost,
	}

	sessionID, err := t.db.InsertSession(ctx, sessionRow)
	if err != nil {
		return fmt.Errorf("failed to insert session: %w", err)
	}

	// Insert messages
	for _, msg := range session.Messages {
		msgRow := &MessageRow{
			SessionID: sessionID,
			Role:      msg.Role,
			Content:   msg.Content,
			Timestamp: msg.Timestamp.Unix(),
		}
		if _, err := t.db.InsertMessage(ctx, msgRow); err != nil {
			return fmt.Errorf("failed to insert message: %w", err)
		}
	}

	return nil
}

// GetUsageStatsAll returns combined usage stats for all agents
func (t *SQLiteTracker) GetUsageStatsAll(ctx context.Context, period Period) (*UsageStatsData, error) {
	// Calculate the time filter
	var startTime time.Time
	now := time.Now()
	switch period {
	case PeriodDay:
		startTime = now.AddDate(0, 0, -1)
	case PeriodWeek:
		startTime = now.AddDate(0, 0, -7)
	case PeriodMonth:
		startTime = now.AddDate(0, 0, -30)
	default:
		startTime = now.AddDate(0, 0, -1)
	}

	startTimestamp := startTime.Unix()

	// Get aggregated stats for all sources
	stats, err := t.db.GetAggregatedStatsAll(ctx, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated stats: %w", err)
	}

	// Get message count across all sources
	msgCount, err := t.db.GetMessageCountAll(ctx, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}

	// Get tool call count across all sources
	toolCallCount, err := t.db.GetToolCallCountAll(ctx, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get tool call count: %w", err)
	}

	// Get top 3 models across all sources
	topModels, err := t.db.GetTopModelsAll(ctx, startTimestamp, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to get top models: %w", err)
	}

	// Get unique projects
	uniqueProjects, err := t.db.GetUniqueProjectsAll(ctx, startTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique projects: %w", err)
	}

	// Get recent sessions (last 5)
	recentSessions, err := t.db.GetRecentSessions(ctx, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}

	// Get last sync times for all enabled agents and use the most recent
	var lastSyncTime int64
	agents := []string{"codex", "claude"}
	for _, agent := range agents {
		syncTime, err := t.db.GetLastSyncTime(ctx, agent)
		if err != nil {
			return nil, fmt.Errorf("failed to get last sync time for %s: %w", agent, err)
		}
		if syncTime > lastSyncTime {
			lastSyncTime = syncTime
		}
	}

	return &UsageStatsData{
		TopModels:          topModels,
		RecentSessions:     recentSessions,
		TotalSessionTime:   stats.TotalSessionTime,
		TotalInputTokens:   stats.TotalInputTokens,
		TotalOutputTokens:  stats.TotalOutputTokens,
		TotalCacheCreation: stats.TotalCacheCreation,
		TotalCacheRead:     stats.TotalCacheRead,
		TotalTokens:        stats.TotalTokens,
		TotalCost:          stats.TotalCost,
		TotalMessages:      msgCount,
		TotalToolCalls:     toolCallCount,
		UniqueProjects:     uniqueProjects,
		SessionCount:       stats.SessionCount,
		LastSyncTime:       lastSyncTime,
	}, nil
}

// GetPerAgentStats returns per-agent breakdown
func (t *SQLiteTracker) GetPerAgentStats(ctx context.Context, period Period) ([]PerAgentStats, error) {
	// Calculate the time filter
	var startTime time.Time
	now := time.Now()
	switch period {
	case PeriodDay:
		startTime = now.AddDate(0, 0, -1)
	case PeriodWeek:
		startTime = now.AddDate(0, 0, -7)
	case PeriodMonth:
		startTime = now.AddDate(0, 0, -30)
	default:
		startTime = now.AddDate(0, 0, -1)
	}

	startTimestamp := startTime.Unix()

	return t.db.GetPerAgentStats(ctx, startTimestamp)
}
