package room

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/engine"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

func newTestTimer() (*PhaseTimer, *[]types.CommandEnvelope, *sync.Mutex) {
	var mu sync.Mutex
	var dispatched []types.CommandEnvelope
	pt := NewPhaseTimer("test-room", func(cmd types.CommandEnvelope) {
		mu.Lock()
		dispatched = append(dispatched, cmd)
		mu.Unlock()
	}, zap.NewNop())
	return pt, &dispatched, &mu
}

func TestPhaseTimerIdempotencyKey(t *testing.T) {
	pt, dispatched, mu := newTestTimer()
	pt.Schedule(10*time.Millisecond, "advance_phase", map[string]string{"phase": "day"})
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	cmd := (*dispatched)[0]
	if cmd.IdempotencyKey == "" {
		t.Error("IdempotencyKey must not be empty")
	}
	if cmd.CommandID == "" {
		t.Error("CommandID must not be empty")
	}
	if cmd.IdempotencyKey == cmd.CommandID {
		t.Error("IdempotencyKey and CommandID should be distinct UUIDs")
	}
}

func TestPhaseTimerCancelPrevious(t *testing.T) {
	pt, dispatched, mu := newTestTimer()

	// Schedule first (long timeout)
	pt.Schedule(200*time.Millisecond, "old_cmd", nil)
	// Override with second (short timeout)
	pt.Schedule(10*time.Millisecond, "new_cmd", nil)
	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	if (*dispatched)[0].Type != "new_cmd" {
		t.Errorf("expected new_cmd, got %s", (*dispatched)[0].Type)
	}
}

func TestPhaseTimerGenerationGuard(t *testing.T) {
	// Simulate scenario where old callback races past Stop().
	// We test by scheduling, then immediately scheduling again (bumping generation).
	// Only the second callback should dispatch.
	pt, dispatched, mu := newTestTimer()

	pt.Schedule(5*time.Millisecond, "stale", nil)
	// Immediately reschedule — old timer.Stop may return false if callback is queued
	pt.Schedule(5*time.Millisecond, "fresh", nil)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	for _, cmd := range *dispatched {
		if cmd.Type == "stale" {
			t.Error("stale timer should not have dispatched")
		}
	}
	found := false
	for _, cmd := range *dispatched {
		if cmd.Type == "fresh" {
			found = true
		}
	}
	if !found {
		t.Error("fresh timer should have dispatched")
	}
}

func TestPhaseTimerCancel(t *testing.T) {
	pt, dispatched, mu := newTestTimer()
	pt.Schedule(10*time.Millisecond, "should_cancel", nil)
	pt.Cancel()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 0 {
		t.Errorf("expected 0 dispatches after cancel, got %d", len(*dispatched))
	}
}

// [P1] Cancel must bump generation so in-flight callbacks are invalidated.
func TestPhaseTimerCancelBumpsGeneration(t *testing.T) {
	pt, dispatched, mu := newTestTimer()

	// Schedule with a tiny delay so the callback may already be queued
	pt.Schedule(1*time.Millisecond, "stale_after_cancel", nil)
	// Immediately cancel — even if Stop returns false, generation should block dispatch
	pt.Cancel()

	// Now schedule a fresh one to confirm the timer is still usable
	pt.Schedule(10*time.Millisecond, "fresh_after_cancel", nil)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	for _, cmd := range *dispatched {
		if cmd.Type == "stale_after_cancel" {
			t.Error("stale callback should not dispatch after Cancel()")
		}
	}
	found := false
	for _, cmd := range *dispatched {
		if cmd.Type == "fresh_after_cancel" {
			found = true
		}
	}
	if !found {
		t.Error("fresh timer after cancel should dispatch")
	}
}

// --- recoverTimeoutFromState tests ---

// newTestRoomActor builds a minimal RoomActor for recovery testing.
func newTestRoomActor(state engine.State) (*RoomActor, *[]types.CommandEnvelope, *sync.Mutex) {
	var mu sync.Mutex
	var dispatched []types.CommandEnvelope
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

func TestRecoverTimeout_Night(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseNight
	state.Config.NightActionTimeoutSec = 1 // short for test

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	if (*dispatched)[0].Type != "night_timeout" {
		t.Errorf("expected night_timeout, got %s", (*dispatched)[0].Type)
	}
}

func TestRecoverTimeout_DayDiscussion(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseDay
	state.SubPhase = engine.SubPhaseDiscussion
	state.Config.DiscussionDurationSec = 1

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	if (*dispatched)[0].Type != "advance_phase" {
		t.Errorf("expected advance_phase, got %s", (*dispatched)[0].Type)
	}
}

func TestRecoverTimeout_DayDefense(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseDay
	state.SubPhase = engine.SubPhaseDefense
	state.Config.DefenseDurationSec = 1

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()
	time.Sleep(1200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	if (*dispatched)[0].Type != "end_defense" {
		t.Errorf("expected end_defense, got %s", (*dispatched)[0].Type)
	}
}

func TestRecoverTimeout_DayVoting(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseDay
	state.SubPhase = engine.SubPhaseVoting
	state.Config.VotingDurationSec = 1
	state.Players = map[string]engine.Player{
		"p1": {UserID: "p1"},
		"p2": {UserID: "p2"},
	}

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()
	time.Sleep(2500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	if (*dispatched)[0].Type != "close_vote" {
		t.Errorf("expected close_vote, got %s", (*dispatched)[0].Type)
	}
}

func TestRecoverTimeout_Lobby(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseLobby

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 0 {
		t.Errorf("lobby should not schedule timer, got %d dispatches", len(*dispatched))
	}
}

func TestRecoverTimeout_Ended(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseEnded

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 0 {
		t.Errorf("ended phase should not schedule timer, got %d dispatches", len(*dispatched))
	}
}

// [P2] nomination_open subphase should use NominationPhaseDurationSec.
func TestRecoverTimeout_DayNominationOpen(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseDay
	state.SubPhase = engine.SubPhaseNominationOpen
	state.Config.NominationPhaseDurationSec = 1
	state.Config.DiscussionDurationSec = 10 // must NOT use this

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()

	// Should fire at ~1s (NominationTimeoutSec), not 10s
	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(*dispatched))
	}
	if (*dispatched)[0].Type != "advance_phase" {
		t.Errorf("expected advance_phase, got %s", (*dispatched)[0].Type)
	}
}

// [P3] PhaseNomination recovery path should schedule NominationPhaseDurationSec.
func TestRecoverTimeout_PhaseNomination(t *testing.T) {
	state := engine.NewState("test-room")
	state.Phase = engine.PhaseNomination
	state.Config.NominationPhaseDurationSec = 1

	ra, dispatched, mu := newTestRoomActor(state)
	ra.recoverTimeoutFromState()
	time.Sleep(1500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(*dispatched) != 1 {
		t.Fatalf("expected 1 dispatch for PhaseNomination recovery, got %d", len(*dispatched))
	}
	if (*dispatched)[0].Type != "advance_phase" {
		t.Errorf("expected advance_phase, got %s", (*dispatched)[0].Type)
	}
}
