// engine_night_info.go — 夜晚信息分发层
//
// 在统一结算完成后，基于结算后的最终状态为每位信息角色生成 night.info 事件，
// 并在首夜生成 team.recognition（邪恶阵营互认）事件。
//
// [IN]  internal/game（NightAgent.ResolveAbility / spy.BuildGrimoireSnapshot）
// [POS] 三层架构的分发层，紧跟 engine_night_resolve.go 结算层
package engine

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// infoRoles 列出所有会获得夜晚信息的角色 ID。
var infoRoles = map[string]bool{
	"washerwoman":   true,
	"librarian":     true,
	"investigator":  true,
	"chef":          true,
	"empath":        true,
	"fortuneteller": true,
	"undertaker":    true,
	"ravenkeeper":   true,
	"spy":           true,
}

// distributeNightInfo 根据结算后的状态，为每位信息角色生成 night.info 事件。
// state 参数是已经执行过 applyResolveEffects 的副本（包含最终的死亡/中毒状态）。
func distributeNightInfo(state State, cmd types.CommandEnvelope) []types.Event {
	events := []types.Event{}
	isFirstNight := state.Phase == PhaseFirstNight

	// 构建 GameContext 供 NightAgent 使用
	ctx := buildGameContext(state)
	agent := game.NewNightAgent(ctx)

	for _, action := range state.NightActions {
		if !action.Completed {
			continue
		}
		if !infoRoles[action.RoleID] {
			continue
		}

		// 守鸦人特殊处理：只有今晚死亡才获得信息
		if action.RoleID == "ravenkeeper" {
			if !isPlayerDeadInState(action.UserID, state) {
				continue
			}
		}

		infoEvents := generateRoleInfo(agent, ctx, action, isFirstNight, state, cmd)
		events = append(events, infoEvents...)
	}

	// 首夜邪恶阵营互认已移至 handleStartGame (phase.first_night 之后立即发送)
	// 不再在此处重复生成

	return events
}

// generateRoleInfo 为单个角色生成 night.info 事件。
func generateRoleInfo(agent *game.NightAgent, ctx *game.GameContext,
	action NightAction, isFirstNight bool, state State,
	cmd types.CommandEnvelope) []types.Event {

	events := []types.Event{}

	// 间谍特殊处理：使用完整魔典快照
	if action.RoleID == "spy" {
		return generateSpyGrimoire(ctx, action, cmd)
	}

	req := game.AbilityRequest{
		UserID:       action.UserID,
		RoleID:       action.RoleID,
		TargetIDs:    action.TargetIDs,
		IsFirstNight: isFirstNight,
		NightNumber:  state.NightCount,
	}

	result, err := agent.ResolveAbility(req)
	if err != nil {
		slog.Warn("night.info: resolve failed",
			"role", action.RoleID, "user", action.UserID, "err", err)
		return events
	}
	if result == nil || result.Information == nil {
		return events
	}

	contentJSON, _ := json.Marshal(result.Information.Content)
	infoPayload := map[string]string{
		"user_id":   action.UserID,
		"role_id":   action.RoleID,
		"info_type": result.Information.Type,
		"content":   string(contentJSON),
		"message":   result.Message,
	}
	if result.Information.IsFalse {
		infoPayload["is_false"] = "true"
	}

	events = append(events, newEvent(cmd, "night.info", infoPayload))
	slog.Info("night.info: distributed",
		"role", action.RoleID, "user", action.UserID,
		"poisoned", result.IsPoisoned)
	return events
}

// generateSpyGrimoire 为间谍生成包含完整魔典快照的 night.info 事件。
func generateSpyGrimoire(ctx *game.GameContext, action NightAction,
	cmd types.CommandEnvelope) []types.Event {

	grimoire := game.BuildGrimoireSnapshot(ctx)
	grimoireJSON, _ := json.Marshal(grimoire)

	return []types.Event{
		newEvent(cmd, "night.info", map[string]string{
			"user_id":   action.UserID,
			"role_id":   "spy",
			"info_type": "grimoire",
			"content":   string(grimoireJSON),
			"message":   "你查看了魔典",
		}),
	}
}

// generateTeamRecognition 在首夜为邪恶阵营生成 team.recognition 事件。
// 爪牙得知恶魔身份，恶魔得知爪牙身份 + 三个不在场角色（bluffs）。
func generateTeamRecognition(state State, cmd types.CommandEnvelope) []types.Event {
	events := []types.Event{}

	if state.DemonID == "" {
		return events
	}

	// 收集存活爪牙信息
	minionNames := []string{}
	for _, mid := range state.MinionIDs {
		if p, ok := state.Players[mid]; ok {
			minionNames = append(minionNames, mid)
			_ = p // use for seat info if needed
		}
	}
	minionIDsJSON, _ := json.Marshal(minionNames)
	bluffsJSON, _ := json.Marshal(state.BluffRoles)

	// 每个爪牙收到：恶魔身份
	for _, mid := range state.MinionIDs {
		p, ok := state.Players[mid]
		if !ok {
			continue
		}
		events = append(events, newEvent(cmd, "team.recognition", map[string]string{
			"user_id":    mid,
			"team":       "evil",
			"role":       p.TrueRole,
			"demon_id":   state.DemonID,
			"minion_ids": string(minionIDsJSON),
		}))
	}

	// 恶魔收到：爪牙身份 + bluffs
	events = append(events, newEvent(cmd, "team.recognition", map[string]string{
		"user_id":    state.DemonID,
		"team":       "evil",
		"role":       "imp",
		"demon_id":   state.DemonID,
		"minion_ids": string(minionIDsJSON),
		"bluffs":     string(bluffsJSON),
	}))

	slog.Info("night.info: team recognition distributed",
		"demon", state.DemonID,
		"minions", strings.Join(minionNames, ","))
	return events
}

// isPlayerDeadInState 检查玩家在给定状态中是否已死亡。
func isPlayerDeadInState(userID string, state State) bool {
	if p, ok := state.Players[userID]; ok {
		return !p.Alive
	}
	return false
}
