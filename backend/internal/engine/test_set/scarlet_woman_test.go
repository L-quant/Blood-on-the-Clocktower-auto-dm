package engine

import (
	"encoding/json"
	"testing"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// ensureTestState creates a clean state for testing
func ensureTestState() State {
	s := NewState("room_test")
	
	// State definition (checked state.go): Players map[string]Player  (VALUE, not pointer)
	
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

// TestScarletWomanPriority ensures Scarlet Woman is chosen over other minions during Starpass
func TestScarletWomanPriority(t *testing.T) {
	state := ensureTestState()
	
	payload := map[string]interface{}{
		"target":      "demon",  // Self-kill
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
	
	// Check for 'demon.changed' event
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

// TestRandomMinionFallback ensures that if SW is dead or absent, another minion is picked
func TestRandomMinionFallback(t *testing.T) {
	state := ensureTestState()
	// Kill the Scarlet Woman effectively removing her from candidates
	// Players is map[string]Player (struct value), so we must rewrite the entry
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
			// Since SW is dead, only 'minion' is alive candidate
			if newDemon != "minion" {
				t.Errorf("Starpass failed fallback. Expected 'minion', got '%s'", newDemon)
			}
		}
	}
	
	if !found {
		t.Error("Starpass did not trigger a 'demon.changed' event when SW was dead")
	}
}
