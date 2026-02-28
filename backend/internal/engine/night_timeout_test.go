package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestCompleteRemainingNightActionsInfoOnly(t *testing.T) {
	state := NewState("room1")
	state.NightActions = []NightAction{
		{UserID: "p1", RoleID: "empath", ActionType: "info", Completed: false},
		{UserID: "p2", RoleID: "chef", ActionType: "info", Completed: true},
	}
	cmd := types.CommandEnvelope{
		CommandID: uuid.NewString(),
		RoomID:    "room1",
		Type:      "advance_phase",
	}

	events, hasEvilPending := CompleteRemainingNightActions(state, cmd)
	if hasEvilPending {
		t.Error("expected no evil pending for info-only actions")
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 timed_out event, got %d", len(events))
	}
	var payload map[string]string
	_ = json.Unmarshal(events[0].Payload, &payload)
	if payload["result"] != "timed_out" {
		t.Errorf("result = %q, want timed_out", payload["result"])
	}
	if payload["user_id"] != "p1" {
		t.Errorf("user_id = %q, want p1", payload["user_id"])
	}
}

func TestCompleteRemainingNightActionsSkipsEvil(t *testing.T) {
	state := NewState("room1")
	state.NightActions = []NightAction{
		{UserID: "p1", RoleID: "imp", ActionType: "select_one", Completed: false},
		{UserID: "p2", RoleID: "poisoner", ActionType: "select_one", Completed: false},
		{UserID: "p3", RoleID: "empath", ActionType: "info", Completed: false},
	}
	cmd := types.CommandEnvelope{
		CommandID: uuid.NewString(),
		RoomID:    "room1",
		Type:      "advance_phase",
	}

	events, hasEvilPending := CompleteRemainingNightActions(state, cmd)
	if !hasEvilPending {
		t.Error("expected evil pending for imp/poisoner")
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event (empath only), got %d", len(events))
	}
	var payload map[string]string
	_ = json.Unmarshal(events[0].Payload, &payload)
	if payload["role_id"] != "empath" {
		t.Errorf("expected empath to be timed out, got %s", payload["role_id"])
	}
}

func TestCompleteRemainingNightActionsAllComplete(t *testing.T) {
	state := NewState("room1")
	state.NightActions = []NightAction{
		{UserID: "p1", RoleID: "imp", ActionType: "select_one", Completed: true},
		{UserID: "p2", RoleID: "empath", ActionType: "info", Completed: true},
	}
	cmd := types.CommandEnvelope{
		CommandID: uuid.NewString(),
		RoomID:    "room1",
		Type:      "advance_phase",
	}

	events, hasEvilPending := CompleteRemainingNightActions(state, cmd)
	if hasEvilPending {
		t.Error("expected no evil pending when all complete")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestIsEvilCriticalAction(t *testing.T) {
	tests := []struct {
		name     string
		action   NightAction
		expected bool
	}{
		{"imp select", NightAction{RoleID: "imp", ActionType: "select_one"}, true},
		{"poisoner select", NightAction{RoleID: "poisoner", ActionType: "select_one"}, true},
		{"imp info (first night)", NightAction{RoleID: "imp", ActionType: "no_action"}, false},
		{"empath info", NightAction{RoleID: "empath", ActionType: "info"}, false},
		{"spy info", NightAction{RoleID: "spy", ActionType: "info"}, false},
		{"monk select", NightAction{RoleID: "monk", ActionType: "select_one"}, false},
		{"empty action type", NightAction{RoleID: "imp", ActionType: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isEvilCriticalAction(tt.action)
			if got != tt.expected {
				t.Errorf("isEvilCriticalAction(%+v) = %v, want %v", tt.action, got, tt.expected)
			}
		})
	}
}
