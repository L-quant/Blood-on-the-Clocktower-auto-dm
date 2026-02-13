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
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/core"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/memory"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/tools"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/engine"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/mcp"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

const (
	autoDMEventTaskType = "autodm_event"
	defaultEventTimeout = 8 * time.Second
)

// AutoDM is the main Auto-DM agent.
type AutoDM struct {
	mu           sync.RWMutex
	orchestrator *core.Orchestrator
	logger       *slog.Logger
	enabled      bool
	dispatcher   CommandDispatcher
	stateGetter  func() interface{}
	retriever    RuleRetriever
	taskQueue    TaskQueue
	eventTimeout time.Duration
	mcpRegistry  *mcp.Registry
}

// CommandDispatcher dispatches commands to the game engine.
type CommandDispatcher interface {
	DispatchAsync(cmd types.CommandEnvelope) error
}

// Type aliases for external use
type LLMRoutingConfig = llm.RoutingConfig
type LLMClientConfig = llm.Config
type MemoryConfig = memory.Config

// RuleRetriever interface for RAG
type RuleRetriever interface {
	Retrieve(ctx context.Context, query string, limit int) ([]RetrieveResult, error)
}

// RetrieveResult is the result from RAG retrieval
type RetrieveResult struct {
	Content  string
	Score    float64
	Metadata map[string]interface{}
}

// TaskQueue interface for async tasks
type TaskQueue interface {
	Publish(ctx context.Context, task interface{}) error
}

// AsyncEventTask is the payload published to RabbitMQ for out-of-band AutoDM processing.
type AsyncEventTask struct {
	Type   string
	RoomID string
	Event  types.Event
}

// Config configures the Auto-DM.
type Config struct {
	RoomID    string
	LLM       LLMRoutingConfig
	Memory    MemoryConfig
	Logger    *slog.Logger
	Enabled   bool
	Retriever RuleRetriever
	TaskQueue TaskQueue
}

// NewAutoDM creates a new Auto-DM instance.
func NewAutoDM(cfg Config) *AutoDM {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	eventTimeout := cfg.LLM.Default.Timeout
	if eventTimeout <= 0 {
		eventTimeout = defaultEventTimeout
	}

	orch := core.New(core.Config{
		RoomID:       cfg.RoomID,
		LLMConfig:    cfg.LLM,
		MemoryConfig: cfg.Memory,
		Logger:       cfg.Logger,
	})

	a := &AutoDM{
		orchestrator: orch,
		logger:       cfg.Logger,
		enabled:      cfg.Enabled,
		retriever:    cfg.Retriever,
		taskQueue:    cfg.TaskQueue,
		eventTimeout: eventTimeout,
	}
	a.initMCPRegistry()
	return a
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
	fmt.Printf("[AutoDM] DEBUG: OnEvent Triggered: %s (Actor: %s)\n", ev.EventType, ev.ActorUserID)
	if !a.Enabled() {
		fmt.Println("[AutoDM] DEBUG: AutoDM is disabled")
		return
	}
	if (ev.ActorUserID == "autodm" || ev.ActorUserID == "auto-dm") &&
		(ev.EventType == "public.chat" || ev.EventType == "whisper.sent") {
		return
	}

	// Skip non-game events that don't need AI narration
	switch ev.EventType {
	case "player.joined", "player.left", "seat.claimed", "room.settings.changed":
		return
	}

	a.updateGameStateFromEngineState(state)

	if a.publishAsyncTask(ctx, ev) {
		fmt.Println("[AutoDM] DEBUG: Processing Async")
		return
	}
	fmt.Println("[AutoDM] DEBUG: Processing Inline")
	if err := a.ProcessQueuedEvent(ctx, ev); err != nil {
		a.logger.Error("AutoDM failed to process event", "error", err, "event_type", ev.EventType)
	}
}

