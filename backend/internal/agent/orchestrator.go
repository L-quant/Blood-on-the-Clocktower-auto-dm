package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Orchestrator coordinates the AutoDM agent system with the control loop:
// Sense -> BuildContext -> Plan -> ExecuteActions -> Observe -> Reflect -> PersistMemory
type Orchestrator struct {
	roomID        string
	logger        *zap.Logger
	toolRegistry  *ToolRegistry
	modelRouter   *ModelRouter
	memoryManager *MemoryManager

	// Sub-agents
	moderator     SubAgent
	rules         SubAgent
	narrator      SubAgent
	summarizer    SubAgent
	playerModeler SubAgent

	// State
	mu         sync.RWMutex
	active     bool
	lastRunID  string
	lastRunAt  time.Time
	runCount   int64
	errorCount int64
	startedAt  time.Time
	stopCh     chan struct{}

	// Storage
	runStore AgentRunStore

	// Configuration
	config OrchestratorConfig
}

// OrchestratorConfig holds configuration for the orchestrator
type OrchestratorConfig struct {
	MaxActionsPerRun     int           `json:"max_actions_per_run"`
	RunInterval          time.Duration `json:"run_interval"`
	ActionTimeout        time.Duration `json:"action_timeout"`
	MaxRetriesPerAction  int           `json:"max_retries_per_action"`
	ShortTermMemorySize  int           `json:"short_term_memory_size"`
	EnableReflection     bool          `json:"enable_reflection"`
	EnablePlayerModeling bool          `json:"enable_player_modeling"`
}

// DefaultOrchestratorConfig returns sensible defaults
func DefaultOrchestratorConfig() OrchestratorConfig {
	return OrchestratorConfig{
		MaxActionsPerRun:     10,
		RunInterval:          2 * time.Second,
		ActionTimeout:        30 * time.Second,
		MaxRetriesPerAction:  3,
		ShortTermMemorySize:  50,
		EnableReflection:     true,
		EnablePlayerModeling: true,
	}
}

// AgentRunStore interface for persisting agent runs
type AgentRunStore interface {
	SaveRun(ctx context.Context, run AgentRun) error
	GetRun(ctx context.Context, runID string) (*AgentRun, error)
	ListRuns(ctx context.Context, roomID string, limit int) ([]AgentRun, error)
	SaveToolCall(ctx context.Context, call ToolCallAudit) error
}

// NewOrchestrator creates a new AutoDM orchestrator
func NewOrchestrator(
	roomID string,
	logger *zap.Logger,
	toolRegistry *ToolRegistry,
	modelRouter *ModelRouter,
	memoryManager *MemoryManager,
	runStore AgentRunStore,
	config OrchestratorConfig,
) *Orchestrator {
	o := &Orchestrator{
		roomID:        roomID,
		logger:        logger.With(zap.String("room_id", roomID)),
		toolRegistry:  toolRegistry,
		modelRouter:   modelRouter,
		memoryManager: memoryManager,
		runStore:      runStore,
		config:        config,
		stopCh:        make(chan struct{}),
	}

	// Initialize sub-agents
	o.moderator = NewModeratorAgent(modelRouter, toolRegistry)
	o.rules = NewRulesAgent(modelRouter, memoryManager)
	o.narrator = NewNarratorAgent(modelRouter)
	o.summarizer = NewSummarizerAgent(modelRouter, memoryManager)
	o.playerModeler = NewPlayerModelAgent(modelRouter, memoryManager)

	return o
}

// Start begins the AutoDM control loop
func (o *Orchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	if o.active {
		o.mu.Unlock()
		return fmt.Errorf("orchestrator already active")
	}
	o.active = true
	o.startedAt = time.Now()
	o.stopCh = make(chan struct{})
	o.mu.Unlock()

	o.logger.Info("starting AutoDM orchestrator")

	go o.runLoop(ctx)
	return nil
}

