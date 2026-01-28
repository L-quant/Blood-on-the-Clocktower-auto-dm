package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ModeratorAgent drives game phases, prompts players, and handles timeouts
type ModeratorAgent struct {
	router *ModelRouter
	tools  *ToolRegistry
}

// NewModeratorAgent creates a new moderator agent
func NewModeratorAgent(router *ModelRouter, tools *ToolRegistry) *ModeratorAgent {
	return &ModeratorAgent{
		router: router,
		tools:  tools,
	}
}

func (a *ModeratorAgent) Name() string {
	return "moderator"
}

func (a *ModeratorAgent) Description() string {
	return "Drives game phases, prompts players for actions, and handles timeouts"
}

func (a *ModeratorAgent) Execute(ctx context.Context, agentCtx *AgentContext) (*AgentOutput, error) {
	output := &AgentOutput{
		AgentName: a.Name(),
		Actions:   []Action{},
	}

	// Analyze current state and determine needed actions
	switch agentCtx.Phase {
	case PhaseLobby:
		// Check if we have enough players to start
		output.Message = "Waiting for players to join..."

	case PhaseDay:
		actions := a.handleDayPhase(ctx, agentCtx)
		output.Actions = append(output.Actions, actions...)
		output.Message = "Managing day phase activities"

	case PhaseNight:
		actions := a.handleNightPhase(ctx, agentCtx)
		output.Actions = append(output.Actions, actions...)
		output.Message = "Managing night phase activities"

	case PhaseEnded:
		output.Message = "Game has ended"
	}

	return output, nil
}

func (a *ModeratorAgent) handleDayPhase(ctx context.Context, agentCtx *AgentContext) []Action {
	var actions []Action

	// Check for pending nominations
	hasActiveNomination := false
	for _, event := range agentCtx.RecentEvents {
		if event.EventType == "nomination.created" {
			hasActiveNomination = true
			break
		}
		if event.EventType == "execution.resolved" {
			hasActiveNomination = false
		}
	}

	// Check for expired timers
	now := time.Now()
	for timerType, deadline := range agentCtx.Timers {
		if deadline.Before(now) {
			// Timer expired - handle timeout
			switch timerType {
			case "vote":
				actions = append(actions, Action{
					ID:   uuid.NewString(),
					Type: ActionCloseVote,
					Args: mustMarshalArgs(map[string]string{"room_id": agentCtx.RoomID}),
				})
			case "day":
				actions = append(actions, Action{
					ID:   uuid.NewString(),
					Type: ActionAdvancePhase,
					Args: mustMarshalArgs(AdvancePhaseArgs{
						RoomID:    agentCtx.RoomID,
						NextPhase: PhaseNight,
						Reason:    "Day time has expired",
					}),
				})
			}
		}
	}

	// If no active nomination and day has been going for a while, prompt players
	if !hasActiveNomination && len(agentCtx.RecentEvents) > 0 {
		lastEventTime := agentCtx.RecentEvents[len(agentCtx.RecentEvents)-1].Timestamp
		if time.Since(lastEventTime) > 30*time.Second {
			actions = append(actions, Action{
				ID:   uuid.NewString(),
				Type: ActionSendPublicMessage,
				Args: mustMarshalArgs(SendMessageArgs{
					RoomID: agentCtx.RoomID,
					Text:   "☀️ The sun is high. Does anyone wish to make a nomination?",
					Metadata: map[string]string{
						"type": "prompt",
					},
				}),
			})
		}
	}

	return actions
}

