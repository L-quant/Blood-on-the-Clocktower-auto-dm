#!/bin/bash
# run_loadtest.sh - Run specific load test scenarios
# Usage: ./run_loadtest.sh [OPTIONS] [SCENARIOS]
#
# Examples:
#   ./run_loadtest.sh S1              # Run single scenario
#   ./run_loadtest.sh S1,S2,S3        # Run multiple scenarios
#   ./run_loadtest.sh -u 50 S2        # Run with 50 users
#   ./run_loadtest.sh -d 60s S4       # Run for 60 seconds
#   ./run_loadtest.sh                 # Run all scenarios

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOADTEST_DIR="$(dirname "$SCRIPT_DIR")"
BACKEND_DIR="$(dirname "$LOADTEST_DIR")"

# Default values
USERS="${LOADTEST_USERS:-10}"
DURATION="${LOADTEST_DURATION:-30s}"
TARGET="${LOADTEST_TARGET:-http://localhost:8080}"
WS_TARGET="${LOADTEST_WS_TARGET:-ws://localhost:8080/ws}"
GEMINI_CONCURRENCY="${GEMINI_MAX_CONCURRENCY:-5}"
GEMINI_RPS="${GEMINI_RPS_LIMIT:-10}"
GEMINI_BUDGET="${GEMINI_REQUEST_BUDGET:-100}"
VERBOSE=""
SCENARIOS=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--users)
            USERS="$2"
            shift 2
            ;;
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -t|--target)
            TARGET="$2"
            shift 2
            ;;
        -w|--ws-target)
            WS_TARGET="$2"
            shift 2
            ;;
        --gemini-budget)
            GEMINI_BUDGET="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="-verbose"
            shift
            ;;
        -l|--list)
            echo "Available scenarios:"
            echo "  S1  - WS Handshake Storm"
            echo "  S2  - Single Room Join Storm"
            echo "  S3  - Idempotency Verification"
            echo "  S4  - Command Seq Monotonicity"
            echo "  S5  - Visibility Leak Detection"
            echo "  S6  - Gemini Call Monitoring"
            echo "  S7  - Multi-Room Isolation"
            echo "  S8  - Reconnect Seq Gap"
            echo "  S9  - RabbitMQ DLQ Monitoring"
            echo "  S10 - Full Game Flow"
            echo "  S11 - Chaos Test"
            exit 0
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS] [SCENARIOS]"
            echo ""
            echo "Options:"
            echo "  -u, --users N        Number of concurrent users (default: 10)"
            echo "  -d, --duration D     Test duration (default: 30s)"
            echo "  -t, --target URL     Target HTTP server (default: http://localhost:8080)"
            echo "  -w, --ws-target URL  Target WebSocket server (default: ws://localhost:8080/ws)"
            echo "  --gemini-budget N    Gemini request budget (default: 100)"
            echo "  -v, --verbose        Enable verbose output"
            echo "  -l, --list           List available scenarios"
            echo "  -h, --help           Show this help"
            echo ""
            echo "Examples:"
            echo "  $0 S1                    Run single scenario"
            echo "  $0 S1,S2,S3              Run multiple scenarios"
            echo "  $0 -u 50 -d 60s S2       Run S2 with 50 users for 60s"
            echo "  $0                       Run all scenarios"
            exit 0
            ;;
        -*)
            echo "Unknown option: $1"
            echo "Use -h for help"
            exit 1
            ;;
        *)
            SCENARIOS="$1"
            shift
            ;;
    esac
done

# Build loadgen if needed
LOADGEN_BIN="$BACKEND_DIR/bin/autodm_loadgen"
if [ ! -f "$LOADGEN_BIN" ]; then
    echo "Building loadgen..."
    cd "$LOADTEST_DIR"
    go build -o "$LOADGEN_BIN" ./cmd/autodm_loadgen
fi

# Build command
CMD="$LOADGEN_BIN"
CMD="$CMD -users $USERS"
CMD="$CMD -duration $DURATION"
CMD="$CMD -target $TARGET"
CMD="$CMD -ws-target $WS_TARGET"
CMD="$CMD -gemini-max-concurrency $GEMINI_CONCURRENCY"
CMD="$CMD -gemini-rps-limit $GEMINI_RPS"
CMD="$CMD -gemini-request-budget $GEMINI_BUDGET"

if [ -n "$SCENARIOS" ]; then
    CMD="$CMD -scenario $SCENARIOS"
fi

if [ -n "$VERBOSE" ]; then
    CMD="$CMD $VERBOSE"
fi

echo "Running: $CMD"
echo ""

exec $CMD