// Stop halts the AutoDM control loop
func (o *Orchestrator) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.active {
		return fmt.Errorf("orchestrator not active")
	}

	o.logger.Info("stopping AutoDM orchestrator")
	close(o.stopCh)
	o.active = false
	return nil
}

// Status returns the current status of the orchestrator
func (o *Orchestrator) Status() AutoDMStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return AutoDMStatus{
		RoomID:     o.roomID,
		Active:     o.active,
		LastRunID:  o.lastRunID,
		LastRunAt:  o.lastRunAt,
		RunCount:   o.runCount,
		ErrorCount: o.errorCount,
		StartedAt:  o.startedAt,
	}
}

// runLoop is the main control loop
func (o *Orchestrator) runLoop(ctx context.Context) {
	ticker := time.NewTicker(o.config.RunInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			o.logger.Info("context cancelled, stopping loop")
			return
		case <-o.stopCh:
			o.logger.Info("stop signal received")
			return
		case <-ticker.C:
			if err := o.executeRun(ctx); err != nil {
				o.logger.Error("run failed", zap.Error(err))
				o.mu.Lock()
				o.errorCount++
				o.mu.Unlock()
			}
		}
	}
}

// executeRun executes a single iteration of the control loop
func (o *Orchestrator) executeRun(ctx context.Context) error {
	runID := uuid.NewString()
	startTime := time.Now()

	o.logger.Debug("starting run", zap.String("run_id", runID))

	run := AgentRun{
		ID:        runID,
		RoomID:    o.roomID,
		AgentName: "orchestrator",
		Status:    "running",
		CreatedAt: startTime,
	}

	// Step 1: Sense - Gather current state and events
	agentCtx, err := o.sense(ctx, runID)
	if err != nil {
		run.Status = "error"
		run.ErrorText = fmt.Sprintf("sense failed: %v", err)
		run.LatencyMs = time.Since(startTime).Milliseconds()
		o.saveRun(ctx, run)
		return fmt.Errorf("sense: %w", err)
	}

	run.SeqFrom = agentCtx.MemoryContext.ShortTerm[0].Seq
	if len(agentCtx.MemoryContext.ShortTerm) > 0 {
		run.SeqTo = agentCtx.MemoryContext.ShortTerm[len(agentCtx.MemoryContext.ShortTerm)-1].Seq
	}

	// Generate input digest
	inputBytes, _ := json.Marshal(agentCtx)
	run.InputDigest = hashDigest(inputBytes)

	// Step 2: Build Context - Augment with memory and RAG
	memCtx, err := o.buildContext(ctx, agentCtx)
	if err != nil {
		o.logger.Warn("build context failed", zap.Error(err))
		// Continue with partial context
	}
	agentCtx.MemoryContext = memCtx

	// Step 3: Plan - Generate action plan
	plan, err := o.plan(ctx, agentCtx)
	if err != nil {
		run.Status = "error"
		run.ErrorText = fmt.Sprintf("plan failed: %v", err)
		run.LatencyMs = time.Since(startTime).Milliseconds()
		o.saveRun(ctx, run)
		return fmt.Errorf("plan: %w", err)
	}

	planJSON, _ := json.Marshal(plan)
	run.PlanJSON = planJSON

	// Step 4: Execute Actions
	results, toolCalls := o.executeActions(ctx, runID, plan.Actions)
	run.ToolCalls = toolCalls

	// Step 5: Observe - Gather results
	observation := o.observe(ctx, agentCtx, results)

	// Step 6: Reflect - Learn from the execution
	if o.config.EnableReflection {
		reflection := o.reflect(ctx, agentCtx, plan, observation)
		if reflection != nil {
			o.logger.Debug("reflection", zap.String("summary", reflection.Summary))
		}
	}

	// Step 7: Persist Memory
	if err := o.persistMemory(ctx, agentCtx, plan, observation); err != nil {
		o.logger.Warn("persist memory failed", zap.Error(err))
	}

	// Complete run record
	run.Status = "completed"
	run.LatencyMs = time.Since(startTime).Milliseconds()
	outputBytes, _ := json.Marshal(observation)
	run.OutputDigest = hashDigest(outputBytes)

	o.saveRun(ctx, run)

	// Update orchestrator state
	o.mu.Lock()
	o.lastRunID = runID
	o.lastRunAt = time.Now()
	o.runCount++
	o.mu.Unlock()

	o.logger.Debug("run completed",
		zap.String("run_id", runID),
		zap.Int("actions", len(plan.Actions)),
		zap.Int64("latency_ms", run.LatencyMs),
	)

	return nil
}

