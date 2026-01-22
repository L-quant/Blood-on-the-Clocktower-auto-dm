// Package subagent provides the Narrator sub-agent.
package subagent

import (
	"context"
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
)

const narratorPrompt = `You are the Narrator for Blood on the Clocktower.
Create immersive, atmospheric narration. Keep it concise but evocative.
Current game state: %s`

// Narrator generates atmospheric game narration.
type Narrator struct {
	router *llm.Router
}

// NewNarrator creates a new Narrator agent.
func NewNarrator(router *llm.Router) *Narrator {
	return &Narrator{router: router}
}

// NarratePhaseChange creates narration for phase transitions.
func (n *Narrator) NarratePhaseChange(ctx context.Context, gs GameStateView, oldPhase, newPhase string) (string, error) {
	prompt := fmt.Sprintf("Create a brief atmospheric narration for phase change from %s to %s. Day %d, %d alive.",
		oldPhase, newPhase, gs.DayNumber, CountLiving(gs.Players))
	return n.router.SimpleChat(ctx, llm.TaskNarration, narratorPrompt, prompt)
}

// NarrateDeath creates narration for a player's death.
func (n *Narrator) NarrateDeath(ctx context.Context, gs GameStateView, playerName, cause string) (string, error) {
	prompt := fmt.Sprintf("Create a brief death announcement for %s. Cause: %s. Day %d.",
		playerName, cause, gs.DayNumber)
	return n.router.SimpleChat(ctx, llm.TaskNarration, narratorPrompt, prompt)
}
