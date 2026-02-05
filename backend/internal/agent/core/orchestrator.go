// Package core provides the main Orchestrator that coordinates sub-agents.
package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/memory"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/subagent"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/tools"
)

// Orchestrator coordinates sub-agents and manages the agent loop.
type Orchestrator struct {
	mu sync.RWMutex

	router *llm.Router
	memory *memory.Manager
	tools  *tools.Registry
	logger *slog.Logger

	moderator     *subagent.Moderator
	narrator      *subagent.Narrator
	rules         *subagent.Rules
	summarizer    *subagent.Summarizer
	playerModeler *subagent.PlayerModeler

	roomID    string
	gameState *GameState
	isActive  bool
}

// GameState represents the current game state.
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

// Player represents a player in the game.
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

// Config for the orchestrator.
type Config struct {
	RoomID       string
	LLMConfig    llm.RoutingConfig
	MemoryConfig memory.Config
	Logger       *slog.Logger
}

// New creates a new Orchestrator.
func New(cfg Config) *Orchestrator {
	router := llm.NewRouterFromConfig(cfg.LLMConfig)
	memMgr := memory.NewManager(cfg.MemoryConfig)

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Orchestrator{
		router:        router,
		memory:        memMgr,
		tools:         tools.NewRegistry(),
		logger:        logger,
		roomID:        cfg.RoomID,
		gameState:     &GameState{RoomID: cfg.RoomID, Phase: "setup"},
		moderator:     subagent.NewModerator(router),
		narrator:      subagent.NewNarrator(router),
		rules:         subagent.NewRules(router),
		summarizer:    subagent.NewSummarizer(router),
		playerModeler: subagent.NewPlayerModeler(router),
	}
}

// SetCommander sets the game commander for tool execution.
func (o *Orchestrator) SetCommander(commander tools.GameCommander) {
	tools.RegisterGameTools(o.tools, commander, o.roomID)
}

// SetRulesProvider sets the rules provider for info tools.
func (o *Orchestrator) SetRulesProvider(rules tools.RulesProvider) {
	tools.RegisterInfoTools(o.tools, rules)
}

// Start activates the orchestrator.
func (o *Orchestrator) Start() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.isActive = true
	o.logger.Info("Orchestrator started", "room", o.roomID)
}

// Stop deactivates the orchestrator.
func (o *Orchestrator) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.isActive = false
	o.logger.Info("Orchestrator stopped", "room", o.roomID)
}

// IsActive returns whether the orchestrator is running.
func (o *Orchestrator) IsActive() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.isActive
}

// UpdateGameState updates the game state.
func (o *Orchestrator) UpdateGameState(state *GameState) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.gameState = state
}

// Event represents a game event.
type Event struct {
	Type        string
	Description string
	PlayerID    string
	Data        map[string]interface{}
}

// Response is the orchestrator's response.
type Response struct {
	Message     string
	Actions     []Action
	ShouldSpeak bool
}

// Action is an action to perform.
type Action struct {
	Type   string
	Target string
	Data   map[string]interface{}
}

// ProcessEvent handles a game event.
func (o *Orchestrator) ProcessEvent(ctx context.Context, event Event) (*Response, error) {
	o.mu.RLock()
	if !o.isActive {
		o.mu.RUnlock()
		return nil, fmt.Errorf("orchestrator not active")
	}
	roomID := o.roomID
	phase := o.gameState.Phase
	dayNumber := o.gameState.DayNumber
	o.mu.RUnlock()

	o.memory.AddEvent(ctx, roomID, phase, dayNumber, event.Description)
	o.logger.Debug("Processing event", "type", event.Type, "description", event.Description)

	return o.routeEvent(ctx, event)
}

func (o *Orchestrator) routeEvent(ctx context.Context, event Event) (*Response, error) {
	gsView := o.toGameStateView()

	switch event.Type {
	case "phase_change":
		return o.handlePhaseChange(ctx, gsView, event)
	case "nomination":
		return o.handleNomination(ctx, gsView, event)
	case "death":
		return o.handleDeath(ctx, gsView, event)
	case "question":
		return o.handleQuestion(ctx, gsView, event)
	default:
		return o.handleGeneral(ctx, gsView, event)
	}
}

