-- Migration 011: Add Tab and Chip fields for market card display
-- ============================================================================
-- This migration adds tab_id and chip_id fields to the markets table
-- to support the market card display scheme with tabs and chips
-- ============================================================================
-- This migration is IDEMPOTENT - it can be run multiple times safely
-- All table and column creation use IF NOT EXISTS
-- ============================================================================

BEGIN;

-- 1. Add tab_id and chip_id columns to markets table
ALTER TABLE markets
ADD COLUMN IF NOT EXISTS tab_id VARCHAR(50),
ADD COLUMN IF NOT EXISTS chip_id VARCHAR(200);

-- Create indexes for better query performance
-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_markets_tab_id ON markets(tab_id);
CREATE INDEX IF NOT EXISTS idx_markets_chip_id ON markets(chip_id);
CREATE INDEX IF NOT EXISTS idx_markets_event_tab_chip ON markets(event_id, tab_id, chip_id);

-- 2. Create tabs table to store tab configurations
CREATE TABLE IF NOT EXISTS market_tabs (
    id VARCHAR(50) PRIMARY KEY,
    label VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,  -- 'group' or 'specifier_aggregate'
    market_count INTEGER DEFAULT 0,
    chip_specifiers TEXT,  -- comma-separated list of specifier names
    group_name VARCHAR(50),  -- for group-based tabs
    primary_specifier VARCHAR(50),  -- for specifier_aggregate tabs
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on market_tabs
CREATE INDEX IF NOT EXISTS idx_market_tabs_type ON market_tabs(type);
CREATE INDEX IF NOT EXISTS idx_market_tabs_display_order ON market_tabs(display_order);

-- 3. Create chips table to store chip configurations
CREATE TABLE IF NOT EXISTS market_chips (
    id VARCHAR(200) PRIMARY KEY,
    tab_id VARCHAR(50) NOT NULL REFERENCES market_tabs(id) ON DELETE CASCADE,
    specifier VARCHAR(50),
    value VARCHAR(100),
    label VARCHAR(200) NOT NULL,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on market_chips
CREATE INDEX IF NOT EXISTS idx_market_chips_tab_id ON market_chips(tab_id);
CREATE INDEX IF NOT EXISTS idx_market_chips_specifier ON market_chips(specifier, value);

-- 4. Create market_tab_chip_mapping table to track market -> tab -> chip relationships
CREATE TABLE IF NOT EXISTS market_tab_chip_mapping (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    event_id VARCHAR(100) NOT NULL,
    tab_id VARCHAR(50) NOT NULL REFERENCES market_tabs(id) ON DELETE CASCADE,
    chip_id VARCHAR(200),
    specifier_name VARCHAR(50),
    specifier_value VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(market_id, tab_id, chip_id)
);

-- Create indexes on market_tab_chip_mapping
CREATE INDEX IF NOT EXISTS idx_mapping_market_id ON market_tab_chip_mapping(market_id);
CREATE INDEX IF NOT EXISTS idx_mapping_event_id ON market_tab_chip_mapping(event_id);
CREATE INDEX IF NOT EXISTS idx_mapping_tab_id ON market_tab_chip_mapping(tab_id);
CREATE INDEX IF NOT EXISTS idx_mapping_chip_id ON market_tab_chip_mapping(chip_id);
CREATE INDEX IF NOT EXISTS idx_mapping_event_tab ON market_tab_chip_mapping(event_id, tab_id);

-- 5. Create market_groups_cache table to cache market groups for faster lookup
CREATE TABLE IF NOT EXISTS market_groups_cache (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL UNIQUE REFERENCES markets(id) ON DELETE CASCADE,
    groups TEXT NOT NULL,  -- JSON array of group names
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on market_groups_cache
CREATE INDEX IF NOT EXISTS idx_groups_cache_market_id ON market_groups_cache(market_id);

-- 6. Create market_specifiers_cache table to cache parsed specifiers
CREATE TABLE IF NOT EXISTS market_specifiers_cache (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL UNIQUE REFERENCES markets(id) ON DELETE CASCADE,
    specifiers_json JSONB,  -- parsed specifiers as JSON
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index on market_specifiers_cache
CREATE INDEX IF NOT EXISTS idx_specifiers_cache_market_id ON market_specifiers_cache(market_id);

-- 7. Create market_tab_chip_unmapped table to track markets that couldn't be mapped
-- This table is used to identify and maintain markets that don't fit the current tab/chip scheme
CREATE TABLE IF NOT EXISTS market_tab_chip_unmapped (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL UNIQUE REFERENCES markets(id) ON DELETE CASCADE,
    event_id VARCHAR(100) NOT NULL,
    market_type VARCHAR(100),
    market_name VARCHAR(255),
    groups TEXT,  -- comma-separated list of groups
    specifiers TEXT,  -- specifiers string
    reason TEXT,  -- reason why mapping failed (e.g., 'unknown_group', 'no_matching_specifier')
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on market_tab_chip_unmapped
CREATE INDEX IF NOT EXISTS idx_unmapped_market_id ON market_tab_chip_unmapped(market_id);
CREATE INDEX IF NOT EXISTS idx_unmapped_event_id ON market_tab_chip_unmapped(event_id);
CREATE INDEX IF NOT EXISTS idx_unmapped_reason ON market_tab_chip_unmapped(reason);
CREATE INDEX IF NOT EXISTS idx_unmapped_created_at ON market_tab_chip_unmapped(created_at);

-- 8. Create market_tab_chip_assignment_log table to track assignment history
CREATE TABLE IF NOT EXISTS market_tab_chip_assignment_log (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
    event_id VARCHAR(100) NOT NULL,
    old_tab_id VARCHAR(50),
    new_tab_id VARCHAR(50),
    old_chip_id VARCHAR(200),
    new_chip_id VARCHAR(200),
    assignment_type VARCHAR(50),  -- 'initial', 'update', 'correction'
    assigned_by VARCHAR(100),  -- 'import_tool', 'api', 'manual'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes on market_tab_chip_assignment_log
CREATE INDEX IF NOT EXISTS idx_assignment_log_market_id ON market_tab_chip_assignment_log(market_id);
CREATE INDEX IF NOT EXISTS idx_assignment_log_event_id ON market_tab_chip_assignment_log(event_id);
CREATE INDEX IF NOT EXISTS idx_assignment_log_created_at ON market_tab_chip_assignment_log(created_at);

-- 9. Add comment to markets table explaining new fields
COMMENT ON COLUMN markets.tab_id IS 'Tab ID for market card display (e.g., regular_play, quarters, player_props)';
COMMENT ON COLUMN markets.chip_id IS 'Chip ID for market card display (e.g., quarter_1, goal_1)';

-- 8. Create a view for easier querying of markets with tab and chip info
CREATE OR REPLACE VIEW market_tab_chip_view AS
SELECT 
    m.id as market_id,
    m.event_id,
    m.sr_market_id,
    m.market_type,
    m.market_name,
    m.specifiers,
    m.status,
    m.tab_id,
    m.chip_id,
    mt.label as tab_label,
    mt.type as tab_type,
    mc.label as chip_label,
    mc.specifier as chip_specifier,
    mc.value as chip_value,
    m.created_at,
    m.updated_at
FROM markets m
LEFT JOIN market_tabs mt ON m.tab_id = mt.id
LEFT JOIN market_chips mc ON m.chip_id = mc.id;

-- Verify all tables exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_tabs') THEN
        RAISE EXCEPTION 'market_tabs table was not created successfully';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_chips') THEN
        RAISE EXCEPTION 'market_chips table was not created successfully';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_tab_chip_mapping') THEN
        RAISE EXCEPTION 'market_tab_chip_mapping table was not created successfully';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_groups_cache') THEN
        RAISE EXCEPTION 'market_groups_cache table was not created successfully';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'market_specifiers_cache') THEN
        RAISE EXCEPTION 'market_specifiers_cache table was not created successfully';
    END IF;
    RAISE NOTICE 'All required tables exist';
END $$;

-- Verify all columns exist in markets table
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'markets' AND column_name = 'tab_id') THEN
        RAISE EXCEPTION 'tab_id column was not created in markets table';
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'markets' AND column_name = 'chip_id') THEN
        RAISE EXCEPTION 'chip_id column was not created in markets table';
    END IF;
    RAISE NOTICE 'All required columns exist in markets table';
END $$;

COMMIT;

-- Commit message: Add tab_id and chip_id support for market card display
-- This migration is safe to run multiple times (idempotent)
