package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// BotNames is the default set of bot display names.
var BotNames = []string{
	"Alice", "Bob", "Charlie", "Diana", "Eve",
	"Frank", "Grace", "Henry", "Ivy", "Jack",
	"Kate", "Leo", "Mia", "Noah", "Olivia",
}

// Manager manages bot players across rooms.
type Manager struct {
	mu     sync.RWMutex
	bots   map[string][]*Bot // roomID -> bots
	logger *slog.Logger
}

// NewManager creates a new bot manager.
func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		bots:   make(map[string][]*Bot),
		logger: logger,
	}
}

// AddBotsRequest is the request to add bots to a room.
type AddBotsRequest struct {
	RoomID      string      `json:"room_id"`
	Count       int         `json:"count"`
	Personality Personality `json:"personality,omitempty"`
}

// AddBots creates and adds bot players to a room.
// Returns the list of bot user IDs created.
func (m *Manager) AddBots(ctx context.Context, req AddBotsRequest, dispatcher CommandDispatcher) ([]string, error) {
	if req.Count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}
	if req.Count > 14 {
		return nil, fmt.Errorf("cannot add more than 14 bots")
	}

	m.mu.Lock()
	existing := len(m.bots[req.RoomID])
	m.mu.Unlock()

	if existing+req.Count > 14 {
		return nil, fmt.Errorf("too many bots: have %d, adding %d, max 14", existing, req.Count)
	}

	personality := req.Personality
	if personality == "" {
		personality = PersonalityRandom
	}

	var botIDs []string
	var newBots []*Bot

	for i := 0; i < req.Count; i++ {
		nameIdx := existing + i
		name := BotNames[nameIdx%len(BotNames)]
		if nameIdx >= len(BotNames) {
			name = fmt.Sprintf("%s_%d", name, nameIdx/len(BotNames))
		}

		botID := fmt.Sprintf("bot-%s", uuid.NewString()[:8])
		b := NewBot(BotConfig{
			UserID:      botID,
			Name:        name,
			Personality: personality,
			Logger:      m.logger,
		})
		b.SetDispatcher(dispatcher, req.RoomID)

		newBots = append(newBots, b)
		botIDs = append(botIDs, botID)

		// Join the room as a player
		joinPayload, _ := json.Marshal(map[string]string{
			"name":        name,
			"seat_number": fmt.Sprintf("%d", existing+i+1),
			"role":        "player",
		})
		if err := dispatcher.DispatchAsync(types.CommandEnvelope{
			CommandID:   fmt.Sprintf("bot-join-%s", botID),
			RoomID:      req.RoomID,
			Type:        "join",
			ActorUserID: botID,
			Payload:     joinPayload,
		}); err != nil {
			m.logger.Error("bot failed to join", "bot", name, "error", err)
		}
	}

	m.mu.Lock()
	m.bots[req.RoomID] = append(m.bots[req.RoomID], newBots...)
	m.mu.Unlock()

	m.logger.Info("bots added", "room", req.RoomID, "count", req.Count, "total", existing+req.Count)
	return botIDs, nil
}

// OnEvent broadcasts an event to all bots in a room.
func (m *Manager) OnEvent(ctx context.Context, roomID string, ev types.Event) {
	m.mu.RLock()
	bots := m.bots[roomID]
	m.mu.RUnlock()

	for _, b := range bots {
		b.OnEvent(ctx, ev)
	}
}

// GetBots returns all bots in a room.
func (m *Manager) GetBots(roomID string) []*Bot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.bots[roomID]
}

// RemoveBots removes all bots from a room.
func (m *Manager) RemoveBots(roomID string) {
	m.mu.Lock()
	delete(m.bots, roomID)
	m.mu.Unlock()
}

// BotCount returns the number of bots in a room.
func (m *Manager) BotCount(roomID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.bots[roomID])
}
