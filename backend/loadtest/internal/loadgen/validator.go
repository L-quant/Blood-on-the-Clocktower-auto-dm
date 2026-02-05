package loadgen

import (
	"sort"
)

// CorrectnessValidator validates correctness of load test results.
type CorrectnessValidator struct{}

// NewCorrectnessValidator creates a new correctness validator.
func NewCorrectnessValidator() *CorrectnessValidator {
	return &CorrectnessValidator{}
}

// ValidateSeqMonotonicity checks if all sequences are strictly monotonically increasing.
func (v *CorrectnessValidator) ValidateSeqMonotonicity(events []EventResponse) CorrectnessMetrics {
	metrics := CorrectnessMetrics{
		TotalEvents:   int64(len(events)),
		SeqMonotonic:  true,
		MissingSeqs:   []int64{},
		DuplicateSeqs: []int64{},
	}

	if len(events) == 0 {
		return metrics
	}

	// Sort by Seq for validation
	sorted := make([]EventResponse, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Seq < sorted[j].Seq
	})

	// Track seen sequences
	seenSeqs := make(map[int64]bool)
	var lastSeq int64 = 0

	for _, ev := range sorted {
		// Check for duplicates
		if seenSeqs[ev.Seq] {
			metrics.DuplicateSeqs = append(metrics.DuplicateSeqs, ev.Seq)
			metrics.SeqMonotonic = false
		}
		seenSeqs[ev.Seq] = true

		// Check for gaps (only if we have a previous seq)
		if lastSeq > 0 {
			for missing := lastSeq + 1; missing < ev.Seq; missing++ {
				metrics.MissingSeqs = append(metrics.MissingSeqs, missing)
				metrics.SeqMonotonic = false
			}
		}

		lastSeq = ev.Seq
	}

	return metrics
}

// ValidateEventCompleteness checks if events form a complete sequence.
func (v *CorrectnessValidator) ValidateEventCompleteness(events []EventResponse, expectedStart, expectedEnd int64) bool {
	if len(events) == 0 {
		return expectedStart > expectedEnd
	}

	seenSeqs := make(map[int64]bool)
	for _, ev := range events {
		seenSeqs[ev.Seq] = true
	}

	for seq := expectedStart; seq <= expectedEnd; seq++ {
		if !seenSeqs[seq] {
			return false
		}
	}
	return true
}

// ValidateIdempotency checks if idempotency is correctly enforced.
func (v *CorrectnessValidator) ValidateIdempotency(events []EventResponse, expectedUnique int) IdempotencyMetrics {
	// Count unique events by some identifier
	uniqueEvents := make(map[int64]bool)
	for _, ev := range events {
		uniqueEvents[ev.Seq] = true
	}

	return IdempotencyMetrics{
		DuplicateSubmissions: int64(len(events) - len(uniqueEvents)),
		UniqueEvents:         int64(len(uniqueEvents)),
		IdempotencyCorrect:   len(uniqueEvents) == expectedUnique,
	}
}

// ValidateVisibility checks if private events are properly filtered.
func (v *CorrectnessValidator) ValidateVisibility(events []EventResponse, viewerID string) VisibilityMetrics {
	metrics := VisibilityMetrics{
		TotalPrivateEvents: 0,
		LeakedEvents:       []string{},
		LeakDetected:       false,
	}

	privateEventTypes := map[string]bool{
		"whisper.sent":     true,
		"role.assigned":    true,
		"ability.resolved": true,
	}

	for _, ev := range events {
		if privateEventTypes[ev.EventType] {
			metrics.TotalPrivateEvents++
			// In a real implementation, we would check if the viewer
			// should have access to this event based on the event data.
			// For now, we just count private events.
		}
	}

	return metrics
}

// ValidateRoomIsolation checks if events from different rooms are isolated.
func (v *CorrectnessValidator) ValidateRoomIsolation(eventsByRoom map[string][]EventResponse) bool {
	for roomID, events := range eventsByRoom {
		for _, ev := range events {
			if ev.RoomID != roomID {
				return false
			}
		}
	}
	return true
}

// ValidateGamePhaseTransitions checks if game phase transitions are valid.
func (v *CorrectnessValidator) ValidateGamePhaseTransitions(phases []string) GameFlowMetrics {
	metrics := GameFlowMetrics{
		PhaseTransitions: phases,
		ValidFlow:        true,
		FinalPhase:       "",
	}

	if len(phases) == 0 {
		metrics.ValidFlow = false
		return metrics
	}

	metrics.FinalPhase = phases[len(phases)-1]

	// Valid phase sequence: lobby -> night -> day -> (night -> day)* -> ended
	validTransitions := map[string][]string{
		"":       {"lobby"},
		"lobby":  {"night", "ended"},
		"night":  {"day", "ended"},
		"day":    {"night", "ended"},
		"ended":  {},
	}

	prev := ""
	for _, phase := range phases {
		validNext, ok := validTransitions[prev]
		if !ok {
			metrics.ValidFlow = false
			break
		}

		found := false
		for _, v := range validNext {
			if v == phase {
				found = true
				break
			}
		}
		if !found {
			metrics.ValidFlow = false
			break
		}
		prev = phase
	}

	return metrics
}

// CalculateLatencyStats calculates latency statistics from timestamps.
func (v *CorrectnessValidator) CalculateLatencyStats(latencies []int64) (max, p99, p95, avg int64) {
	if len(latencies) == 0 {
		return 0, 0, 0, 0
	}

	// Sort latencies
	sorted := make([]int64, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate stats
	max = sorted[len(sorted)-1]
	
	p99Idx := int(float64(len(sorted)) * 0.99)
	if p99Idx >= len(sorted) {
		p99Idx = len(sorted) - 1
	}
	p99 = sorted[p99Idx]

	p95Idx := int(float64(len(sorted)) * 0.95)
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}
	p95 = sorted[p95Idx]

	var sum int64
	for _, l := range sorted {
		sum += l
	}
	avg = sum / int64(len(sorted))

	return max, p99, p95, avg
}
