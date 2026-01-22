// Package agent provides the Auto-DM agent system for Blood on the Clocktower.
package agent

import (
	"context"
	"encoding/json"
	"time"
)

// Phase represents game phases
type Phase string

const (
	PhaseLobby Phase = "lobby"
	PhaseDay   Phase = "day"
	PhaseNight Phase = "night"
	PhaseEnded Phase = "ended"
)

// ActionType defines the types of actions the agent can take
type ActionType string

const (
	ActionSendPublicMessage   ActionType = "send_public_message"
	ActionSendWhisper         ActionType = "send_whisper"
	ActionRequestPlayerAction ActionType = "request_player_action"
	ActionStartVote           ActionType = "start_vote"
	ActionCollectVotes        ActionType = "collect_votes"
	ActionCloseVote           ActionType = "close_vote"
	ActionAdvancePhase        ActionType = "advance_phase"
	ActionSetTimer            ActionType = "set_timer"
	ActionGetRoomState        ActionType = "get_room_state"
	ActionGetRecentEvents     ActionType = "get_recent_events"
	ActionLookupRule          ActionType = "lookup_rule"
	ActionGenerateNarration   ActionType = "generate_narration"
	ActionCreateSummary       ActionType = "create_summary"
	ActionTTSNarrate          ActionType = "tts_narrate"
)

// Action represents an action the AutoDM wants to execute
type Action struct {
	ID         string          `json:"id"`
	Type       ActionType      `json:"type"`
	Args       json.RawMessage `json:"args"`
	Priority   int             `json:"priority"`
	Timeout    time.Duration   `json:"timeout,omitempty"`
	RetryCount int             `json:"retry_count,omitempty"`
}

// ActionResult represents the result of executing an action
type ActionResult struct {
	ActionID  string          `json:"action_id"`
	Success   bool            `json:"success"`
	Output    json.RawMessage `json:"output,omitempty"`
	Error     string          `json:"error,omitempty"`
	Duration  time.Duration   `json:"duration"`
	Timestamp time.Time       `json:"timestamp"`
}

// Plan represents the agent's plan for a turn
type Plan struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"room_id"`
	Reasoning  string    `json:"reasoning"`
	Actions    []Action  `json:"actions"`
	Priority   string    `json:"priority,omitempty"`
	Confidence float64   `json:"confidence,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Observation represents what the agent observed after executing actions
type Observation struct {
	RoomID       string         `json:"room_id"`
	Results      []ActionResult `json:"results"`
	NewEvents    []Event        `json:"new_events"`
	StateChanged bool           `json:"state_changed"`
	Timestamp    time.Time      `json:"timestamp"`
}

