package engine

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/game"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestHandleJoin(t *testing.T) {
	state := NewState("room1")
	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "join",
		ActorUserID: "alice",
	}
	events, result, err := HandleCommand(state, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != "player.joined" {
		t.Errorf("unexpected event type: %s", events[0].EventType)
	}
	if result.Status != "accepted" {
		t.Errorf("expected accepted, got %s", result.Status)
	}
}

func TestHandleJoinDuplicate(t *testing.T) {
	state := NewState("room1")
	state.Players["alice"] = Player{UserID: "alice"}
	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "join",
		ActorUserID: "alice",
	}
	_, _, err := HandleCommand(state, cmd)
	if err == nil {
		t.Fatalf("expected error for duplicate join")
	}
}

func TestReduceJoin(t *testing.T) {
	s := NewState("room1")
	s.Reduce(EventPayload{Seq: 1, Type: "player.joined", Actor: "bob", Payload: map[string]string{}})
	if _, ok := s.Players["bob"]; !ok {
		t.Errorf("player bob not found")
	}
}

func TestReduceGameStartedPhaseDayNight(t *testing.T) {
	s := NewState("room1")
	s.Reduce(EventPayload{Seq: 1, Type: "game.started", Actor: "system", Payload: nil})
	// Game starts in first_night phase according to Blood on the Clocktower rules
	s.Reduce(EventPayload{Seq: 2, Type: "phase.first_night", Actor: "system", Payload: nil})
	if s.Phase != PhaseFirstNight {
		t.Errorf("expected first_night phase, got %s", s.Phase)
	}
	// Transition to day after first night
	s.Reduce(EventPayload{Seq: 3, Type: "phase.day", Actor: "system", Payload: nil})
	if s.Phase != PhaseDay {
		t.Errorf("expected day phase, got %s", s.Phase)
	}
	s.Reduce(EventPayload{Seq: 4, Type: "phase.night", Actor: "system", Payload: nil})
	if s.Phase != PhaseNight {
		t.Errorf("expected night phase, got %s", s.Phase)
	}
}

func TestVoteResolution(t *testing.T) {
	s := NewState("room1")
	s.Players["a"] = Player{UserID: "a", Alive: true, SeatNumber: 1}
	s.Players["b"] = Player{UserID: "b", Alive: true, SeatNumber: 2}
	s.Players["c"] = Player{UserID: "c", Alive: true, SeatNumber: 3}
	s.Phase = PhaseDay

	// First create a nomination
	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "nominate",
		ActorUserID: "a",
		Payload:     []byte(`{"nominee":"b"}`),
	}
	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("nominate failed: %v", err)
	}
	for _, e := range events {
		s.Reduce(toEventPayload(e))
	}

	// Transition to voting phase (defense ends)
	s.SubPhase = SubPhaseVoting

	voteCmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "vote",
		ActorUserID: "a",
		Payload:     []byte(`{"vote":"yes"}`),
	}
	events, _, err = HandleCommand(s, voteCmd)
	if err != nil {
		t.Fatalf("vote failed: %v", err)
	}
	for _, e := range events {
		s.Reduce(toEventPayload(e))
	}
	// Second vote (player c votes yes)
	voteCmd2 := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "vote",
		ActorUserID: "c",
		Payload:     []byte(`{"vote":"yes"}`),
	}
	events, _, err = HandleCommand(s, voteCmd2)
	if err != nil {
		t.Fatalf("vote2 failed: %v", err)
	}
	for _, e := range events {
		s.Reduce(toEventPayload(e))
	}

	// Third vote (nominee also needs to vote for all-voted check)
	// In Blood on the Clocktower, the nominee can also vote
	voteCmd3 := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "vote",
		ActorUserID: "b",
		Payload:     []byte(`{"vote":"no"}`),
	}
	events, _, err = HandleCommand(s, voteCmd3)
	if err != nil {
		t.Fatalf("vote3 failed: %v", err)
	}

	resolved := false
	for _, e := range events {
		if e.EventType == "execution.resolved" {
			resolved = true
		}
	}
	if !resolved {
		t.Fatalf("expected execution resolved with majority vote (2 yes out of 3 players, threshold is 2)")
	}
}

func TestHandleWriteEventByDM(t *testing.T) {
	state := NewState("room1")
	state.Players["dm"] = Player{UserID: "dm", IsDM: true}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "write_event",
		ActorUserID: "dm",
		Payload:     []byte(`{"event_type":"audit.note","data":{"reason":"manual override","count":2}}`),
	}

	events, result, err := HandleCommand(state, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Status != "accepted" {
		t.Fatalf("expected accepted result")
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != "audit.note" {
		t.Fatalf("expected event type audit.note, got %s", events[0].EventType)
	}
}

func TestHandleWriteEventRejectNonDM(t *testing.T) {
	state := NewState("room1")
	state.Players["p1"] = Player{UserID: "p1", IsDM: false}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "write_event",
		ActorUserID: "p1",
		Payload:     []byte(`{"event_type":"audit.note","data":{"reason":"x"}}`),
	}

	_, _, err := HandleCommand(state, cmd)
	if err == nil {
		t.Fatalf("expected non-DM write_event to be rejected")
	}
}

