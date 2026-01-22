// Package subagent provides the PlayerModeler sub-agent.
package subagent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
)

const playerModelerPrompt = `You are the Player Modeler for Blood on the Clocktower.
Analyze player behavior to help the DM understand dynamics. This is DM-only information.`

// PlayerModeler analyzes player behavior.
type PlayerModeler struct {
	mu           sync.RWMutex
	router       *llm.Router
	observations map[string]*PlayerProfile
}

// PlayerProfile tracks a player's behavior.
type PlayerProfile struct {
	PlayerID    string
	PlayerName  string
	ClaimedRole string
	ActualRole  string
	VotesFor    []string
	AccusedBy   []string
	Notes       []string
}

// NewPlayerModeler creates a new PlayerModeler agent.
func NewPlayerModeler(router *llm.Router) *PlayerModeler {
	return &PlayerModeler{
		router:       router,
		observations: make(map[string]*PlayerProfile),
	}
}

// RecordVote records a vote observation.
func (p *PlayerModeler) RecordVote(voterID, voterName, targetID, targetName string, votedYes bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	profile := p.getOrCreate(voterID, voterName)
	if votedYes {
		profile.VotesFor = append(profile.VotesFor, targetName)
	}
}

// RecordAccusation records when one player accuses another.
func (p *PlayerModeler) RecordAccusation(accuserID, accuserName, targetID, targetName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	target := p.getOrCreate(targetID, targetName)
	target.AccusedBy = append(target.AccusedBy, accuserName)
}

// IdentifySuspects identifies suspicious players.
func (p *PlayerModeler) IdentifySuspects(ctx context.Context, gs GameStateView) (string, error) {
	history := p.formatHistory()
	prompt := fmt.Sprintf("%s\n\nPlayer history:\n%s\n\nIdentify the most suspicious players.",
		playerModelerPrompt, history)
	return p.router.SimpleChat(ctx, llm.TaskReasoning, prompt, "Rank suspects with reasoning.")
}

func (p *PlayerModeler) getOrCreate(id, name string) *PlayerProfile {
	if profile, ok := p.observations[id]; ok {
		return profile
	}
	profile := &PlayerProfile{PlayerID: id, PlayerName: name}
	p.observations[id] = profile
	return profile
}

func (p *PlayerModeler) formatHistory() string {
	var sb strings.Builder
	for _, profile := range p.observations {
		sb.WriteString(fmt.Sprintf("- %s: voted against %v, accused by %v\n",
			profile.PlayerName, profile.VotesFor, profile.AccusedBy))
	}
	return sb.String()
}

// Clear resets observations.
func (p *PlayerModeler) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observations = make(map[string]*PlayerProfile)
}
