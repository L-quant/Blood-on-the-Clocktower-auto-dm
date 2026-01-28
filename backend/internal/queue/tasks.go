package queue

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AgentTaskType defines task types for agent operations.
const (
	TaskTypeLLMCall      = "llm_call"
	TaskTypeRAGQuery     = "rag_query"
	TaskTypeNightResolve = "night_resolve"
	TaskTypeGenerateTTS  = "generate_tts"
	TaskTypeSummarize    = "summarize"
)

// LLMCallData represents data for an LLM call task.
type LLMCallData struct {
	SystemPrompt string                 `json:"system_prompt"`
	UserPrompt   string                 `json:"user_prompt"`
	Model        string                 `json:"model"`
	MaxTokens    int                    `json:"max_tokens"`
	Temperature  float64                `json:"temperature"`
	Context      map[string]interface{} `json:"context"`
}

// RAGQueryData represents data for a RAG query task.
type RAGQueryData struct {
	Query  string            `json:"query"`
	Limit  int               `json:"limit"`
	Filter map[string]string `json:"filter"`
}

// NightResolveData represents data for night resolution.
type NightResolveData struct {
	DayNumber int      `json:"day_number"`
	Order     []string `json:"order"` // Role IDs in night order
}

// TaskFactory creates tasks for common operations.
type TaskFactory struct {
	DefaultPriority int
}

// NewTaskFactory creates a task factory.
func NewTaskFactory() *TaskFactory {
	return &TaskFactory{DefaultPriority: 5}
}

// CreateLLMCallTask creates an LLM call task.
func (f *TaskFactory) CreateLLMCallTask(roomID string, data LLMCallData) Task {
	return Task{
		ID:        uuid.New().String(),
		Type:      TaskTypeLLMCall,
		RoomID:    roomID,
		Data:      structToMap(data),
		Priority:  f.DefaultPriority,
		CreatedAt: time.Now(),
		MaxRetry:  2,
	}
}

// CreateRAGQueryTask creates a RAG query task.
func (f *TaskFactory) CreateRAGQueryTask(roomID string, data RAGQueryData) Task {
	return Task{
		ID:        uuid.New().String(),
		Type:      TaskTypeRAGQuery,
		RoomID:    roomID,
		Data:      structToMap(data),
		Priority:  f.DefaultPriority + 1, // Higher priority for RAG
		CreatedAt: time.Now(),
		MaxRetry:  3,
	}
}

// CreateNightResolveTask creates a night resolution task.
func (f *TaskFactory) CreateNightResolveTask(roomID string, data NightResolveData) Task {
	return Task{
		ID:        uuid.New().String(),
		Type:      TaskTypeNightResolve,
		RoomID:    roomID,
		Data:      structToMap(data),
		Priority:  8, // High priority for game flow
		CreatedAt: time.Now(),
		MaxRetry:  1,
	}
}

// CreateSummarizeTask creates a summarize task.
func (f *TaskFactory) CreateSummarizeTask(roomID string, context map[string]interface{}) Task {
	return Task{
		ID:        uuid.New().String(),
		Type:      TaskTypeSummarize,
		RoomID:    roomID,
		Data:      context,
		Priority:  3, // Lower priority
		CreatedAt: time.Now(),
		MaxRetry:  2,
	}
}

func structToMap(v interface{}) map[string]interface{} {
	// Use reflection or encoding/json for a generic solution
	// For simplicity, we'll handle known types
	switch data := v.(type) {
	case LLMCallData:
		return map[string]interface{}{
			"system_prompt": data.SystemPrompt,
			"user_prompt":   data.UserPrompt,
			"model":         data.Model,
			"max_tokens":    data.MaxTokens,
			"temperature":   data.Temperature,
			"context":       data.Context,
		}
	case RAGQueryData:
		return map[string]interface{}{
			"query":  data.Query,
			"limit":  data.Limit,
			"filter": data.Filter,
		}
	case NightResolveData:
		return map[string]interface{}{
			"day_number": data.DayNumber,
			"order":      data.Order,
		}
	default:
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
		return nil
	}
}

// AgentTaskHandlers provides handlers for agent-related tasks.
type AgentTaskHandlers struct {
	LLMHandler   TaskHandler
	RAGHandler   TaskHandler
	NightHandler TaskHandler
	TTSHandler   TaskHandler
}

// RegisterHandlers registers all agent task handlers.
func (h *AgentTaskHandlers) RegisterHandlers(q *Queue) {
	if h.LLMHandler != nil {
		q.RegisterHandler(TaskTypeLLMCall, h.LLMHandler)
	}
	if h.RAGHandler != nil {
		q.RegisterHandler(TaskTypeRAGQuery, h.RAGHandler)
	}
	if h.NightHandler != nil {
		q.RegisterHandler(TaskTypeNightResolve, h.NightHandler)
	}
	if h.TTSHandler != nil {
		q.RegisterHandler(TaskTypeGenerateTTS, h.TTSHandler)
	}
}

// CreateLLMHandler creates a handler for LLM calls.
func CreateLLMHandler(llmClient interface {
	Chat(ctx context.Context, system, user string) (string, error)
}) TaskHandler {
	return func(ctx context.Context, task Task) (map[string]interface{}, error) {
		systemPrompt, _ := task.Data["system_prompt"].(string)
		userPrompt, _ := task.Data["user_prompt"].(string)

		response, err := llmClient.Chat(ctx, systemPrompt, userPrompt)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"response": response,
		}, nil
	}
}

// CreateRAGHandler creates a handler for RAG queries.
func CreateRAGHandler(retriever interface {
	Retrieve(ctx context.Context, query string, limit int) ([]interface{}, error)
}) TaskHandler {
	return func(ctx context.Context, task Task) (map[string]interface{}, error) {
		query, _ := task.Data["query"].(string)
		limit := 5
		if l, ok := task.Data["limit"].(float64); ok {
			limit = int(l)
		}

		results, err := retriever.Retrieve(ctx, query, limit)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"results": results,
		}, nil
	}
}
