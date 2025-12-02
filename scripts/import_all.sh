#!/bin/bash

# Import Tab and Chip Configuration Script
# This script imports tab and chip configurations from CSV files into the database
# and assigns them to all markets

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
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
print_info "Tab config: $TAB_CONFIG_FILE"
print_info "Chip config: $CHIP_CONFIG_FILE"

# Step 1: Run database migration
print_info "\n=== Step 1: Running Database Migration ==="
if [ -f "$PROJECT_DIR/database/migrations/011_add_tab_chip_fields.sql" ]; then
    print_info "Applying migration: 011_add_tab_chip_fields.sql"
    psql "$DATABASE_URL" -f "$PROJECT_DIR/database/migrations/011_add_tab_chip_fields.sql" > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        print_success "Database migration completed"
    else
        print_warning "Database migration may have already been applied"
    fi
else
    print_warning "Migration file not found: 011_add_tab_chip_fields.sql"
fi

# Step 2: Build import_tab_chip binary
print_info "\n=== Step 2: Building Import Binary ==="
IMPORT_BINARY="$PROJECT_DIR/bin/import_tab_chip"
mkdir -p "$PROJECT_DIR/bin"

if [ -f "$PROJECT_DIR/cmd/import_tab_chip/main.go" ]; then
    print_info "Building import_tab_chip..."
    cd "$PROJECT_DIR"
    go build -o "$IMPORT_BINARY" ./cmd/import_tab_chip/main.go
    if [ $? -eq 0 ]; then
        print_success "Binary built successfully: $IMPORT_BINARY"
    else
        print_error "Failed to build import_tab_chip binary"
        exit 1
    fi
else
    print_error "import_tab_chip source file not found"
    exit 1
fi

# Step 3: Import tab and chip configurations
print_info "\n=== Step 3: Importing Tab and Chip Configurations ==="
print_info "Running: $IMPORT_BINARY -db '$DATABASE_URL' -tabs '$TAB_CONFIG_FILE' -chips '$CHIP_CONFIG_FILE'"

"$IMPORT_BINARY" -db "$DATABASE_URL" -tabs "$TAB_CONFIG_FILE" -chips "$CHIP_CONFIG_FILE"
if [ $? -eq 0 ]; then
    print_success "Tab and chip configurations imported successfully"
else
    print_error "Failed to import configurations"
    exit 1
fi

# Step 4: Build assign_tab_chip binary
print_info "\n=== Step 4: Building Assign Binary ==="
ASSIGN_BINARY="$PROJECT_DIR/bin/assign_tab_chip"

if [ -f "$PROJECT_DIR/cmd/assign_tab_chip/main.go" ]; then
    print_info "Building assign_tab_chip..."
    cd "$PROJECT_DIR"
    go build -o "$ASSIGN_BINARY" ./cmd/assign_tab_chip/main.go
    if [ $? -eq 0 ]; then
        print_success "Binary built successfully: $ASSIGN_BINARY"
    else
        print_error "Failed to build assign_tab_chip binary"
        exit 1
    fi
else
    print_warning "assign_tab_chip source file not found, skipping market assignment"
fi

# Step 5: Assign tab/chip to markets
print_info "\n=== Step 5: Assigning Tab/Chip to Markets ==="
if [ -f "$ASSIGN_BINARY" ]; then
    print_info "Running: $ASSIGN_BINARY -db '$DATABASE_URL'"
    "$ASSIGN_BINARY" -db "$DATABASE_URL"
    if [ $? -eq 0 ]; then
        print_success "Tab/Chip assignment completed successfully"
    else
        print_error "Failed to assign tab/chip to markets"
        exit 1
    fi
else
    print_warning "assign_tab_chip binary not found, skipping market assignment"
fi

# Step 6: Verify import
print_info "\n=== Step 6: Verifying Import ==="

# Count tabs
TAB_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_tabs;")
print_info "Total tabs in database: $TAB_COUNT"

# Count chips
CHIP_COUNT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_chips;")
print_info "Total chips in database: $CHIP_COUNT"

# Count markets with tab_id assigned
MARKET_WITH_TAB=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets WHERE tab_id IS NOT NULL AND tab_id != '';")
print_info "Markets with tab_id assigned: $MARKET_WITH_TAB"

# Count markets with chip_id assigned
MARKET_WITH_CHIP=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets WHERE chip_id IS NOT NULL AND chip_id != '';")
print_info "Markets with chip_id assigned: $MARKET_WITH_CHIP"

print_success "\n=== Import Completed Successfully ==="
print_info "Summary:"
print_info "  - Tabs: $TAB_COUNT"
print_info "  - Chips: $CHIP_COUNT"
print_info "  - Markets with tab_id: $MARKET_WITH_TAB"
print_info "  - Markets with chip_id: $MARKET_WITH_CHIP"
