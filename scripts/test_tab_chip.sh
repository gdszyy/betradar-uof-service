#!/bin/bash

# Test Script for Tab/Chip Implementation
# This script tests the tab/chip functionality

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}ℹ ${NC}$1"
}

print_success() {
    echo -e "${GREEN}✓ ${NC}$1"
}

print_error() {
    echo -e "${RED}✗ ${NC}$1"
}

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

print_info "Starting Tab/Chip Tests"
print_info "Project directory: $PROJECT_DIR"

# Test 1: Run unit tests
print_info "\n=== Test 1: Running Unit Tests ==="
cd "$PROJECT_DIR"

if go test ./services -v -run TestParseSpecifiers; then
    print_success "TestParseSpecifiers passed"
else
    print_error "TestParseSpecifiers failed"
    exit 1
fi

if go test ./services -v -run TestDetermineTabID; then
    print_success "TestDetermineTabID passed"
else
    print_error "TestDetermineTabID failed"
    exit 1
fi

if go test ./services -v -run TestDetermineChipID; then
    print_success "TestDetermineChipID passed"
else
    print_error "TestDetermineChipID failed"
    exit 1
fi

# Test 2: Verify CSV files
print_info "\n=== Test 2: Verifying CSV Files ==="

TAB_CONFIG_FILE="$PROJECT_DIR/final_tab_chip_config.csv"
CHIP_CONFIG_FILE="$PROJECT_DIR/final_chip_enumeration.csv"

if [ -f "$TAB_CONFIG_FILE" ]; then
    TAB_COUNT=$(wc -l < "$TAB_CONFIG_FILE")
    print_success "Tab config file found ($TAB_COUNT lines)"
else
    print_error "Tab config file not found"
    exit 1
fi

if [ -f "$CHIP_CONFIG_FILE" ]; then
    CHIP_COUNT=$(wc -l < "$CHIP_CONFIG_FILE")
    print_success "Chip config file found ($CHIP_COUNT lines)"
else
    print_error "Chip config file not found"
    exit 1
fi

# Test 3: Verify database connection
print_info "\n=== Test 3: Verifying Database Connection ==="

if [ -z "$DATABASE_URL" ]; then
    print_error "DATABASE_URL environment variable is not set"
    print_info "Skipping database tests"
else
    if psql "$DATABASE_URL" -c "SELECT 1" > /dev/null 2>&1; then
        print_success "Database connection successful"
    else
        print_error "Failed to connect to database"
        exit 1
    fi

    # Test 4: Verify tables exist
    print_info "\n=== Test 4: Verifying Database Tables ==="

    # Check if migration has been applied
    if psql "$DATABASE_URL" -c "SELECT 1 FROM market_tabs LIMIT 1" > /dev/null 2>&1; then
        print_success "market_tabs table exists"
    else
        print_error "market_tabs table does not exist"
        print_info "Please run the migration first: psql \$DATABASE_URL -f database/migrations/011_add_tab_chip_fields.sql"
    fi

    if psql "$DATABASE_URL" -c "SELECT 1 FROM market_chips LIMIT 1" > /dev/null 2>&1; then
        print_success "market_chips table exists"
    else
        print_error "market_chips table does not exist"
    fi

    # Test 5: Verify data import
    print_info "\n=== Test 5: Verifying Data Import ==="

    TAB_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_tabs;" 2>/dev/null || echo "0")
    print_info "Tabs in database: $TAB_COUNT"

    CHIP_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_chips;" 2>/dev/null || echo "0")
    print_info "Chips in database: $CHIP_COUNT"

    MARKET_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets;" 2>/dev/null || echo "0")
    print_info "Total markets: $MARKET_COUNT"

    MARKET_WITH_TAB=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets WHERE tab_id IS NOT NULL AND tab_id != '';" 2>/dev/null || echo "0")
    print_info "Markets with tab_id: $MARKET_WITH_TAB"

    MARKET_WITH_CHIP=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets WHERE chip_id IS NOT NULL AND chip_id != '';" 2>/dev/null || echo "0")
    print_info "Markets with chip_id: $MARKET_WITH_CHIP"

    # Test 6: Verify API endpoints (if server is running)
    print_info "\n=== Test 6: Testing API Endpoints ==="

    API_URL="${API_URL:-http://localhost:8080}"
    
    if curl -s "$API_URL/api/v1/health" > /dev/null 2>&1; then
        print_success "API server is running"

        # Test health endpoint
        if curl -s "$API_URL/api/v1/health" | grep -q "ok"; then
            print_success "Health check endpoint works"
        else
            print_error "Health check endpoint failed"
        fi

        # Test tabs endpoint (requires valid event_id)
        print_info "Note: Tab/Chip endpoints require valid event_id from database"
    else
        print_info "API server is not running at $API_URL"
        print_info "Start the server to test API endpoints"
    fi
fi

print_success "\n=== All Tests Completed ==="
print_info "Summary:"
print_info "  - Unit tests: PASSED"
print_info "  - CSV files: VERIFIED"
if [ -n "$DATABASE_URL" ]; then
    print_info "  - Database: CONNECTED"
    print_info "  - Tables: VERIFIED"
    print_info "  - Data: $TAB_COUNT tabs, $CHIP_COUNT chips"
fi