// ProcessQueuedEvent executes an event that was dequeued by RabbitMQ workers.
// It bypasses queue publish to avoid enqueue loops.
func (a *AutoDM) ProcessQueuedEvent(ctx context.Context, ev types.Event) error {
	if !a.Enabled() {
		return nil
	}

	event := a.convertEvent(ev)
	a.logger.Info("AutoDM processing event",
		"type", event.Type,
		"description", event.Description,
		"enabled", a.Enabled())

	a.injectRuleContext(ctx, &event)

	processCtx, cancel := context.WithTimeout(ctx, a.eventTimeout)
	defer cancel()

	resp, err := a.ProcessEvent(processCtx, event)
	if err != nil {
		if fallback := defaultMessageForEvent(ev.EventType); fallback != "" {
			a.sendMessage(ctx, ev.RoomID, fallback)
		}
		return err
	}

	if resp != nil && resp.ShouldSpeak && resp.Message != "" {
		a.sendMessage(ctx, ev.RoomID, resp.Message)
	}
	return nil
}

func (a *AutoDM) publishAsyncTask(ctx context.Context, ev types.Event) bool {
	a.mu.RLock()
	taskQueue := a.taskQueue
	a.mu.RUnlock()
	if taskQueue == nil {
		return false
	}

	task := AsyncEventTask{
		Type:   autoDMEventTaskType,
		RoomID: ev.RoomID,
		Event:  ev,
	}
	if err := taskQueue.Publish(ctx, task); err != nil {
		a.logger.Warn("failed to enqueue AutoDM event task, falling back to inline processing", "error", err, "event_type", ev.EventType)
		return false
	}
	return true
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
	case "phase.first_night":
		event.Type = "phase_change"
		event.Data["new_phase"] = "night"
		event.Data["night_type"] = "first_night"
	case "phase.night":
		event.Type = "phase_change"
		event.Data["new_phase"] = "night"
	case "phase.day":
		event.Type = "phase_change"
		event.Data["new_phase"] = "day"
	case "phase.nomination":
		event.Type = "phase_change"
		event.Data["new_phase"] = "nomination"
	case "nomination.created":
		event.Type = "nomination"
		event.Data["nominator"] = ev.ActorUserID
	case "vote.cast":
		event.Type = "vote"
	case "execution.resolved":
		event.Type = "death"
		event.Data["cause"] = "execution"
		if executed, ok := event.Data["executed"]; ok {
			event.Data["player_name"] = executed
		}
	case "game.started", "game.ended":
		event.Type = "phase_change"
	}

	event.PlayerID = ev.ActorUserID
	event.Description = formatEventDescription(ev.EventType, event.Data)

	return event
}

func (a *AutoDM) injectRuleContext(ctx context.Context, event *Event) {
	if event == nil {
		return
	}
	a.mu.RLock()
	retriever := a.retriever
	a.mu.RUnlock()
	if retriever == nil {
		return
	}

	query := buildRuleQuery(*event)
	if query == "" {
		return
	}

	retrieveCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	results, err := retriever.Retrieve(retrieveCtx, query, 2)
	if err != nil || len(results) == 0 {
		return
	}

	snippets := make([]string, 0, len(results))
	for _, r := range results {
		content := strings.TrimSpace(r.Content)
		if content == "" {
			continue
		}
		if len(content) > 180 {
			content = content[:180] + "..."
		}
		snippets = append(snippets, content)
	}
	if len(snippets) == 0 {
		return
	}

	event.Data["rule_context"] = snippets
	event.Description = event.Description + "\nRelevant rule context:\n- " + strings.Join(snippets, "\n- ")
}

func buildRuleQuery(event Event) string {
	switch event.Type {
	case "phase_change":
		if nightType, ok := event.Data["night_type"].(string); ok && nightType == "first_night" {
			return "first night setup rules in Blood on the Clocktower"
		}
		if phase, ok := event.Data["new_phase"].(string); ok && phase != "" {
			return "phase transition to " + phase + " in Blood on the Clocktower"
		}
		return "phase transition in Blood on the Clocktower"
	case "nomination":
		return "nomination and voting rules in Blood on the Clocktower"
	case "vote":
		return "voting threshold and ghost vote rules in Blood on the Clocktower"
	case "death":
		return "execution and death resolution rules in Blood on the Clocktower"
	default:
		return ""
	}
}

