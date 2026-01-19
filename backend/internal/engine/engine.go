package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

var (
	ErrPhaseEnded = errors.New("game already ended")
)

func HandleCommand(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase == PhaseEnded {
		return nil, nil, ErrPhaseEnded
	}
	switch cmd.Type {
	case "join":
		return handleJoin(state, cmd)
	case "leave":
		return handleLeave(state, cmd)
	case "start_game":
		return handleStartGame(state, cmd)
	case "public_chat":
		return handlePublicChat(state, cmd)
	case "whisper":
		return handleWhisper(state, cmd)
	case "nominate":
		return handleNomination(state, cmd)
	case "vote":
		return handleVote(state, cmd)
	case "ability.use":
		return handleAbility(state, cmd)
	default:
		return nil, nil, fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

func handleJoin(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if _, exists := state.Players[cmd.ActorUserID]; exists {
		return nil, nil, fmt.Errorf("player already joined")
	}
	payload := map[string]string{"role": "player"}
	return []types.Event{newEvent(cmd, "player.joined", payload)}, acceptedResult(cmd.CommandID), nil
}

func handleLeave(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if _, exists := state.Players[cmd.ActorUserID]; !exists {
		return nil, nil, fmt.Errorf("player not in room")
	}
	return []types.Event{newEvent(cmd, "player.left", nil)}, acceptedResult(cmd.CommandID), nil
}

func handleStartGame(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseLobby {
		return nil, nil, fmt.Errorf("cannot start game outside lobby")
	}
	if len(state.Players) < 3 {
		return nil, nil, fmt.Errorf("need at least 3 players")
	}
	events := []types.Event{newEvent(cmd, "game.started", nil)}
	roles := []string{"villager", "villager", "demon", "seer", "soldier"}
	i := 0
	for userID, p := range state.Players {
		role := roles[i%len(roles)]
		if p.IsDM {
			role = "dm"
		}
		i++
		payload := map[string]string{"user_id": userID, "role": role}
		events = append(events, newEvent(cmd, "role.assigned", payload))
	}
	events = append(events, newEvent(cmd, "phase.day", map[string]string{"day": "1"}))
	return events, acceptedResult(cmd.CommandID), nil
}

func handlePublicChat(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	if payload == nil {
		payload = map[string]string{}
	}
	if payload["message"] == "" {
		return nil, nil, fmt.Errorf("message required")
	}
	return []types.Event{newEvent(cmd, "public.chat", payload)}, acceptedResult(cmd.CommandID), nil
}

func handleWhisper(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	if payload == nil || payload["to_user_id"] == "" || payload["message"] == "" {
		return nil, nil, fmt.Errorf("invalid whisper payload")
	}
	if _, ok := state.Players[payload["to_user_id"]]; !ok {
		return nil, nil, fmt.Errorf("recipient not in room")
	}
	return []types.Event{newEvent(cmd, "whisper.sent", payload)}, acceptedResult(cmd.CommandID), nil
}

func handleNomination(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseDay {
		return nil, nil, fmt.Errorf("nominations only allowed during day")
	}
	if state.Nomination != nil && !state.Nomination.Resolved {
		return nil, nil, fmt.Errorf("nomination in progress")
	}
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	nominee := payload["nominee"]
	if nominee == "" {
		return nil, nil, fmt.Errorf("nominee required")
	}
	if _, ok := state.Players[nominee]; !ok {
		return nil, nil, fmt.Errorf("nominee not in room")
	}
	return []types.Event{newEvent(cmd, "nomination.created", map[string]string{"nominee": nominee})}, acceptedResult(cmd.CommandID), nil
}

func handleVote(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Nomination == nil || state.Nomination.Resolved {
		return nil, nil, fmt.Errorf("no active nomination")
	}
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	vote := payload["vote"]
	if vote != "yes" && vote != "no" {
		return nil, nil, fmt.Errorf("vote must be yes or no")
	}
	events := []types.Event{newEvent(cmd, "vote.cast", map[string]string{"vote": vote})}
	yesCount := 0
	alive := 0
	for _, p := range state.Players {
		if p.Alive {
			alive++
		}
	}
	for _, v := range state.Nomination.Votes {
		if v {
			yesCount++
		}
	}
	if vote == "yes" {
		yesCount++
	}
	if yesCount*2 >= alive {
		result := "executed"
		payloadRes := map[string]string{"result": result, "nominee": state.Nomination.Nominee}
		if p, ok := state.Players[state.Nomination.Nominee]; ok && p.Role == "demon" {
			payloadRes["win"] = "good"
			events = append(events, newEvent(cmd, "execution.resolved", payloadRes))
			events = append(events, newEvent(cmd, "game.ended", map[string]string{"winner": "good"}))
			return events, acceptedResult(cmd.CommandID), nil
		}
		events = append(events, newEvent(cmd, "execution.resolved", payloadRes))
	}
	return events, acceptedResult(cmd.CommandID), nil
}

func handleAbility(state State, cmd types.CommandEnvelope) ([]types.Event, *types.CommandResult, error) {
	if state.Phase != PhaseNight {
		return nil, nil, fmt.Errorf("abilities only at night")
	}
	var payload map[string]string
	_ = json.Unmarshal(cmd.Payload, &payload)
	if payload == nil {
		payload = map[string]string{}
	}
	return []types.Event{newEvent(cmd, "ability.resolved", payload)}, acceptedResult(cmd.CommandID), nil
}

func newEvent(cmd types.CommandEnvelope, eventType string, payload map[string]string) types.Event {
	b, _ := json.Marshal(payload)
	return types.Event{
		RoomID:            cmd.RoomID,
		Seq:               0,
		EventID:           uuid.NewString(),
		EventType:         eventType,
		ActorUserID:       cmd.ActorUserID,
		CausationCommand:  cmd.CommandID,
		Payload:           b,
		ServerTimestampMs: time.Now().UnixMilli(),
	}
}

func acceptedResult(commandID string) *types.CommandResult {
	return &types.CommandResult{CommandID: commandID, Status: "accepted"}
}
