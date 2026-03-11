// Package engine 夜晚超时路径（当前版本显式禁用）
//
// 项目当前规则要求夜晚只能自然结束，不能因计时器强制结束。
// night_timeout 命令保留为兼容入口，但一律返回错误，避免外部误调用。
package engine

import (
	"encoding/json"
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// handleNightTimeout processes the night_timeout command from phase timer.
// Auto-completes info/good actions, keeps evil critical actions pending
// with a reminder, or advances to day if all done.
func handleNightTimeout(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	_ = state
	_ = cmd
	return nil, nil, fmt.Errorf("engine.handleNightTimeout: night timeout is disabled in current version")
}

func finalizeNightFromCompletions(state State, cmd types.CommandEnvelope,
	completionEvents []types.Event) []types.Event {
	workingState := state.Copy()
	applyEventsToState(&workingState, completionEvents)

	resolveEvents := resolveNight(workingState, cmd)
	events := append([]types.Event{}, resolveEvents...)

	resolvedState := workingState.Copy()
	applyResolveEffects(&resolvedState, resolveEvents)

	infoEvents := distributeNightInfo(resolvedState, cmd)
	events = append(events, infoEvents...)
	events = append(events, newEvent(cmd, "phase.day", buildPhaseDayPayload(resolvedState, resolveEvents)))

	winEvents := checkWinCondition(resolvedState, cmd)
	events = append(events, winEvents...)

	return events
}

func applyEventsToState(state *State, events []types.Event) {
	for _, event := range events {
		var payload map[string]string
		if len(event.Payload) > 0 {
			_ = json.Unmarshal(event.Payload, &payload)
		}
		state.Reduce(EventPayload{
			Seq:     event.Seq,
			Type:    event.EventType,
			Payload: payload,
		})
	}
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
