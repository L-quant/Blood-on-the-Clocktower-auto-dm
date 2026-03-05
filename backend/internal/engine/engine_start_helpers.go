// engine_start_helpers.go — handleStartGame 的辅助函数
//
// [IN]  game (角色定义, NightAction)
// [POS] 从 handleStartGame 提取的 custom_roles 解析与首夜 no_action 自动完成逻辑
package engine

import (
	"encoding/json"
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// parseCustomRoles extracts custom_roles from a start_game command payload.
// Returns nil slice if no custom_roles present or on parse failure.
func parseCustomRoles(rawPayload json.RawMessage) ([]string, error) {
	if len(rawPayload) == 0 {
		return nil, nil
	}

	var payload map[string]string
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return nil, fmt.Errorf("parseCustomRoles: unmarshal payload: %w", err)
	}

	cr, ok := payload["custom_roles"]
	if !ok || cr == "" {
		return nil, nil
	}

	var roles []string
	if err := json.Unmarshal([]byte(cr), &roles); err != nil {
		return nil, fmt.Errorf("parseCustomRoles: unmarshal custom_roles: %w", err)
	}

	return roles, nil
}

// buildNoActionCompletions generates night.action.completed events for
// roles that have no_action on first night (e.g. Imp).
func buildNoActionCompletions(cmd types.CommandEnvelope, nightOrder []game.NightAction) []types.Event {
	var events []types.Event
	for _, action := range nightOrder {
		actionType := ""
		if r := game.GetRoleByID(action.RoleID); r != nil {
			actionType = string(r.FirstNightActionType)
		}
		if actionType != string(game.ActionNoAction) {
			continue
		}
		events = append(events, newEvent(cmd, "night.action.completed", map[string]string{
			"user_id": action.UserID,
			"role_id": action.RoleID,
			"result":  "首夜无行动",
		}))
	}
	return events
}

// buildTeamRecognitionFromSetup 根据 SetupResult 生成首夜邪恶阵营互认事件。
// 爪牙得知恶魔身份 + 其他爪牙，恶魔得知爪牙身份 + 伪装角色。
func buildTeamRecognitionFromSetup(cmd types.CommandEnvelope, result *game.SetupResult) []types.Event {
	var demonID string
	var minionIDs []string

	for uid, a := range result.Assignments {
		if a.TrueRole == "imp" {
			demonID = uid
		}
		r := game.GetRoleByID(a.TrueRole)
		if r != nil && r.Type == game.RoleMinion {
			minionIDs = append(minionIDs, uid)
		}
	}
	if demonID == "" {
		return nil
	}

	minionIDsJSON, _ := json.Marshal(minionIDs)
	bluffsJSON, _ := json.Marshal(result.BluffRoles)

	var events []types.Event
	for _, mid := range minionIDs {
		a := result.Assignments[mid]
		events = append(events, newEvent(cmd, "team.recognition", map[string]string{
			"user_id":    mid,
			"team":       "evil",
			"role":       a.TrueRole,
			"demon_id":   demonID,
			"minion_ids": string(minionIDsJSON),
		}))
	}
	events = append(events, newEvent(cmd, "team.recognition", map[string]string{
		"user_id":    demonID,
		"team":       "evil",
		"role":       "imp",
		"demon_id":   demonID,
		"minion_ids": string(minionIDsJSON),
		"bluffs":     string(bluffsJSON),
	}))
	return events
}
