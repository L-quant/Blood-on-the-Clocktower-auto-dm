// Package engine 夜晚超时差异化处理
//
// 当夜晚计时器超时时，自动补全信息类和善良方行动，
// 邪恶方关键行动（imp杀人、poisoner选毒）不强制超时，
// 而是发送 action.reminder 提醒并等待下一轮超时。
package engine

import (
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// handleNightTimeout processes the night_timeout command from phase timer.
// Auto-completes info/good actions, keeps evil critical actions pending
// with a reminder, or advances to day if all done.
func handleNightTimeout(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseFirstNight && state.Phase != PhaseNight {
		return nil, nil, fmt.Errorf("engine.handleNightTimeout: not in night phase")
	}

	timeoutEvents, hasEvilPending := CompleteRemainingNightActions(state, cmd)
	events := append([]types.Event{}, timeoutEvents...)

	if hasEvilPending {
		events = append(events, buildEvilReminders(state, cmd)...)
	} else {
		events = append(events, newEvent(cmd, "phase.day", map[string]string{}))
	}

	return events, acceptedResult(cmd.CommandID), nil
}

// buildEvilReminders generates action.reminder events for each incomplete
// evil critical action, prompting the player to act.
func buildEvilReminders(state State, cmd types.CommandEnvelope) []types.Event {
	var events []types.Event
	for _, a := range state.NightActions {
		if a.Completed || !isEvilCriticalAction(a) {
			continue
		}
		events = append(events, newEvent(cmd, "action.reminder", map[string]string{
			"user_id": a.UserID,
			"role_id": a.RoleID,
			"message": "请尽快完成你的夜晚行动",
		}))
	}
	return events
}