func (a *ModeratorAgent) handleNightPhase(ctx context.Context, agentCtx *AgentContext) []Action {
	var actions []Action

	// Check for pending abilities
	pendingAbilities := a.getPendingNightAbilities(agentCtx)

	if len(pendingAbilities) > 0 {
		// Request actions from players with night abilities
		for _, pending := range pendingAbilities {
			actions = append(actions, Action{
				ID:   uuid.NewString(),
				Type: ActionSendWhisper,
				Args: mustMarshalArgs(SendWhisperArgs{
					RoomID:   agentCtx.RoomID,
					ToUserID: pending.UserID,
					Text:     fmt.Sprintf("🌙 It is time to use your %s ability. Please make your choice.", pending.ActionType),
				}),
			})
			actions = append(actions, Action{
				ID:   uuid.NewString(),
				Type: ActionRequestPlayerAction,
				Args: mustMarshalArgs(RequestActionArgs{
					RoomID:     agentCtx.RoomID,
					UserID:     pending.UserID,
					ActionType: "ability",
					Deadline:   60 * time.Second,
					Prompt:     "Use your night ability",
				}),
			})
		}
	}

	// Check for expired timers
	now := time.Now()
	if deadline, ok := agentCtx.Timers["night"]; ok && deadline.Before(now) {
		// Night is over, transition to day
		actions = append(actions, Action{
			ID:   uuid.NewString(),
			Type: ActionAdvancePhase,
			Args: mustMarshalArgs(AdvancePhaseArgs{
				RoomID:    agentCtx.RoomID,
				NextPhase: PhaseDay,
				Reason:    "Night has passed",
			}),
		})
	}

	return actions
}

func (a *ModeratorAgent) getPendingNightAbilities(agentCtx *AgentContext) []PendingInput {
	// In a real implementation, this would check game state for:
	// - Which players have night abilities
	// - Which abilities haven't been used yet this night
	// - The order in which abilities should resolve
	return agentCtx.PendingInputs
}

// RulesAgent looks up rules and resolves disputes
type RulesAgent struct {
	router *ModelRouter
	memory *MemoryManager
}

// NewRulesAgent creates a new rules agent
func NewRulesAgent(router *ModelRouter, memory *MemoryManager) *RulesAgent {
	return &RulesAgent{
		router: router,
		memory: memory,
	}
}

func (a *RulesAgent) Name() string {
	return "rules"
}

func (a *RulesAgent) Description() string {
	return "Looks up game rules and resolves disputes with citations"
}

func (a *RulesAgent) Execute(ctx context.Context, agentCtx *AgentContext) (*AgentOutput, error) {
	output := &AgentOutput{
		AgentName:  a.Name(),
		Actions:    []Action{},
		Confidence: 0.8,
	}

	// Look for rule questions or disputes in recent events
	for _, event := range agentCtx.RecentEvents {
		if event.EventType == "rule_question" || event.EventType == "dispute" {
			var payload map[string]string
			json.Unmarshal(event.Payload, &payload)

			query := payload["question"]
			if query == "" {
				query = payload["message"]
			}

			// Search rules knowledge base
			results := a.memory.SearchRules(ctx, query, 3)

			if len(results) > 0 {
				// Build response with citations
				var response string
				var citations []string

				for i, result := range results {
					var meta map[string]interface{}
					json.Unmarshal(result.Metadata, &meta)
					source := "rules"
					if s, ok := meta["source"].(string); ok {
						source = s
					}
					citations = append(citations, fmt.Sprintf("[%d] %s", i+1, source))
				}

				// Generate response using LLM
				messages := []Message{
					{
						Role: "system",
						Content: `You are an expert Blood on the Clocktower rules judge.
Answer the question based on the provided rules context.
Be clear, concise, and cite your sources using [N] notation.
If the rules are unclear, explain the most common interpretation.`,
					},
					{
						Role: "user",
						Content: fmt.Sprintf("Rules Context:\n%s\n\nQuestion: %s",
							formatRulesContext(results), query),
					},
				}

				resp, err := a.router.Chat(ctx, "rules", messages, nil)

				if err == nil && len(resp.Choices) > 0 {
					response = resp.Choices[0].Message.Content
				} else {
					response = "I found the following relevant rules:\n"
					for i, result := range results {
						response += fmt.Sprintf("\n[%d] %s", i+1, result.Content)
					}
				}

				output.Actions = append(output.Actions, Action{
					ID:   uuid.NewString(),
					Type: ActionSendPublicMessage,
					Args: mustMarshalArgs(SendMessageArgs{
						RoomID: agentCtx.RoomID,
						Text:   fmt.Sprintf("📜 **Rules Clarification**\n\n%s\n\nSources: %s", response, formatCitations(citations)),
						Metadata: map[string]string{
							"type": "rules_response",
						},
					}),
				})

				output.Message = "Provided rules clarification"
			}
		}
	}

	return output, nil
}

