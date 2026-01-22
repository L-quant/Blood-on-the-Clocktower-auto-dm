// Package agent provides the Auto-DM agent for Blood on the Clocktower.
//
// This package implements a multi-agent system that can automatically run
// Blood on the Clocktower games as the Storyteller (DM).
//
// Architecture:
//
//	┌────────────────────────────────────────────────────────────┐
//	│                        Orchestrator                        │
//	│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
//	│  │Moderator │  │ Narrator │  │  Rules   │  │Summarizer│   │
//	│  └──────────┘  └──────────┘  └──────────┘  └──────────┘   │
//	│                    ┌──────────────┐                        │
//	│                    │PlayerModeler │                        │
//	│                    └──────────────┘                        │
//	├────────────────────────────────────────────────────────────┤
//	│                       LLM Router                           │
//	│         (Routes to different models by task type)          │
//	├────────────────────────────────────────────────────────────┤
//	│     Memory Manager            │      Tool Registry         │
//	│  (Short-term + Long-term)     │   (Game operations)        │
//	└────────────────────────────────────────────────────────────┘
//
// # Sub-Agents
//
//   - Moderator: Manages game flow, phases, nominations, voting
//   - Narrator: Generates atmospheric narration and announcements
//   - Rules: Answers rules questions and validates actions
//   - Summarizer: Creates summaries of game state and events
//   - PlayerModeler: Analyzes player behavior (DM-only tool)
//
// # Usage
//
//	cfg := agent.Config{
//	    RoomID: "room-123",
//	    LLM: llm.RoutingConfig{
//	        Default: llm.Config{
//	            BaseURL: "https://api.openai.com/v1",
//	            APIKey:  os.Getenv("OPENAI_API_KEY"),
//	            Model:   "gpt-4o",
//	        },
//	    },
//	}
//
//	dm := agent.NewAutoDM(cfg)
//	dm.SetCommander(gameCommander)
//	dm.Start()
//
//	// Process game events
//	response, err := dm.ProcessEvent(ctx, event)
package agent

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/core"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/memory"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/tools"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// AutoDM is the main Auto-DM agent.
type AutoDM struct {
	mu           sync.RWMutex
	orchestrator *core.Orchestrator
	logger       *slog.Logger
	enabled      bool
	dispatcher   CommandDispatcher
	stateGetter  func() interface{}
}

// CommandDispatcher dispatches commands to the game engine.
type CommandDispatcher interface {
	DispatchAsync(cmd types.CommandEnvelope) error
}

// Type aliases for external use
type LLMRoutingConfig = llm.RoutingConfig
type LLMClientConfig = llm.Config
type MemoryConfig = memory.Config

// Config configures the Auto-DM.
type Config struct {
	RoomID  string
	LLM     LLMRoutingConfig
	Memory  MemoryConfig
	Logger  *slog.Logger
	Enabled bool
}

// NewAutoDM creates a new Auto-DM instance.
func NewAutoDM(cfg Config) *AutoDM {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	orch := core.New(core.Config{
		RoomID:       cfg.RoomID,
		LLMConfig:    cfg.LLM,
		MemoryConfig: cfg.Memory,
		Logger:       cfg.Logger,
	})

	return &AutoDM{
		orchestrator: orch,
		logger:       cfg.Logger,
		enabled:      cfg.Enabled,
	}
}

// Enabled returns whether the Auto-DM is enabled.
func (a *AutoDM) Enabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// SetEnabled enables or disables the Auto-DM.
func (a *AutoDM) SetEnabled(enabled bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.enabled = enabled
}

// SetDispatcher sets the command dispatcher and state getter for integration with RoomActor.
func (a *AutoDM) SetDispatcher(dispatcher CommandDispatcher, stateGetter func() interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.dispatcher = dispatcher
	a.stateGetter = stateGetter
}

// SetCommander sets the game commander for executing game actions.
func (a *AutoDM) SetCommander(commander tools.GameCommander) {
	a.orchestrator.SetCommander(commander)
}

// SetRulesProvider sets the rules provider for game rules lookup.
func (a *AutoDM) SetRulesProvider(rules tools.RulesProvider) {
	a.orchestrator.SetRulesProvider(rules)
}

// Start activates the Auto-DM.
func (a *AutoDM) Start() {
	a.orchestrator.Start()
}

// Stop deactivates the Auto-DM.
func (a *AutoDM) Stop() {
	a.orchestrator.Stop()
}

// IsActive returns whether the Auto-DM is running.
func (a *AutoDM) IsActive() bool {
	return a.orchestrator.IsActive()
}

// ProcessEvent handles a game event and returns a response.
func (a *AutoDM) ProcessEvent(ctx context.Context, event Event) (*Response, error) {
	coreEvent := core.Event{
		Type:        event.Type,
		Description: event.Description,
		PlayerID:    event.PlayerID,
		Data:        event.Data,
	}

	coreResp, err := a.orchestrator.ProcessEvent(ctx, coreEvent)
	if err != nil {
		return nil, err
	}

	return &Response{
		Message:     coreResp.Message,
		ShouldSpeak: coreResp.ShouldSpeak,
	}, nil
}

