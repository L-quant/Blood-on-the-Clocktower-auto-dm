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
// all incomplete night actions with result="timed_out".
func CompleteRemainingNightActions(state State, cmd types.CommandEnvelope) []types.Event {
	var events []types.Event
	for _, a := range state.NightActions {
		if a.Completed {
			continue
		}
		emptyTargets, _ := json.Marshal([]string{})
		events = append(events, newEvent(cmd, "night.action.completed", map[string]string{
			"user_id": a.UserID,
			"role_id": a.RoleID,
			"targets": string(emptyTargets),
			"result":  "timed_out",
		}))
	}
	return events
}
