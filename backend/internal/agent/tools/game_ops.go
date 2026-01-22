// Package tools provides game operation tools for the Auto-DM agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// GameCommander executes game commands.
type GameCommander interface {
	SendMessage(ctx context.Context, roomID, message string) error
	KillPlayer(ctx context.Context, roomID, playerID string) error
	RevivePlayer(ctx context.Context, roomID, playerID string) error
	SetPhase(ctx context.Context, roomID, phase string) error
	StartVote(ctx context.Context, roomID, nominatorID, nomineeID string) error
	EndVote(ctx context.Context, roomID string) error
	RevealRole(ctx context.Context, roomID, playerID, role string) error
	AssignRole(ctx context.Context, roomID, playerID, role string) error
	SetReminder(ctx context.Context, roomID, playerID, reminder string) error
	EndGame(ctx context.Context, roomID, winner string) error
	GetPlayers(ctx context.Context, roomID string) ([]PlayerInfo, error)
	GetGameState(ctx context.Context, roomID string) (*GameInfo, error)
}

// PlayerInfo contains player information.
type PlayerInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	IsAlive  bool   `json:"is_alive"`
	Seat     int    `json:"seat"`
	HasVoted bool   `json:"has_voted"`
}

// GameInfo contains game information.
type GameInfo struct {
	RoomID     string       `json:"room_id"`
	Phase      string       `json:"phase"`
	DayNumber  int          `json:"day_number"`
	Players    []PlayerInfo `json:"players"`
	Edition    string       `json:"edition"`
	IsStarted  bool         `json:"is_started"`
	IsFinished bool         `json:"is_finished"`
}

// RegisterGameTools registers all game operation tools.
func RegisterGameTools(registry *Registry, commander GameCommander, roomID string) {
	registry.Register(
		"send_message",
		"Send a message to all players in the room",
		NewParamSchema().AddString("message", "The message to send", true),
		func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", err
			}
			if err := commander.SendMessage(ctx, roomID, params.Message); err != nil {
				return "", err
			}
			return fmt.Sprintf("Message sent: %s", params.Message), nil
		},
	)

	registry.Register(
		"kill_player",
		"Mark a player as dead",
		NewParamSchema().AddString("player_id", "ID of the player to kill", true),
		func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				PlayerID string `json:"player_id"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", err
			}
			if err := commander.KillPlayer(ctx, roomID, params.PlayerID); err != nil {
				return "", err
			}
			return fmt.Sprintf("Player %s marked as dead", params.PlayerID), nil
		},
	)

	registry.Register(
		"set_phase",
		"Change the current game phase",
		NewParamSchema().AddEnum("phase", "The new phase",
			[]string{"setup", "night", "day", "nomination", "vote", "execution", "end"}, true),
		func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Phase string `json:"phase"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", err
			}
			if err := commander.SetPhase(ctx, roomID, params.Phase); err != nil {
				return "", err
			}
			return fmt.Sprintf("Phase changed to: %s", params.Phase), nil
		},
	)

	registry.Register(
		"get_game_state",
		"Get the current game state",
		NewParamSchema(),
		func(ctx context.Context, args json.RawMessage) (string, error) {
			state, err := commander.GetGameState(ctx, roomID)
			if err != nil {
				return "", err
			}
			data, _ := json.Marshal(state)
			return string(data), nil
		},
	)
}

// RulesProvider provides game rules information.
type RulesProvider interface {
	GetRoleInfo(role string) (string, error)
	GetNightOrder(roles []string, isFirstNight bool) ([]string, error)
	SearchRules(query string) ([]string, error)
}

// RegisterInfoTools registers information query tools.
func RegisterInfoTools(registry *Registry, rules RulesProvider) {
	registry.Register(
		"get_role_info",
		"Get detailed information about a specific role",
		NewParamSchema().AddString("role", "Name of the role to look up", true),
		func(ctx context.Context, args json.RawMessage) (string, error) {
			var params struct {
				Role string `json:"role"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", err
			}
			return rules.GetRoleInfo(params.Role)
		},
	)
}
