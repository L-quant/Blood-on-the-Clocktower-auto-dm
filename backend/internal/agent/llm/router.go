// Package llm provides model routing for different task types.
package llm

import (
	"context"
	"fmt"
	"sync"
)

// TaskType represents the type of task for model routing.
type TaskType string

const (
	TaskReasoning TaskType = "reasoning"
	TaskNarration TaskType = "narration"
	TaskRules     TaskType = "rules"
	TaskSummarize TaskType = "summarize"
	TaskQuick     TaskType = "quick"
	TaskDefault   TaskType = "default"
)

// Router routes requests to appropriate models based on task type.
type Router struct {
	mu       sync.RWMutex
	models   map[TaskType]Provider
	fallback Provider
}

// NewRouter creates a new model router.
func NewRouter(defaultCfg Config) *Router {
	return &Router{
		models:   make(map[TaskType]Provider),
		fallback: NewClient(defaultCfg),
	}
}

// RegisterModel registers a model for a specific task type.
func (r *Router) RegisterModel(taskType TaskType, cfg Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[taskType] = NewClient(cfg)
}

// GetClient returns the appropriate client for a task type.
func (r *Router) GetClient(taskType TaskType) Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if client, ok := r.models[taskType]; ok {
		return client
	}
	return r.fallback
}

// Chat routes a chat request to the appropriate model.
func (r *Router) Chat(ctx context.Context, taskType TaskType, messages []Message, tools []Tool) (*ChatResponse, error) {
	client := r.GetClient(taskType)
	return client.Chat(ctx, messages, tools)
}

// SimpleChat routes a simple chat to the appropriate model.
func (r *Router) SimpleChat(ctx context.Context, taskType TaskType, systemPrompt, userMessage string) (string, error) {
	client := r.GetClient(taskType)
	return client.SimpleChat(ctx, systemPrompt, userMessage)
}

// ModelInfo returns info about which model is used for a task.
func (r *Router) ModelInfo(taskType TaskType) string {
	client := r.GetClient(taskType)
	return fmt.Sprintf("task=%s model=%s", taskType, client.Model())
}

// RoutingConfig defines the complete routing configuration.
type RoutingConfig struct {
	Default   Config
	Reasoning Config
	Narration Config
	Quick     Config
}

// NewRouterFromConfig creates a router with full configuration.
func NewRouterFromConfig(cfg RoutingConfig) *Router {
	router := NewRouter(cfg.Default)

	if cfg.Reasoning.Model != "" {
		router.RegisterModel(TaskReasoning, cfg.Reasoning)
	}
	if cfg.Narration.Model != "" {
		router.RegisterModel(TaskNarration, cfg.Narration)
	}
	if cfg.Quick.Model != "" {
		router.RegisterModel(TaskQuick, cfg.Quick)
		router.RegisterModel(TaskSummarize, cfg.Quick)
		router.RegisterModel(TaskRules, cfg.Quick)
	}

	return router
}

// SingleModelRouter creates a router that uses one model for all tasks.
func SingleModelRouter(cfg Config) *Router {
	return NewRouter(cfg)
}
