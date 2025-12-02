#!/bin/bash

# Railway Post-Deployment Script
# This script runs after Railway deploys the application
# It handles database migrations, configuration imports, and tab/chip assignments

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Verify DATABASE_URL is set
if [ -z "$DATABASE_URL" ]; then
    print_error "DATABASE_URL environment variable is not set"
    exit 1
fi

print_info "Starting post-deployment setup..."
print_info "Database: $DATABASE_URL"

# Step 1: Run database migrations
print_info "\n=== Step 1: Running Database Migrations ==="
if [ -f "database/migrations/011_add_tab_chip_fields.sql" ]; then
    print_info "Applying migration: 011_add_tab_chip_fields.sql"
    psql "$DATABASE_URL" -f database/migrations/011_add_tab_chip_fields.sql > /dev/null 2>&1
    print_success "Migration applied successfully"
else
    print_warning "Migration file not found: database/migrations/011_add_tab_chip_fields.sql"
fi

# Step 2: Check if configurations need to be imported
print_info "\n=== Step 2: Checking Configuration Status ==="
EXISTING_TABS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_tabs;" 2>/dev/null || echo "0")
EXISTING_TABS=$(echo $EXISTING_TABS | xargs)

if [ "$EXISTING_TABS" -eq 0 ]; then
    print_info "No existing configurations found - importing..."
    
    # Build and run import tool
    if [ -f "cmd/import_tab_chip/main.go" ]; then
        print_info "Building import_tab_chip tool..."
        go build -o /tmp/import_tab_chip ./cmd/import_tab_chip/main.go
        
        if [ -f "final_tab_chip_config.csv" ] && [ -f "final_chip_enumeration.csv" ]; then
            print_info "Running import..."
            /tmp/import_tab_chip -db "$DATABASE_URL" \
                -tabs final_tab_chip_config.csv \
                -chips final_chip_enumeration.csv
            print_success "Configurations imported successfully"
        else
            print_warning "Configuration CSV files not found"
        fi
    else
        print_warning "Import tool source not found"
    fi
else
    print_success "Configurations already imported ($EXISTING_TABS tabs found)"
fi

# Step 3: Run incremental tab/chip assignment
print_info "\n=== Step 3: Running Incremental Tab/Chip Assignment ==="
if [ -f "cmd/assign_tab_chip_incremental/main.go" ]; then
    print_info "Building assign_tab_chip_incremental tool..."
    go build -o /tmp/assign_tab_chip_incremental ./cmd/assign_tab_chip_incremental/main.go
    
    print_info "Running incremental assignment (only new markets)..."
    /tmp/assign_tab_chip_incremental -db "$DATABASE_URL" -mode incremental
    print_success "Incremental assignment completed"
else
    print_warning "Assignment tool source not found"
fi

# Step 4: Verify results
print_info "\n=== Step 4: Verifying Results ==="
TOTAL_MARKETS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets;" 2>/dev/null || echo "0")
MAPPED_MARKETS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM markets WHERE tab_id IS NOT NULL AND tab_id != '';" 2>/dev/null || echo "0")
UNMAPPED_MARKETS=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM market_tab_chip_unmapped;" 2>/dev/null || echo "0")

TOTAL_MARKETS=$(echo $TOTAL_MARKETS | xargs)
MAPPED_MARKETS=$(echo $MAPPED_MARKETS | xargs)
UNMAPPED_MARKETS=$(echo $UNMAPPED_MARKETS | xargs)

if [ "$TOTAL_MARKETS" -gt 0 ]; then
    MAPPED_PERCENTAGE=$(echo "scale=2; $MAPPED_MARKETS * 100 / $TOTAL_MARKETS" | bc)
else
    MAPPED_PERCENTAGE="0"
fi

print_info "Total markets: $TOTAL_MARKETS"
print_info "Mapped markets: $MAPPED_MARKETS ($MAPPED_PERCENTAGE%)"
print_info "Unmapped markets: $UNMAPPED_MARKETS"

if [ "$UNMAPPED_MARKETS" -gt 0 ]; then
    print_warning "Found $UNMAPPED_MARKETS unmapped markets"
    print_info "Top unmapped reasons:"
    psql "$DATABASE_URL" -c "SELECT reason, COUNT(*) as count FROM market_tab_chip_unmapped GROUP BY reason ORDER BY count DESC LIMIT 3;" 2>/dev/null || true
fi

print_success "\n=== Post-Deployment Setup Completed Successfully ==="
print_info "Application is ready to start"