// NarratorAgent generates engaging narration and announcements
type NarratorAgent struct {
	router *ModelRouter
}

// NewNarratorAgent creates a new narrator agent
func NewNarratorAgent(router *ModelRouter) *NarratorAgent {
	return &NarratorAgent{
		router: router,
	}
}

func (a *NarratorAgent) Name() string {
	return "narrator"
}

func (a *NarratorAgent) Description() string {
	return "Generates engaging narration and atmospheric announcements"
}

func (a *NarratorAgent) Execute(ctx context.Context, agentCtx *AgentContext) (*AgentOutput, error) {
	output := &AgentOutput{
		AgentName: a.Name(),
		Actions:   []Action{},
	}

	// Check for events that need narration
	for _, event := range agentCtx.RecentEvents {
		var narration string

		switch event.EventType {
		case "game.started":
			narration = a.generateGameStartNarration(ctx, agentCtx)
		case "phase.day":
			narration = a.generateDawnNarration(ctx, agentCtx)
		case "phase.night":
			narration = a.generateNightfallNarration(ctx, agentCtx)
		case "execution.resolved":
			narration = a.generateExecutionNarration(ctx, agentCtx, event)
		case "game.ended":
			narration = a.generateGameEndNarration(ctx, agentCtx, event)
		}

		if narration != "" {
			output.Actions = append(output.Actions, Action{
				ID:   uuid.NewString(),
				Type: ActionSendPublicMessage,
				Args: mustMarshalArgs(SendMessageArgs{
					RoomID: agentCtx.RoomID,
					Text:   narration,
					Metadata: map[string]string{
						"type": "narration",
					},
				}),
			})
		}
	}

	return output, nil
}

func (a *NarratorAgent) generateGameStartNarration(ctx context.Context, agentCtx *AgentContext) string {
	messages := []Message{
		{
			Role: "system",
			Content: `You are a dramatic storyteller for Blood on the Clocktower.
Generate an atmospheric, mysterious opening narration to start the game.
Keep it under 100 words. Use evocative language about the village, darkness, and hidden evil.`,
		},
		{
			Role: "user",
			Content: fmt.Sprintf("Generate opening narration for a game with %d players.",
				len(agentCtx.MemoryContext.ShortTerm)),
		},
	}

	resp, err := a.router.Chat(ctx, "narrator", messages, nil)

	if err != nil || len(resp.Choices) == 0 {
		return "🌑 **Night falls upon the village...**\n\nA sinister presence lurks among you. One of your number is not what they seem. Trust no one completely. The fate of the village rests in your hands. Let the game begin!"
	}

	return "🌑 " + resp.Choices[0].Message.Content
}

func (a *NarratorAgent) generateDawnNarration(ctx context.Context, agentCtx *AgentContext) string {
	messages := []Message{
		{
			Role: "system",
			Content: `You are a dramatic storyteller. Generate a brief (2-3 sentences) dawn announcement.
Reference any deaths that occurred, or if no deaths, note the village's relief.
Keep it atmospheric and suspenseful.`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Day begins with %d recent events. Generate dawn narration.", len(agentCtx.MemoryContext.ShortTerm)),
		},
	}

	resp, err := a.router.Chat(ctx, "narrator", messages, nil)

	if err != nil || len(resp.Choices) == 0 {
		return "☀️ **Dawn breaks over the village.**\n\nThe survivors emerge from their homes, wary eyes scanning for signs of evil. The debate begins anew."
	}

	return "☀️ " + resp.Choices[0].Message.Content
}

