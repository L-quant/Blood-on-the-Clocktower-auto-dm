package engine

import (
	"testing"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func TestHandleEndDefenseRequiresBothSides(t *testing.T) {
	state := NewState("room")
	state.Phase = PhaseNomination
	state.SubPhase = SubPhaseDefense
	state.SeatOrder = []string{"nominator", "nominee"}
	state.Players["nominator"] = Player{UserID: "nominator", SeatNumber: 1, Alive: true}
	state.Players["nominee"] = Player{UserID: "nominee", SeatNumber: 2, Alive: true}
	state.Nomination = &Nomination{
		Nominator: "nominator",
		Nominee:   "nominee",
	}

	cmd1 := types.CommandEnvelope{ActorUserID: "nominator", CommandID: "c1"}
	events1, _, err := handleEndDefense(state, cmd1)
	if err != nil {
		t.Fatalf("first end_defense returned err: %v", err)
	}
	if hasTestEventType(events1, "defense.ended") {
		t.Fatal("first end_defense should not end defense before nominee confirms")
	}
	if !hasTestEventType(events1, "defense.progress") {
		t.Fatal("first end_defense should emit defense.progress")
	}

	applyEventsToState(&state, events1)

	cmd2 := types.CommandEnvelope{ActorUserID: "nominee", CommandID: "c2"}
	events2, _, err := handleEndDefense(state, cmd2)
	if err != nil {
		t.Fatalf("second end_defense returned err: %v", err)
	}
	if !hasTestEventType(events2, "defense.ended") {
		t.Fatal("second end_defense should end defense after both sides confirmed")
	}
}
