// Package mcp implements the Model Context Protocol for standardized tool access.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ToolDefinition defines a tool's schema and metadata.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]ParamSchema `json:"parameters"`
	Required    []string               `json:"required,omitempty"`
	Returns     *ParamSchema           `json:"returns,omitempty"`
	Category    ToolCategory           `json:"category"`
	Async       bool                   `json:"async"`
}

// ParamSchema defines a parameter's JSON Schema.
type ParamSchema struct {
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Items       *ParamSchema           `json:"items,omitempty"`
	Properties  map[string]ParamSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Minimum     *float64               `json:"minimum,omitempty"`
	Maximum     *float64               `json:"maximum,omitempty"`
	MinLength   *int                   `json:"minLength,omitempty"`
	MaxLength   *int                   `json:"maxLength,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
}

// ToolCategory categorizes tools for organization.
type ToolCategory string

const (
	CategoryGameControl   ToolCategory = "game_control"
	CategoryCommunication ToolCategory = "communication"
	CategoryInformation   ToolCategory = "information"
	CategoryModeration    ToolCategory = "moderation"
)

// ToolCall represents a request to invoke a tool.
type ToolCall struct {
	ID         string          `json:"id"`
	ToolName   string          `json:"tool_name"`
	Parameters json.RawMessage `json:"parameters"`
	Timestamp  int64           `json:"timestamp"`
}

// ToolResult represents the result of a tool invocation.
type ToolResult struct {
	CallID    string      `json:"call_id"`
	ToolName  string      `json:"tool_name"`
	Success   bool        `json:"success"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
	TaskID    string      `json:"task_id,omitempty"`
}

