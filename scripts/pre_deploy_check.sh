#!/bin/bash

# Pre-Deployment Check Script
# This script verifies that all migration files are present and valid
# before deploying to Railway

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

echo "=== Pre-Deployment Check ==="
echo ""

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

print_info "Checking project structure..."

# Check 1: Verify migration file exists
print_info "\n1. Checking migration files..."
MIGRATION_FILE="$PROJECT_DIR/database/migrations/011_add_tab_chip_fields.sql"
if [ -f "$MIGRATION_FILE" ]; then
    print_success "Migration file exists: $MIGRATION_FILE"
    LINES=$(wc -l < "$MIGRATION_FILE")
    print_info "   File size: $LINES lines"
else
    print_error "Migration file not found: $MIGRATION_FILE"
    exit 1
fi

# Check 2: Verify CSV files exist
print_info "\n2. Checking configuration files..."
TAB_CONFIG="$PROJECT_DIR/final_tab_chip_config.csv"
CHIP_CONFIG="$PROJECT_DIR/final_chip_enumeration.csv"

if [ -f "$TAB_CONFIG" ]; then
    print_success "Tab config file exists"
    TAB_LINES=$(wc -l < "$TAB_CONFIG")
    print_info "   Rows: $TAB_LINES"
else
    print_error "Tab config file not found: $TAB_CONFIG"
    exit 1
fi

if [ -f "$CHIP_CONFIG" ]; then
    print_success "Chip config file exists"
    CHIP_LINES=$(wc -l < "$CHIP_CONFIG")
    print_info "   Rows: $CHIP_LINES"
else
    print_error "Chip config file not found: $CHIP_CONFIG"
    exit 1
fi

# Check 3: Verify Go source files exist
print_info "\n3. Checking Go source files..."
GO_FILES=(
    "database/models_tab_chip.go"
    "services/market_tab_chip_service.go"
    "handlers/market_tab_chip_handler.go"
    "cmd/import_tab_chip/main.go"
    "cmd/assign_tab_chip/main.go"
)

for file in "${GO_FILES[@]}"; do
    if [ -f "$PROJECT_DIR/$file" ]; then
        print_success "Found: $file"
    else
        print_error "Missing: $file"
        exit 1
    fi
done

# Check 4: Verify shell scripts exist and are executable
print_info "\n4. Checking shell scripts..."
SHELL_SCRIPTS=(
    "scripts/import_all.sh"
    "scripts/test_tab_chip.sh"
    "railway-post-deploy.sh"
)

for script in "${SHELL_SCRIPTS[@]}"; do
    if [ -f "$PROJECT_DIR/$script" ]; then
        if [ -x "$PROJECT_DIR/$script" ]; then
            print_success "Found and executable: $script"
        else
            print_warning "Found but not executable: $script"
            chmod +x "$PROJECT_DIR/$script"
            print_success "Made executable: $script"
        fi
    else
        print_error "Missing: $script"
        exit 1
    fi
done

# Check 5: Verify documentation files exist
print_info "\n5. Checking documentation files..."
DOC_FILES=(
    "MARKET_TAB_CHIP_IMPLEMENTATION.md"
    "DEPLOYMENT_GUIDE.md"
    "QUICKSTART.md"
    "IMPLEMENTATION_SUMMARY.md"
    "CHANGELOG_TAB_CHIP.md"
)

for doc in "${DOC_FILES[@]}"; do
    if [ -f "$PROJECT_DIR/$doc" ]; then
        print_success "Found: $doc"
    else
        print_error "Missing: $doc"
        exit 1
    fi
done

# Check 6: Verify migration file contains required elements
print_info "\n6. Checking migration file content..."
CHECKS=(
    "CREATE TABLE IF NOT EXISTS market_tabs"
    "CREATE TABLE IF NOT EXISTS market_chips"
    "CREATE TABLE IF NOT EXISTS market_tab_chip_mapping"
    "CREATE TABLE IF NOT EXISTS market_groups_cache"
    "CREATE TABLE IF NOT EXISTS market_specifiers_cache"
    "ADD COLUMN IF NOT EXISTS tab_id"
    "ADD COLUMN IF NOT EXISTS chip_id"
    "BEGIN;"
    "COMMIT;"
)

for check in "${CHECKS[@]}"; do
    if grep -q "$check" "$MIGRATION_FILE"; then
        print_success "Found: $check"
    else
        print_error "Missing: $check"
        exit 1
    fi
done

# Check 7: Verify git status
print_info "\n7. Checking git status..."
cd "$PROJECT_DIR"
if git rev-parse --git-dir > /dev/null 2>&1; then
    print_success "Project is a git repository"
    
    # Check if there are uncommitted changes
    if git diff-index --quiet HEAD -- 2>/dev/null; then
        print_success "No uncommitted changes"
    else
        print_warning "There are uncommitted changes"
        echo "   Run: git status"
    fi
else
    print_warning "Not a git repository"
fi

# Check 8: Verify database connectivity (if DATABASE_URL is set)
print_info "\n8. Checking database connectivity..."
if [ -z "$DATABASE_URL" ]; then
    print_warning "DATABASE_URL not set - skipping database check"
else
    print_info "DATABASE_URL is set"
    if psql "$DATABASE_URL" -c "SELECT 1" > /dev/null 2>&1; then
        print_success "Database connection successful"
    else
        print_error "Failed to connect to database"
        exit 1
    fi
fi

# Summary
print_success "\n=== Pre-Deployment Check Passed ==="
echo ""
echo "All checks passed! Ready to deploy."
echo ""
echo "Next steps:"
echo "1. Commit changes: git add . && git commit -m 'Add market tab/chip display implementation'"
echo "2. Push to main: git push origin main"
echo "3. Railway will automatically deploy"
echo "4. Monitor deployment: railway logs"
