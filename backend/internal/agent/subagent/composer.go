// AI 角色组合器：根据玩家人数和版本智能选配角色
//
// [IN]  internal/agent/llm（LLM 路由）
// [IN]  internal/game（角色定义与分配规则）
// [OUT] room（游戏启动时调用）
// [POS] 通过 AI 分析为每局生成平衡且有趣的角色组合
package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
)

const composerSystemPrompt = `You are a Blood on the Clocktower Storyteller creating a balanced and fun game setup.

Your task: choose which specific roles to include in the game.

RULES:
- You must pick EXACTLY the number of roles specified for each type
- All role IDs must come from the available list
- No duplicate roles
- If you include Baron (minion), its +2 Outsider effect is handled automatically — just select it normally

DESIGN GOALS:
1. Balance: mix information-gathering and active roles for Good team
2. Interaction: create interesting combos (e.g. Poisoner+Empath, Drunk+Washerwoman, Spy+Chef)
3. Variety: avoid always picking the same "strongest" roles
4. Fun: include at least one unusual or underused role for surprise

Respond with ONLY a JSON object, no explanation:
{"roles": ["role_id_1", "role_id_2", ...], "reasoning": "brief explanation"}`

// AIComposer uses LLM to compose game roles.
type AIComposer struct {
	router *llm.Router
}

// NewAIComposer creates a new AI-powered composer.
func NewAIComposer(router *llm.Router) *AIComposer {
	return &AIComposer{router: router}
}

// ComposeRoles asks the LLM to compose a balanced role set.
func (c *AIComposer) ComposeRoles(ctx context.Context, req game.ComposeRequest) (*game.ComposeResult, error) {
	dist := game.GetDistribution(req.PlayerCount)
	if dist == nil {
		return nil, fmt.Errorf("subagent.AIComposer: no distribution for %d players", req.PlayerCount)
	}

	userMsg := buildComposePrompt(req.PlayerCount, dist)
	response, err := c.router.SimpleChat(ctx, llm.TaskReasoning, composerSystemPrompt, userMsg)
	if err != nil {
		return nil, fmt.Errorf("subagent.AIComposer: llm call failed: %w", err)
	}

	return parseComposeResponse(response, req.PlayerCount)
}

// buildComposePrompt creates the user message with game parameters.
func buildComposePrompt(playerCount int, dist *game.PlayerDistribution) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Game: %d players (Trouble Brewing edition)\n", playerCount)
	fmt.Fprintf(&sb, "Required: %d Townsfolk, %d Outsiders, %d Minions, %d Demon\n\n",
		dist.Townsfolk, dist.Outsiders, dist.Minions, dist.Demons)

	sb.WriteString("Available Townsfolk: ")
	writeRoleList(&sb, game.GetRolesByType(game.RoleTownsfolk))

	sb.WriteString("\nAvailable Outsiders: ")
	writeRoleList(&sb, game.GetRolesByType(game.RoleOutsider))

	sb.WriteString("\nAvailable Minions: ")
	writeRoleList(&sb, game.GetRolesByType(game.RoleMinion))

	sb.WriteString("\nAvailable Demons: ")
	writeRoleList(&sb, game.GetRolesByType(game.RoleDemon))

	return sb.String()
}

// writeRoleList formats a list of roles with abilities.
func writeRoleList(sb *strings.Builder, roles []game.Role) {
	for i, r := range roles {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(sb, "%s (%s)", r.ID, r.Ability)
	}
}

// parseComposeResponse extracts role IDs from the LLM JSON response.
func parseComposeResponse(raw string, expectedCount int) (*game.ComposeResult, error) {
	// Extract JSON from response (LLM may wrap in markdown code block)
	jsonStr := extractJSON(raw)

	var resp struct {
		Roles     []string `json:"roles"`
		Reasoning string   `json:"reasoning"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return nil, fmt.Errorf("subagent.AIComposer: parse response: %w", err)
	}

	if len(resp.Roles) != expectedCount {
		return nil, fmt.Errorf("subagent.AIComposer: expected %d roles, got %d", expectedCount, len(resp.Roles))
	}

	// Validate all role IDs exist and no duplicates
	seen := make(map[string]bool, len(resp.Roles))
	for _, id := range resp.Roles {
		if game.GetRoleByID(id) == nil {
			return nil, fmt.Errorf("subagent.AIComposer: unknown role: %s", id)
		}
		if seen[id] {
			return nil, fmt.Errorf("subagent.AIComposer: duplicate role: %s", id)
		}
		seen[id] = true
	}

	return &game.ComposeResult{
		Roles:     resp.Roles,
		Reasoning: resp.Reasoning,
	}, nil
}

// extractJSON finds the first JSON object in a string.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	end := strings.LastIndex(s, "}")
	if end < start {
		return s
	}
	return s[start : end+1]
}