// AsyncTask represents a long-running task.
type AsyncTask struct {
	ID        string      `json:"id"`
	ToolName  string      `json:"tool_name"`
	Status    TaskStatus  `json:"status"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	CreatedAt int64       `json:"created_at"`
	UpdatedAt int64       `json:"updated_at"`
}

// TaskStatus represents the status of an async task.
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// ToolHandler is a function that handles a tool call.
type ToolHandler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// Registry manages tool definitions and handlers.
type Registry struct {
	mu       sync.RWMutex
	tools    map[string]ToolDefinition
	handlers map[string]ToolHandler
	tasks    map[string]*AsyncTask
	taskCh   chan *AsyncTask
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:    make(map[string]ToolDefinition),
		handlers: make(map[string]ToolHandler),
		tasks:    make(map[string]*AsyncTask),
		taskCh:   make(chan *AsyncTask, 100),
	}
}

// Register registers a tool with its handler.
func (r *Registry) Register(def ToolDefinition, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[def.Name]; exists {
		return fmt.Errorf("tool already registered: %s", def.Name)
	}
	r.tools[def.Name] = def
	r.handlers[def.Name] = handler
	return nil
}

// GetTool returns a tool definition by name.
func (r *Registry) GetTool(name string) (ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// ListTools returns all registered tool definitions.
func (r *Registry) ListTools() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ListToolsByCategory returns tools filtered by category.
func (r *Registry) ListToolsByCategory(category ToolCategory) []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var tools []ToolDefinition
	for _, t := range r.tools {
		if t.Category == category {
			tools = append(tools, t)
		}
	}
	return tools
}

// Invoke calls a tool with the given parameters.
func (r *Registry) Invoke(ctx context.Context, call ToolCall) *ToolResult {
	r.mu.RLock()
	def, defOk := r.tools[call.ToolName]
	handler, handlerOk := r.handlers[call.ToolName]
	r.mu.RUnlock()

	if !defOk || !handlerOk {
		return &ToolResult{
			CallID:    call.ID,
			ToolName:  call.ToolName,
			Success:   false,
			Error:     fmt.Sprintf("tool not found: %s", call.ToolName),
			Timestamp: time.Now().UnixMilli(),
		}
	}

	if err := r.validateParams(def, call.Parameters); err != nil {
		return &ToolResult{
			CallID:    call.ID,
			ToolName:  call.ToolName,
			Success:   false,
			Error:     fmt.Sprintf("parameter validation failed: %v", err),
			Timestamp: time.Now().UnixMilli(),
		}
	}

	if def.Async {
		return r.invokeAsync(ctx, call, handler)
	}

	result, err := handler(ctx, call.Parameters)
	if err != nil {
		return &ToolResult{
			CallID:    call.ID,
			ToolName:  call.ToolName,
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now().UnixMilli(),
		}
	}

	return &ToolResult{
		CallID:    call.ID,
		ToolName:  call.ToolName,
		Success:   true,
		Result:    result,
		Timestamp: time.Now().UnixMilli(),
	}
}

func (r *Registry) invokeAsync(ctx context.Context, call ToolCall, handler ToolHandler) *ToolResult {
	taskID := uuid.NewString()
	task := &AsyncTask{
		ID:        taskID,
		ToolName:  call.ToolName,
		Status:    TaskPending,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}
	r.mu.Lock()
	r.tasks[taskID] = task
	r.mu.Unlock()

	go func() {
		r.mu.Lock()
		task.Status = TaskRunning
		task.UpdatedAt = time.Now().UnixMilli()
		r.mu.Unlock()

		result, err := handler(ctx, call.Parameters)

		r.mu.Lock()
		if err != nil {
			task.Error = err.Error()
			task.Status = TaskFailed
		} else {
			task.Result = result
			task.Status = TaskCompleted
		}
		task.UpdatedAt = time.Now().UnixMilli()
		r.mu.Unlock()

		select {
		case r.taskCh <- task:
		default:
		}
	}()

	return &ToolResult{
		CallID:    call.ID,
		ToolName:  call.ToolName,
		Success:   true,
		TaskID:    taskID,
		Timestamp: time.Now().UnixMilli(),
	}
}

// GetTask returns an async task by ID.
func (r *Registry) GetTask(taskID string) (*AsyncTask, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	task, ok := r.tasks[taskID]
	return task, ok
}

// TaskChannel returns the channel for task completion notifications.
func (r *Registry) TaskChannel() <-chan *AsyncTask {
	return r.taskCh
}

func (r *Registry) validateParams(def ToolDefinition, params json.RawMessage) error {
	var paramMap map[string]interface{}
	if err := json.Unmarshal(params, &paramMap); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	for _, req := range def.Required {
		if _, ok := paramMap[req]; !ok {
			return fmt.Errorf("missing required parameter: %s", req)
		}
	}

	for name, schema := range def.Parameters {
		if val, ok := paramMap[name]; ok {
			if err := validateValue(name, val, schema); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateValue(name string, val interface{}, schema ParamSchema) error {
	switch schema.Type {
	case "string":
		s, ok := val.(string)
		if !ok {
			return fmt.Errorf("%s: expected string", name)
		}
		if len(schema.Enum) > 0 {
			found := false
			for _, e := range schema.Enum {
				if s == e {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("%s: value not in enum", name)
			}
		}
		if schema.MinLength != nil && len(s) < *schema.MinLength {
			return fmt.Errorf("%s: string too short", name)
		}
		if schema.MaxLength != nil && len(s) > *schema.MaxLength {
			return fmt.Errorf("%s: string too long", name)
		}
	case "number", "integer":
		var n float64
		switch v := val.(type) {
		case float64:
			n = v
		case int:
			n = float64(v)
		case int64:
			n = float64(v)
		default:
			return fmt.Errorf("%s: expected number", name)
		}
		if schema.Minimum != nil && n < *schema.Minimum {
			return fmt.Errorf("%s: value below minimum", name)
		}
		if schema.Maximum != nil && n > *schema.Maximum {
			return fmt.Errorf("%s: value above maximum", name)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("%s: expected boolean", name)
		}
	case "array":
		arr, ok := val.([]interface{})
		if !ok {
			return fmt.Errorf("%s: expected array", name)
		}
		if schema.Items != nil {
			for i, item := range arr {
				if err := validateValue(fmt.Sprintf("%s[%d]", name, i), item, *schema.Items); err != nil {
					return err
				}
			}
		}
	case "object":
		obj, ok := val.(map[string]interface{})
		if !ok {
			return fmt.Errorf("%s: expected object", name)
		}
		for propName, propSchema := range schema.Properties {
			if propVal, exists := obj[propName]; exists {
				if err := validateValue(propName, propVal, propSchema); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// AuditLog represents an audit log entry for tool calls.
type AuditLog struct {
	ID        string          `json:"id"`
	Timestamp int64           `json:"timestamp"`
	RoomID    string          `json:"room_id"`
	AgentID   string          `json:"agent_id"`
	ToolName  string          `json:"tool_name"`
	Params    json.RawMessage `json:"params"`
	Result    *ToolResult     `json:"result"`
	Duration  int64           `json:"duration_ms"`
}

// Auditor records tool call audit logs.
type Auditor struct {
	mu   sync.Mutex
	logs []AuditLog
}

// NewAuditor creates a new auditor.
func NewAuditor() *Auditor {
	return &Auditor{logs: make([]AuditLog, 0)}
}

// Record records a tool call.
func (a *Auditor) Record(roomID, agentID string, call ToolCall, result *ToolResult, duration time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logs = append(a.logs, AuditLog{
		ID:        uuid.NewString(),
		Timestamp: time.Now().UnixMilli(),
		RoomID:    roomID,
		AgentID:   agentID,
		ToolName:  call.ToolName,
		Params:    call.Parameters,
		Result:    result,
		Duration:  duration.Milliseconds(),
	})
}

// GetLogs returns audit logs, optionally filtered.
func (a *Auditor) GetLogs(roomID string, limit int) []AuditLog {
	a.mu.Lock()
	defer a.mu.Unlock()
	var logs []AuditLog
	for i := len(a.logs) - 1; i >= 0 && len(logs) < limit; i-- {
		if roomID == "" || a.logs[i].RoomID == roomID {
			logs = append(logs, a.logs[i])
		}
	}
	return logs
}
