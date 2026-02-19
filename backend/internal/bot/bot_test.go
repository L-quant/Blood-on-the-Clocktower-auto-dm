package bot

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// mockDispatcher records dispatched commands.
type mockDispatcher struct {
	mu       sync.Mutex
	commands []types.CommandEnvelope
}

func (m *mockDispatcher) DispatchAsync(cmd types.CommandEnvelope) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.commands = append(m.commands, cmd)
	return nil
}

func (m *mockDispatcher) getCommands() []types.CommandEnvelope {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]types.CommandEnvelope, len(m.commands))
	copy(result, m.commands)
	return result
}

func TestNewBot(t *testing.T) {
	b := NewBot(BotConfig{
		UserID:      "bot-1",
		Name:        "Alice",
		Personality: PersonalityAggressive,
	})

	if b.UserID() != "bot-1" {
		t.Errorf("expected UserID bot-1, got %s", b.UserID())
	}
	if b.Name() != "Alice" {
		t.Errorf("expected Name Alice, got %s", b.Name())
	}
	if b.personality != PersonalityAggressive {
		t.Errorf("expected aggressive personality")
	}
	if !b.alive {
		t.Errorf("expected bot to start alive")
	}
}

func TestBotDefaultPersonality(t *testing.T) {
	b := NewBot(BotConfig{
		UserID: "bot-2",
		Name:   "Bob",
	})
	if b.personality != PersonalityRandom {
		t.Errorf("expected default random personality, got %s", b.personality)
	}
}

func TestBotProcessesRoleAssignment(t *testing.T) {
	b := NewBot(BotConfig{UserID: "bot-1", Name: "Alice"})

	payload, _ := json.Marshal(map[string]string{
		"user_id":   "bot-1",
		"role":      "empath",
		"true_role": "empath",
		"team":      "good",
	})

	b.OnEvent(context.Background(), types.Event{
		EventType: "role.assigned",
		Payload:   payload,
	})

	if b.role != "empath" {
		t.Errorf("expected role empath, got %s", b.role)
	}
	if b.team != "good" {
		t.Errorf("expected team good, got %s", b.team)
	}
}

func TestBotIgnoresOtherPlayerRoleAssignment(t *testing.T) {
	b := NewBot(BotConfig{UserID: "bot-1", Name: "Alice"})

	payload, _ := json.Marshal(map[string]string{
		"user_id":   "bot-2",
		"role":      "imp",
		"true_role": "imp",
		"team":      "evil",
	})

	b.OnEvent(context.Background(), types.Event{
		EventType: "role.assigned",
		Payload:   payload,
	})

	if b.role != "" {
		t.Errorf("bot should not have been assigned role for another player")
	}
}

func TestBotDiesOnPlayerDied(t *testing.T) {
	b := NewBot(BotConfig{UserID: "bot-1", Name: "Alice"})

	payload, _ := json.Marshal(map[string]string{
		"user_id": "bot-1",
		"cause":   "demon",
	})

	b.OnEvent(context.Background(), types.Event{
		EventType: "player.died",
		Payload:   payload,
	})

	if b.alive {
		t.Errorf("expected bot to be dead")
	}
}

func TestBotSurvivesOtherPlayerDeath(t *testing.T) {
	b := NewBot(BotConfig{UserID: "bot-1", Name: "Alice"})

	payload, _ := json.Marshal(map[string]string{
		"user_id": "bot-2",
		"cause":   "demon",
	})

	b.OnEvent(context.Background(), types.Event{
		EventType: "player.died",
		Payload:   payload,
	})

	if !b.alive {
		t.Errorf("bot should still be alive")
	}
}

