// 角色组合器工厂：创建 AI 或随机角色组合器
//
// [OUT] cmd/server（main.go 初始化 Composer）
// [POS] 组合器创建入口，隔离 subagent/llm 内部依赖
package agent

import (
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/subagent"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
)

// NewComposer creates a game.Composer based on LLM config.
// If LLM is configured, returns AI composer with random fallback.
// Otherwise returns a pure random composer.
func NewComposer(cfg LLMRoutingConfig) game.Composer {
	random := &game.RandomComposer{}

	if cfg.Default.Model == "" || cfg.Default.APIKey == "" {
		return random
	}

	router := llm.NewRouterFromConfig(cfg)
	aiComposer := subagent.NewAIComposer(router)

	return &game.FallbackComposer{
		Primary:  aiComposer,
		Fallback: random,
	}
}
