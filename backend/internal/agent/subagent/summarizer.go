// Package subagent provides the Summarizer sub-agent.
package subagent

import (
	"context"
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
)

const summarizerPrompt = `You are the Summarizer for Blood on the Clocktower.
Create clear, concise summaries of game events and status.`

// Summarizer creates summaries of game state and events.
type Summarizer struct {
	router *llm.Router
}

// NewSummarizer creates a new Summarizer agent.
func NewSummarizer(router *llm.Router) *Summarizer {
	return &Summarizer{router: router}
}

// SummarizeGameState creates a summary of current game state.
func (s *Summarizer) SummarizeGameState(ctx context.Context, gs GameStateView, forDM bool) (string, error) {
	prompt := "Create a game state summary."
	if forDM {
		prompt = "Create a comprehensive game state summary for the Storyteller."
	}
	systemPrompt := fmt.Sprintf("%s\n\nCurrent state:\n%s", summarizerPrompt, FormatGameState(gs))
	return s.router.SimpleChat(ctx, llm.TaskSummarize, systemPrompt, prompt)
}

// QuickStatus returns a one-line status.
func (s *Summarizer) QuickStatus(gs GameStateView) string {
	return fmt.Sprintf("Day %d | %s | %d alive | %d nominations",
		gs.DayNumber, gs.Phase, CountLiving(gs.Players), len(gs.Nominations))
}
