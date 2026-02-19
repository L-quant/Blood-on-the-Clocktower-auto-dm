package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

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
