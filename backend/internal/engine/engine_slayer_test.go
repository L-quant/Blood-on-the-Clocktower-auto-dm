package engine

import (
	"encoding/json"
	"testing"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestHandleSlayerShotAllowedInAllDaytimePhases(t *testing.T) {
	daytimePhases := []Phase{PhaseDay, PhaseNomination, PhaseVoting}

	for _, phase := range daytimePhases {
		t.Run(string(phase), func(t *testing.T) {
			state := NewState("room-1")
			state.Phase = phase
			state.DemonID = "target"
			state.Players["slayer"] = Player{UserID: "slayer", TrueRole: "slayer", Alive: true}
			state.Players["target"] = Player{UserID: "target", TrueRole: "imp", Alive: true, SeatNumber: 2}

			payload, err := json.Marshal(map[string]string{"target": "target"})
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}

			events, _, err := handleSlayerShot(state, types.CommandEnvelope{
				CommandID:   "cmd-1",
				ActorUserID: "slayer",
				Payload:     payload,
			})
			if err != nil {
				t.Fatalf("handleSlayerShot returned error: %v", err)
			}

			hasSlayerShot := false
			hasPlayerDied := false
			for _, event := range events {
				if event.EventType == "slayer.shot" {
					hasSlayerShot = true
				}
				if event.EventType == "player.died" {
					hasPlayerDied = true
				}
			}
			if !hasSlayerShot {
				t.Fatal("expected slayer.shot event")
			}
			if !hasPlayerDied {
				t.Fatal("expected player.died event")
			}
		})
	}
}

func TestHandleSlayerShotAllowsFalseClaimWithoutKill(t *testing.T) {
	state := NewState("room-1")
	state.Phase = PhaseDay
	state.DemonID = "target"
	state.Players["faker"] = Player{UserID: "faker", TrueRole: "washerwoman", Alive: true, SeatNumber: 1}
	state.Players["target"] = Player{UserID: "target", TrueRole: "imp", Alive: true, SeatNumber: 2}

	payload, err := json.Marshal(map[string]string{"target": "target"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	events, _, err := handleSlayerShot(state, types.CommandEnvelope{
		CommandID:   "cmd-1",
		ActorUserID: "faker",
		Payload:     payload,
	})
	if err != nil {
		t.Fatalf("expected false slayer claim to resolve, got error: %v", err)
	}

	if !hasTestEventType(events, "slayer.shot") {
		t.Fatal("expected slayer.shot event for false claim")
	}
	if hasTestEventType(events, "player.died") {
		t.Fatal("expected false slayer claim to have no kill effect")
	}
	if hasTestEventType(events, "reminder.added") {
		t.Fatal("expected false slayer claim not to consume a real slayer ability")
	}
}

func TestHandleSlayerShotConsumedButIneffectiveWhenPoisoned(t *testing.T) {
	state := NewState("room-1")
	state.Phase = PhaseVoting
	state.DemonID = "target"
	state.Players["slayer"] = Player{UserID: "slayer", TrueRole: "slayer", Alive: true, IsPoisoned: true}
	state.Players["target"] = Player{UserID: "target", TrueRole: "imp", Alive: true, SeatNumber: 2}

	payload, err := json.Marshal(map[string]string{"target": "target"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	events, _, err := handleSlayerShot(state, types.CommandEnvelope{
		CommandID:   "cmd-1",
		ActorUserID: "slayer",
		Payload:     payload,
	})
	if err != nil {
		t.Fatalf("expected poisoned slayer shot to resolve, got error: %v", err)
	}

	hasSlayerShot := false
	hasReminder := false
	hasPlayerDied := false
	for _, event := range events {
		if event.EventType == "slayer.shot" {
			hasSlayerShot = true
		}
		if event.EventType == "reminder.added" {
			hasReminder = true
		}
		if event.EventType == "player.died" {
			hasPlayerDied = true
		}
	}
	if !hasSlayerShot {
		t.Fatal("expected slayer.shot event")
	}
	if !hasReminder {
		t.Fatal("expected no_ability reminder to consume the skill")
	}
	if hasPlayerDied {
		t.Fatal("expected poisoned slayer shot to have no kill effect")
	}
}

func TestHandleSlayerShotTriggersNightAfterScarletWomanTakeover(t *testing.T) {
	state := NewState("room-1")
	state.Phase = PhaseDay
	state.DemonID = "demon"
	state.MinionIDs = []string{"scarlet"}
	state.Players["slayer"] = Player{UserID: "slayer", TrueRole: "slayer", Alive: true, SeatNumber: 1}
	state.Players["demon"] = Player{UserID: "demon", TrueRole: "imp", Alive: true, SeatNumber: 2, Team: "evil"}
	state.Players["scarlet"] = Player{UserID: "scarlet", TrueRole: "scarletwoman", Alive: true, SeatNumber: 3, Team: "evil"}
	state.Players["town1"] = Player{UserID: "town1", TrueRole: "washerwoman", Alive: true, SeatNumber: 4, Team: "good"}
	state.Players["town2"] = Player{UserID: "town2", TrueRole: "chef", Alive: true, SeatNumber: 5, Team: "good"}
	state.Players["town3"] = Player{UserID: "town3", TrueRole: "librarian", Alive: true, SeatNumber: 6, Team: "good"}
	state.SeatOrder = []string{"slayer", "demon", "scarlet", "town1", "town2", "town3"}

	payload, err := json.Marshal(map[string]string{"target": "demon"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	events, _, err := handleSlayerShot(state, types.CommandEnvelope{
		CommandID:   "cmd-1",
		ActorUserID: "slayer",
		Payload:     payload,
	})
	if err != nil {
		t.Fatalf("handleSlayerShot returned error: %v", err)
	}

	if !hasTestEventType(events, "slayer.shot") {
		t.Fatal("expected slayer.shot event")
	}
	if !hasTestEventType(events, "player.died") {
		t.Fatal("expected player.died event")
	}
	if !hasTestEventType(events, "demon.changed") {
		t.Fatal("expected demon.changed event after scarlet woman takeover")
	}
	if !hasTestEventType(events, "phase.night") {
		t.Fatal("expected direct transition to night after scarlet woman takeover")
	}
	if hasTestEventType(events, "game.ended") {
		t.Fatal("expected no immediate game end when scarlet woman takes over")
	}
}

func hasTestEventType(events []types.Event, eventType string) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}
