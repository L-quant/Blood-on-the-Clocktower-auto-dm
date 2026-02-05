package loadgen

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Runner executes load test scenarios.
type Runner struct {
	cfg Config

	// HTTP client pool
	httpClient *HTTPClient

	// Gemini protection
	geminiSem     chan struct{}
	geminiLimiter *rate.Limiter
	geminiCount   int64
	geminiBudget  int64

	// Metrics collection
	mu      sync.Mutex
	metrics map[string]interface{}
}

// NewRunner creates a new load test runner.
func NewRunner(cfg Config) (*Runner, error) {
	httpClient, err := NewHTTPClient(cfg.TargetHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &Runner{
		cfg:           cfg,
		httpClient:    httpClient,
		geminiSem:     make(chan struct{}, cfg.GeminiMaxConcurrency),
		geminiLimiter: rate.NewLimiter(rate.Limit(cfg.GeminiRPSLimit), cfg.GeminiRPSLimit),
		geminiBudget:  cfg.GeminiRequestBudget,
		metrics:       make(map[string]interface{}),
	}, nil
}

// Run executes the specified scenarios.
func (r *Runner) Run(ctx context.Context, scenarios []string) ([]ScenarioResult, error) {
	var results []ScenarioResult

	for _, scenarioID := range scenarios {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		name, _ := ScenarioInfo(scenarioID)
		log.Printf("Running scenario %s: %s", scenarioID, name)

		result, err := r.runScenario(ctx, scenarioID)
		if err != nil {
			result = ScenarioResult{
				Scenario:   scenarioID,
				Passed:     false,
				DurationMs: 0,
				Metrics:    nil,
				Errors:     []string{fmt.Sprintf("scenario failed: %v", err)},
			}
		}

		results = append(results, result)

		status := "✅"
		if !result.Passed {
			status = "❌"
		}
		log.Printf("  %s Completed in %dms", status, result.DurationMs)
	}

	return results, nil
}

// runScenario executes a single scenario.
func (r *Runner) runScenario(ctx context.Context, scenarioID string) (ScenarioResult, error) {
	start := time.Now()

	var result ScenarioResult
	var err error

	switch scenarioID {
	case "S1":
		result, err = r.runS1WSHandshakeStorm(ctx)
	case "S2":
		result, err = r.runS2JoinStorm(ctx)
	case "S3":
		result, err = r.runS3IdempotencyVerification(ctx)
	case "S4":
		result, err = r.runS4SeqMonotonicity(ctx)
	case "S5":
		result, err = r.runS5VisibilityLeakDetection(ctx)
	case "S6":
		result, err = r.runS6GeminiCallMonitoring(ctx)
	case "S7":
		result, err = r.runS7MultiRoomIsolation(ctx)
	case "S8":
		result, err = r.runS8ReconnectSeqGap(ctx)
	case "S9":
		result, err = r.runS9RabbitMQDLQMonitoring(ctx)
	case "S10":
		result, err = r.runS10FullGameFlow(ctx)
	case "S11":
		result, err = r.runS11ChaosTest(ctx)
	default:
		return ScenarioResult{}, fmt.Errorf("unknown scenario: %s", scenarioID)
	}

	result.Scenario = scenarioID
	result.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Passed = false
		if result.Errors == nil {
			result.Errors = []string{}
		}
		result.Errors = append(result.Errors, err.Error())
	}

	return result, nil
}

// GetGeminiStats returns current Gemini API usage statistics.
func (r *Runner) GetGeminiStats() GeminiStats {
	return GeminiStats{
		TotalRequests:   atomic.LoadInt64(&r.geminiCount),
		BudgetRemaining: r.geminiBudget - atomic.LoadInt64(&r.geminiCount),
	}
}

// acquireGeminiSlot acquires a slot for Gemini API call.
// Returns false if budget is exhausted.
func (r *Runner) acquireGeminiSlot(ctx context.Context) bool {
	// Check budget first
	if atomic.LoadInt64(&r.geminiCount) >= r.geminiBudget {
		return false
	}

	// Wait for rate limiter
	if err := r.geminiLimiter.Wait(ctx); err != nil {
		return false
	}

	// Acquire semaphore
	select {
	case r.geminiSem <- struct{}{}:
		atomic.AddInt64(&r.geminiCount, 1)
		return true
	case <-ctx.Done():
		return false
	}
}

// releaseGeminiSlot releases a Gemini API slot.
func (r *Runner) releaseGeminiSlot() {
	<-r.geminiSem
}

// createTestUser creates a test user and returns JWT token.
func (r *Runner) createTestUser(ctx context.Context, suffix string) (userID, token string, err error) {
	email := fmt.Sprintf("loadtest_%s_%d@test.local", suffix, time.Now().UnixNano())
	password := "loadtest123"

	// Register user
	regResp, err := r.httpClient.Register(ctx, email, password)
	if err != nil {
		return "", "", fmt.Errorf("failed to register: %w", err)
	}

	// Login to get token
	loginResp, err := r.httpClient.Login(ctx, email, password)
	if err != nil {
		return "", "", fmt.Errorf("failed to login: %w", err)
	}

	return regResp.UserID, loginResp.Token, nil
}

// createTestRoom creates a test room and returns room ID.
func (r *Runner) createTestRoom(ctx context.Context, token string) (roomID string, err error) {
	resp, err := r.httpClient.CreateRoom(ctx, token)
	if err != nil {
		return "", fmt.Errorf("failed to create room: %w", err)
	}
	return resp.RoomID, nil
}
