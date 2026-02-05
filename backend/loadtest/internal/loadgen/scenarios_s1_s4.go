package loadgen

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// runS1WSHandshakeStorm tests N concurrent WebSocket connections.
func (r *Runner) runS1WSHandshakeStorm(ctx context.Context) (ScenarioResult, error) {
	result := ScenarioResult{
		Metrics: make(map[string]interface{}),
		Errors:  []string{},
	}

	// Create test users and tokens
	numUsers := r.cfg.Users
	tokens := make([]string, numUsers)

	for i := 0; i < numUsers; i++ {
		_, token, err := r.createTestUser(ctx, fmt.Sprintf("s1_%d", i))
		if err != nil {
			return result, fmt.Errorf("failed to create user %d: %w", i, err)
		}
		tokens[i] = token
	}

	// Create a test room
	roomID, err := r.createTestRoom(ctx, tokens[0])
	if err != nil {
		return result, fmt.Errorf("failed to create room: %w", err)
	}

	// Concurrent WS connections
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64
	var totalLatency int64
	latencies := make([]int64, 0, numUsers)
	var latMu sync.Mutex

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			start := time.Now()
			ws := NewWSClient(r.cfg.TargetWS, tokens[idx])

			if err := ws.Connect(ctx); err != nil {
				atomic.AddInt64(&failCount, 1)
				return
			}
			defer ws.Close()

			// Subscribe to room
			if err := ws.Subscribe(ctx, roomID, 0); err != nil {
				atomic.AddInt64(&failCount, 1)
				return
			}

			latency := time.Since(start).Milliseconds()
			atomic.AddInt64(&successCount, 1)
			atomic.AddInt64(&totalLatency, latency)

			latMu.Lock()
			latencies = append(latencies, latency)
			latMu.Unlock()
		}(i)
	}

	wg.Wait()

	// Calculate metrics
	validator := NewCorrectnessValidator()
	maxLat, p99Lat, p95Lat, avgLat := validator.CalculateLatencyStats(latencies)

	result.Metrics["total_connections"] = numUsers
	result.Metrics["successful_connections"] = successCount
	result.Metrics["failed_connections"] = failCount
	result.Metrics["max_latency_ms"] = maxLat
	result.Metrics["p99_latency_ms"] = p99Lat
	result.Metrics["p95_latency_ms"] = p95Lat
	result.Metrics["avg_latency_ms"] = avgLat

	// Pass if all connections succeeded
	result.Passed = failCount == 0

	if failCount > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("%d connections failed", failCount))
	}

	return result, nil
}

// runS2JoinStorm tests M users joining the same room simultaneously.
func (r *Runner) runS2JoinStorm(ctx context.Context) (ScenarioResult, error) {
	result := ScenarioResult{
		Metrics: make(map[string]interface{}),
		Errors:  []string{},
	}

	numUsers := r.cfg.Users
	tokens := make([]string, numUsers)
	userIDs := make([]string, numUsers)

	// Create test users
	for i := 0; i < numUsers; i++ {
		userID, token, err := r.createTestUser(ctx, fmt.Sprintf("s2_%d", i))
		if err != nil {
			return result, fmt.Errorf("failed to create user %d: %w", i, err)
		}
		tokens[i] = token
		userIDs[i] = userID
	}

	// First user creates room
	roomID, err := r.createTestRoom(ctx, tokens[0])
	if err != nil {
		return result, fmt.Errorf("failed to create room: %w", err)
	}

	// All users join simultaneously via WS
	var wg sync.WaitGroup
	allEvents := make([][]EventResponse, numUsers)

	for i := 0; i < numUsers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ws := NewWSClient(r.cfg.TargetWS, tokens[idx])
			if err := ws.Connect(ctx); err != nil {
				return
			}
			defer ws.Close()

			// Subscribe
			if err := ws.Subscribe(ctx, roomID, 0); err != nil {
				return
			}

			// Send join command
			idempotencyKey := fmt.Sprintf("join_%s_%d", roomID, idx)
			if err := ws.SendCommand(ctx, roomID, "join", idempotencyKey, nil); err != nil {
				return
			}

			// Collect events for a short period
			events, _ := ws.WaitForEvents(ctx, 10, 5*time.Second)
			allEvents[idx] = events
		}(i)
	}

	wg.Wait()

	// Validate: collect all events and check seq monotonicity
	var combinedEvents []EventResponse
	for _, events := range allEvents {
		combinedEvents = append(combinedEvents, events...)
	}

	validator := NewCorrectnessValidator()
	metrics := validator.ValidateSeqMonotonicity(combinedEvents)

	result.Metrics["total_events"] = metrics.TotalEvents
	result.Metrics["seq_monotonic"] = metrics.SeqMonotonic
	result.Metrics["missing_seqs"] = metrics.MissingSeqs
	result.Metrics["duplicate_seqs"] = metrics.DuplicateSeqs

	result.Passed = metrics.SeqMonotonic && len(metrics.MissingSeqs) == 0 && len(metrics.DuplicateSeqs) == 0

	if !metrics.SeqMonotonic {
		result.Errors = append(result.Errors, "sequence is not monotonic")
	}
	if len(metrics.MissingSeqs) > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("missing sequences: %v", metrics.MissingSeqs))
	}
	if len(metrics.DuplicateSeqs) > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("duplicate sequences: %v", metrics.DuplicateSeqs))
	}

	return result, nil
}