func (o *Orchestrator) handlePhaseChange(ctx context.Context, gs subagent.GameStateView, event Event) (*Response, error) {
	newPhase, _ := event.Data["new_phase"].(string)
	oldPhase, _ := event.Data["old_phase"].(string)

	narration, err := o.narrator.NarratePhaseChange(ctx, gs, oldPhase, newPhase)
	if err != nil {
		o.logger.Error("Failed to generate narration", "error", err)
		narration = fmt.Sprintf("The %s phase begins.", newPhase)
	}

	return &Response{Message: narration, ShouldSpeak: true}, nil
}

func (o *Orchestrator) handleNomination(ctx context.Context, gs subagent.GameStateView, event Event) (*Response, error) {
	nominatorID, _ := event.Data["nominator"].(string)
	nomineeID, _ := event.Data["nominee"].(string)

	valid, reason, err := o.moderator.ValidateNomination(ctx, gs, nominatorID, nomineeID)
	if err != nil {
		return nil, err
	}

	if !valid {
		return &Response{Message: fmt.Sprintf("Nomination not valid: %s", reason), ShouldSpeak: true}, nil
	}

	return &Response{Message: "Nomination accepted. " + reason, ShouldSpeak: true}, nil
}

func (o *Orchestrator) handleDeath(ctx context.Context, gs subagent.GameStateView, event Event) (*Response, error) {
	playerName, _ := event.Data["player_name"].(string)
	cause, _ := event.Data["cause"].(string)

	narration, err := o.narrator.NarrateDeath(ctx, gs, playerName, cause)
	if err != nil {
		narration = fmt.Sprintf("%s has died.", playerName)
	}

	return &Response{Message: narration, ShouldSpeak: true}, nil
}

func (o *Orchestrator) handleQuestion(ctx context.Context, gs subagent.GameStateView, event Event) (*Response, error) {
	question := event.Description

	if isRulesQuestion(question) {
		content, err := o.rules.Process(ctx, gs, question)
		if err != nil {
			return nil, err
		}
		return &Response{Message: content, ShouldSpeak: false}, nil
	}

	content, err := o.moderator.Process(ctx, gs, question)
	if err != nil {
		return nil, err
	}

	return &Response{Message: content, ShouldSpeak: true}, nil
}

func (o *Orchestrator) handleGeneral(ctx context.Context, gs subagent.GameStateView, event Event) (*Response, error) {
	content, err := o.moderator.Process(ctx, gs, event.Description)
	if err != nil {
		return nil, err
	}

	return &Response{Message: content, ShouldSpeak: true}, nil
}

func (o *Orchestrator) toGameStateView() subagent.GameStateView {
	o.mu.RLock()
	defer o.mu.RUnlock()

	players := make([]subagent.PlayerView, len(o.gameState.Players))
	for i, p := range o.gameState.Players {
		players[i] = subagent.PlayerView{
			ID:       p.ID,
			Name:     p.Name,
			Role:     p.Role,
			IsAlive:  p.IsAlive,
			HasVoted: p.HasVoted,
		}
	}

	nominations := make([]subagent.NominationView, len(o.gameState.Nominations))
	for i, n := range o.gameState.Nominations {
		nominations[i] = subagent.NominationView{
			Nominator: n.Nominator,
			Nominee:   n.Nominee,
			Votes:     n.Votes,
			Threshold: n.Threshold,
		}
	}

	return subagent.GameStateView{
		RoomID:      o.gameState.RoomID,
		Phase:       o.gameState.Phase,
		DayNumber:   o.gameState.DayNumber,
		Players:     players,
		Nominations: nominations,
		Edition:     o.gameState.Edition,
		Script:      o.gameState.Script,
	}
}

func isRulesQuestion(q string) bool {
	q = strings.ToLower(q)
	keywords := []string{"rule", "ability", "power", "when does", "how does", "can i"}
	for _, kw := range keywords {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

// GetSummary returns a summary of the current game.
func (o *Orchestrator) GetSummary(ctx context.Context, forDM bool) (string, error) {
	gsView := o.toGameStateView()
	return o.summarizer.SummarizeGameState(ctx, gsView, forDM)
}

// AnalyzePlayers returns player analysis.
func (o *Orchestrator) AnalyzePlayers(ctx context.Context) (string, error) {
	gsView := o.toGameStateView()
	return o.playerModeler.IdentifySuspects(ctx, gsView)
}
