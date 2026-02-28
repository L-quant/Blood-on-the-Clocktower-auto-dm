package room

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/engine"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/store"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func unmarshalPayload(raw json.RawMessage, dst interface{}) error {
	return json.Unmarshal(raw, dst)
}

// newTestScheduleActor builds a RoomActor for scheduleTimeouts testing.
func newTestScheduleActor(cfg engine.GameConfig, players int) (*RoomActor, *[]types.CommandEnvelope, *sync.Mutex) {
	var mu sync.Mutex
	var dispatched []types.CommandEnvelope

	state := engine.NewState("test-room")
	state.Config = cfg
	for i := 0; i < players; i++ {
		uid := string(rune('a' + i))
		state.Players[uid] = engine.Player{UserID: uid, Alive: true}
	}

	ra := &RoomActor{
		RoomID: "test-room",
		state:  state,
		logger: zap.NewNop(),
	}
	ra.phaseTimer = NewPhaseTimer(ra.RoomID, func(cmd types.CommandEnvelope) {
		mu.Lock()
		dispatched = append(dispatched, cmd)
		mu.Unlock()
	}, ra.logger)
	return ra, &dispatched, &mu
}

func makeEvent(eventType string) store.StoredEvent {
	return store.StoredEvent{EventType: eventType, RoomID: "test-room"}
}

func TestScheduleTimeouts_NominationResolved(t *testing.T) {
	cfg := engine.GameConfig{NominationPhaseDurationSec: 1}
	ra, dispatched, mu := newTestScheduleActor(cfg, 5)

	ra.scheduleTimeouts([]store.StoredEvent{makeEvent("nomination.resolved")}, cfg)
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	cmd := (*dispatched)[0]
	if cmd.Type != "advance_phase" {
		t.Errorf("expected advance_phase, got %s", cmd.Type)
	}
	var data map[string]string
	if err := unmarshalPayload(cmd.Payload, &data); err != nil {
		t.Fatalf("failed to parse payload: %v", err)
	}
	if data["phase"] != "night" {
		t.Errorf("expected phase=night, got %s", data["phase"])
	}
}

func TestScheduleTimeouts_NominationResolved_ThenGameEnded(t *testing.T) {
	cfg := engine.GameConfig{NominationPhaseDurationSec: 1}
	ra, dispatched, mu := newTestScheduleActor(cfg, 5)

	events := []store.StoredEvent{
		makeEvent("nomination.resolved"),
		makeEvent("game.ended"),
	}
	ra.scheduleTimeouts(events, cfg)
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 0 {
		t.Errorf("game.ended should cancel timer, got %d dispatches", len(*dispatched))
	}
}

func TestScheduleTimeouts_FullCycle(t *testing.T) {
	cfg := engine.GameConfig{
		DiscussionDurationSec:      1,
		DefenseDurationSec:         1,
		VotingDurationSec:          1,
		NominationPhaseDurationSec: 1,
	}
	ra, dispatched, mu := newTestScheduleActor(cfg, 3)

	// Step 1: phase.day → schedules advance_phase(nomination)
	ra.scheduleTimeouts([]store.StoredEvent{makeEvent("phase.day")}, cfg)
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	if len(*dispatched) != 1 || (*dispatched)[0].Type != "advance_phase" {
		t.Fatalf("step 1: expected advance_phase, got %v", *dispatched)
	}
	*dispatched = nil
	mu.Unlock()

	// Step 2: nomination.created → schedules end_defense
	ra.scheduleTimeouts([]store.StoredEvent{makeEvent("nomination.created")}, cfg)
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	if len(*dispatched) != 1 || (*dispatched)[0].Type != "end_defense" {
		t.Fatalf("step 2: expected end_defense, got %v", *dispatched)
	}
	*dispatched = nil
	mu.Unlock()

	// Step 3: defense.ended → schedules close_vote
	ra.scheduleTimeouts([]store.StoredEvent{makeEvent("defense.ended")}, cfg)
	// VotingDurationSec * len(players) = 1 * 3 = 3 seconds
	time.Sleep(3500 * time.Millisecond)

	mu.Lock()
	if len(*dispatched) != 1 || (*dispatched)[0].Type != "close_vote" {
		t.Fatalf("step 3: expected close_vote, got %v", *dispatched)
	}
	*dispatched = nil
	mu.Unlock()

	// Step 4: nomination.resolved → schedules advance_phase(night)
	ra.scheduleTimeouts([]store.StoredEvent{makeEvent("nomination.resolved")}, cfg)
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	if len(*dispatched) != 1 || (*dispatched)[0].Type != "advance_phase" {
		t.Fatalf("step 4: expected advance_phase, got %v", *dispatched)
	}
	mu.Unlock()
}
