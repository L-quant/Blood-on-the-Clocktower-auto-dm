package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RoomBridge bridges the agent tools with the room management system
type RoomBridge struct {
	roomManager RoomManagerInterface
}

// RoomManagerInterface defines the interface for room management
type RoomManagerInterface interface {
	GetState(ctx context.Context, roomID string) (*RoomState, error)
	GetEvents(ctx context.Context, roomID string, afterSeq int64, limit int) ([]GameEvent, error)
	SendCommand(ctx context.Context, roomID string, cmd Command) error
}

// Command represents a game command
type Command struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	RoomID    string          `json:"room_id"`
	ActorID   string          `json:"actor_id"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewRoomBridge creates a new room bridge
func NewRoomBridge(rm RoomManagerInterface) *RoomBridge {
	return &RoomBridge{
		roomManager: rm,
	}
}

func (b *RoomBridge) GetRoomState(ctx context.Context, args GetRoomStateArgs) (json.RawMessage, error) {
	state, err := b.roomManager.GetState(ctx, args.RoomID)
	if err != nil {
		return nil, fmt.Errorf("get room state: %w", err)
	}
	return json.Marshal(state)
}

func (b *RoomBridge) GetRecentEvents(ctx context.Context, args GetRecentEventsArgs) (json.RawMessage, error) {
	limit := args.Limit
	if limit <= 0 {
		limit = 50
	}
	events, err := b.roomManager.GetEvents(ctx, args.RoomID, args.SinceSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("get events: %w", err)
	}
	return json.Marshal(events)
}

func (b *RoomBridge) SendPublicMessage(ctx context.Context, args SendMessageArgs) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"message":  args.Text,
		"metadata": args.Metadata,
	})

	cmd := Command{
		ID:        uuid.NewString(),
		Type:      "public_chat",
		RoomID:    args.RoomID,
		ActorID:   "autodm",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := b.roomManager.SendCommand(ctx, args.RoomID, cmd); err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	return json.Marshal(map[string]string{"status": "sent"})
}

func (b *RoomBridge) SendWhisper(ctx context.Context, args SendWhisperArgs) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"to_user_id": args.ToUserID,
		"message":    args.Text,
		"metadata":   args.Metadata,
	})

	cmd := Command{
		ID:        uuid.NewString(),
		Type:      "whisper",
		RoomID:    args.RoomID,
		ActorID:   "autodm",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := b.roomManager.SendCommand(ctx, args.RoomID, cmd); err != nil {
		return nil, fmt.Errorf("send whisper: %w", err)
	}

	return json.Marshal(map[string]string{"status": "sent", "to": args.ToUserID})
}

func (b *RoomBridge) RequestPlayerAction(ctx context.Context, args RequestActionArgs) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"user_id":     args.UserID,
		"action_type": args.ActionType,
		"deadline":    time.Now().Add(args.Deadline).Unix(),
		"prompt":      args.Prompt,
	})

	cmd := Command{
		ID:        uuid.NewString(),
		Type:      "request_action",
		RoomID:    args.RoomID,
		ActorID:   "autodm",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := b.roomManager.SendCommand(ctx, args.RoomID, cmd); err != nil {
		return nil, fmt.Errorf("request action: %w", err)
	}

	return json.Marshal(map[string]string{"status": "requested", "user_id": args.UserID})
}

func (b *RoomBridge) StartVote(ctx context.Context, args StartVoteArgs) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"nominee":  args.Target,
		"deadline": time.Now().Add(args.Deadline).Unix(),
	})

	cmd := Command{
		ID:        uuid.NewString(),
		Type:      "nominate",
		RoomID:    args.RoomID,
		ActorID:   "autodm",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := b.roomManager.SendCommand(ctx, args.RoomID, cmd); err != nil {
		return nil, fmt.Errorf("start vote: %w", err)
	}

	return json.Marshal(map[string]string{"status": "started", "target": args.Target})
}

func (b *RoomBridge) CollectVotes(ctx context.Context, roomID string) (json.RawMessage, error) {
	state, err := b.roomManager.GetState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	if state.Nomination == nil {
		return json.Marshal(map[string]interface{}{
			"active": false,
		})
	}

	return json.Marshal(map[string]interface{}{
		"active":    true,
		"nominee":   state.Nomination.Nominee,
		"nominator": state.Nomination.Nominator,
		"votes":     state.Nomination.Votes,
		"resolved":  state.Nomination.Resolved,
	})
}

func (b *RoomBridge) CloseVote(ctx context.Context, roomID string) (json.RawMessage, error) {
	state, err := b.roomManager.GetState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	if state.Nomination == nil || state.Nomination.Resolved {
		return json.Marshal(map[string]string{"status": "no_active_vote"})
	}

	// Count votes
	yesCount := 0
	aliveCount := 0
	for _, player := range state.Players {
		if player.Alive {
			aliveCount++
		}
	}
	for _, vote := range state.Nomination.Votes {
		if vote {
			yesCount++
		}
	}

	result := "not_executed"
	if yesCount*2 >= aliveCount {
		result = "executed"
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"result":      result,
		"yes_votes":   yesCount,
		"alive_count": aliveCount,
	})

	cmd := Command{
		ID:        uuid.NewString(),
		Type:      "close_vote",
		RoomID:    roomID,
		ActorID:   "autodm",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := b.roomManager.SendCommand(ctx, roomID, cmd); err != nil {
		return nil, fmt.Errorf("close vote: %w", err)
	}

	return json.Marshal(map[string]string{"status": "closed", "result": result})
}

func (b *RoomBridge) AdvancePhase(ctx context.Context, args AdvancePhaseArgs) (json.RawMessage, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"phase":  args.NextPhase, // FIX-1: engine reads "phase" not "next_phase"
		"reason": args.Reason,
	})

	cmd := Command{
		ID:        uuid.NewString(),
		Type:      "advance_phase",
		RoomID:    args.RoomID,
		ActorID:   "autodm",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := b.roomManager.SendCommand(ctx, args.RoomID, cmd); err != nil {
		return nil, fmt.Errorf("advance phase: %w", err)
	}

	return json.Marshal(map[string]string{"status": "advanced", "phase": string(args.NextPhase)})
}

func (b *RoomBridge) SetTimer(ctx context.Context, args SetTimerArgs) (json.RawMessage, error) {
	deadline := time.Now().Add(args.Duration)

	payload, _ := json.Marshal(map[string]interface{}{
		"timer_type": args.TimerType,
		"deadline":   deadline.Unix(),
	})

	cmd := Command{
		ID:        uuid.NewString(),
		Type:      "set_timer",
		RoomID:    args.RoomID,
		ActorID:   "autodm",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	if err := b.roomManager.SendCommand(ctx, args.RoomID, cmd); err != nil {
		return nil, fmt.Errorf("set timer: %w", err)
	}

	return json.Marshal(map[string]interface{}{
		"status":     "set",
		"timer_type": args.TimerType,
		"deadline":   deadline,
	})
}

func (b *RoomBridge) TTSNarrate(ctx context.Context, text, voice string) (json.RawMessage, error) {
	// Stub implementation - would integrate with TTS service
	return json.Marshal(map[string]interface{}{
		"status":    "stub",
		"text":      text,
		"voice":     voice,
		"audio_url": "",
		"message":   "TTS not implemented - would generate audio here",
	})
}

// MockRoomManager is a mock implementation for testing
type MockRoomManager struct {
	state    *RoomState
	events   []GameEvent
	commands []Command
}

// NewMockRoomManager creates a mock room manager
func NewMockRoomManager() *MockRoomManager {
	return &MockRoomManager{
		state: &RoomState{
			RoomID:   "test-room",
			Phase:    PhaseLobby,
			Players:  make(map[string]PlayerState),
			Timers:   make(map[string]time.Time),
			Metadata: make(map[string]interface{}),
		},
		events:   make([]GameEvent, 0),
		commands: make([]Command, 0),
	}
}

func (m *MockRoomManager) GetState(ctx context.Context, roomID string) (*RoomState, error) {
	return m.state, nil
}

func (m *MockRoomManager) GetEvents(ctx context.Context, roomID string, afterSeq int64, limit int) ([]GameEvent, error) {
	var result []GameEvent
	for _, e := range m.events {
		if e.Seq > afterSeq {
			result = append(result, e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *MockRoomManager) SendCommand(ctx context.Context, roomID string, cmd Command) error {
	m.commands = append(m.commands, cmd)

	// Simulate event generation
	event := GameEvent{
		RoomID:    roomID,
		Seq:       int64(len(m.events) + 1),
		EventID:   uuid.NewString(),
		EventType: cmd.Type,
		ActorID:   cmd.ActorID,
		Payload:   cmd.Payload,
		Timestamp: time.Now(),
	}
	m.events = append(m.events, event)

	// Update state based on command
	switch cmd.Type {
	case "advance_phase":
		var payload struct {
			NextPhase Phase `json:"next_phase"`
		}
		json.Unmarshal(cmd.Payload, &payload)
		m.state.Phase = payload.NextPhase
		if payload.NextPhase == PhaseDay {
			m.state.DayCount++
		} else if payload.NextPhase == PhaseNight {
			m.state.NightCount++
		}
	}

	return nil
}

// AddPlayer adds a player to the mock state
func (m *MockRoomManager) AddPlayer(userID string, role string, alive bool) {
	m.state.Players[userID] = PlayerState{
		UserID: userID,
		Role:   role,
		Alive:  alive,
	}
}

// SetPhase sets the game phase
func (m *MockRoomManager) SetPhase(phase Phase) {
	m.state.Phase = phase
}

// AddEvent adds an event
func (m *MockRoomManager) AddEvent(eventType string, actorID string, payload map[string]string) {
	payloadJSON, _ := json.Marshal(payload)
	m.events = append(m.events, GameEvent{
		RoomID:    m.state.RoomID,
		Seq:       int64(len(m.events) + 1),
		EventID:   uuid.NewString(),
		EventType: eventType,
		ActorID:   actorID,
		Payload:   payloadJSON,
		Timestamp: time.Now(),
	})
	m.state.LastSeq = int64(len(m.events))
}

// GetCommands returns all sent commands
func (m *MockRoomManager) GetCommands() []Command {
	return m.commands
}