// sense gathers the current state and recent events
func (o *Orchestrator) sense(ctx context.Context, runID string) (*AgentContext, error) {
	// Get room state via tool
	stateResult, err := o.toolRegistry.Execute(ctx, "get_room_state", GetRoomStateArgs{
		RoomID: o.roomID,
	})
	if err != nil {
		return nil, fmt.Errorf("get room state: %w", err)
	}

	var roomState RoomState
	if err := json.Unmarshal(stateResult.Output, &roomState); err != nil {
		return nil, fmt.Errorf("unmarshal room state: %w", err)
	}

	// Get recent events
	eventsResult, err := o.toolRegistry.Execute(ctx, "get_recent_events", GetRecentEventsArgs{
		RoomID:   o.roomID,
		SinceSeq: roomState.LastSeq - int64(o.config.ShortTermMemorySize),
		Limit:    o.config.ShortTermMemorySize,
	})
	if err != nil {
		return nil, fmt.Errorf("get recent events: %w", err)
	}

	var events []Event
	if err := json.Unmarshal(eventsResult.Output, &events); err != nil {
		return nil, fmt.Errorf("unmarshal events: %w", err)
	}

	// Build pending inputs from timers
	var pendingInputs []PendingInput
	for userID, player := range roomState.Players {
		if player.Alive && !player.IsDM {
			pendingInputs = append(pendingInputs, PendingInput{
				UserID:     userID,
				ActionType: "awaiting_action",
			})
		}
	}

	agentCtx := &AgentContext{
		RoomID:        o.roomID,
		RunID:         runID,
		Phase:         roomState.Phase,
		RecentEvents:  events,
		PendingInputs: pendingInputs,
		Timers:        roomState.Timers,
		StartTime:     time.Now(),
		MemoryContext: &MemoryContext{
			ShortTerm: events,
		},
	}

	return agentCtx, nil
}

// buildContext augments the context with memory and RAG
func (o *Orchestrator) buildContext(ctx context.Context, agentCtx *AgentContext) (*MemoryContext, error) {
	memCtx := &MemoryContext{
		ShortTerm:    agentCtx.MemoryContext.ShortTerm,
		PlayerModels: make(map[string]PlayerModel),
	}

	// Retrieve relevant long-term memories
	longTerm, err := o.memoryManager.RetrieveRelevant(ctx, o.roomID, buildQueryFromContext(agentCtx), 5)
	if err != nil {
		o.logger.Warn("failed to retrieve long-term memory", zap.Error(err))
	} else {
		memCtx.LongTerm = longTerm
	}

	// Get game summary
	summary, err := o.memoryManager.GetGameSummary(ctx, o.roomID)
	if err != nil {
		o.logger.Warn("failed to get game summary", zap.Error(err))
	} else {
		memCtx.GameSummary = summary
	}

	// Get player models if enabled
	if o.config.EnablePlayerModeling {
		models, err := o.memoryManager.GetPlayerModels(ctx, o.roomID)
		if err != nil {
			o.logger.Warn("failed to get player models", zap.Error(err))
		} else {
			memCtx.PlayerModels = models
		}
	}

	return memCtx, nil
}