func formatEventDescription(eventType string, data map[string]interface{}) string {
	switch eventType {
	case "phase.first_night":
		return "First night phase begins"
	case "phase.night":
		return "Night phase begins"
	case "phase.day":
		return "Day phase begins"
	case "phase.nomination":
		return "Nomination phase begins"
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
	case "public.chat":
		sender, _ := data["sender_name"].(string)
		msg, _ := data["message"].(string)
		if sender == "" {
			sender = "Player"
		}
		return fmt.Sprintf("%s says: %s", sender, msg)
	case "whisper.sent":
		sender, _ := data["sender_name"].(string)
		msg, _ := data["message"].(string)
		if sender == "" {
			sender = "Player"
		}
		return fmt.Sprintf("%s whispers to DM: %s", sender, msg)
	default:
		return eventType
	}
}

func (a *AutoDM) sendMessage(ctx context.Context, roomID, message string) {
	if strings.TrimSpace(message) == "" || strings.TrimSpace(roomID) == "" {
		return
	}

	a.mu.RLock()
	registry := a.mcpRegistry
	a.mu.RUnlock()
	if registry != nil {
		params, _ := json.Marshal(map[string]string{
			"room_id": roomID,
			"message": message,
		})
		result := registry.Invoke(ctx, mcp.ToolCall{
			ID:         generateCommandID(),
			ToolName:   "send_public_message",
			Parameters: params,
			Timestamp:  time.Now().UnixMilli(),
		})
		if result.Success {
			return
		}
		a.logger.Error("MCP send_public_message failed", "error", result.Error)
	}

	payload, _ := json.Marshal(map[string]string{
		"message": message,
		"from":    "auto-dm",
	})
	cmdID := generateCommandID()
	cmd := types.CommandEnvelope{
		CommandID:      cmdID,
		IdempotencyKey: cmdID,
		RoomID:         roomID,
		Type:           "public_chat",
		ActorUserID:    "autodm",
		Payload:        payload,
	}

	if err := a.dispatchCommand(cmd); err != nil {
		a.logger.Error("Failed to send AutoDM message", "error", err)
	}
}

func generateCommandID() string {
	return uuid.NewString()
}

func defaultMessageForEvent(eventType string) string {
	switch eventType {
	case "phase.day":
		return "☀️ 天亮了，开始讨论并寻找隐藏的邪恶吧。"
	case "phase.night":
		return "🌙 夜幕降临，请等待夜晚行动结算。"
	case "nomination.created":
		return "📣 提名已发起，请进行陈述与投票。"
	case "game.started":
		return "🎲 游戏开始，愿好运站在你这边。"
	case "game.ended":
		return "🏁 对局结束，感谢各位参与。"
	default:
		return ""
	}
}

func (a *AutoDM) dispatchCommand(cmd types.CommandEnvelope) error {
	a.mu.RLock()
	dispatcher := a.dispatcher
	a.mu.RUnlock()
	if dispatcher == nil {
		return errors.New("AutoDM dispatcher is not configured")
	}
	return dispatcher.DispatchAsync(cmd)
}

