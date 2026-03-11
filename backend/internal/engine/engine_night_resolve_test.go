package engine

import (
	"testing"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestResolveNightImpSelfKillTriggersStarpass(t *testing.T) {
	state := NewState("room-1")
	state.Phase = PhaseNight
	state.DemonID = "imp"
	state.MinionIDs = []string{"scarlet"}
	state.SeatOrder = []string{"imp", "scarlet", "town1", "town2", "town3", "town4"}
	state.Players["imp"] = Player{UserID: "imp", TrueRole: "imp", Alive: true, SeatNumber: 1, Team: "evil"}
	state.Players["scarlet"] = Player{UserID: "scarlet", TrueRole: "scarletwoman", Alive: true, SeatNumber: 2, Team: "evil"}
	state.Players["town1"] = Player{UserID: "town1", TrueRole: "washerwoman", Alive: true, SeatNumber: 3, Team: "good"}
	state.Players["town2"] = Player{UserID: "town2", TrueRole: "chef", Alive: true, SeatNumber: 4, Team: "good"}
	state.Players["town3"] = Player{UserID: "town3", TrueRole: "librarian", Alive: true, SeatNumber: 5, Team: "good"}
	state.Players["town4"] = Player{UserID: "town4", TrueRole: "empath", Alive: true, SeatNumber: 6, Team: "good"}
	state.NightActions = []NightAction{{
		UserID:     "imp",
		RoleID:     "imp",
		Completed:  true,
		TargetIDs:  []string{"imp"},
		ActionType: "select_one",
	}}

	events := resolveNight(state, types.CommandEnvelope{CommandID: "cmd-1", ActorUserID: "imp", RoomID: state.RoomID})
	if !hasTestEventType(events, "player.died") {
		t.Fatal("expected imp self-kill to create player.died event")
	}
	if !hasTestEventType(events, "demon.changed") {
		t.Fatal("expected imp self-kill to trigger demon.changed")
	}
}

func TestResolveNightImpSelectingDeadTargetHasNoEffect(t *testing.T) {
	state := NewState("room-1")
	state.Phase = PhaseNight
	state.DemonID = "imp"
	state.Players["imp"] = Player{UserID: "imp", TrueRole: "imp", Alive: true, SeatNumber: 1, Team: "evil"}
	state.Players["dead-good"] = Player{UserID: "dead-good", TrueRole: "chef", Alive: false, SeatNumber: 2, Team: "good"}
	state.NightActions = []NightAction{{
		UserID:     "imp",
		RoleID:     "imp",
		Completed:  true,
		TargetIDs:  []string{"dead-good"},
		ActionType: "select_one",
	}}

	events := resolveNight(state, types.CommandEnvelope{CommandID: "cmd-2", ActorUserID: "imp", RoomID: state.RoomID})
	if hasTestEventType(events, "player.died") {
		t.Fatal("expected dead imp target to behave like a skipped attack")
	}
}

func TestResolveNightPoisonerSelectingDeadTargetHasNoEffect(t *testing.T) {
	state := NewState("room-1")
	state.Phase = PhaseNight
	state.Players["poisoner"] = Player{UserID: "poisoner", TrueRole: "poisoner", Alive: true, SeatNumber: 1, Team: "evil"}
	state.Players["dead-good"] = Player{UserID: "dead-good", TrueRole: "chef", Alive: false, SeatNumber: 2, Team: "good"}
	state.NightActions = []NightAction{{
		UserID:     "poisoner",
		RoleID:     "poisoner",
		Completed:  true,
		TargetIDs:  []string{"dead-good"},
		ActionType: "select_one",
	}}

	events := resolveNight(state, types.CommandEnvelope{CommandID: "cmd-3", ActorUserID: "poisoner", RoomID: state.RoomID})
	if hasTestEventType(events, "player.poisoned") {
		t.Fatal("expected dead poison target to behave like a skipped poison")
	}
}
