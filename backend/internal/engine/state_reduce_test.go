package engine

import "testing"

func TestReduceGameRecapStoresSummary(t *testing.T) {
	state := NewState("room-1")
	state.Reduce(EventPayload{
		Seq:  1,
		Type: "game.recap",
		Payload: map[string]string{
			"summary": "good wins recap",
		},
	})

	if state.GameRecap != "good wins recap" {
		t.Fatalf("expected game recap to be stored, got %q", state.GameRecap)
	}
}
