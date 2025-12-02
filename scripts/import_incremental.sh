#!/bin/bash

# Incremental Import and Assignment Script
# This script handles incremental updates for tab/chip assignment
# It only processes new markets (those without tab_id), making it efficient for production

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

print_warning() {
    echo -e "${YELLOW}⚠ ${NC}$1"
}

print_error() {
    echo -e "${RED}✗ ${NC}$1"
}

# Check required environment variables
if [ -z "$DATABASE_URL" ]; then
    print_error "DATABASE_URL environment variable is not set"
    exit 1
fi

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

print_info "Project directory: $PROJECT_DIR"
print_info "Script directory: $SCRIPT_DIR"

# Check if CSV files exist
TAB_CONFIG_FILE="${SCRIPT_DIR}/../final_tab_chip_config.csv"
CHIP_CONFIG_FILE="${SCRIPT_DIR}/../final_chip_enumeration.csv"

if [ ! -f "$TAB_CONFIG_FILE" ]; then
    print_error "Tab configuration file not found: $TAB_CONFIG_FILE"
    exit 1
fi

if [ ! -f "$CHIP_CONFIG_FILE" ]; then
    print_error "Chip configuration file not found: $CHIP_CONFIG_FILE"
    exit 1
fi

print_success "Found configuration files"

# Step 1: Check if tables exist
print_info "\n=== Step 1: Checking Database Tables ==="
if psql "$DATABASE_URL" -c "SELECT 1 FROM market_tabs LIMIT 1" > /dev/null 2>&1; then
    print_success "market_tabs table exists"
    EXISTING_TABS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_tabs;")
    print_info "Existing tabs: $EXISTING_TABS"
else
    print_warning "market_tabs table does not exist - running migration first"
    if [ -f "$PROJECT_DIR/database/migrations/011_add_tab_chip_fields.sql" ]; then
        psql "$DATABASE_URL" -f "$PROJECT_DIR/database/migrations/011_add_tab_chip_fields.sql" > /dev/null 2>&1
        print_success "Migration applied"
    fi
fi

# Step 2: Import tab and chip configurations (if not already imported)
print_info "\n=== Step 2: Importing Tab and Chip Configurations ==="
EXISTING_TABS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_tabs;")
if [ "$EXISTING_TABS" -eq 0 ]; then
    print_info "No existing tabs found - importing configurations"
    
    IMPORT_BINARY="$PROJECT_DIR/bin/import_tab_chip"
    mkdir -p "$PROJECT_DIR/bin"
    
    if [ ! -f "$IMPORT_BINARY" ]; then
        print_info "Building import_tab_chip..."
        cd "$PROJECT_DIR"
        go build -o "$IMPORT_BINARY" ./cmd/import_tab_chip/main.go
    fi
    
    print_info "Running import..."
    "$IMPORT_BINARY" -db "$DATABASE_URL" -tabs "$TAB_CONFIG_FILE" -chips "$CHIP_CONFIG_FILE"
    print_success "Configurations imported"
else
    print_success "Configurations already imported ($EXISTING_TABS tabs found)"
fi

# Step 3: Run incremental assignment
print_info "\n=== Step 3: Running Incremental Tab/Chip Assignment ==="
ASSIGN_BINARY="$PROJECT_DIR/bin/assign_tab_chip_incremental"

if [ ! -f "$ASSIGN_BINARY" ]; then
    print_info "Building assign_tab_chip_incremental..."
    cd "$PROJECT_DIR"
    go build -o "$ASSIGN_BINARY" ./cmd/assign_tab_chip_incremental/main.go
fi

print_info "Running incremental assignment (only new markets)..."
"$ASSIGN_BINARY" -db "$DATABASE_URL" -mode incremental
print_success "Incremental assignment completed"

# Step 4: Verify results
print_info "\n=== Step 4: Verifying Results ==="

TOTAL_MARKETS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets;")
MAPPED_MARKETS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets WHERE tab_id IS NOT NULL AND tab_id != '';")
UNMAPPED_MARKETS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_tab_chip_unmapped;")
MAPPED_PERCENTAGE=$(echo "scale=2; $MAPPED_MARKETS * 100 / $TOTAL_MARKETS" | bc)

print_info "Total markets: $TOTAL_MARKETS"
print_info "Mapped markets: $MAPPED_MARKETS ($MAPPED_PERCENTAGE%)"
print_info "Unmapped markets: $UNMAPPED_MARKETS"

if [ "$UNMAPPED_MARKETS" -gt 0 ]; then
    print_warning "Found unmapped markets. Showing summary:"
    psql "$DATABASE_URL" -c "SELECT reason, COUNT(*) as count FROM market_tab_chip_unmapped GROUP BY reason ORDER BY count DESC LIMIT 5;"
fi

print_success "\n=== Incremental Import Completed Successfully ==="
print_info "Summary:"
print_info "  - Total markets: $TOTAL_MARKETS"
print_info "  - Mapped: $MAPPED_MARKETS ($MAPPED_PERCENTAGE%)"
print_info "  - Unmapped: $UNMAPPED_MARKETS"
