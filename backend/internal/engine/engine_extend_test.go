package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestExtendTimeDuringDiscussion(t *testing.T) {
	s := NewState("room1")
	s.Phase = PhaseDay
	s.SubPhase = SubPhaseDiscussion
	s.Players["p1"] = Player{UserID: "p1", Alive: true}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "extend_time",
		ActorUserID: "p1",
	}

	events, result, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("extend_time failed: %v", err)
	}
	if result.Status != "accepted" {
		t.Errorf("expected accepted, got %s", result.Status)
	}
	if len(events) != 1 || events[0].EventType != "time.extended" {
		t.Fatalf("expected 1 time.extended event, got %d events", len(events))
	}

	var payload map[string]string
	_ = json.Unmarshal(events[0].Payload, &payload)
	if payload["extensions_remaining"] != "2" {
		t.Errorf("extensions_remaining = %s, want 2", payload["extensions_remaining"])
	}
}

func TestExtendTimeMaxReached(t *testing.T) {
	s := NewState("room1")
	s.Phase = PhaseDay
	s.SubPhase = SubPhaseDiscussion
	s.ExtensionsUsed = 3 // max is 3
	s.Players["p1"] = Player{UserID: "p1", Alive: true}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "extend_time",
		ActorUserID: "p1",
	}

	_, _, err := HandleCommand(s, cmd)
	if err == nil {
		t.Fatal("expected error when max extensions reached")
	}
}

func TestExtendTimeWrongPhase(t *testing.T) {
	s := NewState("room1")
	s.Phase = PhaseNight
	s.Players["p1"] = Player{UserID: "p1", Alive: true}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "extend_time",
		ActorUserID: "p1",
	}

	_, _, err := HandleCommand(s, cmd)
	if err == nil {
		t.Fatal("expected error when not in day discussion phase")
	}
}

func TestReduceTimeExtended(t *testing.T) {
	s := NewState("room1")
	s.Phase = PhaseDay
	s.SubPhase = SubPhaseDiscussion
	s.ExtensionsUsed = 1

	s.Reduce(EventPayload{
		Seq:   1,
		Type:  "time.extended",
		Actor: "p1",
		Payload: map[string]string{
			"deadline":             "9999999999999",
			"extensions_remaining": "1",
		},
	})

	if s.ExtensionsUsed != 2 {
		t.Errorf("ExtensionsUsed = %d, want 2", s.ExtensionsUsed)
	}
	if s.PhaseEndsAt != 9999999999999 {
		t.Errorf("PhaseEndsAt = %d, want 9999999999999", s.PhaseEndsAt)
	}
}
