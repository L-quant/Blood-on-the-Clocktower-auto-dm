// Package loadgen provides load testing functionality for Blood on the Clocktower Auto-DM.
package loadgen

import (
	"errors"
	"fmt"
	"time"
)

// Config holds load test configuration.
type Config struct {
	// Server targets
	TargetHTTP string
	TargetWS   string

	// Load parameters
	Users    int
	Duration time.Duration

	// Output settings
	Verbose bool

	// Gemini protection
	GeminiMaxConcurrency int
	GeminiRPSLimit       int
	GeminiRequestBudget  int64

	// Internal - JWT token for authenticated requests
	JWTSecret string
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.TargetHTTP == "" {
		return errors.New("target HTTP URL is required")
	}
	if c.TargetWS == "" {
		return errors.New("target WebSocket URL is required")
	}
	if c.Users < 1 {
		return errors.New("users must be at least 1")
	}
	if c.Duration < time.Second {
		return errors.New("duration must be at least 1 second")
	}
	if c.GeminiMaxConcurrency < 1 {
		c.GeminiMaxConcurrency = 5
	}
	if c.GeminiRPSLimit < 1 {
		c.GeminiRPSLimit = 10
	}
	if c.GeminiRequestBudget < 1 {
		c.GeminiRequestBudget = 100
	}
	return nil
}

// Report is the final load test report.
type Report struct {
	Timestamp time.Time        `json:"timestamp"`
	Target    string           `json:"target"`
	Scenarios []ScenarioResult `json:"scenarios"`
	Summary   Summary          `json:"summary"`
}

// Summary holds aggregate statistics.
type Summary struct {
	TotalScenarios        int   `json:"total_scenarios"`
	Passed                int   `json:"passed"`
	Failed                int   `json:"failed"`
	TotalDurationMs       int64 `json:"total_duration_ms"`
	GeminiRequests        int64 `json:"gemini_requests"`
	GeminiBudgetRemaining int64 `json:"gemini_budget_remaining"`
}

// ScenarioResult holds the result of a single scenario.
type ScenarioResult struct {
	Scenario   string                 `json:"scenario"`
	Passed     bool                   `json:"passed"`
	DurationMs int64                  `json:"duration_ms"`
	Metrics    map[string]interface{} `json:"metrics"`
	Errors     []string               `json:"errors"`
}

// GeminiStats holds Gemini API usage statistics.
type GeminiStats struct {
	TotalRequests   int64
	BudgetRemaining int64
}

// CorrectnessMetrics holds correctness validation metrics.
type CorrectnessMetrics struct {
	TotalEvents   int64   `json:"total_events"`
	SeqMonotonic  bool    `json:"seq_monotonic"`
	MissingSeqs   []int64 `json:"missing_seqs"`
	DuplicateSeqs []int64 `json:"duplicate_seqs"`
	MaxLatencyMs  int64   `json:"max_latency_ms"`
	P99LatencyMs  int64   `json:"p99_latency_ms"`
	P95LatencyMs  int64   `json:"p95_latency_ms"`
	AvgLatencyMs  int64   `json:"avg_latency_ms"`
}

// VisibilityMetrics holds visibility check metrics.
type VisibilityMetrics struct {
	TotalPrivateEvents int64    `json:"total_private_events"`
	LeakedEvents       []string `json:"leaked_events"`
	LeakDetected       bool     `json:"leak_detected"`
}

// IdempotencyMetrics holds idempotency check metrics.
type IdempotencyMetrics struct {
	DuplicateSubmissions int64 `json:"duplicate_submissions"`
	UniqueEvents         int64 `json:"unique_events"`
	IdempotencyCorrect   bool  `json:"idempotency_correct"`
}

// GameFlowMetrics holds game flow validation metrics.
type GameFlowMetrics struct {
	PhaseTransitions []string `json:"phase_transitions"`
	ValidFlow        bool     `json:"valid_flow"`
	FinalPhase       string   `json:"final_phase"`
}

// AllScenarios returns all available scenario IDs.
func AllScenarios() []string {
	return []string{"S1", "S2", "S3", "S4", "S5", "S6", "S7", "S8", "S9", "S10", "S11"}
}

// ScenarioInfo returns human-readable info about a scenario.
func ScenarioInfo(id string) (name, description string) {
	switch id {
	case "S1":
		return "WS Handshake Storm", "N concurrent WebSocket connections + subscribe"
	case "S2":
		return "Single Room Join Storm", "M users join same room simultaneously"
	case "S3":
		return "Idempotency Verification", "Duplicate idempotency_key submission"
	case "S4":
		return "Command Seq Monotonicity", "Rapid sequential commands, verify Seq order"
	case "S5":
		return "Visibility Leak Detection", "Verify private events not leaked"
	case "S6":
		return "Gemini Call Monitoring", "Monitor AutoDM Gemini calls within budget"
	case "S7":
		return "Multi-Room Isolation", "K rooms in parallel, verify no cross-talk"
	case "S8":
		return "Reconnect Seq Gap", "Disconnect/reconnect with last_seq replay"
	case "S9":
		return "RabbitMQ DLQ Monitoring", "Verify DLQ message count on failures"
	case "S10":
		return "Full Game Flow", "Lobby -> Night -> Day -> Vote -> End"
	case "S11":
		return "Chaos Test", "Random disconnects and commands"
	default:
		return "Unknown", fmt.Sprintf("Unknown scenario: %s", id)
	}
}
