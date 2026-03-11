// engine_day_flow.go — 白天阶段辅助逻辑
//
// 提供白天阶段通用判断，以及少数白天能力直接触发入夜时的
// 夜晚过渡事件构造（如猎手命中恶魔后红衣女郎接任）。
//
// [IN]  internal/game（NightAction 生成）
// [IN]  internal/types（Event 类型）
// [POS] 白天到夜晚的过渡辅助层，避免把阶段跳转细节堆回 engine.go
package engine

import (
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func isDaytimePhase(phase Phase) bool {
	return phase == PhaseDay || phase == PhaseNomination || phase == PhaseVoting
}

func buildNightTransitionEvents(state State, cmd types.CommandEnvelope) []types.Event {
	events := []types.Event{
		newEvent(cmd, "poison.cleared", nil),
		newEvent(cmd, "phase.night", nil),
	}

	assignments := make(map[string]game.Assignment)
	for uid, player := range state.Players {
		if player.Alive {
			assignments[uid] = game.Assignment{
				UserID:   uid,
				TrueRole: player.TrueRole,
				Team:     game.Team(player.Team),
			}
		}
	}

	allRoles := game.GetAllRoles()
	nightActions := game.GenerateNightOrder(allRoles, assignments, false)
	for _, action := range nightActions {
		actionType := ""
		if role := game.GetRoleByID(action.RoleID); role != nil {
			actionType = string(role.NightActionType)
		}
		events = append(events, newEvent(cmd, "night.action.queued", map[string]string{
			"user_id":     action.UserID,
			"role_id":     action.RoleID,
			"order":       fmt.Sprintf("%d", action.Order),
			"action_type": actionType,
		}))
	}

	queuedActions := buildEngineNightActions(nightActions, false)
	events = append(events, buildFirstPrompt(cmd, queuedActions)...)
	return events
}

func hasEventType(events []types.Event, eventType string) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