func (a *AutoDM) initMCPRegistry() {
	registry := mcp.NewRegistry()
	minLen, maxLen := 1, 2000
	phaseEnum := []string{"day", "night", "nomination"}

	_ = registry.Register(mcp.ToolDefinition{
		Name:        "send_public_message",
		Description: "Send a public message into a room",
		Category:    mcp.CategoryCommunication,
		Parameters: map[string]mcp.ParamSchema{
			"room_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"message": {
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
		},
		Required: []string{"room_id", "message"},
	}, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p struct {
			RoomID  string `json:"room_id"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		cmdID := generateCommandID()
		payload, _ := json.Marshal(map[string]string{
			"message": p.Message,
			"from":    "auto-dm",
		})
		cmd := types.CommandEnvelope{
			CommandID:      cmdID,
			IdempotencyKey: cmdID,
			RoomID:         p.RoomID,
			Type:           "public_chat",
			ActorUserID:    "autodm",
			Payload:        payload,
		}
		if err := a.dispatchCommand(cmd); err != nil {
			return nil, err
		}
		return map[string]string{"status": "sent"}, nil
	})

	_ = registry.Register(mcp.ToolDefinition{
		Name:        "send_private_message",
		Description: "Send a private whisper to one player",
		Category:    mcp.CategoryCommunication,
		Parameters: map[string]mcp.ParamSchema{
			"room_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"to_user_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"message": {
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
		},
		Required: []string{"room_id", "to_user_id", "message"},
	}, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p struct {
			RoomID   string `json:"room_id"`
			ToUserID string `json:"to_user_id"`
			Message  string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}

		cmdID := generateCommandID()
		payload, _ := json.Marshal(map[string]string{
			"to_user_id": p.ToUserID,
			"message":    p.Message,
			"from":       "auto-dm",
		})
		cmd := types.CommandEnvelope{
			CommandID:      cmdID,
			IdempotencyKey: cmdID,
			RoomID:         p.RoomID,
			Type:           "whisper",
			ActorUserID:    "autodm",
			Payload:        payload,
		}
		if err := a.dispatchCommand(cmd); err != nil {
			return nil, err
		}
		return map[string]string{"status": "sent"}, nil
	})

	_ = registry.Register(mcp.ToolDefinition{
		Name:        "request_player_confirmation",
		Description: "Ask a player to confirm or reject an action",
		Category:    mcp.CategoryModeration,
		Parameters: map[string]mcp.ParamSchema{
			"room_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"to_user_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"prompt": {
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
		},
		Required: []string{"room_id", "to_user_id", "prompt"},
	}, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p struct {
			RoomID   string `json:"room_id"`
			ToUserID string `json:"to_user_id"`
			Prompt   string `json:"prompt"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}

		cmdID := generateCommandID()
		whisperPayload, _ := json.Marshal(map[string]string{
			"to_user_id": p.ToUserID,
			"message":    "[确认请求] " + p.Prompt + "（回复 yes/no）",
			"from":       "auto-dm",
		})
		whisperCmd := types.CommandEnvelope{
			CommandID:      cmdID,
			IdempotencyKey: cmdID,
			RoomID:         p.RoomID,
			Type:           "whisper",
			ActorUserID:    "autodm",
			Payload:        whisperPayload,
		}
		if err := a.dispatchCommand(whisperCmd); err != nil {
			return nil, err
		}

		eventCmdID := generateCommandID()
		eventPayload, _ := json.Marshal(map[string]interface{}{
			"event_type": "confirmation.requested",
			"data": map[string]string{
				"to_user_id": p.ToUserID,
				"prompt":     p.Prompt,
			},
		})
		eventCmd := types.CommandEnvelope{
			CommandID:      eventCmdID,
			IdempotencyKey: eventCmdID,
			RoomID:         p.RoomID,
			Type:           "write_event",
			ActorUserID:    "autodm",
			Payload:        eventPayload,
		}
		if err := a.dispatchCommand(eventCmd); err != nil {
			return nil, err
		}
		return map[string]string{"status": "requested"}, nil
	})

	_ = registry.Register(mcp.ToolDefinition{
		Name:        "toggle_voting",
		Description: "Enable or disable voting mode",
		Category:    mcp.CategoryGameControl,
		Parameters: map[string]mcp.ParamSchema{
			"room_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"enabled": {
				Type: "boolean",
			},
			"reason": {
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
		},
		Required: []string{"room_id", "enabled"},
	}, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p struct {
			RoomID  string `json:"room_id"`
			Enabled bool   `json:"enabled"`
			Reason  string `json:"reason"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}

		targetPhase := "day"
		if p.Enabled {
			targetPhase = "nomination"
		}

		cmdID := generateCommandID()
		payload, _ := json.Marshal(map[string]string{
			"phase":  targetPhase,
			"reason": p.Reason,
		})
		cmd := types.CommandEnvelope{
			CommandID:      cmdID,
			IdempotencyKey: cmdID,
			RoomID:         p.RoomID,
			Type:           "advance_phase",
			ActorUserID:    "autodm",
			Payload:        payload,
		}
		if err := a.dispatchCommand(cmd); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "updated", "enabled": p.Enabled}, nil
	})

	_ = registry.Register(mcp.ToolDefinition{
		Name:        "advance_phase",
		Description: "Advance game phase deterministically",
		Category:    mcp.CategoryGameControl,
		Parameters: map[string]mcp.ParamSchema{
			"room_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"phase": {
				Type: "string",
				Enum: phaseEnum,
			},
			"reason": {
				Type:      "string",
				MinLength: &minLen,
				MaxLength: &maxLen,
			},
		},
		Required: []string{"room_id", "phase"},
	}, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p struct {
			RoomID string `json:"room_id"`
			Phase  string `json:"phase"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}

		cmdID := generateCommandID()
		payload, _ := json.Marshal(map[string]string{
			"phase":  p.Phase,
			"reason": p.Reason,
		})
		cmd := types.CommandEnvelope{
			CommandID:      cmdID,
			IdempotencyKey: cmdID,
			RoomID:         p.RoomID,
			Type:           "advance_phase",
			ActorUserID:    "autodm",
			Payload:        payload,
		}
		if err := a.dispatchCommand(cmd); err != nil {
			return nil, err
		}
		return map[string]string{"status": "advanced", "phase": p.Phase}, nil
	})

	_ = registry.Register(mcp.ToolDefinition{
		Name:        "write_event",
		Description: "Write an auditable custom event into the immutable stream",
		Category:    mcp.CategoryModeration,
		Parameters: map[string]mcp.ParamSchema{
			"room_id": {
				Type:      "string",
				MinLength: &minLen,
			},
			"event_type": {
				Type:      "string",
				MinLength: &minLen,
			},
			"data": {
				Type: "object",
			},
		},
		Required: []string{"room_id", "event_type"},
	}, func(ctx context.Context, params json.RawMessage) (interface{}, error) {
		var p struct {
			RoomID    string                 `json:"room_id"`
			EventType string                 `json:"event_type"`
			Data      map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		if p.Data == nil {
			p.Data = map[string]interface{}{}
		}

		cmdID := generateCommandID()
		payload, _ := json.Marshal(map[string]interface{}{
			"event_type": p.EventType,
			"data":       normalizeEventData(p.Data),
		})
		cmd := types.CommandEnvelope{
			CommandID:      cmdID,
			IdempotencyKey: cmdID,
			RoomID:         p.RoomID,
			Type:           "write_event",
			ActorUserID:    "autodm",
			Payload:        payload,
		}
		if err := a.dispatchCommand(cmd); err != nil {
			return nil, err
		}
		return map[string]string{"status": "written", "event_type": p.EventType}, nil
	})

	a.mu.Lock()
	a.mcpRegistry = registry
	a.mu.Unlock()
}

