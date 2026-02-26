package tracker

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB represents the database connection
type DB struct {
	db *sql.DB
}

const messageCountSubquery = "COALESCE((SELECT COUNT(*) FROM messages m WHERE m.session_id = s.id), 0) as message_count"

// Open opens the database at the given path
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	d := &DB{db: db}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return d, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.db.Close()
}

// migrate creates the database tables if they don't exist
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		external_id TEXT UNIQUE,
		source TEXT NOT NULL,
		project_path TEXT,
		model TEXT,
		provider TEXT,
		started_at INTEGER NOT NULL,
		ended_at INTEGER,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		cache_creation_tokens INTEGER DEFAULT 0,
		cache_read_tokens INTEGER DEFAULT 0,
		total_tokens INTEGER DEFAULT 0,
		cost REAL DEFAULT 0,
		reasoning_tokens INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		role TEXT NOT NULL,
		content TEXT,
		timestamp INTEGER NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE TABLE IF NOT EXISTS tool_calls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL,
		tool_name TEXT NOT NULL,
		arguments TEXT,
		result TEXT,
		timestamp INTEGER NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_external_id ON sessions(external_id);
	CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
	CREATE INDEX IF NOT EXISTS idx_tool_calls_session_id ON tool_calls(session_id);

	CREATE TABLE IF NOT EXISTS metadata (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at INTEGER
	);
	`

	_, err := db.db.Exec(schema)
	if err != nil {
		return err
	}

	// Migrate existing database if columns are missing
	// SQLite doesn't support IF NOT EXISTS in ALTER TABLE, so we try and ignore errors
	db.db.Exec("ALTER TABLE sessions ADD COLUMN cache_creation_tokens INTEGER DEFAULT 0")
	db.db.Exec("ALTER TABLE sessions ADD COLUMN cache_read_tokens INTEGER DEFAULT 0")
	db.db.Exec("ALTER TABLE sessions ADD COLUMN reasoning_tokens INTEGER DEFAULT 0")

	return nil
}

// SessionRow represents a session database row
type SessionRow struct {
	ID                  int64
	ExternalID          string
	Source              string
	ProjectPath         string
	Model               string
	Provider            string
	StartedAt           int64
	EndedAt             *int64
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	ReasoningTokens     int64
	TotalTokens         int64
	Cost                float64
	MessageCount        int64
}

// InsertSession inserts a new session and returns its ID
func (db *DB) InsertSession(ctx context.Context, s *SessionRow) (int64, error) {
	query := `
	INSERT INTO sessions (external_id, source, project_path, model, provider, started_at, ended_at,
		input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens, reasoning_tokens, total_tokens, cost)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.db.ExecContext(ctx, query,
		s.ExternalID, s.Source, s.ProjectPath, s.Model, s.Provider, s.StartedAt, s.EndedAt,
		s.InputTokens, s.OutputTokens, s.CacheCreationTokens, s.CacheReadTokens, s.ReasoningTokens, s.TotalTokens, s.Cost)
	if err != nil {
		return 0, fmt.Errorf("failed to insert session: %w", err)
	}
	return result.LastInsertId()
}