// Reflection represents the agent's reflection on its actions
type Reflection struct {
	RoomID      string    `json:"room_id"`
	Summary     string    `json:"summary"`
	Lessons     []string  `json:"lessons,omitempty"`
	Adjustments []string  `json:"adjustments,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Event represents a game event
type Event struct {
	RoomID    string          `json:"room_id"`
	Seq       int64           `json:"seq"`
	EventID   string          `json:"event_id"`
	EventType string          `json:"event_type"`
	ActorID   string          `json:"actor_id"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// PlayerState represents the state of a player
type PlayerState struct {
	UserID string   `json:"user_id"`
	Name   string   `json:"name"`
	Role   string   `json:"role"`
	Alive  bool     `json:"alive"`
	IsDM   bool     `json:"is_dm"`
	Tags   []string `json:"tags,omitempty"`
}

// NominationState represents an active nomination
type NominationState struct {
	Nominator string          `json:"nominator"`
	Nominee   string          `json:"nominee"`
	Votes     map[string]bool `json:"votes"`
	Resolved  bool            `json:"resolved"`
	Result    string          `json:"result"`
	Deadline  time.Time       `json:"deadline,omitempty"`
}

// RoomState represents the current state of a game room
type RoomState struct {
	RoomID     string                 `json:"room_id"`
	Phase      Phase                  `json:"phase"`
	DayCount   int                    `json:"day_count"`
	NightCount int                    `json:"night_count"`
	Players    map[string]PlayerState `json:"players"`
	Nomination *NominationState       `json:"nomination,omitempty"`
	LastSeq    int64                  `json:"last_seq"`
	Timers     map[string]time.Time   `json:"timers,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// AgentContext provides context for agent execution
type AgentContext struct {
	RoomID        string
	RunID         string
	Phase         Phase
	RecentEvents  []Event
	PendingInputs []PendingInput
	Timers        map[string]time.Time
	MemoryContext *MemoryContext
	ViewerID      string
	StartTime     time.Time
}

// PendingInput represents an input the agent is waiting for
type PendingInput struct {
	UserID     string          `json:"user_id"`
	ActionType string          `json:"action_type"`
	Deadline   time.Time       `json:"deadline"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// MemoryContext holds memory-related context for agent execution
type MemoryContext struct {
	ShortTerm    []Event                `json:"short_term"`
	LongTerm     []MemoryEntry          `json:"long_term"`
	PlayerModels map[string]PlayerModel `json:"player_models,omitempty"`
	GameSummary  string                 `json:"game_summary,omitempty"`
}

// MemoryEntry represents a long-term memory entry
type MemoryEntry struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"` // "rule", "summary", "profile", "event"
	Content   string          `json:"content"`
	Embedding []float32       `json:"embedding,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	Score     float64         `json:"score,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// PlayerModel represents behavioral model of a player
type PlayerModel struct {
	UserID            string    `json:"user_id"`
	Playstyle         string    `json:"playstyle"` // aggressive, passive, analytical, etc.
	TrustScore        float64   `json:"trust_score"`
	DeceptionScore    float64   `json:"deception_score"`
	ParticipationRate float64   `json:"participation_rate"`
	VotingPatterns    []string  `json:"voting_patterns,omitempty"`
	LastUpdated       time.Time `json:"last_updated"`
}

// AgentRun records a single agent execution run
type AgentRun struct {
	ID           string          `json:"id"`
	RoomID       string          `json:"room_id"`
	AgentName    string          `json:"agent_name"`
	SeqFrom      int64           `json:"seq_from"`
	SeqTo        int64           `json:"seq_to"`
	InputDigest  string          `json:"input_digest"`
	PlanJSON     json.RawMessage `json:"plan_json,omitempty"`
	ToolCalls    []ToolCallAudit `json:"tool_calls,omitempty"`
	OutputDigest string          `json:"output_digest"`
	Status       string          `json:"status"`
	LatencyMs    int64           `json:"latency_ms"`
	ErrorText    string          `json:"error_text,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ToolCallAudit records a single tool call for auditing
type ToolCallAudit struct {
	ID         string          `json:"id"`
	RunID      string          `json:"run_id"`
	ToolName   string          `json:"tool_name"`
	Args       json.RawMessage `json:"args"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	DurationMs int64           `json:"duration_ms"`
	CreatedAt  time.Time       `json:"created_at"`
}

// SubAgent defines the interface for all sub-agents in the AutoDM system
type SubAgent interface {
	Name() string
	Description() string
	Execute(ctx context.Context, agentCtx *AgentContext) (*AgentOutput, error)
}

// AgentOutput represents the output from an agent execution
type AgentOutput struct {
	AgentName  string          `json:"agent_name"`
	Actions    []Action        `json:"actions,omitempty"`
	Message    string          `json:"message,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
	Confidence float64         `json:"confidence,omitempty"`
}

// AutoDMStatus represents the status of the AutoDM for a room
type AutoDMStatus struct {
	RoomID     string    `json:"room_id"`
	Active     bool      `json:"active"`
	Phase      Phase     `json:"phase"`
	LastRunID  string    `json:"last_run_id,omitempty"`
	LastRunAt  time.Time `json:"last_run_at,omitempty"`
	RunCount   int64     `json:"run_count"`
	ErrorCount int64     `json:"error_count"`
	StartedAt  time.Time `json:"started_at,omitempty"`
}

// SendMessageArgs arguments for sending a public message
type SendMessageArgs struct {
	RoomID   string            `json:"room_id"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SendWhisperArgs arguments for sending a whisper
type SendWhisperArgs struct {
	RoomID   string            `json:"room_id"`
	ToUserID string            `json:"to_user_id"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RequestActionArgs arguments for requesting player action
type RequestActionArgs struct {
	RoomID     string        `json:"room_id"`
	UserID     string        `json:"user_id"`
	ActionType string        `json:"action_type"`
	Deadline   time.Duration `json:"deadline"`
	Prompt     string        `json:"prompt,omitempty"`
}

// StartVoteArgs arguments for starting a vote
type StartVoteArgs struct {
	RoomID   string        `json:"room_id"`
	Target   string        `json:"target"`
	Deadline time.Duration `json:"deadline"`
}

// AdvancePhaseArgs arguments for advancing the phase
type AdvancePhaseArgs struct {
	RoomID    string `json:"room_id"`
	NextPhase Phase  `json:"next_phase"`
	Reason    string `json:"reason,omitempty"`
}

// SetTimerArgs arguments for setting a timer
type SetTimerArgs struct {
	RoomID    string        `json:"room_id"`
	TimerType string        `json:"timer_type"`
	Duration  time.Duration `json:"duration"`
}

// GetRoomStateArgs arguments for getting room state
type GetRoomStateArgs struct {
	RoomID   string `json:"room_id"`
	ViewerID string `json:"viewer_id,omitempty"`
}

// GetRecentEventsArgs arguments for getting recent events
type GetRecentEventsArgs struct {
	RoomID   string `json:"room_id"`
	SinceSeq int64  `json:"since_seq"`
	Limit    int    `json:"limit,omitempty"`
}

// LookupRuleArgs arguments for looking up rules
type LookupRuleArgs struct {
	Query     string   `json:"query"`
	RoleNames []string `json:"role_names,omitempty"`
	TopK      int      `json:"top_k,omitempty"`
}

// GenerateNarrationArgs arguments for generating narration
type GenerateNarrationArgs struct {
	Context   string `json:"context"`
	Style     string `json:"style,omitempty"` // dramatic, humorous, mysterious
	MaxLength int    `json:"max_length,omitempty"`
}

// CreateSummaryArgs arguments for creating a summary
type CreateSummaryArgs struct {
	RoomID    string `json:"room_id"`
	FromSeq   int64  `json:"from_seq"`
	ToSeq     int64  `json:"to_seq"`
	ForPlayer string `json:"for_player,omitempty"`
}