func TestManagerAddBots(t *testing.T) {
	mgr := NewManager(nil)
	disp := &mockDispatcher{}

	botIDs, err := mgr.AddBots(context.Background(), AddBotsRequest{
		RoomID: "room-1",
		Count:  3,
	}, disp)

	if err != nil {
		t.Fatal(err)
	}
	if len(botIDs) != 3 {
		t.Errorf("expected 3 bot IDs, got %d", len(botIDs))
	}
	if mgr.BotCount("room-1") != 3 {
		t.Errorf("expected 3 bots in room, got %d", mgr.BotCount("room-1"))
	}

	// Check that join commands were dispatched
	cmds := disp.getCommands()
	if len(cmds) != 3 {
		t.Errorf("expected 3 join commands, got %d", len(cmds))
	}
	for _, cmd := range cmds {
		if cmd.Type != "join" {
			t.Errorf("expected join command, got %s", cmd.Type)
		}
	}
}

func TestManagerAddBotsMaxLimit(t *testing.T) {
	mgr := NewManager(nil)
	disp := &mockDispatcher{}

	_, err := mgr.AddBots(context.Background(), AddBotsRequest{
		RoomID: "room-1",
		Count:  15,
	}, disp)

	if err == nil {
		t.Errorf("expected error for >14 bots")
	}
}

func TestManagerAddBotsZeroCount(t *testing.T) {
	mgr := NewManager(nil)
	disp := &mockDispatcher{}

	_, err := mgr.AddBots(context.Background(), AddBotsRequest{
		RoomID: "room-1",
		Count:  0,
	}, disp)

	if err == nil {
		t.Errorf("expected error for 0 count")
	}
}

func TestManagerRemoveBots(t *testing.T) {
	mgr := NewManager(nil)
	disp := &mockDispatcher{}

	mgr.AddBots(context.Background(), AddBotsRequest{
		RoomID: "room-1",
		Count:  3,
	}, disp)

	mgr.RemoveBots("room-1")

	if mgr.BotCount("room-1") != 0 {
		t.Errorf("expected 0 bots after removal")
	}
}

func TestManagerOnEventBroadcasts(t *testing.T) {
	mgr := NewManager(nil)
	disp := &mockDispatcher{}

	mgr.AddBots(context.Background(), AddBotsRequest{
		RoomID: "room-1",
		Count:  2,
	}, disp)

	// Get bot IDs
	bots := mgr.GetBots("room-1")
	if len(bots) != 2 {
		t.Fatalf("expected 2 bots")
	}

	// Send role assignment to first bot
	payload, _ := json.Marshal(map[string]string{
		"user_id":   bots[0].UserID(),
		"role":      "washerwoman",
		"true_role": "washerwoman",
		"team":      "good",
	})

	mgr.OnEvent(context.Background(), "room-1", types.Event{
		EventType: "role.assigned",
		Payload:   payload,
	})

	// First bot should have the role
	if bots[0].role != "washerwoman" {
		t.Errorf("expected first bot to have role washerwoman, got %s", bots[0].role)
	}
	// Second bot should not
	if bots[1].role != "" {
		t.Errorf("expected second bot to have no role, got %s", bots[1].role)
	}
}

func TestBotNamesUnique(t *testing.T) {
	mgr := NewManager(nil)
	disp := &mockDispatcher{}

	mgr.AddBots(context.Background(), AddBotsRequest{
		RoomID: "room-1",
		Count:  10,
	}, disp)

	bots := mgr.GetBots("room-1")
	names := make(map[string]bool)
	for _, b := range bots {
		if names[b.Name()] {
			t.Errorf("duplicate bot name: %s", b.Name())
		}
		names[b.Name()] = true
	}
}

func TestRandomChance(t *testing.T) {
	// Test that randomChance(0) always returns false
	for i := 0; i < 100; i++ {
		if randomChance(0) {
			t.Errorf("randomChance(0) should always return false")
			break
		}
	}

	// Test that randomChance(100) always returns true
	for i := 0; i < 100; i++ {
		if !randomChance(100) {
			t.Errorf("randomChance(100) should always return true")
			break
		}
	}
}

func TestGenerateChat(t *testing.T) {
	b := NewBot(BotConfig{
		UserID:      "bot-1",
		Name:        "Alice",
		Personality: PersonalityAggressive,
	})
	b.dayCount = 1

	msg := b.generateChat()
	if msg == "" {
		t.Errorf("expected non-empty chat message")
	}
}
