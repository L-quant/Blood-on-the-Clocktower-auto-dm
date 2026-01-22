// Package tools provides a registry for agent tools.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/qingchang/Blood-on-the-Clocktower-auto-dm/internal/agent/llm"
)

// Tool is a function the agent can call.
type Tool struct {
	Definition llm.Tool
	Handler    ToolHandler
}

// ToolHandler processes a tool call and returns a result.
type ToolHandler func(ctx context.Context, args json.RawMessage) (string, error)

// Registry manages available tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(name, description string, params interface{}, handler ToolHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	paramBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal params: %w", err)
	}

	r.tools[name] = Tool{
		Definition: llm.Tool{
			Type: "function",
			Function: llm.ToolFunction{
				Name:        name,
				Description: description,
				Parameters:  paramBytes,
			},
		},
		Handler: handler,
	}

	return nil
}

// Definitions returns all tool definitions for LLM.
func (r *Registry) Definitions() []llm.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	defs := make([]llm.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition)
	}
	return defs
}

// Execute runs a tool by name with the given arguments.
func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	r.mu.RLock()
	tool, ok := r.tools[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	return tool.Handler(ctx, args)
}

// ParamSchema helps build JSON Schema for parameters.
type ParamSchema struct {
	Type       string                `json:"type"`
	Properties map[string]PropSchema `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
}

// PropSchema defines a property in the schema.
type PropSchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// NewParamSchema creates a new parameter schema.
func NewParamSchema() *ParamSchema {
	return &ParamSchema{
		Type:       "object",
		Properties: make(map[string]PropSchema),
	}
}

// AddString adds a string property.
func (p *ParamSchema) AddString(name, description string, required bool) *ParamSchema {
	p.Properties[name] = PropSchema{Type: "string", Description: description}
	if required {
		p.Required = append(p.Required, name)
	}
	return p
}

// AddNumber adds a number property.
func (p *ParamSchema) AddNumber(name, description string, required bool) *ParamSchema {
	p.Properties[name] = PropSchema{Type: "number", Description: description}
	if required {
		p.Required = append(p.Required, name)
	}
	return p
}

// AddBoolean adds a boolean property.
func (p *ParamSchema) AddBoolean(name, description string, required bool) *ParamSchema {
	p.Properties[name] = PropSchema{Type: "boolean", Description: description}
	if required {
		p.Required = append(p.Required, name)
	}
	return p
}

// AddEnum adds an enum property.
func (p *ParamSchema) AddEnum(name, description string, values []string, required bool) *ParamSchema {
	p.Properties[name] = PropSchema{Type: "string", Description: description, Enum: values}
	if required {
		p.Required = append(p.Required, name)
	}
	return p
}
