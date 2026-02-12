package engine

import (
	"encoding/json"
	"testing"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestDemonDeath_WinCondition(t *testing.T) {
	// Setup state: Demon (Imp), Slayer. No Scarlet Woman.
	state := NewState("room_win_check")
	state.Phase = PhaseDay // Slayer shoots at Day

	state.Players["demon"] = Player{
		UserID: "demon", Name: "Demon", Role: "imp", TrueRole: "imp",
		Team: "evil", Alive: true,
	}
	state.DemonID = "demon"

	state.Players["slayer"] = Player{
		UserID: "slayer", Name: "Slayer", Role: "slayer", TrueRole: "slayer",
		Team: "good", Alive: true,
		Reminders: []string{}, // No "Used Ability" reminder
	}

	state.SeatOrder = []string{"demon", "slayer"}

	// Slayer shoots Demon
	payload := map[string]string{
		"target": "demon",
	}
	payloadBytes, _ := json.Marshal(payload)

	cmd := types.CommandEnvelope{
		CommandID:   "cmd_slayer",
		RoomID:      "room_win_check",
		Type:        "slayer_shot",
		ActorUserID: "slayer",
		Payload:     payloadBytes,
	}

	// Invoke handler directly to avoid wrapper overhead, 
	// or use HandleCommand if we want full integration logic.
	// Since handleSlayerShot is private but in same package, we can call it if testing package engine.
	// However, engine_test.go is usually package engine_test if blackbox, 
	// but here files are package engine. So we can access handleSlayerShot.
	
	events, _, err := handleSlayerShot(state, cmd)
	if err != nil {
		t.Fatalf("handleSlayerShot failed: %v", err)
	}

	// Check results
	// 1. demon died
	// 2. game ended
	
	demonDied := false
	gameEnded := false

	for _, e := range events {
		if e.EventType == "player.died" {
			var data map[string]string
			json.Unmarshal(e.Payload, &data)
			if data["user_id"] == "demon" {
				demonDied = true
			}
		}
		if e.EventType == "game.ended" {
			gameEnded = true
			var data map[string]string
			json.Unmarshal(e.Payload, &data)
			if data["winner"] != "good" {
				t.Errorf("Expected winner 'good', got '%s'", data["winner"])
			}
		}
	}

	if !demonDied {
		t.Error("Demon did not die from slayer shot")
	}
	if !gameEnded {
		t.Error("Game did NOT end after demon death (Win condition check failed?)")
	}
}
