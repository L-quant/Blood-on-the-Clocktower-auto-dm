// Package main provides the entry point for the Blood on the Clocktower load testing tool.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Blood-on-the-Clocktower-auto-dm/backend/loadtest/internal/loadgen"
)

func main() {
	// Parse command line flags
	var (
		scenario          = flag.String("scenario", "", "Specific scenario to run (S1-S11), empty for all")
		users             = flag.Int("users", 10, "Number of concurrent users")
		duration          = flag.Duration("duration", 30*time.Second, "Test duration")
		target            = flag.String("target", "http://localhost:8080", "Target HTTP server")
		wsTarget          = flag.String("ws-target", "ws://localhost:8080/ws", "Target WebSocket server")
		outputFile        = flag.String("output", "", "Output report file (default: loadtest_report_{timestamp}.json)")
		verbose           = flag.Bool("verbose", false, "Verbose output")
		listScenarios     = flag.Bool("list", false, "List all available scenarios")
		geminiMaxConcurrency = flag.Int("gemini-max-concurrency", 5, "Max concurrent Gemini requests")
		geminiRPSLimit    = flag.Int("gemini-rps-limit", 10, "Gemini requests per second limit")
		geminiRequestBudget = flag.Int("gemini-request-budget", 100, "Total Gemini request budget")
	)
	flag.Parse()

	// List scenarios and exit
	if *listScenarios {
		printScenarios()
		return
	}

	// Build configuration
	cfg := loadgen.Config{
		TargetHTTP:           envOrDefault("LOADTEST_TARGET", *target),
		TargetWS:             envOrDefault("LOADTEST_WS_TARGET", *wsTarget),
		Users:                envIntOrDefault("LOADTEST_USERS", *users),
		Duration:             envDurationOrDefault("LOADTEST_DURATION", *duration),
		Verbose:              *verbose,
		GeminiMaxConcurrency: envIntOrDefault("GEMINI_MAX_CONCURRENCY", *geminiMaxConcurrency),
		GeminiRPSLimit:       envIntOrDefault("GEMINI_RPS_LIMIT", *geminiRPSLimit),
		GeminiRequestBudget:  int64(envIntOrDefault("GEMINI_REQUEST_BUDGET", *geminiRequestBudget)),
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Received shutdown signal, cancelling tests...")
		cancel()
	}()

	// Create runner
	runner, err := loadgen.NewRunner(cfg)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// Determine which scenarios to run
	scenarios := parseScenarios(*scenario)
	if len(scenarios) == 0 {
		log.Println("No scenarios specified, running all scenarios")
		scenarios = loadgen.AllScenarios()
	}

	log.Printf("Starting load test:")
	log.Printf("  Target HTTP: %s", cfg.TargetHTTP)
	log.Printf("  Target WS: %s", cfg.TargetWS)
	log.Printf("  Users: %d", cfg.Users)
	log.Printf("  Duration: %s", cfg.Duration)
	log.Printf("  Scenarios: %v", scenarios)
	log.Printf("  Gemini Budget: %d requests", cfg.GeminiRequestBudget)
	log.Println()

	// Run scenarios
	startTime := time.Now()
	results, err := runner.Run(ctx, scenarios)
	if err != nil {
		log.Fatalf("Load test failed: %v", err)
	}
	totalDuration := time.Since(startTime)

	// Build report
	report := loadgen.Report{
		Timestamp: time.Now().UTC(),
		Target:    cfg.TargetHTTP,
		Scenarios: results,
		Summary:   buildSummary(results, totalDuration, runner.GetGeminiStats()),
	}

	// Output report
	outputPath := *outputFile
	if outputPath == "" {
		outputPath = fmt.Sprintf("loadtest_report_%s.json", time.Now().Format("20060102_150405"))
	}

	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal report: %v", err)
	}

	if err := os.WriteFile(outputPath, reportJSON, 0644); err != nil {
		log.Fatalf("Failed to write report: %v", err)
	}

	// Print summary
	printSummary(report)

	log.Printf("\nFull report written to: %s", outputPath)

	// Exit with appropriate code
	if report.Summary.Failed > 0 {
		os.Exit(1)
	}
}

func parseScenarios(input string) []string {
	if input == "" {
		return nil
	}
	parts := strings.Split(input, ",")
	var result []string
	for _, p := range parts {
		s := strings.TrimSpace(strings.ToUpper(p))
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func buildSummary(results []loadgen.ScenarioResult, totalDuration time.Duration, geminiStats loadgen.GeminiStats) loadgen.Summary {
	var passed, failed int
	for _, r := range results {
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}
	return loadgen.Summary{
		TotalScenarios:        len(results),
		Passed:                passed,
		Failed:                failed,
		TotalDurationMs:       totalDuration.Milliseconds(),
		GeminiRequests:        geminiStats.TotalRequests,
		GeminiBudgetRemaining: geminiStats.BudgetRemaining,
	}
}

func printSummary(report loadgen.Report) {
	log.Println("\n" + strings.Repeat("=", 60))
	log.Println("LOAD TEST SUMMARY")
	log.Println(strings.Repeat("=", 60))

	for _, r := range report.Scenarios {
		status := "✅ PASS"
		if !r.Passed {
			status = "❌ FAIL"
		}
		log.Printf("  %s: %s (%dms)", r.Scenario, status, r.DurationMs)
		if len(r.Errors) > 0 {
			for _, e := range r.Errors {
				log.Printf("    - Error: %s", e)
			}
		}
	}

	log.Println(strings.Repeat("-", 60))
	log.Printf("Total: %d scenarios, %d passed, %d failed",
		report.Summary.TotalScenarios, report.Summary.Passed, report.Summary.Failed)
	log.Printf("Duration: %dms", report.Summary.TotalDurationMs)
	log.Printf("Gemini: %d requests, %d budget remaining",
		report.Summary.GeminiRequests, report.Summary.GeminiBudgetRemaining)
	log.Println(strings.Repeat("=", 60))
}

func printScenarios() {
	fmt.Println("Available Load Test Scenarios:")
	fmt.Println()
	scenarios := []struct {
		id   string
		name string
		desc string
	}{
		{"S1", "WS Handshake Storm", "N concurrent WebSocket connections + subscribe"},
		{"S2", "Single Room Join Storm", "M users join same room simultaneously"},
		{"S3", "Idempotency Verification", "Duplicate idempotency_key submission"},
		{"S4", "Command Seq Monotonicity", "Rapid sequential commands, verify Seq order"},
		{"S5", "Visibility Leak Detection", "Verify private events not leaked"},
		{"S6", "Gemini Call Monitoring", "Monitor AutoDM Gemini calls within budget"},
		{"S7", "Multi-Room Isolation", "K rooms in parallel, verify no cross-talk"},
		{"S8", "Reconnect Seq Gap", "Disconnect/reconnect with last_seq replay"},
		{"S9", "RabbitMQ DLQ Monitoring", "Verify DLQ message count on failures"},
		{"S10", "Full Game Flow", "Lobby → Night → Day → Vote → End"},
		{"S11", "Chaos Test", "Random disconnects and commands"},
	}

	for _, s := range scenarios {
		fmt.Printf("  %s: %s\n      %s\n\n", s.id, s.name, s.desc)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return defaultVal
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
