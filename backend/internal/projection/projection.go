// Package projection 事件可见性过滤与状态投影
//
// [IN]  internal/engine（State 结构体）
// [IN]  internal/types（Event、Viewer、ProjectedEvent 类型）
// [OUT] api（状态脱敏返回前端）
// [OUT] realtime（WebSocket 事件过滤）
// [OUT] room（广播前事件过滤）
// [POS] 安全层，确保每个玩家只能看到自己权限内的信息
package projection

import (
	"encoding/json"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/engine"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func Project(event types.Event, state engine.State, viewer types.Viewer) *types.ProjectedEvent {
	if !allowed(event, state, viewer) {
		return nil
	}
	return &types.ProjectedEvent{
		RoomID:      event.RoomID,
		Seq:         event.Seq,
		EventType:   event.EventType,
		ActorUserID: event.ActorUserID,
		Data:        sanitizePayload(event, viewer),
		ServerTS:    event.ServerTimestampMs,
	}
}

func allowed(event types.Event, state engine.State, viewer types.Viewer) bool {
	if viewer.IsDM {
		return true
	}
	switch event.EventType {
	case "player.poisoned", "player.protected", "demon.changed":
		return false
	case "poison.rollback":
		// Internal resolution event; never shown to players
		return false
	// FIX-6: Filter evil_team.chat so only evil players can see it
	case "evil_team.chat":
		player, ok := state.Players[viewer.UserID]
		if !ok {
			return false
		}
		return player.Team == "evil"
	case "night.info":
		// Only the target player sees their own night info
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		return viewer.UserID == payload["user_id"]
	case "team.recognition":
		// Only the target evil player sees their team recognition
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		return viewer.UserID == payload["user_id"]
	case "night.action.queued":
		// Internal state-building event; players receive night.action.prompt instead
		return false
	case "ai.decision":
		// Contains sensitive data (roles, results, poison status); DM only
		return false
	case "night.action.prompt", "night.action.completed":
		// Allow players to see their own night action events
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		return viewer.UserID == payload["user_id"]
	case "bluffs.assigned":
		// Only the demon should see bluffs
		return viewer.UserID == state.DemonID
	case "whisper.sent":
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		sender := event.ActorUserID
		recipient := payload["to_user_id"]
		return viewer.UserID == sender || viewer.UserID == recipient
	case "role.assigned":
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		return viewer.UserID == payload["user_id"]
	case "ability.resolved":
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		target := payload["target_user_id"]
		return viewer.UserID == event.ActorUserID || viewer.UserID == target
	default:
		return true
	}
}

func sanitizePayload(event types.Event, viewer types.Viewer) json.RawMessage {
	if !viewer.IsDM && event.EventType == "role.assigned" {
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		if viewer.UserID != payload["user_id"] {
			return []byte(`{}`)
		}
		delete(payload, "true_role")
		delete(payload, "is_demon")
		delete(payload, "is_minion")
		delete(payload, "spy_apparent_role")
		b, _ := json.Marshal(payload)
		return b
	}
	// Strip is_false from night.info — players should not know if info is real/fake
	if !viewer.IsDM && event.EventType == "night.info" {
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		delete(payload, "is_false")
		b, _ := json.Marshal(payload)
		return b
	}
	// Strip bluffs from team.recognition for minions (only demon gets bluffs)
	if !viewer.IsDM && event.EventType == "team.recognition" {
		var payload map[string]string
		_ = json.Unmarshal(event.Payload, &payload)
		if payload["user_id"] != payload["demon_id"] {
			delete(payload, "bluffs")
		}
		b, _ := json.Marshal(payload)
		return b
	}
	return event.Payload
}

func ProjectedState(state engine.State, viewer types.Viewer) engine.State {
	cp := state.Copy()
	if !viewer.IsDM {
		cp.DemonID = ""
		cp.MinionIDs = nil
		cp.BluffRoles = nil
		// FIX-5: Clear sensitive fields that leak game info to players
		cp.NightActions = nil
		cp.AIDecisionLog = nil
		cp.RedHerringID = ""
		cp.PendingDeaths = nil

		for id, p := range cp.Players {
			p.TrueRole = ""
			if id == viewer.UserID {
				// FIX-5b: Keep own team info on reconnect
			} else {
				p.Team = ""
			}
			p.NightInfo = nil
			if id != viewer.UserID {
				p.Role = ""
			}
			cp.Players[id] = p
		}
	}
	return cp
}
