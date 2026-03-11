package engine

import (
	"encoding/json"
	"testing"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestHandleAbilityDeliversEmpathInfoWhenLastNightActionCompletes(t *testing.T) {
	state := NewState("room-1")
	state.Phase = PhaseFirstNight
	state.NightCount = 1
	state.SeatOrder = []string{"good-left", "empath", "evil-right"}
	state.Players["good-left"] = Player{UserID: "good-left", TrueRole: "washerwoman", Team: "good", Alive: true, SeatNumber: 1}
	state.Players["empath"] = Player{UserID: "empath", TrueRole: "empath", Team: "good", Alive: true, SeatNumber: 2}
	state.Players["evil-right"] = Player{UserID: "evil-right", TrueRole: "imp", Team: "evil", Alive: true, SeatNumber: 3}
	state.NightActions = []NightAction{{
		UserID:     "empath",
		RoleID:     "empath",
		Order:      36,
		ActionType: "info",
	}}

	events, _, err := handleAbility(state, types.CommandEnvelope{
		CommandID:   "cmd-1",
		ActorUserID: "empath",
	})
	if err != nil {
		t.Fatalf("handleAbility returned error: %v", err)
	}

	infoPayload := findEventPayload(t, events, "night.info")
	if infoPayload["user_id"] != "empath" {
		t.Fatalf("expected empath night.info for empath, got %q", infoPayload["user_id"])
	}
	if infoPayload["role_id"] != "empath" {
		t.Fatalf("expected role_id empath, got %q", infoPayload["role_id"])
	}
	if infoPayload["info_type"] != "empath" {
		t.Fatalf("expected info_type empath, got %q", infoPayload["info_type"])
	}

	var content map[string]int
	if err := json.Unmarshal([]byte(infoPayload["content"]), &content); err != nil {
		t.Fatalf("unmarshal night.info content: %v", err)
	}
	if content["evil_neighbors"] != 1 {
		t.Fatalf("expected empath to see 1 evil neighbor, got %d", content["evil_neighbors"])
	}

	if !hasTestEventType(events, "phase.day") {
		t.Fatal("expected phase.day event after final night action")
	}

	dayPayload := findEventPayload(t, events, "phase.day")
	if dayPayload["night_deaths"] != "[]" {
		t.Fatalf("expected first-night phase.day payload to report no deaths, got %q", dayPayload["night_deaths"])
	}
}

func TestHandleAbilityIncludesNightDeathsInPhaseDayPayload(t *testing.T) {
	state := NewState("room-2")
	state.Phase = PhaseNight
	state.NightCount = 2
	state.SeatOrder = []string{"imp", "target", "monk"}
	state.Players["imp"] = Player{UserID: "imp", TrueRole: "imp", Team: "evil", Alive: true, SeatNumber: 1}
	state.Players["target"] = Player{UserID: "target", TrueRole: "washerwoman", Team: "good", Alive: true, SeatNumber: 2}
	state.Players["monk"] = Player{UserID: "monk", TrueRole: "monk", Team: "good", Alive: true, SeatNumber: 3}
	state.NightActions = []NightAction{{
		UserID:     "imp",
		RoleID:     "imp",
		Order:      100,
		ActionType: "select_one",
	}}

	events, _, err := handleAbility(state, types.CommandEnvelope{
		CommandID:   "cmd-2",
		ActorUserID: "imp",
		RoomID:      state.RoomID,
		Payload:     []byte(`{"targets":"[\"target\"]"}`),
	})
	if err != nil {
		t.Fatalf("handleAbility returned error: %v", err)
	}

	dayPayload := findEventPayload(t, events, "phase.day")
	if dayPayload["night_deaths"] != "[2]" {
		t.Fatalf("expected phase.day payload to include seat 2 death, got %q", dayPayload["night_deaths"])
	}
}

func findEventPayload(t *testing.T, events []types.Event, eventType string) map[string]string {
	t.Helper()
	for _, event := range events {
		if event.EventType != eventType {
			continue
		}
		var payload map[string]string
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatalf("unmarshal %s payload: %v", eventType, err)
		}
		return payload
	}
	t.Fatalf("expected %s event", eventType)
	return nil
}