// GetSessionByExternalID retrieves a session by its external ID
func (db *DB) GetSessionByExternalID(ctx context.Context, externalID string) (*SessionRow, error) {
	query := `SELECT id, external_id, source, project_path, model, provider, started_at, ended_at,
		input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens, reasoning_tokens, total_tokens, cost
		FROM sessions WHERE external_id = ?`

	row := db.db.QueryRowContext(ctx, query, externalID)
	var s SessionRow
	err := row.Scan(
		&s.ID, &s.ExternalID, &s.Source, &s.ProjectPath, &s.Model, &s.Provider,
		&s.StartedAt, &s.EndedAt, &s.InputTokens, &s.OutputTokens, &s.CacheCreationTokens,
		&s.CacheReadTokens, &s.ReasoningTokens, &s.TotalTokens, &s.Cost,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &s, nil
}

// MessageRow represents a message database row
type MessageRow struct {
	ID        int64
	SessionID int64
	Role      string
	Content   string
	Timestamp int64
}

// InsertMessage inserts a new message
func (db *DB) InsertMessage(ctx context.Context, m *MessageRow) (int64, error) {
	query := `INSERT INTO messages (session_id, role, content, timestamp) VALUES (?, ?, ?, ?)`
	result, err := db.db.ExecContext(ctx, query, m.SessionID, m.Role, m.Content, m.Timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to insert message: %w", err)
	}
	return result.LastInsertId()
}

// ToolCallRow represents a tool call database row
type ToolCallRow struct {
	ID        int64
	SessionID int64
	ToolName  string
	Arguments string
	Result    string
	Timestamp int64
}

// InsertToolCall inserts a new tool call
func (db *DB) InsertToolCall(ctx context.Context, t *ToolCallRow) (int64, error) {
	query := `INSERT INTO tool_calls (session_id, tool_name, arguments, result, timestamp) VALUES (?, ?, ?, ?, ?)`
	result, err := db.db.ExecContext(ctx, query, t.SessionID, t.ToolName, t.Arguments, t.Result, t.Timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to insert tool call: %w", err)
	}
	return result.LastInsertId()
}

// GetAllSessions returns all sessions ordered by started_at descending
func (db *DB) GetAllSessions(ctx context.Context) ([]SessionRow, error) {
	query := `SELECT s.id, s.external_id, s.source, s.project_path, s.model, s.provider, s.started_at, s.ended_at,
		s.input_tokens, s.output_tokens, s.cache_creation_tokens, s.cache_read_tokens, s.reasoning_tokens, s.total_tokens, s.cost, ` + messageCountSubquery + `
		FROM sessions s ORDER BY s.started_at DESC`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionRow
	for rows.Next() {
		var s SessionRow
		err := rows.Scan(
			&s.ID, &s.ExternalID, &s.Source, &s.ProjectPath, &s.Model, &s.Provider,
			&s.StartedAt, &s.EndedAt, &s.InputTokens, &s.OutputTokens, &s.CacheCreationTokens,
			&s.CacheReadTokens, &s.ReasoningTokens, &s.TotalTokens, &s.Cost, &s.MessageCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// GetMessagesBySessionID returns all messages for a session
func (db *DB) GetMessagesBySessionID(ctx context.Context, sessionID int64) ([]MessageRow, error) {
	query := `SELECT id, session_id, role, content, timestamp FROM messages WHERE session_id = ? ORDER BY timestamp`

	rows, err := db.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []MessageRow
	for rows.Next() {
		var m MessageRow
		err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// GetToolCallsBySessionID returns all tool calls for a session
func (db *DB) GetToolCallsBySessionID(ctx context.Context, sessionID int64) ([]ToolCallRow, error) {
	query := `SELECT id, session_id, tool_name, arguments, result, timestamp FROM tool_calls WHERE session_id = ? ORDER BY timestamp`

	rows, err := db.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tool calls: %w", err)
	}
	defer rows.Close()

	var toolCalls []ToolCallRow
	for rows.Next() {
		var t ToolCallRow
		err := rows.Scan(&t.ID, &t.SessionID, &t.ToolName, &t.Arguments, &t.Result, &t.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tool call: %w", err)
		}
		toolCalls = append(toolCalls, t)
	}
	return toolCalls, rows.Err()
}

// AggregatedStats holds aggregated statistics
type AggregatedStats struct {
	TotalSessionTime   int64
	TotalInputTokens   int64
	TotalOutputTokens  int64
	TotalCacheCreation int64
	TotalCacheRead     int64
	TotalTokens        int64
	TotalCost          float64
	SessionCount       int64
}

// GetLastSession returns the most recent session within the time period
func (db *DB) GetLastSession(ctx context.Context, source string, since int64) (*SessionRow, error) {
	query := `SELECT s.id, s.external_id, s.source, s.project_path, s.model, s.provider, s.started_at, s.ended_at,
		s.input_tokens, s.output_tokens, s.cache_creation_tokens, s.cache_read_tokens, s.reasoning_tokens, s.total_tokens, ` + messageCountSubquery + `, s.cost
		FROM sessions s WHERE s.source = ? AND s.started_at >= ? ORDER BY s.started_at DESC LIMIT 1`

	row := db.db.QueryRowContext(ctx, query, source, since)
	var s SessionRow
	var endedAt sql.NullInt64
	err := row.Scan(
		&s.ID, &s.ExternalID, &s.Source, &s.ProjectPath, &s.Model, &s.Provider,
		&s.StartedAt, &endedAt, &s.InputTokens, &s.OutputTokens, &s.CacheCreationTokens,
		&s.CacheReadTokens, &s.ReasoningTokens, &s.TotalTokens, &s.MessageCount, &s.Cost,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last session: %w", err)
	}
	if endedAt.Valid {
		s.EndedAt = &endedAt.Int64
	}
	return &s, nil
}

// GetTopModels returns the top N models by session count
func (db *DB) GetTopModels(ctx context.Context, source string, since int64, limit int) ([]ModelUsage, error) {
	query := `SELECT model, COUNT(*) as session_count
		FROM sessions WHERE source = ? AND started_at >= ? AND model IS NOT NULL AND model != ''
		GROUP BY model ORDER BY session_count DESC LIMIT ?`

	rows, err := db.db.QueryContext(ctx, query, source, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top models: %w", err)
	}
	defer rows.Close()

	var models []ModelUsage
	for rows.Next() {
		var m ModelUsage
		if err := rows.Scan(&m.Model, &m.SessionCount); err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

// GetAggregatedStats returns aggregated statistics for the period
func (db *DB) GetAggregatedStats(ctx context.Context, source string, since int64) (*AggregatedStats, error) {
	query := `SELECT
		COALESCE(SUM(CASE WHEN ended_at IS NOT NULL AND ended_at > started_at THEN ended_at - started_at ELSE 0 END), 0) as total_time,
		COALESCE(SUM(input_tokens), 0) as total_input,
		COALESCE(SUM(output_tokens), 0) as total_output,
		COALESCE(SUM(cache_creation_tokens), 0) as total_cache_creation,
		COALESCE(SUM(cache_read_tokens), 0) as total_cache_read,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(cost), 0) as total_cost,
		COUNT(*) as session_count
		FROM sessions WHERE source = ? AND started_at >= ?`

	var stats AggregatedStats
	err := db.db.QueryRowContext(ctx, query, source, since).Scan(
		&stats.TotalSessionTime,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCacheCreation,
		&stats.TotalCacheRead,
		&stats.TotalTokens,
		&stats.TotalCost,
		&stats.SessionCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated stats: %w", err)
	}
	return &stats, nil
}

// GetMessageCount returns the total message count for sessions in the period
func (db *DB) GetMessageCount(ctx context.Context, source string, since int64) (int64, error) {
	query := `SELECT COALESCE(COUNT(m.id), 0)
		FROM messages m
		JOIN sessions s ON m.session_id = s.id
		WHERE s.source = ? AND s.started_at >= ?`

	var count int64
	err := db.db.QueryRowContext(ctx, query, source, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}
	return count, nil
}

// GetMessageCountAll returns the total message count for all sessions in the period
func (db *DB) GetMessageCountAll(ctx context.Context, since int64) (int64, error) {
	query := `SELECT COALESCE(COUNT(m.id), 0)
		FROM messages m
		JOIN sessions s ON m.session_id = s.id
		WHERE s.started_at >= ?`

	var count int64
	err := db.db.QueryRowContext(ctx, query, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}
	return count, nil
}

// GetMessageCountBySessionID returns the total message count for a session
func (db *DB) GetMessageCountBySessionID(ctx context.Context, sessionID int64) (int64, error) {
	query := `SELECT COALESCE(COUNT(*), 0) FROM messages WHERE session_id = ?`

	var count int64
	err := db.db.QueryRowContext(ctx, query, sessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get message count by session: %w", err)
	}
	return count, nil
}

// GetToolCallCount returns the total tool call count for sessions in the period
func (db *DB) GetToolCallCount(ctx context.Context, source string, since int64) (int64, error) {
	query := `SELECT COALESCE(COUNT(t.id), 0)
		FROM tool_calls t
		JOIN sessions s ON t.session_id = s.id
		WHERE s.source = ? AND s.started_at >= ?`

	var count int64
	err := db.db.QueryRowContext(ctx, query, source, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get tool call count: %w", err)
	}
	return count, nil
}

// GetToolCallCountAll returns the total tool call count for all sessions in the period
func (db *DB) GetToolCallCountAll(ctx context.Context, since int64) (int64, error) {
	query := `SELECT COALESCE(COUNT(t.id), 0)
		FROM tool_calls t
		JOIN sessions s ON t.session_id = s.id
		WHERE s.started_at >= ?`

	var count int64
	err := db.db.QueryRowContext(ctx, query, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get tool call count: %w", err)
	}
	return count, nil
}

// GetUniqueProjects returns the count of unique projects in the period
func (db *DB) GetUniqueProjects(ctx context.Context, source string, since int64) (int64, error) {
	query := `SELECT COALESCE(COUNT(DISTINCT project_path), 0)
		FROM sessions WHERE source = ? AND started_at >= ? AND project_path IS NOT NULL`

	var count int64
	err := db.db.QueryRowContext(ctx, query, source, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unique projects: %w", err)
	}
	return count, nil
}

// GetSessionsInPeriod returns all sessions within a time period for debug
func (db *DB) GetSessionsInPeriod(ctx context.Context, source string, since int64) ([]SessionRow, error) {
	query := `SELECT s.id, s.external_id, s.source, s.project_path, s.model, s.provider, s.started_at, s.ended_at,
		s.input_tokens, s.output_tokens, s.cache_creation_tokens, s.cache_read_tokens, s.reasoning_tokens, s.total_tokens, ` + messageCountSubquery + `, s.cost
		FROM sessions s WHERE s.source = ? AND s.started_at >= ? ORDER BY s.started_at DESC`

	rows, err := db.db.QueryContext(ctx, query, source, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionRow
	for rows.Next() {
		var s SessionRow
		var endedAt sql.NullInt64
		err := rows.Scan(
			&s.ID, &s.ExternalID, &s.Source, &s.ProjectPath, &s.Model, &s.Provider,
			&s.StartedAt, &endedAt, &s.InputTokens, &s.OutputTokens, &s.CacheCreationTokens,
			&s.CacheReadTokens, &s.ReasoningTokens, &s.TotalTokens, &s.MessageCount, &s.Cost,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		if endedAt.Valid {
			s.EndedAt = &endedAt.Int64
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// DailySummary represents daily aggregated statistics
type DailySummary struct {
	Date         string
	SessionCount int64
	TotalTime    int64
	TotalTokens  int64
}

// GetDailySummaries returns daily summaries for a time period (used for weekly period)
func (db *DB) GetDailySummaries(ctx context.Context, source string, since int64) ([]DailySummary, error) {
	query := `SELECT date(started_at, 'unixepoch') as day,
		COUNT(*) as sessions,
		COALESCE(SUM(CASE WHEN ended_at IS NOT NULL AND ended_at > started_at THEN ended_at - started_at ELSE 0 END), 0) as total_time,
		COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM sessions
		WHERE source = ? AND started_at >= ?
		GROUP BY day
		ORDER BY day DESC`

	rows, err := db.db.QueryContext(ctx, query, source, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily summaries: %w", err)
	}
	defer rows.Close()

	var summaries []DailySummary
	for rows.Next() {
		var s DailySummary
		if err := rows.Scan(&s.Date, &s.SessionCount, &s.TotalTime, &s.TotalTokens); err != nil {
			return nil, fmt.Errorf("failed to scan daily summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// WeeklySummary represents weekly aggregated statistics
type WeeklySummary struct {
	WeekStart    string
	SessionCount int64
	TotalTime    int64
	TotalTokens  int64
}

// PerAgentStats represents per-agent statistics
type PerAgentStats struct {
	Source             string
	SessionCount       int64
	TotalInputTokens   int64
	TotalOutputTokens  int64
	TotalCacheCreation int64
	TotalCacheRead     int64
	TotalTokens        int64
	TotalCost          float64
	TotalTime          int64
	TotalMessages      int64
}

// GetWeeklySummaries returns weekly summaries for a time period (used for monthly period)
func (db *DB) GetWeeklySummaries(ctx context.Context, source string, since int64) ([]WeeklySummary, error) {
	query := `SELECT strftime('%Y/%m/%d', datetime(min(started_at), 'unixepoch')) as week_start,
		COUNT(*) as sessions,
		COALESCE(SUM(CASE WHEN ended_at IS NOT NULL AND ended_at > started_at THEN ended_at - started_at ELSE 0 END), 0) as total_time,
		COALESCE(SUM(total_tokens), 0) as total_tokens
		FROM sessions
		WHERE source = ? AND started_at >= ?
		GROUP BY strftime('%Y-W%W', started_at, 'unixepoch')
		ORDER BY week_start DESC`

	rows, err := db.db.QueryContext(ctx, query, source, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly summaries: %w", err)
	}
	defer rows.Close()

	var summaries []WeeklySummary
	for rows.Next() {
		var s WeeklySummary
		if err := rows.Scan(&s.WeekStart, &s.SessionCount, &s.TotalTime, &s.TotalTokens); err != nil {
			return nil, fmt.Errorf("failed to scan weekly summary: %w", err)
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// GetAggregatedStatsAll returns aggregated stats for all sources
func (db *DB) GetAggregatedStatsAll(ctx context.Context, since int64) (*AggregatedStats, error) {
	query := `SELECT
		COALESCE(SUM(CASE WHEN ended_at IS NOT NULL AND ended_at > started_at THEN ended_at - started_at ELSE 0 END), 0) as total_time,
		COALESCE(SUM(input_tokens), 0) as total_input,
		COALESCE(SUM(output_tokens), 0) as total_output,
		COALESCE(SUM(cache_creation_tokens), 0) as total_cache_creation,
		COALESCE(SUM(cache_read_tokens), 0) as total_cache_read,
		COALESCE(SUM(total_tokens), 0) as total_tokens,
		COALESCE(SUM(cost), 0) as total_cost,
		COUNT(*) as session_count
		FROM sessions WHERE started_at >= ?`

	var stats AggregatedStats
	err := db.db.QueryRowContext(ctx, query, since).Scan(
		&stats.TotalSessionTime,
		&stats.TotalInputTokens,
		&stats.TotalOutputTokens,
		&stats.TotalCacheCreation,
		&stats.TotalCacheRead,
		&stats.TotalTokens,
		&stats.TotalCost,
		&stats.SessionCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregated stats: %w", err)
	}
	return &stats, nil
}

// GetPerAgentStats returns stats grouped by source
func (db *DB) GetPerAgentStats(ctx context.Context, since int64) ([]PerAgentStats, error) {
	query := `SELECT s.source,
		COUNT(*) as session_count,
		COALESCE(SUM(s.input_tokens), 0) as total_input,
		COALESCE(SUM(s.output_tokens), 0) as total_output,
		COALESCE(SUM(s.cache_creation_tokens), 0) as total_cache_creation,
		COALESCE(SUM(s.cache_read_tokens), 0) as total_cache_read,
		COALESCE(SUM(s.total_tokens), 0) as total_tokens,
		COALESCE(SUM(s.cost), 0) as total_cost,
		COALESCE(SUM(CASE WHEN s.ended_at IS NOT NULL AND s.ended_at > s.started_at THEN s.ended_at - s.started_at ELSE 0 END), 0) as total_time,
		COALESCE(SUM(m.message_count), 0) as total_messages
		FROM sessions s
		LEFT JOIN (
			SELECT session_id, COUNT(*) as message_count FROM messages GROUP BY session_id
		) m ON m.session_id = s.id
		WHERE s.started_at >= ?
		GROUP BY s.source ORDER BY session_count DESC`

	rows, err := db.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query per-agent stats: %w", err)
	}
	defer rows.Close()

	var stats []PerAgentStats
	for rows.Next() {
		var s PerAgentStats
		if err := rows.Scan(&s.Source, &s.SessionCount, &s.TotalInputTokens, &s.TotalOutputTokens, &s.TotalCacheCreation, &s.TotalCacheRead, &s.TotalTokens, &s.TotalCost, &s.TotalTime, &s.TotalMessages); err != nil {
			return nil, fmt.Errorf("failed to scan per-agent stats: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// GetTopModelsAll returns top models across all sources
func (db *DB) GetTopModelsAll(ctx context.Context, since int64, limit int) ([]ModelUsage, error) {
	query := `SELECT model, COUNT(*) as session_count
		FROM sessions WHERE started_at >= ? AND model IS NOT NULL AND model != ''
		GROUP BY model ORDER BY session_count DESC LIMIT ?`

	rows, err := db.db.QueryContext(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top models: %w", err)
	}
	defer rows.Close()

	var models []ModelUsage
	for rows.Next() {
		var m ModelUsage
		if err := rows.Scan(&m.Model, &m.SessionCount); err != nil {
			return nil, fmt.Errorf("failed to scan model: %w", err)
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

// GetUniqueProjectsAll returns unique projects across all agents
func (db *DB) GetUniqueProjectsAll(ctx context.Context, since int64) (int64, error) {
	query := `SELECT COALESCE(COUNT(DISTINCT project_path), 0)
		FROM sessions WHERE started_at >= ? AND project_path IS NOT NULL`

	var count int64
	err := db.db.QueryRowContext(ctx, query, since).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unique projects: %w", err)
	}
	return count, nil
}

// GetRecentSessions returns the most recent N sessions ordered by start time descending
func (db *DB) GetRecentSessions(ctx context.Context, limit int) ([]SessionRow, error) {
	query := `SELECT s.id, s.external_id, s.source, s.project_path, s.model, s.provider, s.started_at, s.ended_at,
		s.input_tokens, s.output_tokens, s.cache_creation_tokens, s.cache_read_tokens, s.reasoning_tokens, s.total_tokens, ` + messageCountSubquery + `, s.cost
		FROM sessions s ORDER BY s.started_at DESC LIMIT ?`

	rows, err := db.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionRow
	for rows.Next() {
		var s SessionRow
		if err := rows.Scan(&s.ID, &s.ExternalID, &s.Source, &s.ProjectPath, &s.Model, &s.Provider,
			&s.StartedAt, &s.EndedAt, &s.InputTokens, &s.OutputTokens, &s.CacheCreationTokens,
			&s.CacheReadTokens, &s.ReasoningTokens, &s.TotalTokens, &s.MessageCount, &s.Cost); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// SetLastSyncTime sets the last sync time for an agent
func (db *DB) SetLastSyncTime(ctx context.Context, agent string, timestamp int64) error {
	query := `INSERT OR REPLACE INTO metadata (key, value, updated_at) VALUES (?, ?, ?)`
	key := "last_sync_" + agent
	_, err := db.db.ExecContext(ctx, query, key, fmt.Sprintf("%d", timestamp), timestamp)
	if err != nil {
		return fmt.Errorf("failed to set last sync time: %w", err)
	}
	return nil
}

// GetLastSyncTime returns the last sync time for an agent (unix timestamp, 0 if never synced)
func (db *DB) GetLastSyncTime(ctx context.Context, agent string) (int64, error) {
	query := `SELECT value FROM metadata WHERE key = ?`
	key := "last_sync_" + agent
	var value string
	err := db.db.QueryRowContext(ctx, query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get last sync time: %w", err)
	}
	var timestamp int64
	_, err = fmt.Sscanf(value, "%d", &timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last sync time: %w", err)
	}
	return timestamp, nil
}
