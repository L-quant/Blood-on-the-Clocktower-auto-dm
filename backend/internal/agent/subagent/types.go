// Package subagent provides specialized sub-agents for the Auto-DM system.
package subagent

import (
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
)

// GameStateView is a read-only view of game state for sub-agents.
type GameStateView struct {
	RoomID      string
	Phase       string
	DayNumber   int
	Players     []PlayerView
	Nominations []NominationView
	Edition     string
	Script      []string
}

// PlayerView is a read-only view of a player.
type PlayerView struct {
	ID       string
	Name     string
	Role     string
	IsAlive  bool
	HasVoted bool
}

// NominationView is a read-only view of a nomination.
type NominationView struct {
	Nominator string
	Nominee   string
	Votes     int
	Threshold int
}

// FormatGameState formats game state for prompts.
func FormatGameState(gs GameStateView) string {
	var result string
	result += fmt.Sprintf("Room: %s | Phase: %s | Day: %d\n", gs.RoomID, gs.Phase, gs.DayNumber)
	result += fmt.Sprintf("Edition: %s\n", gs.Edition)
	result += fmt.Sprintf("Players (%d):\n", len(gs.Players))

	for _, p := range gs.Players {
		status := "alive"
		if !p.IsAlive {
			status = "dead"
		}
		role := p.Role
		if role == "" {
			role = "unknown"
		}
		result += fmt.Sprintf("  - %s (%s): %s\n", p.Name, role, status)
	}

	return result
}

// CountLiving counts living players.
func CountLiving(players []PlayerView) int {
	count := 0
	for _, p := range players {
		if p.IsAlive {
			count++
		}
	}
	return count
}

// Router is a type alias for convenience.
type Router = llm.Router
