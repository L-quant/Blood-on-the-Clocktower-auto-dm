// Package engine 夜晚超时自动补全未完成行动
//
// 当 autodm 触发 advance_phase("day") 时，可能仍有玩家未完成夜晚行动。
// CompleteRemainingNightActions 为这些玩家生成 timed_out 结果事件，
// 确保状态机可以干净地过渡到白天。
package engine

import (
	"encoding/json"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// CompleteRemainingNightActions emits night.action.completed events for
// incomplete night actions. Uses ActionType to differentiate:
//   - info / good-side select: auto-complete with timed_out
//   - evil-side select (imp/poisoner): skip (do not force-complete)
//
// Returns (events, hasEvilPending) where hasEvilPending indicates if
// evil critical actions remain unresolved.
func CompleteRemainingNightActions(state State, cmd types.CommandEnvelope) ([]types.Event, bool) {
	var events []types.Event
	hasEvilPending := false
	emptyTargets, _ := json.Marshal([]string{})

	for _, a := range state.NightActions {
		if a.Completed {
			continue
		}
		if isEvilCriticalAction(a) {
			hasEvilPending = true
			continue
		}
		events = append(events, newEvent(cmd, "night.action.completed", map[string]string{
			"user_id": a.UserID,
			"role_id": a.RoleID,
			"targets": string(emptyTargets),
			"result":  "timed_out",
		}))
	}
	return events, hasEvilPending
}

// isEvilCriticalAction returns true if the action belongs to an evil
// role with a select-type ability (imp kill, poisoner poison).
func isEvilCriticalAction(a NightAction) bool {
	if a.ActionType == "info" || a.ActionType == "no_action" || a.ActionType == "" {
		return false
	}
	return a.RoleID == "imp" || a.RoleID == "poisoner"
}
