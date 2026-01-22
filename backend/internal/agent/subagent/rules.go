// Package subagent provides the Rules sub-agent.
package subagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
)

const rulesPrompt = `You are the Rules Agent for Blood on the Clocktower.
Provide accurate answers about game rules and mechanics.`

// Rules answers questions about game rules.
type Rules struct {
	router   *llm.Router
	roleData map[string]RoleInfo
}

// RoleInfo contains information about a role.
type RoleInfo struct {
	Name        string
	Team        string
	Ability     string
	FirstNight  int
	OtherNights int
}

// NewRules creates a new Rules agent.
func NewRules(router *llm.Router) *Rules {
	return &Rules{
		router:   router,
		roleData: defaultRoleData(),
	}
}

// Process handles rules questions.
func (r *Rules) Process(ctx context.Context, gs GameStateView, query string) (string, error) {
	roleContext := r.getRoleContext(query)
	fullQuery := query
	if roleContext != "" {
		fullQuery = query + "\n\nRelevant roles:\n" + roleContext
	}
	return r.router.SimpleChat(ctx, llm.TaskRules, rulesPrompt, fullQuery)
}

// GetRoleInfo returns information about a specific role.
func (r *Rules) GetRoleInfo(roleName string) (RoleInfo, bool) {
	info, ok := r.roleData[strings.ToLower(roleName)]
	return info, ok
}

func (r *Rules) getRoleContext(query string) string {
	queryLower := strings.ToLower(query)
	var found []string
	for name, info := range r.roleData {
		if strings.Contains(queryLower, name) {
			found = append(found, fmt.Sprintf("%s (%s): %s", info.Name, info.Team, info.Ability))
		}
	}
	return strings.Join(found, "\n")
}

func defaultRoleData() map[string]RoleInfo {
	return map[string]RoleInfo{
		"washerwoman":  {Name: "Washerwoman", Team: "Townsfolk", Ability: "You start knowing that 1 of 2 players is a particular Townsfolk.", FirstNight: 32},
		"librarian":    {Name: "Librarian", Team: "Townsfolk", Ability: "You start knowing that 1 of 2 players is a particular Outsider.", FirstNight: 33},
		"investigator": {Name: "Investigator", Team: "Townsfolk", Ability: "You start knowing that 1 of 2 players is a particular Minion.", FirstNight: 34},
		"chef":         {Name: "Chef", Team: "Townsfolk", Ability: "You start knowing how many pairs of evil players there are.", FirstNight: 35},
		"empath":       {Name: "Empath", Team: "Townsfolk", Ability: "Each night, you learn how many of your 2 alive neighbours are evil.", FirstNight: 36, OtherNights: 53},
		"imp":          {Name: "Imp", Team: "Demon", Ability: "Each night, choose a player: they die.", FirstNight: 25, OtherNights: 16},
		"poisoner":     {Name: "Poisoner", Team: "Minion", Ability: "Each night, choose a player: they are poisoned tonight and tomorrow.", FirstNight: 17, OtherNights: 8},
	}
}