// plan generates an action plan using the planner model
func (o *Orchestrator) plan(ctx context.Context, agentCtx *AgentContext) (*Plan, error) {
	// Determine which sub-agents should contribute
	contributions := make(map[string]*AgentOutput)

	// Always consult the moderator for phase control
	modOutput, err := o.moderator.Execute(ctx, agentCtx)
	if err != nil {
		o.logger.Warn("moderator failed", zap.Error(err))
	} else {
		contributions["moderator"] = modOutput
	}

	// Check if we need rules clarification
	if needsRulesLookup(agentCtx) {
		rulesOutput, err := o.rules.Execute(ctx, agentCtx)
		if err != nil {
			o.logger.Warn("rules agent failed", zap.Error(err))
		} else {
			contributions["rules"] = rulesOutput
		}
	}

	// Generate narration for key moments
	if needsNarration(agentCtx) {
		narrOutput, err := o.narrator.Execute(ctx, agentCtx)
		if err != nil {
			o.logger.Warn("narrator failed", zap.Error(err))
		} else {
			contributions["narrator"] = narrOutput
		}
	}

	// Merge contributions into a unified plan
	plan := o.mergeContributions(agentCtx, contributions)

	return plan, nil
}

// mergeContributions combines outputs from multiple agents
func (o *Orchestrator) mergeContributions(agentCtx *AgentContext, contributions map[string]*AgentOutput) *Plan {
	plan := &Plan{
		ID:        uuid.NewString(),
		RoomID:    o.roomID,
		Actions:   []Action{},
		CreatedAt: time.Now(),
	}

	// Priority: Moderator for control flow > Rules for legality > Narrator for wording
	priorityOrder := []string{"moderator", "rules", "narrator", "summarizer", "player_modeler"}

	for _, agentName := range priorityOrder {
		if output, ok := contributions[agentName]; ok && output != nil {
			for _, action := range output.Actions {
				plan.Actions = append(plan.Actions, action)
			}
			if plan.Reasoning == "" && output.Message != "" {
				plan.Reasoning = output.Message
			}
		}
	}

	// Limit actions per run
	if len(plan.Actions) > o.config.MaxActionsPerRun {
		plan.Actions = plan.Actions[:o.config.MaxActionsPerRun]
	}

	return plan
}

// executeActions executes the planned actions via tools
func (o *Orchestrator) executeActions(ctx context.Context, runID string, actions []Action) ([]ActionResult, []ToolCallAudit) {
	results := make([]ActionResult, 0, len(actions))
	toolCalls := make([]ToolCallAudit, 0, len(actions))

	for _, action := range actions {
		actionCtx, cancel := context.WithTimeout(ctx, o.config.ActionTimeout)
		startTime := time.Now()

		var result ActionResult
		var toolCall ToolCallAudit

		toolCall.ID = uuid.NewString()
		toolCall.RunID = runID
		toolCall.ToolName = string(action.Type)
		toolCall.Args = action.Args
		toolCall.CreatedAt = startTime

		// Execute with retries
		var execResult *ToolResult
		var err error
		for attempt := 0; attempt <= o.config.MaxRetriesPerAction; attempt++ {
			execResult, err = o.toolRegistry.ExecuteRaw(actionCtx, string(action.Type), action.Args)
			if err == nil {
				break
			}
			if attempt < o.config.MaxRetriesPerAction {
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			}
		}

		duration := time.Since(startTime)
		toolCall.DurationMs = duration.Milliseconds()

		if err != nil {
			result = ActionResult{
				ActionID:  action.ID,
				Success:   false,
				Error:     err.Error(),
				Duration:  duration,
				Timestamp: time.Now(),
			}
			toolCall.Error = err.Error()
		} else {
			result = ActionResult{
				ActionID:  action.ID,
				Success:   true,
				Output:    execResult.Output,
				Duration:  duration,
				Timestamp: time.Now(),
			}
			toolCall.Result = execResult.Output
		}

		results = append(results, result)
		toolCalls = append(toolCalls, toolCall)

		// Save tool call audit
		if o.runStore != nil {
			if err := o.runStore.SaveToolCall(ctx, toolCall); err != nil {
				o.logger.Warn("failed to save tool call", zap.Error(err))
			}
		}

		cancel()
	}

	return results, toolCalls
}