// runS3IdempotencyVerification tests idempotency with duplicate keys.
func (r *Runner) runS3IdempotencyVerification(ctx context.Context) (ScenarioResult, error) {
	result := ScenarioResult{
		Metrics: make(map[string]interface{}),
		Errors:  []string{},
	}

	// Create user and room
	_, token, err := r.createTestUser(ctx, "s3")
	if err != nil {
		return result, fmt.Errorf("failed to create user: %w", err)
	}

	roomID, err := r.createTestRoom(ctx, token)
	if err != nil {
		return result, fmt.Errorf("failed to create room: %w", err)
	}

	// Connect WS
	ws := NewWSClient(r.cfg.TargetWS, token)
	if err := ws.Connect(ctx); err != nil {
		return result, fmt.Errorf("failed to connect: %w", err)
	}
	defer ws.Close()

	if err := ws.Subscribe(ctx, roomID, 0); err != nil {
		return result, fmt.Errorf("failed to subscribe: %w", err)
	}

	// Send the same command multiple times with same idempotency key
	idempotencyKey := fmt.Sprintf("test_idem_%d", time.Now().UnixNano())
	duplicateCount := 5

	for i := 0; i < duplicateCount; i++ {
		if err := ws.SendCommand(ctx, roomID, "public_chat", idempotencyKey, map[string]string{
			"message": "test message",
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to send command %d: %v", i, err))
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for events
	time.Sleep(2 * time.Second)

	// Get events via HTTP to validate
	eventsResp, err := r.httpClient.GetEvents(ctx, token, roomID, 0)
	if err != nil {
		return result, fmt.Errorf("failed to get events: %w", err)
	}

	// Count public_chat events
	var chatEvents int
	for _, ev := range eventsResp.Events {
		if ev.EventType == "public.chat" {
			chatEvents++
		}
	}

	result.Metrics["duplicate_submissions"] = duplicateCount
	result.Metrics["unique_events"] = chatEvents
	result.Metrics["idempotency_correct"] = chatEvents == 1

	result.Passed = chatEvents == 1

	if chatEvents != 1 {
		result.Errors = append(result.Errors, fmt.Sprintf("expected 1 event, got %d", chatEvents))
	}

	return result, nil
}

// runS4SeqMonotonicity tests rapid sequential commands.
func (r *Runner) runS4SeqMonotonicity(ctx context.Context) (ScenarioResult, error) {
	result := ScenarioResult{
		Metrics: make(map[string]interface{}),
		Errors:  []string{},
	}

	// Create user and room
	_, token, err := r.createTestUser(ctx, "s4")
	if err != nil {
		return result, fmt.Errorf("failed to create user: %w", err)
	}

	roomID, err := r.createTestRoom(ctx, token)
	if err != nil {
		return result, fmt.Errorf("failed to create room: %w", err)
	}

	// Connect WS
	ws := NewWSClient(r.cfg.TargetWS, token)
	if err := ws.Connect(ctx); err != nil {
		return result, fmt.Errorf("failed to connect: %w", err)
	}
	defer ws.Close()

	if err := ws.Subscribe(ctx, roomID, 0); err != nil {
		return result, fmt.Errorf("failed to subscribe: %w", err)
	}

	// Send many commands rapidly
	commandCount := 50
	for i := 0; i < commandCount; i++ {
		idempotencyKey := fmt.Sprintf("cmd_%d_%d", time.Now().UnixNano(), i)
		if err := ws.SendCommand(ctx, roomID, "public_chat", idempotencyKey, map[string]string{
			"message": fmt.Sprintf("message %d", i),
		}); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to send command %d: %v", i, err))
		}
	}

	// Wait and collect events
	events, _ := ws.WaitForEvents(ctx, commandCount, 10*time.Second)

	// Validate monotonicity
	validator := NewCorrectnessValidator()
	metrics := validator.ValidateSeqMonotonicity(events)

	result.Metrics["commands_sent"] = commandCount
	result.Metrics["events_received"] = len(events)
	result.Metrics["seq_monotonic"] = metrics.SeqMonotonic
	result.Metrics["missing_seqs"] = metrics.MissingSeqs
	result.Metrics["duplicate_seqs"] = metrics.DuplicateSeqs

	result.Passed = metrics.SeqMonotonic && len(metrics.DuplicateSeqs) == 0

	if !metrics.SeqMonotonic {
		result.Errors = append(result.Errors, "sequence is not monotonic")
	}
	if len(metrics.DuplicateSeqs) > 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("duplicate sequences: %v", metrics.DuplicateSeqs))
	}

	return result, nil
}
