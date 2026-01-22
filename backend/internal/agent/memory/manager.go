// Package memory provides short-term and long-term memory management.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// EntryType categorizes memory entries.
type EntryType string

const (
	EntryEvent     EntryType = "event"
	EntryDecision  EntryType = "decision"
	EntryNarration EntryType = "narration"
	EntryPlayer    EntryType = "player"
	EntryRules     EntryType = "rules"
)

// Entry represents a memory entry.
type Entry struct {
	ID        string    `json:"id"`
	Type      EntryType `json:"type"`
	Content   string    `json:"content"`
	Metadata  Metadata  `json:"metadata"`
	Timestamp time.Time `json:"timestamp"`
	Embedding []float32 `json:"embedding,omitempty"`
}

// Metadata holds additional context.
type Metadata struct {
	RoomID    string            `json:"room_id,omitempty"`
	Phase     string            `json:"phase,omitempty"`
	DayNumber int               `json:"day_number,omitempty"`
	Players   []string          `json:"players,omitempty"`
	Tags      []string          `json:"tags,omitempty"`
	Extra     map[string]string `json:"extra,omitempty"`
}

// Config for memory manager.
type Config struct {
	ShortTermCapacity int
	LongTermEnabled   bool
}

// Manager manages short-term and long-term memory.
type Manager struct {
	mu        sync.RWMutex
	shortTerm []Entry
	capacity  int
}

// NewManager creates a new memory manager.
func NewManager(cfg Config) *Manager {
	if cfg.ShortTermCapacity <= 0 {
		cfg.ShortTermCapacity = 100
	}
	return &Manager{
		shortTerm: make([]Entry, 0, cfg.ShortTermCapacity),
		capacity:  cfg.ShortTermCapacity,
	}
}

// Add stores a new memory entry.
func (m *Manager) Add(ctx context.Context, entry Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%s-%d", entry.Type, time.Now().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	m.shortTerm = append(m.shortTerm, entry)
	if len(m.shortTerm) > m.capacity {
		m.shortTerm = m.shortTerm[1:]
	}

	return nil
}

// AddEvent is a convenience method for adding game events.
func (m *Manager) AddEvent(ctx context.Context, roomID, phase string, dayNum int, content string) error {
	return m.Add(ctx, Entry{
		Type:    EntryEvent,
		Content: content,
		Metadata: Metadata{
			RoomID:    roomID,
			Phase:     phase,
			DayNumber: dayNum,
		},
	})
}

// Recent returns the most recent entries.
func (m *Manager) Recent(limit int) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit <= 0 || limit > len(m.shortTerm) {
		limit = len(m.shortTerm)
	}

	start := len(m.shortTerm) - limit
	result := make([]Entry, limit)
	copy(result, m.shortTerm[start:])
	return result
}

// RecentForRoom returns recent entries for a specific room.
func (m *Manager) RecentForRoom(roomID string, limit int) []Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Entry
	for i := len(m.shortTerm) - 1; i >= 0 && len(result) < limit; i-- {
		if m.shortTerm[i].Metadata.RoomID == roomID {
			result = append(result, m.shortTerm[i])
		}
	}
	return result
}

// GetContext retrieves relevant context for a prompt.
func (m *Manager) GetContext(ctx context.Context, roomID string, query string) (MemoryContext, error) {
	return MemoryContext{
		RecentEvents: m.RecentForRoom(roomID, 20),
	}, nil
}

// MemoryContext is bundled context for agent prompts.
type MemoryContext struct {
	RecentEvents    []Entry
	RelevantHistory []Entry
}

// Format formats memory context as a string.
func (mc MemoryContext) Format() string {
	var sb strings.Builder
	if len(mc.RecentEvents) > 0 {
		sb.WriteString("## Recent Events\n")
		for _, e := range mc.RecentEvents {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.Type, e.Content))
		}
	}
	return sb.String()
}

// ToJSON returns the context as JSON.
func (mc MemoryContext) ToJSON() string {
	data, _ := json.Marshal(mc)
	return string(data)
}

// Clear removes all entries for a room.
func (m *Manager) Clear(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	filtered := make([]Entry, 0, len(m.shortTerm))
	for _, e := range m.shortTerm {
		if e.Metadata.RoomID != roomID {
			filtered = append(filtered, e)
		}
	}
	m.shortTerm = filtered
}
