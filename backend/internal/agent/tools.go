package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/engine"
	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// ToolExecutor defines how a tool is executed.
type ToolExecutor func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)

// GameTools provides tools for the AutoDM to interact with the game.
type GameTools struct {
	dispatcher CommandDispatcher
	state      func() engine.State
}

// NewGameTools creates a new GameTools instance.
func NewGameTools(dispatcher CommandDispatcher, stateGetter func() engine.State) *GameTools {
	return &GameTools{
		dispatcher: dispatcher,
		state:      stateGetter,
	}
}

// GetToolDefinitions returns the tool definitions for the LLM.
func (t *GameTools) GetToolDefinitions() []Tool {
	return []Tool{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "send_public_message",
				Description: "Send a public message to all players in the room. Use this for announcements, narration, and game information.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"message": {
							"type": "string",
							"description": "The message to send to all players"
						}
					},
					"required": ["message"]
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "send_whisper",
				Description: "Send a private message to a specific player. Use this for role information, night ability results, or private hints.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"to_user_id": {
							"type": "string",
							"description": "The user ID of the recipient"
						},
						"message": {
							"type": "string",
							"description": "The private message content"
						}
					},
					"required": ["to_user_id", "message"]
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "advance_phase",
				Description: "Advance the game to the next phase. Call this when the current phase activities are complete.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"to_phase": {
							"type": "string",
							"enum": ["day", "night", "ended"],
							"description": "The phase to advance to"
						},
						"reason": {
							"type": "string",
							"description": "Explanation for the phase transition"
						}
					},
					"required": ["to_phase"]
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "start_nomination",
				Description: "Announce that nominations are now open. Call this during the day phase.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"announcement": {
							"type": "string",
							"description": "The nomination opening announcement"
						}
					},
					"required": ["announcement"]
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "resolve_execution",
				Description: "Resolve an execution after voting completes. Announce the result and any consequences.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"executed_user_id": {
							"type": "string",
							"description": "The user ID of the executed player, or empty if no execution"
						},
						"announcement": {
							"type": "string",
							"description": "The execution result announcement"
						}
					},
					"required": ["announcement"]
				}`),
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "end_game",
				Description: "End the game and announce the winner.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"winner": {
							"type": "string",
							"enum": ["good", "evil"],
							"description": "The winning team"
						},
						"announcement": {
							"type": "string",
							"description": "The game ending announcement with explanation"
						}
					},
					"required": ["winner", "announcement"]
				}`),
			},
		},
	}
}

// ExecuteTool executes a tool call and returns the result.
func (t *GameTools) ExecuteTool(ctx context.Context, toolName string, args json.RawMessage, roomID string) (json.RawMessage, error) {
	switch toolName {
	case "send_public_message":
		return t.sendPublicMessage(ctx, args, roomID)
	case "send_whisper":
		return t.sendWhisper(ctx, args, roomID)
	case "advance_phase":
		return t.advancePhase(ctx, args, roomID)
	case "start_nomination":
		return t.startNomination(ctx, args, roomID)
	case "resolve_execution":
		return t.resolveExecution(ctx, args, roomID)
	case "end_game":
		return t.endGame(ctx, args, roomID)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (t *GameTools) sendPublicMessage(ctx context.Context, args json.RawMessage, roomID string) (json.RawMessage, error) {
	var a struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	cmd := types.CommandEnvelope{
		RoomID:      roomID,
		Type:        "public_chat",
		ActorUserID: "autodm",
		Payload:     json.RawMessage(fmt.Sprintf(`{"message": %q}`, a.Message)),
	}

	if err := t.dispatcher.DispatchAsync(cmd); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]string{"status": "sent"})
}

func (t *GameTools) sendWhisper(ctx context.Context, args json.RawMessage, roomID string) (json.RawMessage, error) {
	var a struct {
		ToUserID string `json:"to_user_id"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	cmd := types.CommandEnvelope{
		RoomID:      roomID,
		Type:        "whisper",
		ActorUserID: "autodm",
		Payload:     json.RawMessage(fmt.Sprintf(`{"to_user_id": %q, "message": %q}`, a.ToUserID, a.Message)),
	}

	if err := t.dispatcher.DispatchAsync(cmd); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]string{"status": "sent"})
}

func (t *GameTools) advancePhase(ctx context.Context, args json.RawMessage, roomID string) (json.RawMessage, error) {
	var a struct {
		ToPhase string `json:"to_phase"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	cmd := types.CommandEnvelope{
		RoomID:      roomID,
		Type:        "advance_phase",
		ActorUserID: "autodm",
		Payload:     json.RawMessage(fmt.Sprintf(`{"phase": %q, "reason": %q}`, a.ToPhase, a.Reason)),
	}

	if err := t.dispatcher.DispatchAsync(cmd); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]string{"status": "phase_advanced", "to": a.ToPhase})
}

func (t *GameTools) startNomination(ctx context.Context, args json.RawMessage, roomID string) (json.RawMessage, error) {
	var a struct {
		Announcement string `json:"announcement"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	// First send announcement
	cmd := types.CommandEnvelope{
		RoomID:      roomID,
		Type:        "public_chat",
		ActorUserID: "autodm",
		Payload:     json.RawMessage(fmt.Sprintf(`{"message": %q}`, a.Announcement)),
	}

	if err := t.dispatcher.DispatchAsync(cmd); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]string{"status": "nominations_open"})
}

func (t *GameTools) resolveExecution(ctx context.Context, args json.RawMessage, roomID string) (json.RawMessage, error) {
	var a struct {
		ExecutedUserID string `json:"executed_user_id"`
		Announcement   string `json:"announcement"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	// Send announcement
	cmd := types.CommandEnvelope{
		RoomID:      roomID,
		Type:        "public_chat",
		ActorUserID: "autodm",
		Payload:     json.RawMessage(fmt.Sprintf(`{"message": %q}`, a.Announcement)),
	}

	if err := t.dispatcher.DispatchAsync(cmd); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]string{"status": "resolved", "executed": a.ExecutedUserID})
}

func (t *GameTools) endGame(ctx context.Context, args json.RawMessage, roomID string) (json.RawMessage, error) {
	var a struct {
		Winner       string `json:"winner"`
		Announcement string `json:"announcement"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, err
	}

	// Send announcement
	cmd := types.CommandEnvelope{
		RoomID:      roomID,
		Type:        "public_chat",
		ActorUserID: "autodm",
		Payload:     json.RawMessage(fmt.Sprintf(`{"message": %q}`, a.Announcement)),
	}

	if err := t.dispatcher.DispatchAsync(cmd); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]string{"status": "game_ended", "winner": a.Winner})
}
