package engine

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// ---------- helpers ----------

// buildFivePlayerState returns a 5-player day-phase state with demon="demon".
func buildFivePlayerState() State {
	s := NewState("room-vr")
	s.Phase = PhaseDay
	s.DemonID = "demon"
	s.MinionIDs = []string{"minion"}
	s.Players = map[string]Player{
		"g1":     {UserID: "g1", SeatNumber: 1, TrueRole: "washerwoman", Team: "good", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"g2":     {UserID: "g2", SeatNumber: 2, TrueRole: "empath", Team: "good", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"g3":     {UserID: "g3", SeatNumber: 3, TrueRole: "chef", Team: "good", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"demon":  {UserID: "demon", SeatNumber: 4, TrueRole: "imp", Team: "evil", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"minion": {UserID: "minion", SeatNumber: 5, TrueRole: "poisoner", Team: "evil", Alive: true, HasGhostVote: true, Reminders: []string{}},
	}
	s.SeatOrder = []string{"g1", "g2", "g3", "demon", "minion"}
	return s
}

func makeCmd(roomID, cmdType, actor string, payload map[string]string) types.CommandEnvelope {
	b, _ := json.Marshal(payload)
	return types.CommandEnvelope{
		CommandID:   uuid.NewString(),
		RoomID:      roomID,
		Type:        cmdType,
		ActorUserID: actor,
		Payload:     b,
	}
}

func reduceAll(s *State, events []types.Event) {
	for _, e := range events {
		s.Reduce(toEventPayload(e))
	}
}

func hasEvent(events []types.Event, eventType string) bool {
	for _, e := range events {
		if e.EventType == eventType {
			return true
		}
	}
	return false
}

func eventData(events []types.Event, eventType string) map[string]string {
	for _, e := range events {
		if e.EventType == eventType {
			var d map[string]string
			json.Unmarshal(e.Payload, &d)
			return d
		}
	}
	return nil
}

// ---------- Phase 1.6 tests ----------

func TestExecuteDemon_GameEndedGood(t *testing.T) {
	s := buildFivePlayerState()

	// Nominate demon
	cmd := makeCmd(s.RoomID, "nominate", "g1", map[string]string{"nominee": "demon"})
	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("nominate: %v", err)
	}
	reduceAll(&s, events)

	// Move to voting
	s.SubPhase = SubPhaseVoting

	// All 5 vote yes
	for _, uid := range []string{"g1", "g2", "g3", "demon", "minion"} {
		cmd = makeCmd(s.RoomID, "vote", uid, map[string]string{"vote": "yes"})
		events, _, err = HandleCommand(s, cmd)
		if err != nil {
			t.Fatalf("vote %s: %v", uid, err)
		}
		reduceAll(&s, events)
	}

	if !hasEvent(events, "game.ended") {
		t.Fatal("expected game.ended after executing demon")
	}
	d := eventData(events, "game.ended")
	if d["winner"] != "good" {
		t.Errorf("expected winner=good, got %s", d["winner"])
	}
}

func TestExecuteNormalPlayer_NoGameEnded(t *testing.T) {
	s := buildFivePlayerState()

	// Nominate a good player
	cmd := makeCmd(s.RoomID, "nominate", "g1", map[string]string{"nominee": "g2"})
	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("nominate: %v", err)
	}
	reduceAll(&s, events)

	s.SubPhase = SubPhaseVoting

	for _, uid := range []string{"g1", "g2", "g3", "demon", "minion"} {
		cmd = makeCmd(s.RoomID, "vote", uid, map[string]string{"vote": "yes"})
		events, _, err = HandleCommand(s, cmd)
		if err != nil {
			t.Fatalf("vote %s: %v", uid, err)
		}
		reduceAll(&s, events)
	}

	if hasEvent(events, "game.ended") {
		t.Fatal("game should NOT end when executing a normal player")
	}
	if !hasEvent(events, "execution.resolved") {
		t.Fatal("expected execution.resolved event")
	}
}

func TestExecuteDemon_ScarletWomanTakeover(t *testing.T) {
	// Need 6 players so that after demon dies, 5 remain alive (SW requires >=5).
	s := NewState("room-sw")
	s.Phase = PhaseDay
	s.DemonID = "demon"
	s.MinionIDs = []string{"sw"}
	s.Players = map[string]Player{
		"g1":    {UserID: "g1", SeatNumber: 1, TrueRole: "washerwoman", Team: "good", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"g2":    {UserID: "g2", SeatNumber: 2, TrueRole: "empath", Team: "good", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"g3":    {UserID: "g3", SeatNumber: 3, TrueRole: "chef", Team: "good", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"g4":    {UserID: "g4", SeatNumber: 4, TrueRole: "monk", Team: "good", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"demon": {UserID: "demon", SeatNumber: 5, TrueRole: "imp", Team: "evil", Alive: true, HasGhostVote: true, Reminders: []string{}},
		"sw":    {UserID: "sw", SeatNumber: 6, TrueRole: "scarletwoman", Team: "evil", Alive: true, HasGhostVote: true, Reminders: []string{}},
	}
	s.SeatOrder = []string{"g1", "g2", "g3", "g4", "demon", "sw"}

	// Nominate demon
	cmd := makeCmd(s.RoomID, "nominate", "g1", map[string]string{"nominee": "demon"})
	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("nominate: %v", err)
	}
	reduceAll(&s, events)

	s.SubPhase = SubPhaseVoting

	for _, uid := range []string{"g1", "g2", "g3", "g4", "demon", "sw"} {
		cmd = makeCmd(s.RoomID, "vote", uid, map[string]string{"vote": "yes"})
		events, _, err = HandleCommand(s, cmd)
		if err != nil {
			t.Fatalf("vote %s: %v", uid, err)
		}
		reduceAll(&s, events)
	}

	// With 6 players (5 alive after demon dies) and SW alive, should get demon.changed not game.ended
	if hasEvent(events, "game.ended") {
		t.Fatal("game should NOT end when SW can take over (>=5 alive)")
	}
	d := eventData(events, "demon.changed")
	if d == nil {
		t.Fatal("expected demon.changed event for SW takeover")
	}
	if d["reason"] != "scarletwoman" {
		t.Errorf("expected reason=scarletwoman, got %s", d["reason"])
	}
}

func TestHandleVoteAndCloseVote_EventParity(t *testing.T) {
	// Both paths should produce the same event structure.
	// We run a full-vote path and a close_vote path and compare field names.

	// --- Path A: handleVote (all voted) ---
	sA := buildFivePlayerState()
	cmd := makeCmd(sA.RoomID, "nominate", "g1", map[string]string{"nominee": "g2"})
	events, _, _ := HandleCommand(sA, cmd)
	reduceAll(&sA, events)
	sA.SubPhase = SubPhaseVoting

	var lastEventsA []types.Event
	for _, uid := range []string{"g1", "g2", "g3", "demon", "minion"} {
		cmd = makeCmd(sA.RoomID, "vote", uid, map[string]string{"vote": "yes"})
		lastEventsA, _, _ = HandleCommand(sA, cmd)
		reduceAll(&sA, lastEventsA)
	}

	nomResolvedA := eventData(lastEventsA, "nomination.resolved")
	if nomResolvedA == nil {
		t.Fatal("path A: missing nomination.resolved")
	}

	// --- Path B: handleCloseVote ---
	sB := buildFivePlayerState()
	cmd = makeCmd(sB.RoomID, "nominate", "g1", map[string]string{"nominee": "g2"})
	events, _, _ = HandleCommand(sB, cmd)
	reduceAll(&sB, events)
	sB.SubPhase = SubPhaseVoting

	// Only 3 players vote (not all)
	for _, uid := range []string{"g1", "g3", "demon"} {
		cmd = makeCmd(sB.RoomID, "vote", uid, map[string]string{"vote": "yes"})
		events, _, _ = HandleCommand(sB, cmd)
		reduceAll(&sB, events)
	}

	// autodm closes
	cmd = makeCmd(sB.RoomID, "close_vote", "autodm", nil)
	lastEventsB, _, err := HandleCommand(sB, cmd)
	if err != nil {
		t.Fatalf("close_vote: %v", err)
	}

	nomResolvedB := eventData(lastEventsB, "nomination.resolved")
	if nomResolvedB == nil {
		t.Fatal("path B: missing nomination.resolved")
	}

	// Check same field names exist in both
	for _, key := range []string{"result", "votes_for", "votes_against", "threshold"} {
		if _, ok := nomResolvedA[key]; !ok {
			t.Errorf("path A nomination.resolved missing field %s", key)
		}
		if _, ok := nomResolvedB[key]; !ok {
			t.Errorf("path B nomination.resolved missing field %s", key)
		}
	}

	// Both should emit execution.resolved (not player.executed)
	if !hasEvent(lastEventsA, "execution.resolved") {
		t.Error("path A: missing execution.resolved")
	}
	if !hasEvent(lastEventsB, "execution.resolved") {
		t.Error("path B: missing execution.resolved")
	}
	// Neither should emit player.executed
	if hasEvent(lastEventsA, "player.executed") {
		t.Error("path A: should not emit player.executed (use player.died + execution.resolved)")
	}
	if hasEvent(lastEventsB, "player.executed") {
		t.Error("path B: should not emit player.executed (use player.died + execution.resolved)")
	}
}

func TestAutoDMNomination_ActorSemantics(t *testing.T) {
	s := buildFivePlayerState()

	// autodm proxies a nomination on behalf of g2
	cmd := makeCmd(s.RoomID, "nominate", "autodm", map[string]string{
		"nominator": "g2",
		"nominee":   "g3",
	})
	events, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("autodm nominate: %v", err)
	}

	// nomination.created should have nominator_user_id = g2
	d := eventData(events, "nomination.created")
	if d == nil {
		t.Fatal("missing nomination.created")
	}
	if d["nominator_user_id"] != "g2" {
		t.Errorf("expected nominator_user_id=g2, got %s", d["nominator_user_id"])
	}

	// Apply events and check HasNominated was set on g2 (not autodm)
	reduceAll(&s, events)
	if !s.Players["g2"].HasNominated {
		t.Error("g2 should have HasNominated=true")
	}
}

func TestAutoDMCanEndDefense(t *testing.T) {
	s := buildFivePlayerState()

	// Create a nomination and go to defense phase
	cmd := makeCmd(s.RoomID, "nominate", "g1", map[string]string{"nominee": "g2"})
	events, _, _ := HandleCommand(s, cmd)
	reduceAll(&s, events)

	// Verify we're in defense
	if s.SubPhase != SubPhaseDefense {
		t.Fatalf("expected defense sub-phase, got %s", s.SubPhase)
	}

	// autodm ends defense
	cmd = makeCmd(s.RoomID, "end_defense", "autodm", nil)
	events, result, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("autodm end_defense should succeed: %v", err)
	}
	if result.Status != "accepted" {
		t.Errorf("expected accepted, got %s", result.Status)
	}
	if !hasEvent(events, "defense.ended") {
		t.Error("expected defense.ended event")
	}
}

func TestAutoDMEndDefense_AutoDMHyphenAlias(t *testing.T) {
	s := buildFivePlayerState()

	cmd := makeCmd(s.RoomID, "nominate", "g1", map[string]string{"nominee": "g2"})
	events, _, _ := HandleCommand(s, cmd)
	reduceAll(&s, events)

	// auto-dm (with hyphen) should also work
	cmd = makeCmd(s.RoomID, "end_defense", "auto-dm", nil)
	_, _, err := HandleCommand(s, cmd)
	if err != nil {
		t.Fatalf("auto-dm end_defense should succeed: %v", err)
	}
}

func TestRegularPlayerCannotEndDefense(t *testing.T) {
	s := buildFivePlayerState()

	cmd := makeCmd(s.RoomID, "nominate", "g1", map[string]string{"nominee": "g2"})
	events, _, _ := HandleCommand(s, cmd)
	reduceAll(&s, events)

	// g3 (not nominator or nominee) should fail
	cmd = makeCmd(s.RoomID, "end_defense", "g3", nil)
	_, _, err := HandleCommand(s, cmd)
	if err == nil {
		t.Fatal("regular player should not be able to end defense")
	}
}

func TestThreshold_EvenAlive(t *testing.T) {
	// 10 alive: threshold = (10+1)/2 = 5 (ceil(10/2)=5)
	s := NewState("room-threshold")
	s.Phase = PhaseDay
	s.DemonID = "demon"
	players := map[string]Player{}
	seatOrder := []string{}
	for i := 1; i <= 10; i++ {
		uid := "p" + string(rune('0'+i))
		if i == 10 {
			uid = "demon"
		}
		role := "empath"
		team := "good"
		if i == 10 {
			role = "imp"
			team = "evil"
		}
		players[uid] = Player{
			UserID: uid, SeatNumber: i, TrueRole: role, Team: team,
			Alive: true, HasGhostVote: true, Reminders: []string{},
		}
		seatOrder = append(seatOrder, uid)
	}
	s.Players = players
	s.SeatOrder = seatOrder

	// Nominate
	cmd := makeCmd(s.RoomID, "nominate", "p1", map[string]string{"nominee": "p2"})
	events, _, _ := HandleCommand(s, cmd)
	reduceAll(&s, events)
	s.SubPhase = SubPhaseVoting

	// 5 yes votes, rest no — should trigger execution (5 >= 5)
	voters := s.SeatOrder
	for i, uid := range voters {
		vote := "no"
		if i < 5 {
			vote = "yes"
		}
		cmd = makeCmd(s.RoomID, "vote", uid, map[string]string{"vote": vote})
		events, _, _ = HandleCommand(s, cmd)
		reduceAll(&s, events)
	}

	d := eventData(events, "nomination.resolved")
	if d == nil {
		t.Fatal("missing nomination.resolved")
	}
	if d["threshold"] != "5" {
		t.Errorf("10 alive: expected threshold=5, got %s", d["threshold"])
	}
	if d["result"] != "executed" {
		t.Errorf("5 yes out of 10 alive (threshold 5) should execute, got %s", d["result"])
	}
}

func TestThreshold_OddAlive(t *testing.T) {
	// 9 alive: threshold = (9+1)/2 = 5 (ceil(9/2)=5)
	s := NewState("room-threshold-odd")
	s.Phase = PhaseDay
	s.DemonID = "demon"
	players := map[string]Player{}
	seatOrder := []string{}
	for i := 1; i <= 9; i++ {
		uid := "p" + string(rune('0'+i))
		if i == 9 {
			uid = "demon"
		}
		role := "empath"
		team := "good"
		if i == 9 {
			role = "imp"
			team = "evil"
		}
		players[uid] = Player{
			UserID: uid, SeatNumber: i, TrueRole: role, Team: team,
			Alive: true, HasGhostVote: true, Reminders: []string{},
		}
		seatOrder = append(seatOrder, uid)
	}
	s.Players = players
	s.SeatOrder = seatOrder

	// Nominate
	cmd := makeCmd(s.RoomID, "nominate", "p1", map[string]string{"nominee": "p2"})
	events, _, _ := HandleCommand(s, cmd)
	reduceAll(&s, events)
	s.SubPhase = SubPhaseVoting

	// 4 yes, 5 no — should NOT execute (4 < 5)
	voters := s.SeatOrder
	for i, uid := range voters {
		vote := "no"
		if i < 4 {
			vote = "yes"
		}
		cmd = makeCmd(s.RoomID, "vote", uid, map[string]string{"vote": vote})
		events, _, _ = HandleCommand(s, cmd)
		reduceAll(&s, events)
	}

	d := eventData(events, "nomination.resolved")
	if d == nil {
		t.Fatal("missing nomination.resolved")
	}
	if d["threshold"] != "5" {
		t.Errorf("9 alive: expected threshold=5, got %s", d["threshold"])
	}
	if d["result"] != "not_executed" {
		t.Errorf("4 yes out of 9 alive (threshold 5) should not execute, got %s", d["result"])
	}
}
