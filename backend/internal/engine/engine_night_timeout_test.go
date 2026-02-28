package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestNightTimeoutAllComplete(t *testing.T) {
	s := NewState("room1")
	s.Phase = PhaseNight
	s.NightActions = []NightAction{
		{UserID: "p1", RoleID: "empath", ActionType: "info", Completed: true},
		{UserID: "p2", RoleID: "imp", ActionType: "select_one", Completed: true},
	}
	s.Players["p1"] = Player{UserID: "p1", Alive: true}
	s.Players["p2"] = Player{UserID: "p2", Alive: true}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "night_timeout",
		ActorUserID: "autodm",
	}

	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("night_timeout failed: %v", err)
	}

	hasDayPhase := false
	for _, e := range events {
		if e.EventType == "phase.day" {
			hasDayPhase = true
		}
	}
	if !hasDayPhase {
		t.Error("expected phase.day when all actions complete")
	}
}

func TestNightTimeoutEvilPending(t *testing.T) {
	s := NewState("room1")
	s.Phase = PhaseNight
	s.NightActions = []NightAction{
		{UserID: "p1", RoleID: "empath", ActionType: "info", Completed: false},
		{UserID: "p2", RoleID: "imp", ActionType: "select_one", Completed: false},
	}
	s.Players["p1"] = Player{UserID: "p1", Alive: true}
	s.Players["p2"] = Player{UserID: "p2", Alive: true}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "night_timeout",
		ActorUserID: "autodm",
	}

	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("night_timeout failed: %v", err)
	}

	hasTimedOut := false
	hasReminder := false
	hasDayPhase := false
	for _, e := range events {
		var payload map[string]string
		_ = json.Unmarshal(e.Payload, &payload)
		if e.EventType == "night.action.completed" && payload["result"] == "timed_out" {
			hasTimedOut = true
			if payload["role_id"] != "empath" {
				t.Errorf("timed_out should be for empath, got %s", payload["role_id"])
			}
		}
		if e.EventType == "action.reminder" {
			hasReminder = true
			if payload["role_id"] != "imp" {
				t.Errorf("reminder should be for imp, got %s", payload["role_id"])
			}
		}
		if e.EventType == "phase.day" {
			hasDayPhase = true
		}
	}

	if !hasTimedOut {
		t.Error("expected empath to be timed out")
	}
	if !hasReminder {
		t.Error("expected action.reminder for imp")
	}
	if hasDayPhase {
		t.Error("should NOT advance to day when evil pending")
	}
}

func TestNightTimeoutWrongPhase(t *testing.T) {
	s := NewState("room1")
	s.Phase = PhaseDay
	s.Players["p1"] = Player{UserID: "p1", Alive: true}

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room1",
		Type:        "night_timeout",
		ActorUserID: "autodm",
	}

	_, _, err := HandleCommand(s, cmd)
	if err == nil {
		t.Fatal("expected error when not in night phase")
	}
}
