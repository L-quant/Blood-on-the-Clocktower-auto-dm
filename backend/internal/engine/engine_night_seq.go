// engine_night_seq.go — 夜晚行动顺序控制
//
// 确保夜晚技能按官方行动顺序逐个执行：
// 先行角色（如投毒者）的效果在后行角色（如洗衣妇）行动前已生效。
//
// [IN]  internal/game（角色定义）
// [IN]  internal/types（Event 类型）
// [POS] 从 engine.go 提取的夜晚顺序执行逻辑
package engine

import (
	"fmt"
	"log/slog"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// buildFirstPrompt emits a night.action.prompt for the first actionable
// night action (skipping no_action roles that are already auto-completed).
// Called after all night.action.queued events in start_game / advance_phase.
func buildFirstPrompt(cmd types.CommandEnvelope, nightActions []NightAction) []types.Event {
	for _, a := range nightActions {
		if a.Completed {
			continue
		}
		slog.Info("night.seq: buildFirstPrompt",
			"user_id", a.UserID, "role_id", a.RoleID, "order", a.Order)
		return []types.Event{buildPromptEvent(cmd, a)}
	}
	slog.Info("night.seq: buildFirstPrompt — no actionable actions found")
	return nil
}

// buildNextPrompt emits a night.action.prompt for the next uncompleted
// action after the just-completed one. Returns nil if all done.
func buildNextPrompt(cmd types.CommandEnvelope, nightActions []NightAction, justCompletedUserID string) []types.Event {
	passed := false
	for _, a := range nightActions {
		if a.UserID == justCompletedUserID && !passed {
			passed = true
			continue
		}
		if !passed {
			continue
		}
		if a.Completed {
			continue
		}
		slog.Info("night.seq: buildNextPrompt",
			"justCompleted", justCompletedUserID,
			"nextUser", a.UserID, "nextRole", a.RoleID, "order", a.Order)
		return []types.Event{buildPromptEvent(cmd, a)}
	}
	slog.Info("night.seq: buildNextPrompt — all done after", "justCompleted", justCompletedUserID)
	return nil
}

// buildPromptEvent creates a night.action.prompt event for a specific
// player, telling the frontend to open their ability panel.
func buildPromptEvent(cmd types.CommandEnvelope, a NightAction) types.Event {
	actionType := a.ActionType
	if actionType == "" {
		if r := game.GetRoleByID(a.RoleID); r != nil {
			actionType = string(r.NightActionType)
		}
	}
	return newEvent(cmd, "night.action.prompt", map[string]string{
		"user_id":     a.UserID,
		"role_id":     a.RoleID,
		"order":       fmt.Sprintf("%d", a.Order),
		"action_type": actionType,
	})
}

// validateCurrentNightAction strictly enforces that only the next
// uncompleted action's player may submit an ability. Scans the
// NightActions array to find the first uncompleted action rather than
// trusting CurrentAction index (which can desync with auto-completions).
func validateCurrentNightAction(state State, actorID string) error {
	if len(state.NightActions) == 0 {
		return nil // No actions queued; allow (shouldn't happen in practice)
	}
	// Find first uncompleted action by scanning
	for _, a := range state.NightActions {
		if a.Completed {
			continue
		}
		// Found the first uncompleted action — must be the actor
		if a.UserID != actorID {
			slog.Warn("night.seq: validateCurrentNightAction rejected",
				"actor", actorID, "expected", a.UserID, "role", a.RoleID, "order", a.Order)
			return fmt.Errorf("not your turn to act, waiting for %s (order %d)",
				a.RoleID, a.Order)
		}
		slog.Info("night.seq: validateCurrentNightAction accepted",
			"actor", actorID, "role", a.RoleID, "order", a.Order)
		return nil
	}
	return fmt.Errorf("all night actions already completed")
}

// buildEngineNightActions converts game.NightAction list to engine
// NightAction list, preserving order and action types.
func buildEngineNightActions(gameActions []game.NightAction, isFirstNight bool) []NightAction {
	actions := make([]NightAction, 0, len(gameActions))
	for _, ga := range gameActions {
		actionType := ""
		if r := game.GetRoleByID(ga.RoleID); r != nil {
			if isFirstNight {
				actionType = string(r.FirstNightActionType)
			} else {
				actionType = string(r.NightActionType)
			}
		}
		actions = append(actions, NightAction{
			UserID:     ga.UserID,
			RoleID:     ga.RoleID,
			Order:      ga.Order,
			ActionType: actionType,
		})
	}
	return actions
}

// buildNoActionSet returns a set of user IDs that have no_action on
// first night (used to mark them as completed before prompt selection).
func buildNoActionSet(gameActions []game.NightAction) map[string]bool {
	m := make(map[string]bool)
	for _, ga := range gameActions {
		r := game.GetRoleByID(ga.RoleID)
		if r != nil && r.FirstNightActionType == game.ActionNoAction {
			m[ga.UserID] = true
		}
	}
	return m
}
