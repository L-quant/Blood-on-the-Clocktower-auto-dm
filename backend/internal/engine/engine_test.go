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
	if s.Phase != PhaseDay {
		t.Errorf("expected day phase, got %s", s.Phase)
	}
	s.Reduce(EventPayload{Seq: 2, Type: "phase.night", Actor: "system", Payload: nil})
	if s.Phase != PhaseNight {
		t.Errorf("expected night phase, got %s", s.Phase)
	}
}

func TestVoteResolution(t *testing.T) {
	s := NewState("room1")
	s.Players["a"] = Player{UserID: "a", Alive: true}
	s.Players["b"] = Player{UserID: "b", Alive: true}
	s.Players["c"] = Player{UserID: "c", Alive: true}
	s.Phase = PhaseDay
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
	// Second vote to reach majority (2/3)
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
	resolved := false
	for _, e := range events {
		if e.EventType == "execution.resolved" {
			resolved = true
		}
	}
	if !resolved {
		t.Fatalf("expected execution resolved with majority vote")
	}
}

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
