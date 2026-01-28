package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ModelRouter routes LLM requests to appropriate models based on task type.
type ModelRouter struct {
	mu       sync.RWMutex
	clients  map[string]*LLMClient
	defaultC string
	routing  map[string]string
}

// NewModelRouter creates a new model router.
func NewModelRouter(defaultClient string) *ModelRouter {
	return &ModelRouter{
		clients:  make(map[string]*LLMClient),
		defaultC: defaultClient,
		routing:  make(map[string]string),
	}
}

// RegisterClient registers an LLM client.
func (r *ModelRouter) RegisterClient(name string, client *LLMClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[name] = client
}

// SetRouting sets task type to client routing.
func (r *ModelRouter) SetRouting(taskType, clientName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routing[taskType] = clientName
}

// Chat routes a chat request to the appropriate model.
func (r *ModelRouter) Chat(ctx context.Context, taskType string, messages []Message, tools []Tool) (*ChatResponse, error) {
	r.mu.RLock()
	clientName := r.routing[taskType]
	if clientName == "" {
		clientName = r.defaultC
	}
	client := r.clients[clientName]
	r.mu.RUnlock()

	if client == nil {
		return nil, errors.New("no client configured")
	}

	return client.Chat(ctx, messages, tools)
}

// SimpleChat performs a simple chat without tools.
func (r *ModelRouter) SimpleChat(ctx context.Context, taskType, systemPrompt, userPrompt string) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	resp, err := r.Chat(ctx, taskType, messages, nil)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("no response choices")
	}
	return resp.Choices[0].Message.Content, nil
}

// ToolRegistry manages available tools.
type ToolRegistry struct {
	mu       sync.RWMutex
	tools    map[string]Tool
	handlers map[string]ToolHandlerFunc
}

// ToolHandlerFunc is a function that handles a tool call.
type ToolHandlerFunc func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolHandlerFunc),
	}
}

// Register registers a tool.
func (r *ToolRegistry) Register(tool Tool, handler ToolHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Function.Name] = tool
	r.handlers[tool.Function.Name] = handler
}

// GetTools returns all registered tools.
func (r *ToolRegistry) GetTools() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Execute executes a tool by name.
func (r *ToolRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (json.RawMessage, error) {
	r.mu.RLock()
	handler, ok := r.handlers[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return handler(ctx, args)
}

// ToolResult wraps execution result with metadata.
type ToolResult struct {
	Output   json.RawMessage `json:"output"`
	Success  bool            `json:"success"`
	Error    string          `json:"error,omitempty"`
	Duration int64           `json:"duration_ms"`
}

// ExecuteRaw executes a tool and returns a ToolResult.
func (r *ToolRegistry) ExecuteRaw(ctx context.Context, name string, args json.RawMessage) (*ToolResult, error) {
	start := time.Now()
	output, err := r.Execute(ctx, name, args)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return &ToolResult{
			Success:  false,
			Error:    err.Error(),
			Duration: duration,
		}, err
	}

	return &ToolResult{
		Output:   output,
		Success:  true,
		Duration: duration,
	}, nil
}

// OpenAIProvider is a client for OpenAI-compatible APIs.
type OpenAIProvider struct {
	client *LLMClient
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(cfg LLMConfig) *OpenAIProvider {
	return &OpenAIProvider{
		client: NewLLMClient(cfg),
	}
}

// Chat sends a chat completion request.
func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message, tools []Tool) (*ChatResponse, error) {
	return p.client.Chat(ctx, messages, tools)
}