// UpdateGameState updates the agent's view of the game state.
func (a *AutoDM) UpdateGameState(state *GameState) {
	coreState := &core.GameState{
		RoomID:     state.RoomID,
		Phase:      state.Phase,
		DayNumber:  state.DayNumber,
		Edition:    state.Edition,
		Script:     state.Script,
		IsStarted:  state.IsStarted,
		IsFinished: state.IsFinished,
	}

	for _, p := range state.Players {
		coreState.Players = append(coreState.Players, core.Player{
			ID:        p.ID,
			Name:      p.Name,
			Role:      p.Role,
			IsAlive:   p.IsAlive,
			HasVoted:  p.HasVoted,
			Seat:      p.Seat,
			Reminders: p.Reminders,
		})
	}

	for _, n := range state.Nominations {
		coreState.Nominations = append(coreState.Nominations, core.Nomination{
			Nominator: n.Nominator,
			Nominee:   n.Nominee,
			Votes:     n.Votes,
			Threshold: n.Threshold,
		})
	}

	a.orchestrator.UpdateGameState(coreState)
}

// GetSummary returns a summary of the current game.
func (a *AutoDM) GetSummary(ctx context.Context, forDM bool) (string, error) {
	return a.orchestrator.GetSummary(ctx, forDM)
}

// AnalyzePlayers returns player behavior analysis (DM only).
func (a *AutoDM) AnalyzePlayers(ctx context.Context) (string, error) {
	return a.orchestrator.AnalyzePlayers(ctx)
}

// Event represents a game event.
type Event struct {
	Type        string                 // Event type: phase_change, nomination, vote, death, question, night_action
	Description string                 // Human-readable description
	PlayerID    string                 // Related player ID (if applicable)
	Data        map[string]interface{} // Additional event data
}

// Response is the Auto-DM's response to an event.
type Response struct {
	Message     string // The message/narration to send
	ShouldSpeak bool   // Whether to broadcast to players
}

// GameState represents the game state.
type GameState struct {
	RoomID      string
	Phase       string
	DayNumber   int
	Players     []Player
	Nominations []Nomination
	Edition     string
	Script      []string
	IsStarted   bool
	IsFinished  bool
}

// Player represents a player.
type Player struct {
	ID        string
	Name      string
	Role      string
	IsAlive   bool
	HasVoted  bool
	Seat      int
	Reminders []string
}

// Nomination represents a nomination.
type Nomination struct {
	Nominator string
	Nominee   string
	Votes     int
	Threshold int
}

// OnEvent is called by RoomActor when game events occur.
// It processes the event and generates appropriate responses.
func (a *AutoDM) OnEvent(ctx context.Context, ev types.Event, state interface{}) {
	if !a.Enabled() {
		return
	}

	// Convert types.Event to agent.Event
	event := a.convertEvent(ev)

	// Process the event
	resp, err := a.ProcessEvent(ctx, event)
	if err != nil {
		a.logger.Error("AutoDM failed to process event", "error", err, "event_type", ev.EventType)
		return
	}

	// If we have a response that should be spoken, dispatch a chat command
	if resp != nil && resp.ShouldSpeak && resp.Message != "" {
		a.sendMessage(ctx, ev.RoomID, resp.Message)
	}
}

func (a *AutoDM) convertEvent(ev types.Event) Event {
	event := Event{
		Type:        ev.EventType,
		Description: ev.EventType,
		Data:        make(map[string]interface{}),
	}

	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal(ev.Payload, &payload); err == nil {
		event.Data = payload
	}

	// Map event types to our internal types
	switch ev.EventType {
	case "phase.night", "phase.day":
		event.Type = "phase_change"
		event.Data["new_phase"] = ev.EventType
	case "nomination.created":
		event.Type = "nomination"
	case "vote.cast":
		event.Type = "vote"
	case "execution.resolved":
		event.Type = "death"
		event.Data["cause"] = "execution"
	case "game.started", "game.ended":
		event.Type = "phase_change"
	}

	event.PlayerID = ev.ActorUserID
	event.Description = formatEventDescription(ev.EventType, event.Data)

	return event
}

func formatEventDescription(eventType string, data map[string]interface{}) string {
	switch eventType {
	case "phase.night":
		return "Night phase begins"
	case "phase.day":
		return "Day phase begins"
	case "nomination.created":
		return "A nomination has been made"
	case "vote.cast":
		return "A vote has been cast"
	case "execution.resolved":
		return "An execution has occurred"
	case "game.started":
		return "The game has started"
	case "game.ended":
		return "The game has ended"
	default:
		return eventType
	}
}

func (a *AutoDM) sendMessage(ctx context.Context, roomID, message string) {
	a.mu.RLock()
	dispatcher := a.dispatcher
	a.mu.RUnlock()

	if dispatcher == nil {
		return
	}

	cmd := types.CommandEnvelope{
		RoomID:      roomID,
		Type:        "chat.send",
		ActorUserID: "auto-dm",
		CommandID:   generateCommandID(),
		Payload:     json.RawMessage(`{"message":"` + escapeJSON(message) + `","from":"auto-dm"}`),
	}

	if err := dispatcher.DispatchAsync(cmd); err != nil {
		a.logger.Error("Failed to send AutoDM message", "error", err)
	}
}

func generateCommandID() string {
	return "autodm-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[i%len(letters)]
	}
	return string(b)
}

func escapeJSON(s string) string {
	// Simple JSON string escaping
	result := ""
	for _, c := range s {
		switch c {
		case '"':
			result += `\"`
		case '\\':
			result += `\\`
		case '\n':
			result += `\n`
		case '\r':
			result += `\r`
		case '\t':
			result += `\t`
		default:
			result += string(c)
		}
	}
	return result
}
