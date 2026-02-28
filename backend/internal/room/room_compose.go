// 角色组合集成：start_game 命令拦截，调用 Composer 注入 custom_roles
//
// [IN]  internal/game（Composer 接口）
// [POS] 游戏启动前通过 AI/Random 自动配置角色组合
package room

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

const composeTimeout = 15 * time.Second

// enrichStartGame calls the Composer before start_game reaches the engine.
// On success, injects "custom_roles" into cmd.Data.
// On failure, logs warning and returns original cmd (random fallback).
func (ra *RoomActor) enrichStartGame(ctx context.Context, cmd types.CommandEnvelope) types.CommandEnvelope {
	if ra.composer == nil {
		return cmd
	}

	state := ra.GetState()
	playerCount := 0
	for _, p := range state.Players {
		if !p.IsDM {
			playerCount++
		}
	}
	if playerCount < 5 {
		return cmd // Engine will reject anyway
	}

	composeCtx, cancel := context.WithTimeout(ctx, composeTimeout)
	defer cancel()

	result, err := ra.composer.ComposeRoles(composeCtx, game.ComposeRequest{
		PlayerCount: playerCount,
		Edition:     state.Edition,
	})
	if err != nil {
		ra.logger.Warn("composer failed, falling back to random",
			zap.String("room_id", ra.RoomID),
			zap.Error(err))
		return cmd
	}

	rolesJSON, err := json.Marshal(result.Roles)
	if err != nil {
		ra.logger.Warn("failed to marshal composed roles",
			zap.String("room_id", ra.RoomID),
			zap.Error(err))
		return cmd
	}

	// Merge custom_roles into the command payload
	var payload map[string]string
	if cmd.Payload != nil {
		if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
			ra.logger.Warn("enrichStartGame: failed to unmarshal payload",
				zap.String("room_id", ra.RoomID), zap.Error(err))
		}
	}
	if payload == nil {
		payload = make(map[string]string)
	}
	payload["custom_roles"] = string(rolesJSON)
	merged, err := json.Marshal(payload)
	if err != nil {
		ra.logger.Warn("enrichStartGame: failed to marshal payload",
			zap.String("room_id", ra.RoomID), zap.Error(err))
		return cmd
	}
	cmd.Payload = merged

	ra.logger.Info("AI composed roles",
		zap.String("room_id", ra.RoomID),
		zap.Int("player_count", playerCount),
		zap.String("roles", string(rolesJSON)),
		zap.String("reasoning", result.Reasoning))

	return cmd
}

