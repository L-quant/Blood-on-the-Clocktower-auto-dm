#!/bin/bash
# quick_smoke.sh - Quick smoke test for Blood on the Clocktower Auto-DM
# Runs core scenarios (S1, S2, S4) with minimal load
# Expected duration: < 1 minute

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOADTEST_DIR="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$(dirname "$LOADTEST_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}  Blood on the Clocktower - Quick Smoke Test${NC}"
echo -e "${YELLOW}========================================${NC}"
echo ""

# Default configuration (can be overridden by environment)
export LOADTEST_TARGET="${LOADTEST_TARGET:-http://localhost:8080}"
export LOADTEST_WS_TARGET="${LOADTEST_WS_TARGET:-ws://localhost:8080/ws}"
export LOADTEST_USERS="${LOADTEST_USERS:-5}"
export LOADTEST_DURATION="${LOADTEST_DURATION:-10s}"
export GEMINI_MAX_CONCURRENCY="${GEMINI_MAX_CONCURRENCY:-2}"
export GEMINI_RPS_LIMIT="${GEMINI_RPS_LIMIT:-5}"
export GEMINI_REQUEST_BUDGET="${GEMINI_REQUEST_BUDGET:-10}"

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
    echo "Please ensure the backend is running:"
    echo "  cd $BACKEND_DIR && make run"
    exit 1
fi

# Build loadgen if needed
LOADGEN_BIN="$BACKEND_DIR/bin/autodm_loadgen"
if [ ! -f "$LOADGEN_BIN" ]; then
    echo "Building loadgen..."
    cd "$LOADTEST_DIR"
    go build -o "$LOADGEN_BIN" ./cmd/autodm_loadgen
fi

# Run quick scenarios
echo ""
echo "Running quick smoke tests..."
echo ""

$LOADGEN_BIN \
    -scenario "S1,S2,S4" \
    -users "$LOADTEST_USERS" \
    -duration "$LOADTEST_DURATION" \
    -target "$LOADTEST_TARGET" \
    -ws-target "$LOADTEST_WS_TARGET" \
    -gemini-max-concurrency "$GEMINI_MAX_CONCURRENCY" \
    -gemini-rps-limit "$GEMINI_RPS_LIMIT" \
    -gemini-request-budget "$GEMINI_REQUEST_BUDGET" \
    -output "loadtest_quick_$(date +%Y%m%d_%H%M%S).json"

EXIT_CODE=$?

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}  Quick Smoke Test PASSED${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}  Quick Smoke Test FAILED${NC}"
    echo -e "${RED}========================================${NC}"
fi

exit $EXIT_CODE
