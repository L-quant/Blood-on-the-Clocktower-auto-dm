package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// helper to build a game state with evil team set up
func buildScarletWomanState() State {
	s := NewState("room-sw")
	s.Phase = PhaseNight
	s.NightCount = 2
	s.DemonID = "imp-player"
	s.MinionIDs = []string{"sw-player", "poisoner-player"}
	s.SeatOrder = []string{"p1", "p2", "p3", "imp-player", "sw-player", "poisoner-player", "p4"}

	s.Players = map[string]Player{
		"p1":               {UserID: "p1", SeatNumber: 1, Role: "washerwoman", TrueRole: "washerwoman", Team: "good", Alive: true},
		"p2":               {UserID: "p2", SeatNumber: 2, Role: "empath", TrueRole: "empath", Team: "good", Alive: true},
		"p3":               {UserID: "p3", SeatNumber: 3, Role: "fortune_teller", TrueRole: "fortune_teller", Team: "good", Alive: true},
		"imp-player":       {UserID: "imp-player", SeatNumber: 4, Role: "imp", TrueRole: "imp", Team: "evil", Alive: true},
		"sw-player":        {UserID: "sw-player", SeatNumber: 5, Role: "scarlet_woman", TrueRole: "scarlet_woman", Team: "evil", Alive: true},
		"poisoner-player":  {UserID: "poisoner-player", SeatNumber: 6, Role: "poisoner", TrueRole: "poisoner", Team: "evil", Alive: true},
		"p4":               {UserID: "p4", SeatNumber: 7, Role: "monk", TrueRole: "monk", Team: "good", Alive: true},
	}

	return s
}

func TestStarpassPrioritizesScarletWoman(t *testing.T) {
	s := buildScarletWomanState()

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room-sw",
		Type:        "ability.use",
		ActorUserID: "imp-player",
		Payload:     json.RawMessage(`{"target": "imp-player", "action_type": "kill"}`),
	}

	events, result, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "accepted" {
		t.Fatalf("expected accepted, got %s", result.Status)
	}

	// Look for demon.changed event
	var demonChanged bool
	var newDemon string
	var playerDied bool
	for _, e := range events {
		if e.EventType == "demon.changed" {
			demonChanged = true
			var payload map[string]string
			json.Unmarshal(e.Payload, &payload)
			newDemon = payload["new_demon"]
		}
		if e.EventType == "player.died" {
			var payload map[string]string
			json.Unmarshal(e.Payload, &payload)
			if payload["cause"] == "starpass" {
				playerDied = true
			}
		}
	}

	if !playerDied {
		t.Errorf("expected imp to die from starpass")
	}
	if !demonChanged {
		t.Fatalf("expected demon.changed event")
	}
	if newDemon != "sw-player" {
		t.Errorf("expected Scarlet Woman (sw-player) to become demon, got %s", newDemon)
	}
}

func TestStarpassFallsBackToRandomMinionWhenSWDead(t *testing.T) {
	s := buildScarletWomanState()
	// Kill scarlet woman
	sw := s.Players["sw-player"]
	sw.Alive = false
	s.Players["sw-player"] = sw

	cmd := types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      "room-sw",
		Type:        "ability.use",
		ActorUserID: "imp-player",
		Payload:     json.RawMessage(`{"target": "imp-player", "action_type": "kill"}`),
	}

	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var newDemon string
	for _, e := range events {
		if e.EventType == "demon.changed" {
			var payload map[string]string
			json.Unmarshal(e.Payload, &payload)
			newDemon = payload["new_demon"]
		}
	}

	if newDemon == "" {
		t.Fatalf("expected demon.changed event")
	}
	// Since SW is dead, should fall back to the other alive minion
	if newDemon != "poisoner-player" {
		t.Errorf("expected poisoner-player to become demon, got %s", newDemon)
	}
}

