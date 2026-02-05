#!/bin/bash
# full_suite.sh - Full load test suite for Blood on the Clocktower Auto-DM
# Runs all scenarios (S1-S11) with comprehensive load
# Expected duration: ~10-15 minutes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOADTEST_DIR="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$(dirname "$LOADTEST_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  Blood on the Clocktower - Full Load Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Default configuration (can be overridden by environment)
export LOADTEST_TARGET="${LOADTEST_TARGET:-http://localhost:8080}"
export LOADTEST_WS_TARGET="${LOADTEST_WS_TARGET:-ws://localhost:8080/ws}"
export LOADTEST_USERS="${LOADTEST_USERS:-20}"
export LOADTEST_DURATION="${LOADTEST_DURATION:-30s}"
export GEMINI_MAX_CONCURRENCY="${GEMINI_MAX_CONCURRENCY:-5}"
export GEMINI_RPS_LIMIT="${GEMINI_RPS_LIMIT:-10}"
export GEMINI_REQUEST_BUDGET="${GEMINI_REQUEST_BUDGET:-100}"

echo "Configuration:"
echo "  Target HTTP: $LOADTEST_TARGET"
echo "  Target WS:   $LOADTEST_WS_TARGET"
echo "  Users:       $LOADTEST_USERS"
echo "  Duration:    $LOADTEST_DURATION"
echo "  Gemini Budget: $GEMINI_REQUEST_BUDGET requests"
echo ""

# Check if server is running
echo -n "Checking server health... "
if curl -s -f "$LOADTEST_TARGET/health" > /dev/null 2>&1; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${RED}FAILED${NC}"
    echo "Error: Server is not responding at $LOADTEST_TARGET"
    echo "Please ensure the backend is running with all dependencies:"
    echo "  cd $BACKEND_DIR && make dev"
    exit 1
fi

# Check if required services are available
echo -n "Checking metrics endpoint... "
if curl -s -f "$LOADTEST_TARGET/metrics" > /dev/null 2>&1; then
    echo -e "${GREEN}OK${NC}"
else
    echo -e "${YELLOW}WARN${NC} (metrics not available, some tests may skip)"
fi

# Build loadgen if needed
LOADGEN_BIN="$BACKEND_DIR/bin/autodm_loadgen"
if [ ! -f "$LOADGEN_BIN" ]; then
    echo "Building loadgen..."
    cd "$LOADTEST_DIR"
    go build -o "$LOADGEN_BIN" ./cmd/autodm_loadgen
fi

# Generate timestamp for reports
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="loadtest_full_${TIMESTAMP}.json"

echo ""
echo -e "${BLUE}Starting full test suite at $(date)${NC}"
echo ""

# Run all scenarios
$LOADGEN_BIN \
    -users "$LOADTEST_USERS" \
    -duration "$LOADTEST_DURATION" \
    -target "$LOADTEST_TARGET" \
    -ws-target "$LOADTEST_WS_TARGET" \
    -gemini-max-concurrency "$GEMINI_MAX_CONCURRENCY" \
    -gemini-rps-limit "$GEMINI_RPS_LIMIT" \
    -gemini-request-budget "$GEMINI_REQUEST_BUDGET" \
    -output "$REPORT_FILE" \
    -verbose

EXIT_CODE=$?

echo ""
echo -e "${BLUE}Completed at $(date)${NC}"
echo ""

# Print summary from report
if [ -f "$REPORT_FILE" ]; then
    echo "Report saved to: $REPORT_FILE"
    echo ""
    
    # Parse and display summary using jq if available
    if command -v jq &> /dev/null; then
        echo "Summary:"
        jq -r '.summary | "  Total scenarios: \(.total_scenarios)\n  Passed: \(.passed)\n  Failed: \(.failed)\n  Duration: \(.total_duration_ms)ms\n  Gemini requests: \(.gemini_requests)\n  Gemini budget remaining: \(.gemini_budget_remaining)"' "$REPORT_FILE"
    fi
fi

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Full Test Suite PASSED${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}  Full Test Suite FAILED${NC}"
    echo -e "${RED}========================================${NC}"
    
    # Print failed scenarios
    if [ -f "$REPORT_FILE" ] && command -v jq &> /dev/null; then
        echo ""
        echo "Failed scenarios:"
        jq -r '.scenarios[] | select(.passed == false) | "  \(.scenario): \(.errors | join(", "))"' "$REPORT_FILE"
    fi
fi

exit $EXIT_CODE