// observe gathers and interprets the results of execution
func (o *Orchestrator) observe(ctx context.Context, agentCtx *AgentContext, results []ActionResult) *Observation {
	obs := &Observation{
		RoomID:    o.roomID,
		Results:   results,
		Timestamp: time.Now(),
	}

	// Check for state changes
	for _, r := range results {
		if r.Success {
			obs.StateChanged = true
			break
		}
	}

	// Get new events if state changed
	if obs.StateChanged {
		eventsResult, err := o.toolRegistry.Execute(ctx, "get_recent_events", GetRecentEventsArgs{
			RoomID:   o.roomID,
			SinceSeq: agentCtx.MemoryContext.ShortTerm[len(agentCtx.MemoryContext.ShortTerm)-1].Seq,
			Limit:    20,
		})
		if err == nil {
			var events []Event
			if json.Unmarshal(eventsResult.Output, &events) == nil {
				obs.NewEvents = events
			}
		}
	}

	return obs
}

// reflect generates insights from the execution
func (o *Orchestrator) reflect(ctx context.Context, agentCtx *AgentContext, plan *Plan, obs *Observation) *Reflection {
	// Count successes and failures
	successCount := 0
	failureCount := 0
	for _, r := range obs.Results {
		if r.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	reflection := &Reflection{
		RoomID:    o.roomID,
		Summary:   fmt.Sprintf("Executed %d actions: %d succeeded, %d failed", len(obs.Results), successCount, failureCount),
		CreatedAt: time.Now(),
	}

	if failureCount > 0 {
		reflection.Lessons = append(reflection.Lessons, "Some actions failed - consider adjusting strategy")
	}

	if obs.StateChanged {
		reflection.Lessons = append(reflection.Lessons, "State changed successfully")
	}

	return reflection
}

// persistMemory saves relevant information to memory
func (o *Orchestrator) persistMemory(ctx context.Context, agentCtx *AgentContext, plan *Plan, obs *Observation) error {
	// Store significant events
	for _, event := range obs.NewEvents {
		entry := MemoryEntry{
			ID:        uuid.NewString(),
			Type:      "event",
			Content:   fmt.Sprintf("[%s] %s: %s", event.EventType, event.ActorID, string(event.Payload)),
			CreatedAt: time.Now(),
		}
		if err := o.memoryManager.Store(ctx, o.roomID, entry); err != nil {
			o.logger.Warn("failed to store memory", zap.Error(err))
		}
	}

	return nil
}

// saveRun persists the agent run record
func (o *Orchestrator) saveRun(ctx context.Context, run AgentRun) {
	if o.runStore != nil {
		if err := o.runStore.SaveRun(ctx, run); err != nil {
			o.logger.Error("failed to save run", zap.Error(err))
		}
	}
}

// Helper functions

func hashDigest(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

func buildQueryFromContext(agentCtx *AgentContext) string {
	// Build a query string from recent events for RAG retrieval
	var query string
	if len(agentCtx.RecentEvents) > 0 {
		lastEvent := agentCtx.RecentEvents[len(agentCtx.RecentEvents)-1]
		query = fmt.Sprintf("phase:%s event:%s", agentCtx.Phase, lastEvent.EventType)
	} else {
		query = fmt.Sprintf("phase:%s", agentCtx.Phase)
	}
	return query
}

func needsRulesLookup(agentCtx *AgentContext) bool {
	// Check if recent events include disputes or ability usage
	for _, e := range agentCtx.RecentEvents {
		switch e.EventType {
		case "ability.used", "dispute", "rule_question":
			return true
		}
	}
	return false
}

func needsNarration(agentCtx *AgentContext) bool {
	// Check if we're at a key moment that needs narration
	for _, e := range agentCtx.RecentEvents {
		switch e.EventType {
		case "game.started", "phase.day", "phase.night", "execution.resolved", "game.ended":
			return true
		}
	}
	return false
}
