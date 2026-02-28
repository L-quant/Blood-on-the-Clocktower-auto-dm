// Package game 角色组合接口与随机实现
//
// [OUT] room（游戏启动时调用 Composer）
// [OUT] agent/subagent（AIComposer 实现此接口）
// [POS] 角色选择策略抽象层，支持随机与 AI 两种模式
package game

import (
	"context"
	"fmt"
)

// ComposeRequest contains parameters for role composition.
type ComposeRequest struct {
	PlayerCount int
	Edition     string // "tb", "bmr", "snv"
}

// ComposeResult contains the composed role list and reasoning.
type ComposeResult struct {
	Roles     []string // Role IDs (e.g., ["imp", "poisoner", "washerwoman", ...])
	Reasoning string   // AI reasoning (logged, not shown to players)
}

// Composer generates a role list for a game session.
type Composer interface {
	ComposeRoles(ctx context.Context, req ComposeRequest) (*ComposeResult, error)
}

// FallbackComposer wraps a primary composer with a fallback.
type FallbackComposer struct {
	Primary  Composer
	Fallback Composer
}

// ComposeRoles tries primary, falls back on error.
func (fc *FallbackComposer) ComposeRoles(ctx context.Context, req ComposeRequest) (*ComposeResult, error) {
	result, err := fc.Primary.ComposeRoles(ctx, req)
	if err == nil {
		return result, nil
	}
	fbResult, fbErr := fc.Fallback.ComposeRoles(ctx, req)
	if fbErr != nil {
		return nil, fmt.Errorf("compose.Fallback: primary: %w, fallback: %v", err, fbErr)
	}
	fbResult.Reasoning = "fallback: " + fbResult.Reasoning
	return fbResult, nil
}

// RandomComposer picks roles randomly according to standard distribution rules.
type RandomComposer struct{}

// ComposeRoles selects roles randomly using the standard distribution.
func (rc *RandomComposer) ComposeRoles(_ context.Context, req ComposeRequest) (*ComposeResult, error) {
	dist := GetDistribution(req.PlayerCount)
	if dist == nil {
		return nil, fmt.Errorf("compose.ComposeRoles: no distribution for %d players", req.PlayerCount)
	}

	roles, _, err := selectRolesRandomly(dist, req.PlayerCount)
	if err != nil {
		return nil, fmt.Errorf("compose.ComposeRoles: %w", err)
	}

	ids := make([]string, len(roles))
	for i, r := range roles {
		ids[i] = r.ID
	}

	return &ComposeResult{
		Roles:     ids,
		Reasoning: "random selection",
	}, nil
}