func normalizeEventData(data map[string]interface{}) map[string]string {
	normalized := make(map[string]string, len(data))
	for k, v := range data {
		switch vv := v.(type) {
		case string:
			normalized[k] = vv
		default:
			b, err := json.Marshal(v)
			if err != nil {
				continue
			}
			normalized[k] = string(b)
		}
	}
	return normalized
}

func (a *AutoDM) updateGameStateFromEngineState(raw interface{}) {
	state, ok := raw.(engine.State)
	if !ok {
		return
	}

	gs := &GameState{
		RoomID:      state.RoomID,
		Phase:       string(state.Phase),
		DayNumber:   state.DayCount,
		Edition:     state.Edition,
		IsStarted:   state.Phase != engine.PhaseLobby,
		IsFinished:  state.Phase == engine.PhaseEnded,
		Players:     make([]Player, 0, len(state.Players)),
		Nominations: make([]Nomination, 0, len(state.NominationQueue)+1),
	}

	for _, p := range state.Players {
		gs.Players = append(gs.Players, Player{
			ID:        p.UserID,
			Name:      p.Name,
			Role:      p.Role,
			IsAlive:   p.Alive,
			HasVoted:  false,
			Seat:      p.SeatNumber,
			Reminders: p.Reminders,
		})
	}

	for _, n := range state.NominationQueue {
		gs.Nominations = append(gs.Nominations, Nomination{
			Nominator: n.Nominator,
			Nominee:   n.Nominee,
			Votes:     n.VotesFor,
			Threshold: n.Threshold,
		})
	}
	if state.Nomination != nil && !state.Nomination.Resolved {
		gs.Nominations = append(gs.Nominations, Nomination{
			Nominator: state.Nomination.Nominator,
			Nominee:   state.Nomination.Nominee,
			Votes:     state.Nomination.VotesFor,
			Threshold: state.Nomination.Threshold,
		})
	}

	a.UpdateGameState(gs)
}