// TestStartGameNightActionTypes verifies that night.action.queued events
// include the correct action_type from role definitions.
func TestStartGameNightActionTypes(t *testing.T) {
	state := NewState("room1")
	for i := 1; i <= 7; i++ {
		uid := fmt.Sprintf("p%d", i)
		state.Players[uid] = Player{UserID: uid, SeatNumber: i, Alive: true}
	}
	state.OwnerID = "p1"
	state.Phase = PhaseLobby

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "start_game",
		ActorUserID: "p1",
	}
	events, _, err := HandleCommand(state, cmd)
	if err != nil {
		t.Fatalf("start_game failed: %v", err)
	}

	for _, e := range events {
		if e.EventType != "night.action.queued" {
			continue
		}
		var payload map[string]string
		_ = json.Unmarshal(e.Payload, &payload)
		roleID := payload["role_id"]
		actionType := payload["action_type"]

		if actionType == "" {
			t.Errorf("night.action.queued for %s missing action_type", roleID)
			continue
		}
		role := game.GetRoleByID(roleID)
		if role == nil {
			t.Errorf("unknown role %s", roleID)
			continue
		}
		expected := string(role.FirstNightActionType)
		if actionType != expected {
			t.Errorf("role %s: got action_type %q, want %q", roleID, actionType, expected)
		}
	}
}

// TestImpFirstNightAutoComplete verifies that the imp (no_action on first night)
// gets an automatic night.action.completed event.
func TestImpFirstNightAutoComplete(t *testing.T) {
	state := NewState("room1")
	for i := 1; i <= 7; i++ {
		uid := fmt.Sprintf("p%d", i)
		state.Players[uid] = Player{UserID: uid, SeatNumber: i, Alive: true}
	}
	state.OwnerID = "p1"
	state.Phase = PhaseLobby

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "start_game",
		ActorUserID: "p1",
	}
	events, _, err := HandleCommand(state, cmd)
	if err != nil {
		t.Fatalf("start_game failed: %v", err)
	}

	impQueued := false
	impCompleted := false
	for _, e := range events {
		var payload map[string]string
		_ = json.Unmarshal(e.Payload, &payload)
		if e.EventType == "night.action.queued" && payload["role_id"] == "imp" {
			impQueued = true
			if payload["action_type"] != "no_action" {
				t.Errorf("imp first night action_type = %q, want no_action", payload["action_type"])
			}
		}
		if e.EventType == "night.action.completed" && payload["role_id"] == "imp" {
			impCompleted = true
		}
	}
	if !impQueued {
		t.Error("imp night.action.queued event not found")
	}
	if !impCompleted {
		t.Error("imp night.action.completed auto-event not found for no_action")
	}
}

// TestReduceNightActionActionType verifies that Reduce correctly parses
// action_type from night.action.queued events.
func TestReduceNightActionActionType(t *testing.T) {
	s := NewState("room1")
	s.Reduce(EventPayload{
		Seq:   1,
		Type:  "night.action.queued",
		Actor: "system",
		Payload: map[string]string{
			"user_id":     "p1",
			"role_id":     "fortuneteller",
			"order":       "1",
			"action_type": "select_two",
		},
	})

	if len(s.NightActions) != 1 {
		t.Fatalf("expected 1 night action, got %d", len(s.NightActions))
	}
	if s.NightActions[0].ActionType != "select_two" {
		t.Errorf("action_type = %q, want select_two", s.NightActions[0].ActionType)
	}
}

// toEventPayload converts a types.Event to an EventPayload for state reduction.
func toEventPayload(e types.Event) EventPayload {
	var payload map[string]string
	_ = json.Unmarshal(e.Payload, &payload)
	return EventPayload{
		Seq:     e.Seq,
		Type:    e.EventType,
		Actor:   e.ActorUserID,
		Payload: payload,
	}
}

func TestStartGameWithCustomRoles(t *testing.T) {
	state := NewState("room1")
	for i := 0; i < 7; i++ {
		uid := fmt.Sprintf("user-%d", i)
		state.Players[uid] = Player{UserID: uid, SeatNumber: i + 1, Alive: true}
	}
	state.OwnerID = "user-0"
	state.Phase = PhaseLobby

	// Create start_game command with custom_roles
	customRoles := []string{"imp", "scarletwoman", "washerwoman", "empath", "chef", "butler", "saint"}
	rolesJSON, _ := json.Marshal(customRoles)
	payload, _ := json.Marshal(map[string]string{"custom_roles": string(rolesJSON)})

	cmd := types.CommandEnvelope{
		CommandID:   "start-1",
		RoomID:      "room1",
		Type:        "start_game",
		ActorUserID: "user-0",
		Payload:     payload,
	}
	events, _, err := HandleCommand(state, cmd)
	if err != nil {
		t.Fatalf("start_game with custom_roles failed: %v", err)
	}

	// Verify assigned roles match custom_roles
	assignedRoles := make(map[string]bool)
	for _, e := range events {
		if e.EventType == "role.assigned" {
			var p map[string]string
			_ = json.Unmarshal(e.Payload, &p)
			assignedRoles[p["role"]] = true
		}
	}
	for _, roleID := range customRoles {
		if !assignedRoles[roleID] {
			t.Errorf("custom role %s not found in assignments", roleID)
		}
	}
}