func (a *NarratorAgent) generateNightfallNarration(ctx context.Context, agentCtx *AgentContext) string {
	return "🌙 **Night descends upon the village.**\n\nClose your eyes and await your fate. Dark powers stir in the shadows, and the creatures of the night begin their work..."
}

func (a *NarratorAgent) generateExecutionNarration(ctx context.Context, agentCtx *AgentContext, event GameEvent) string {
	var payload map[string]string
	json.Unmarshal(event.Payload, &payload)

	nominee := payload["nominee"]
	result := payload["result"]

	if result == "executed" {
		return fmt.Sprintf("⚔️ **The village has spoken.**\n\nBy popular vote, %s has been executed. May their fate serve as a lesson to all.", nominee)
	}
	return fmt.Sprintf("🛡️ **The vote concludes.**\n\n%s has been spared. For now, they remain among the living.", nominee)
}

func (a *NarratorAgent) generateGameEndNarration(ctx context.Context, agentCtx *AgentContext, event GameEvent) string {
	var payload map[string]string
	json.Unmarshal(event.Payload, &payload)

	winner := payload["winner"]

	if winner == "good" {
		return "🎉 **Good triumphs over evil!**\n\nThe demon has been vanquished! The village is saved, and peace returns to these troubled lands. Congratulations to the forces of good!"
	}
	return "💀 **Evil prevails!**\n\nDarkness consumes the village. The demon's plan has succeeded, and the forces of evil claim their victory. Better luck next time, villagers..."
}

// SummarizerAgent creates periodic game summaries
type SummarizerAgent struct {
	router *ModelRouter
	memory *MemoryManager
}

// NewSummarizerAgent creates a new summarizer agent
func NewSummarizerAgent(router *ModelRouter, memory *MemoryManager) *SummarizerAgent {
	return &SummarizerAgent{
		router: router,
		memory: memory,
	}
}

func (a *SummarizerAgent) Name() string {
	return "summarizer"
}

func (a *SummarizerAgent) Description() string {
	return "Creates periodic game summaries and recaps"
}

func (a *SummarizerAgent) Execute(ctx context.Context, agentCtx *AgentContext) (*AgentOutput, error) {
	output := &AgentOutput{
		AgentName: a.Name(),
		Actions:   []Action{},
	}

	// Generate summary at end of each day
	shouldSummarize := false
	for _, event := range agentCtx.RecentEvents {
		if event.EventType == "phase.night" {
			shouldSummarize = true
			break
		}
	}

	if !shouldSummarize {
		return output, nil
	}

	// Build events summary
	eventsSummary := a.buildEventsSummary(agentCtx)

	messages := []Message{
		{
			Role: "system",
			Content: `You are a game summarizer for Blood on the Clocktower.
Create a brief, factual summary of the day's events.
Include: key discussions, nominations, votes, and executions.
Keep it under 150 words and use bullet points.`,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Summarize these events:\n%s", eventsSummary),
		},
	}

	resp, err := a.router.Chat(ctx, "summarizer", messages, nil)

	if err != nil || len(resp.Choices) == 0 {
		return output, nil
	}

	summary := resp.Choices[0].Message.Content

	// Save summary to memory
	a.memory.SaveGameSummary(ctx, agentCtx.RoomID, summary)

	// Optionally post public summary
	output.Actions = append(output.Actions, Action{
		ID:   uuid.NewString(),
		Type: ActionSendPublicMessage,
		Args: mustMarshalArgs(SendMessageArgs{
			RoomID: agentCtx.RoomID,
			Text:   fmt.Sprintf("📋 **Day Summary**\n\n%s", summary),
			Metadata: map[string]string{
				"type": "summary",
			},
		}),
	})

	output.Message = "Generated day summary"
	return output, nil
}

