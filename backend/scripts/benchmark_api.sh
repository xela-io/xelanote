#!/bin/bash
# benchmark_api.sh - Measure API response times for performance baseline
#
# Usage:
#   ./benchmark_api.sh <BASE_URL> <AUTH_COOKIE> <NOTE_ID>
#
# Example:
#   ./benchmark_api.sh http://localhost:8080 "access_token=eyJ..." "3ff8270f-930b-7661-1b69-2ab1d2b6186f"
#
# Auth Setup:
#   1. Login via Browser/curl and copy cookie from DevTools
#   2. Or: curl -c cookies.txt -X POST .../api/auth/login -d '{"username":"...","password":"..."}'
#
# Note IDs:
#   - Fixture: Use output from generate_fixture.go (e.g., "3ff8270f-930b-7661-1b69-2ab1d2b6186f")
#   - Staging: SELECT id FROM notes LIMIT 1;

set -e

BASE_URL="${1:-http://localhost:8080}"
COOKIE="${2:-access_token=YOUR_TOKEN}"
NOTE_ID="${3:-FIXTURE_NOTE_ID}"
MEASUREMENTS=5

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== xelanote API Benchmark ===${NC}"
echo "Base URL: $BASE_URL"
echo "Note ID:  $NOTE_ID"
echo "Measurements: $MEASUREMENTS (+ 1 warm-up)"
echo ""

# Check if curl is available
if ! command -v curl &> /dev/null; then
    echo -e "${RED}Error: curl is required but not installed.${NC}"
    exit 1
fi

# Check if bc is available for median calculation
if ! command -v bc &> /dev/null; then
    echo -e "${YELLOW}Warning: bc not found. Using awk for median calculation.${NC}"
    USE_BC=0
else
    USE_BC=1
fi

# Function to calculate median
calculate_median() {
    local sorted=($(printf '%s\n' "$@" | sort -n))
    local len=${#sorted[@]}
    if (( len % 2 == 1 )); then
        echo "${sorted[$((len/2))]}"
    else
        local mid=$((len/2))
        if [[ $USE_BC -eq 1 ]]; then
            echo "scale=3; (${sorted[$((mid-1))]} + ${sorted[$mid]}) / 2" | bc
        else
            echo "${sorted[$((mid-1))]}" # Use lower value if bc not available
        fi
    fi
}

# Function to benchmark an endpoint
benchmark() {
    local name="$1"
    local endpoint="$2"
    local method="${3:-GET}"
    local url="$BASE_URL$endpoint"

    echo -e "${YELLOW}Testing: ${NC}${name}"
    echo -e "  URL: ${url}"

    # Warm-up request (discarded)
    curl -s -o /dev/null -w "" -b "$COOKIE" -X "$method" "$url" 2>/dev/null || true

    # Collect measurements
    local times=()
    for i in $(seq 1 $MEASUREMENTS); do
        local time=$(curl -s -w "%{time_total}" -o /dev/null -b "$COOKIE" -X "$method" "$url" 2>/dev/null)
        times+=("$time")
        echo -e "    Run $i: ${time}s"
    done

    # Calculate median
    local median=$(calculate_median "${times[@]}")

    # Calculate min/max
    local sorted=($(printf '%s\n' "${times[@]}" | sort -n))
    local min="${sorted[0]}"
    local max="${sorted[-1]}"

    echo -e "  ${GREEN}Median: ${median}s${NC} (min: ${min}s, max: ${max}s)"
    echo ""

    # Store result for summary
    RESULTS+=("$name|$median|$min|$max")
}

# Array to store results
RESULTS=()

echo -e "${BLUE}--- Starting Benchmarks ---${NC}"
echo ""

# Test health endpoint (no auth needed)
echo -e "${YELLOW}Testing: ${NC}Health Check (no auth)"
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
if [[ "$HEALTH_STATUS" == "200" ]]; then
    echo -e "  ${GREEN}Server is healthy (HTTP $HEALTH_STATUS)${NC}"
else
    echo -e "  ${RED}Server returned HTTP $HEALTH_STATUS${NC}"
    echo -e "  ${RED}Please check if the server is running and the URL is correct.${NC}"
    exit 1
fi
echo ""

# Test auth by checking if cookie is valid
echo -e "${YELLOW}Verifying: ${NC}Authentication"
AUTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -b "$COOKIE" "$BASE_URL/api/notes?limit=1")
if [[ "$AUTH_STATUS" == "200" ]]; then
    echo -e "  ${GREEN}Authentication valid (HTTP $AUTH_STATUS)${NC}"
elif [[ "$AUTH_STATUS" == "401" ]]; then
    echo -e "  ${RED}Authentication failed (HTTP $AUTH_STATUS)${NC}"
    echo -e "  ${RED}Please provide a valid access_token cookie.${NC}"
    exit 1
else
    echo -e "  ${YELLOW}Unexpected status (HTTP $AUTH_STATUS). Continuing anyway...${NC}"
fi
echo ""

# Run benchmarks
benchmark "GET /api/notes (List all)" "/api/notes"
benchmark "GET /api/notes/:id (Single note)" "/api/notes/$NOTE_ID"
benchmark "GET /api/notes/:id/backlinks" "/api/notes/$NOTE_ID/backlinks"
benchmark "GET /api/graph" "/api/graph"
benchmark "GET /api/search?q=test" "/api/search?q=test"
benchmark "GET /api/folders" "/api/folders"
benchmark "GET /api/tags" "/api/tags"

# Print summary
echo -e "${BLUE}=== Summary ===${NC}"
echo ""
printf "%-35s %10s %10s %10s\n" "Endpoint" "Median" "Min" "Max"
printf "%s\n" "---------------------------------------------------------------------"

for result in "${RESULTS[@]}"; do
    IFS='|' read -r name median min max <<< "$result"
    printf "%-35s %9ss %9ss %9ss\n" "$name" "$median" "$min" "$max"
done

echo ""
echo -e "${BLUE}Target Thresholds (Staging):${NC}"
echo "  GET /api/notes:           <100ms"
echo "  GET /api/notes/:id:       <50ms"
echo "  GET /api/notes/:id/backlinks: <200ms"
echo "  GET /api/graph:           <300ms"
echo "  GET /api/search:          <200ms"

echo ""
echo -e "${GREEN}Done!${NC}"