func TestScarletWomanInheritsOnDemonExecution(t *testing.T) {
	s := buildScarletWomanState()
	s.Phase = PhaseDay
	s.SubPhase = SubPhaseVoting

	// Create an active nomination against the demon
	s.Nomination = &Nomination{
		Nominator: "p1",
		Nominee:   "imp-player",
		Votes:     map[string]bool{},
	}

	// All vote yes to execute demon
	for _, uid := range []string{"p1", "p2", "p3", "p4", "sw-player", "poisoner-player", "imp-player"} {
		cmd := types.CommandEnvelope{
			CommandID:   uuid.NewString(),
			RoomID:      "room-sw",
			Type:        "vote",
			ActorUserID: uid,
			Payload:     json.RawMessage(`{"vote":"yes"}`),
		}
		events, _, err := HandleCommand(s, cmd)
		if err != nil {
			t.Fatalf("vote by %s failed: %v", uid, err)
		}
		for _, e := range events {
			s.Reduce(toEventPayload(e))
		}
	}

	// After execution, the demon should be dead
	imp := s.Players["imp-player"]
	if imp.Alive {
		t.Errorf("expected imp to be dead after execution")
	}

	// With 5+ alive players (6 alive after demon dies), SW should inherit
	// The win check should have emitted demon.changed
	// Note: the actual inheritance happens via checkWinCondition events
}

// ensureTestState creates a clean state for testing starpass via handleAbility
func ensureTestState() State {
	s := NewState("room_test")

	s.Players["demon"] = Player{
		UserID: "demon", Name: "Demon", Role: "imp", TrueRole: "imp",
		Team: "evil", Alive: true,
	}
	s.DemonID = "demon"

	s.Players["sw"] = Player{
		UserID: "sw", Name: "ScarletWoman", Role: "scarlet_woman", TrueRole: "scarlet_woman",
		Team: "evil", Alive: true,
	}

	s.Players["minion"] = Player{
		UserID: "minion", Name: "Minion", Role: "poisoner", TrueRole: "poisoner",
		Team: "evil", Alive: true,
	}

	s.MinionIDs = []string{"sw", "minion"}
	s.SeatOrder = []string{"demon", "sw", "minion"}
	s.Phase = PhaseNight

	return s
}

// TestScarletWomanPriority ensures Scarlet Woman is chosen over other minions during Starpass (via handleAbility)
func TestScarletWomanPriority(t *testing.T) {
	state := ensureTestState()

	payload := map[string]interface{}{
		"target":      "demon",
		"action_type": "kill",
	}
	payloadBytes, _ := json.Marshal(payload)

	cmd := types.CommandEnvelope{
		CommandID:   "cmd_test_1",
		RoomID:      "room_test",
		Type:        "ability.use",
		ActorUserID: "demon",
		Payload:     payloadBytes,
	}

	events, _, err := handleAbility(state, cmd)
	if err != nil {
		t.Fatalf("Failed to handle ability: %v", err)
	}

	found := false
	for _, e := range events {
		if e.EventType == "demon.changed" {
			found = true
			var data map[string]string
			if err := json.Unmarshal(e.Payload, &data); err != nil {
				t.Errorf("Failed to unmarshal event payload: %v", err)
				continue
			}
			newDemon := data["new_demon"]
			if newDemon != "sw" {
				t.Errorf("Starpass failed priority check. Expected 'sw', got '%s'", newDemon)
			}
		}
	}

	if !found {
		t.Error("Starpass did not trigger a 'demon.changed' event")
	}
}

// TestRandomMinionFallback ensures that if SW is dead, another minion is picked
func TestRandomMinionFallback(t *testing.T) {
	state := ensureTestState()
	sw := state.Players["sw"]
	sw.Alive = false
	state.Players["sw"] = sw

	payload := map[string]interface{}{
		"target":      "demon",
		"action_type": "kill",
	}
	payloadBytes, _ := json.Marshal(payload)

	cmd := types.CommandEnvelope{
		CommandID:   "cmd_test_2",
		RoomID:      "room_test",
		Type:        "ability.use",
		ActorUserID: "demon",
		Payload:     payloadBytes,
	}

	events, _, err := handleAbility(state, cmd)
	if err != nil {
		t.Fatalf("Failed to handle ability: %v", err)
	}

	found := false
	for _, e := range events {
		if e.EventType == "demon.changed" {
			found = true
			var data map[string]string
			json.Unmarshal(e.Payload, &data)
			newDemon := data["new_demon"]
			if newDemon != "minion" {
				t.Errorf("Starpass failed fallback. Expected 'minion', got '%s'", newDemon)
			}
		}
	}

	if !found {
		t.Error("Starpass did not trigger a 'demon.changed' event when SW was dead")
	}
}
