package agent

import (
	"context"
	"testing"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/memory"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/tools"
)

// MockCommander implements GameCommander for testing.
type MockCommander struct {
	messages []string
	killed   []string
	phase    string
}

func (m *MockCommander) SendMessage(ctx context.Context, roomID, message string) error {
	m.messages = append(m.messages, message)
	return nil
}

func (m *MockCommander) KillPlayer(ctx context.Context, roomID, playerID string) error {
	m.killed = append(m.killed, playerID)
	return nil
}

func (m *MockCommander) RevivePlayer(ctx context.Context, roomID, playerID string) error {
	return nil
}

func (m *MockCommander) SetPhase(ctx context.Context, roomID, phase string) error {
	m.phase = phase
	return nil
}

func (m *MockCommander) StartVote(ctx context.Context, roomID, nominatorID, nomineeID string) error {
	return nil
}

func (m *MockCommander) EndVote(ctx context.Context, roomID string) error {
	return nil
}

func (m *MockCommander) RevealRole(ctx context.Context, roomID, playerID, role string) error {
	return nil
}

func (m *MockCommander) AssignRole(ctx context.Context, roomID, playerID, role string) error {
	return nil
}

func (m *MockCommander) SetReminder(ctx context.Context, roomID, playerID, reminder string) error {
	return nil
}

func (m *MockCommander) EndGame(ctx context.Context, roomID, winner string) error {
	return nil
}

func (m *MockCommander) GetPlayers(ctx context.Context, roomID string) ([]tools.PlayerInfo, error) {
	return []tools.PlayerInfo{
		{ID: "p1", Name: "Alice", Role: "Washerwoman", IsAlive: true, Seat: 1},
		{ID: "p2", Name: "Bob", Role: "Imp", IsAlive: true, Seat: 2},
		{ID: "p3", Name: "Charlie", Role: "Empath", IsAlive: true, Seat: 3},
	}, nil
}

func (m *MockCommander) GetGameState(ctx context.Context, roomID string) (*tools.GameInfo, error) {
	return &tools.GameInfo{
		RoomID:    roomID,
		Phase:     "day",
		DayNumber: 1,
		Edition:   "trouble_brewing",
		IsStarted: true,
	}, nil
}

func TestAutoDMCreation(t *testing.T) {
	cfg := Config{
		RoomID: "test-room",
		LLM: llm.RoutingConfig{
			Default: llm.Config{
				BaseURL: "http://localhost:11434/v1",
				Model:   "llama3.2",
			},
		},
		Memory: memory.Config{
			ShortTermCapacity: 50,
		},
	}
	
	dm := NewAutoDM(cfg)
	
	if dm == nil {
		t.Fatal("Expected AutoDM to be created")
	}
	
	if dm.IsActive() {
		t.Error("Expected AutoDM to be inactive before Start()")
	}
	
	dm.Start()
	if !dm.IsActive() {
		t.Error("Expected AutoDM to be active after Start()")
	}
	
	dm.Stop()
	if dm.IsActive() {
		t.Error("Expected AutoDM to be inactive after Stop()")
	}
}

func TestGameStateUpdate(t *testing.T) {
	cfg := Config{
		RoomID: "test-room",
		LLM: llm.RoutingConfig{
			Default: llm.Config{
				BaseURL: "http://localhost:11434/v1",
				Model:   "llama3.2",
			},
		},
	}
	
	dm := NewAutoDM(cfg)
	
	state := &GameState{
		RoomID:    "test-room",
		Phase:     "night",
		DayNumber: 2,
		Players: []Player{
			{ID: "p1", Name: "Alice", Role: "Washerwoman", IsAlive: true},
			{ID: "p2", Name: "Bob", Role: "Imp", IsAlive: true},
		},
		Edition: "trouble_brewing",
	}
	
	// Should not panic
	dm.UpdateGameState(state)
}

func TestEventTypes(t *testing.T) {
	events := []Event{
		{Type: "phase_change", Description: "Night begins", Data: map[string]interface{}{"new_phase": "night", "old_phase": "day"}},
		{Type: "nomination", Description: "Alice nominates Bob", Data: map[string]interface{}{"nominator": "p1", "nominee": "p2"}},
		{Type: "vote", Description: "Voting on Bob", Data: map[string]interface{}{"nominee": "p2", "votes": 3}},
		{Type: "death", Description: "Charlie died", Data: map[string]interface{}{"player_name": "Charlie", "cause": "demon"}},
		{Type: "question", Description: "How does the Empath work?"},
		{Type: "night_action", Description: "Imp chooses a target", Data: map[string]interface{}{"role": "Imp", "action": "kill"}},
	}
	
	for _, e := range events {
		if e.Type == "" {
			t.Errorf("Event type should not be empty")
		}
		if e.Description == "" {
			t.Errorf("Event description should not be empty")
		}
	}
}

func TestMockCommander(t *testing.T) {
	ctx := context.Background()
	mock := &MockCommander{}
	
	// Test SendMessage
	err := mock.SendMessage(ctx, "room", "Hello")
	if err != nil {
		t.Errorf("SendMessage failed: %v", err)
	}
	if len(mock.messages) != 1 || mock.messages[0] != "Hello" {
		t.Errorf("Message not recorded correctly")
	}
	
	// Test KillPlayer
	err = mock.KillPlayer(ctx, "room", "p1")
	if err != nil {
		t.Errorf("KillPlayer failed: %v", err)
	}
	if len(mock.killed) != 1 || mock.killed[0] != "p1" {
		t.Errorf("Kill not recorded correctly")
	}
	
	// Test SetPhase
	err = mock.SetPhase(ctx, "room", "night")
	if err != nil {
		t.Errorf("SetPhase failed: %v", err)
	}
	if mock.phase != "night" {
		t.Errorf("Phase not set correctly")
	}
	
	// Test GetPlayers
	players, err := mock.GetPlayers(ctx, "room")
	if err != nil {
		t.Errorf("GetPlayers failed: %v", err)
	}
	if len(players) != 3 {
		t.Errorf("Expected 3 players, got %d", len(players))
	}
}
