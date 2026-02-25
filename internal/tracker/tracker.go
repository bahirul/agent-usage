package tracker

// Agent represents the type of AI coding agent
type Agent string

const (
	AgentCodex      Agent = "codex"
	AgentClaudeCode Agent = "claude"
)

// Session represents a tracking session for an agent
type Session struct {
	Agent      Agent
	StartTime  int64 // Unix timestamp
	EndTime    int64 // Unix timestamp
	InputTokens  int
	OutputTokens int
}

// Tracker is the interface for tracking agent usage
type Tracker interface {
	StartSession(agent Agent) (*Session, error)
	EndSession(session *Session) error
	GetUsage(agent Agent) (*UsageStats, error)
}

// UsageStats represents aggregated usage statistics
type UsageStats struct {
	Agent           Agent
	TotalSessions   int
	TotalInputTokens  int
	TotalOutputTokens int
}