func (a *SummarizerAgent) buildEventsSummary(agentCtx *AgentContext) string {
	var events []string
	for _, event := range agentCtx.RecentEvents {
		var payload map[string]string
		json.Unmarshal(event.Payload, &payload)

		switch event.EventType {
		case "public.chat":
			events = append(events, fmt.Sprintf("%s said: %s", event.ActorID, payload["message"]))
		case "nomination.created":
			events = append(events, fmt.Sprintf("%s nominated %s", event.ActorID, payload["nominee"]))
		case "vote.cast":
			events = append(events, fmt.Sprintf("%s voted %s", event.ActorID, payload["vote"]))
		case "execution.resolved":
			events = append(events, fmt.Sprintf("Execution result: %s - %s", payload["nominee"], payload["result"]))
		}
	}

	if len(events) > 20 {
		events = events[len(events)-20:]
	}

	result := ""
	for _, e := range events {
		result += "- " + e + "\n"
	}
	return result
}

// PlayerModelAgent builds behavioral models of players
type PlayerModelAgent struct {
	router *ModelRouter
	memory *MemoryManager
}

// NewPlayerModelAgent creates a new player model agent
func NewPlayerModelAgent(router *ModelRouter, memory *MemoryManager) *PlayerModelAgent {
	return &PlayerModelAgent{
		router: router,
		memory: memory,
	}
}

func (a *PlayerModelAgent) Name() string {
	return "player_modeler"
}

func (a *PlayerModelAgent) Description() string {
	return "Builds behavioral profiles of players from their actions"
}

func (a *PlayerModelAgent) Execute(ctx context.Context, agentCtx *AgentContext) (*AgentOutput, error) {
	output := &AgentOutput{
		AgentName: a.Name(),
		Actions:   []Action{},
	}

	// Analyze player behavior from recent events
	playerStats := make(map[string]*playerBehavior)

	for _, event := range agentCtx.RecentEvents {
		actor := event.ActorID
		if actor == "" {
			continue
		}

		if playerStats[actor] == nil {
			playerStats[actor] = &playerBehavior{}
		}

		stats := playerStats[actor]
		stats.totalActions++

		switch event.EventType {
		case "public.chat":
			stats.messages++
		case "nomination.created":
			stats.nominations++
		case "vote.cast":
			stats.votes++
			var payload map[string]string
			json.Unmarshal(event.Payload, &payload)
			if payload["vote"] == "yes" {
				stats.yesVotes++
			}
		}
	}

	// Update player models
	for userID, stats := range playerStats {
		model := PlayerModel{
			UserID:            userID,
			LastUpdated:       time.Now(),
			ParticipationRate: float64(stats.totalActions) / float64(len(agentCtx.RecentEvents)),
		}

		// Determine playstyle
		if stats.nominations > 2 {
			model.Playstyle = "aggressive"
		} else if stats.messages > 10 {
			model.Playstyle = "talkative"
		} else if stats.totalActions < 3 {
			model.Playstyle = "quiet"
		} else {
			model.Playstyle = "balanced"
		}

		// Calculate voting patterns
		if stats.votes > 0 {
			yesRatio := float64(stats.yesVotes) / float64(stats.votes)
			if yesRatio > 0.7 {
				model.VotingPatterns = append(model.VotingPatterns, "tends_to_vote_yes")
			} else if yesRatio < 0.3 {
				model.VotingPatterns = append(model.VotingPatterns, "tends_to_vote_no")
			}
		}

		a.memory.SavePlayerModel(ctx, agentCtx.RoomID, model)
	}

	output.Message = fmt.Sprintf("Updated models for %d players", len(playerStats))
	return output, nil
}

type playerBehavior struct {
	totalActions int
	messages     int
	nominations  int
	votes        int
	yesVotes     int
}

// Helper functions

func mustMarshalArgs(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func formatRulesContext(results []MemoryEntry) string {
	var parts []string
	for i, result := range results {
		parts = append(parts, fmt.Sprintf("[%d] %s", i+1, result.Content))
	}
	return fmt.Sprintf("%s", parts)
}

func formatCitations(citations []string) string {
	if len(citations) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", citations)
}
