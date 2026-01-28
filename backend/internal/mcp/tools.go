package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/types"
)

// GameToolsConfig configures the game tools.
type GameToolsConfig struct {
	Dispatcher CommandDispatcher
	RoomID     string
}

// CommandDispatcher dispatches commands to the game engine.
type CommandDispatcher interface {
	DispatchAsync(cmd types.CommandEnvelope) error
}

// RegisterGameTools registers all game-related tools.
func RegisterGameTools(registry *Registry, cfg GameToolsConfig) error {
	tools := []struct {
		def     ToolDefinition
		handler ToolHandler
	}{
		{
			def: ToolDefinition{
				Name:        "send_public_message",
				Description: "发送公开消息到房间所有玩家",
				Category:    CategoryCommunication,
				Parameters: map[string]ParamSchema{
					"message": {
						Type:        "string",
						Description: "要发送的消息内容",
						MinLength:   intPtr(1),
						MaxLength:   intPtr(500),
					},
				},
				Required: []string{"message"},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var p struct {
					Message string `json:"message"`
				}
				if err := json.Unmarshal(params, &p); err != nil {
					return nil, err
				}
				cmd := types.CommandEnvelope{
					RoomID:  cfg.RoomID,
					Type:    "public_chat",
					Payload: mustMarshalJSON(map[string]string{"message": p.Message}),
				}
				return nil, cfg.Dispatcher.DispatchAsync(cmd)
			},
		},
		{
			def: ToolDefinition{
				Name:        "send_whisper",
				Description: "发送私聊消息给特定玩家",
				Category:    CategoryCommunication,
				Parameters: map[string]ParamSchema{
					"to_user_id": {
						Type:        "string",
						Description: "接收消息的玩家ID",
					},
					"message": {
						Type:        "string",
						Description: "私聊消息内容",
						MinLength:   intPtr(1),
						MaxLength:   intPtr(500),
					},
				},
				Required: []string{"to_user_id", "message"},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var p struct {
					ToUserID string `json:"to_user_id"`
					Message  string `json:"message"`
				}
				if err := json.Unmarshal(params, &p); err != nil {
					return nil, err
				}
				cmd := types.CommandEnvelope{
					RoomID: cfg.RoomID,
					Type:   "whisper",
					Payload: mustMarshalJSON(map[string]string{
						"to_user_id": p.ToUserID,
						"message":    p.Message,
					}),
				}
				return nil, cfg.Dispatcher.DispatchAsync(cmd)
			},
		},
		{
			def: ToolDefinition{
				Name:        "advance_phase",
				Description: "推进游戏阶段（如从夜晚到白天）",
				Category:    CategoryGameControl,
				Parameters: map[string]ParamSchema{
					"phase": {
						Type:        "string",
						Description: "目标阶段",
						Enum:        []string{"day", "night", "nomination"},
					},
				},
				Required: []string{"phase"},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var p struct {
					Phase string `json:"phase"`
				}
				if err := json.Unmarshal(params, &p); err != nil {
					return nil, err
				}
				cmd := types.CommandEnvelope{
					RoomID:  cfg.RoomID,
					Type:    "advance_phase",
					Payload: mustMarshalJSON(map[string]string{"phase": p.Phase}),
				}
				return nil, cfg.Dispatcher.DispatchAsync(cmd)
			},
		},
		{
			def: ToolDefinition{
				Name:        "request_player_action",
				Description: "请求玩家执行夜间行动",
				Category:    CategoryGameControl,
				Parameters: map[string]ParamSchema{
					"user_id": {
						Type:        "string",
						Description: "玩家ID",
					},
					"action_type": {
						Type:        "string",
						Description: "行动类型",
						Enum:        []string{"select_target", "select_two_targets", "confirm", "choose_yes_no"},
					},
					"prompt": {
						Type:        "string",
						Description: "显示给玩家的提示信息",
					},
					"options": {
						Type:        "array",
						Description: "可选项列表",
						Items: &ParamSchema{
							Type: "string",
						},
					},
					"timeout_seconds": {
						Type:        "integer",
						Description: "超时时间（秒）",
						Minimum:     floatPtr(5),
						Maximum:     floatPtr(120),
					},
				},
				Required: []string{"user_id", "action_type", "prompt"},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				// This would trigger a UI prompt for the player
				return map[string]string{"status": "action_requested"}, nil
			},
		},
		{
			def: ToolDefinition{
				Name:        "deliver_information",
				Description: "向玩家发送夜间获得的信息",
				Category:    CategoryCommunication,
				Parameters: map[string]ParamSchema{
					"user_id": {
						Type:        "string",
						Description: "接收信息的玩家ID",
					},
					"info_type": {
						Type:        "string",
						Description: "信息类型",
						Enum:        []string{"role_info", "number", "yes_no", "player_list"},
					},
					"content": {
						Type:        "string",
						Description: "信息内容",
					},
					"is_false": {
						Type:        "boolean",
						Description: "是否为虚假信息（因中毒/醉酒）",
					},
				},
				Required: []string{"user_id", "info_type", "content"},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var p struct {
					UserID   string `json:"user_id"`
					InfoType string `json:"info_type"`
					Content  string `json:"content"`
					IsFalse  bool   `json:"is_false"`
				}
				if err := json.Unmarshal(params, &p); err != nil {
					return nil, err
				}
				cmd := types.CommandEnvelope{
					RoomID: cfg.RoomID,
					Type:   "whisper",
					Payload: mustMarshalJSON(map[string]string{
						"to_user_id": p.UserID,
						"message":    fmt.Sprintf("[夜间信息] %s", p.Content),
					}),
				}
				return nil, cfg.Dispatcher.DispatchAsync(cmd)
			},
		},
		{
			def: ToolDefinition{
				Name:        "start_nomination_phase",
				Description: "开启提名阶段",
				Category:    CategoryGameControl,
				Parameters: map[string]ParamSchema{
					"timeout_seconds": {
						Type:        "integer",
						Description: "提名超时时间",
						Minimum:     floatPtr(5),
						Maximum:     floatPtr(60),
					},
				},
				Required: []string{},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				cmd := types.CommandEnvelope{
					RoomID:  cfg.RoomID,
					Type:    "advance_phase",
					Payload: mustMarshalJSON(map[string]string{"phase": "nomination"}),
				}
				return nil, cfg.Dispatcher.DispatchAsync(cmd)
			},
		},
		{
			def: ToolDefinition{
				Name:        "resolve_execution",
				Description: "解决处决结果",
				Category:    CategoryGameControl,
				Parameters: map[string]ParamSchema{
					"executed_user_id": {
						Type:        "string",
						Description: "被处决的玩家ID（为空则无人被处决）",
					},
				},
				Required: []string{},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var p struct {
					ExecutedUserID string `json:"executed_user_id"`
				}
				if err := json.Unmarshal(params, &p); err != nil {
					return nil, err
				}
				if p.ExecutedUserID != "" {
					cmd := types.CommandEnvelope{
						RoomID: cfg.RoomID,
						Type:   "resolve_nomination",
						Payload: mustMarshalJSON(map[string]string{
							"result":   "executed",
							"executed": p.ExecutedUserID,
						}),
					}
					return nil, cfg.Dispatcher.DispatchAsync(cmd)
				}
				return map[string]string{"status": "no_execution"}, nil
			},
		},
		{
			def: ToolDefinition{
				Name:        "end_game",
				Description: "结束游戏并宣布获胜方",
				Category:    CategoryGameControl,
				Parameters: map[string]ParamSchema{
					"winner": {
						Type:        "string",
						Description: "获胜方",
						Enum:        []string{"good", "evil"},
					},
					"reason": {
						Type:        "string",
						Description: "获胜原因",
					},
				},
				Required: []string{"winner", "reason"},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var p struct {
					Winner string `json:"winner"`
					Reason string `json:"reason"`
				}
				if err := json.Unmarshal(params, &p); err != nil {
					return nil, err
				}
				// Game end is typically triggered by win conditions
				return map[string]string{
					"status": "game_ended",
					"winner": p.Winner,
					"reason": p.Reason,
				}, nil
			},
		},
		{
			def: ToolDefinition{
				Name:        "write_event",
				Description: "写入自定义游戏事件到事件日志",
				Category:    CategoryModeration,
				Parameters: map[string]ParamSchema{
					"event_type": {
						Type:        "string",
						Description: "事件类型",
					},
					"data": {
						Type:        "object",
						Description: "事件数据",
					},
				},
				Required: []string{"event_type"},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				return map[string]string{"status": "event_written"}, nil
			},
		},
		{
			def: ToolDefinition{
				Name:        "get_game_state",
				Description: "获取当前游戏状态",
				Category:    CategoryInformation,
				Parameters:  map[string]ParamSchema{},
				Required:    []string{},
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				// This would return current game state
				return map[string]string{"status": "state_retrieved"}, nil
			},
		},
		{
			def: ToolDefinition{
				Name:        "query_rules",
				Description: "查询游戏规则",
				Category:    CategoryInformation,
				Parameters: map[string]ParamSchema{
					"query": {
						Type:        "string",
						Description: "规则查询内容",
					},
					"role_id": {
						Type:        "string",
						Description: "特定角色ID（可选）",
					},
				},
				Required: []string{"query"},
				Async:    true, // RAG query is async
			},
			handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
				var p struct {
					Query  string `json:"query"`
					RoleID string `json:"role_id"`
				}
				if err := json.Unmarshal(params, &p); err != nil {
					return nil, err
				}
				// This would trigger RAG query
				return map[string]string{
					"query":  p.Query,
					"status": "searching",
				}, nil
			},
		},
	}

	for _, t := range tools {
		if err := registry.Register(t.def, t.handler); err != nil {
			return err
		}
	}

	return nil
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func intPtr(i int) *int {
	return &i
}

func floatPtr(f float64) *float64 {
	return &f
}
