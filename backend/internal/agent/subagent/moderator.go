// Package subagent provides the Moderator sub-agent.
package subagent

import (
	"context"
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
)

const moderatorPrompt = `You are the Moderator Agent for Blood on the Clocktower.
Manage game flow, phases, nominations, and voting. Be impartial and follow rules precisely.
Current game state: %s`

// Moderator manages game flow and player interactions.
type Moderator struct {
	router *llm.Router
}

// NewModerator creates a new Moderator agent.
func NewModerator(router *llm.Router) *Moderator {
	return &Moderator{router: router}
}

// Process handles moderator requests.
func (m *Moderator) Process(ctx context.Context, gs GameStateView, query string) (string, error) {
	systemPrompt := fmt.Sprintf(moderatorPrompt, FormatGameState(gs))
	return m.router.SimpleChat(ctx, llm.TaskReasoning, systemPrompt, query)
}

// ValidateNomination checks if a nomination is valid.
func (m *Moderator) ValidateNomination(ctx context.Context, gs GameStateView, nominator, nominee string) (bool, string, error) {
	// Simple validation
	for _, p := range gs.Players {
		if p.ID == nominator && !p.IsAlive {
			return false, "Nominator is dead", nil
		}
		if p.ID == nominee && !p.IsAlive {
			return false, "Nominee is dead", nil
		}
	}
	return true, "Nomination is valid", nil
}
