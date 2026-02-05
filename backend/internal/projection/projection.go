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
		RoomID:    event.RoomID,
		Seq:       event.Seq,
		EventType: event.EventType,
		Data:      sanitizePayload(event, viewer),
		ServerTS:  event.ServerTimestampMs,
	}
}

func allowed(event types.Event, state engine.State, viewer types.Viewer) bool {
	if viewer.IsDM {
		return true
	}
	switch event.EventType {
	case "bluffs.assigned", "night.action.queued", "night.action.completed", "player.poisoned", "player.protected", "demon.changed":
		return false
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

		for id, p := range cp.Players {
			p.TrueRole = ""
			p.Team = ""
			p.NightInfo = nil
			if id != viewer.UserID {
				p.Role = ""
			}
			cp.Players[id] = p
		}
	}
	return cp
}
