-- ============================================================================
-- Betradar UOF Service - Missing Tables Initialization Script
-- ============================================================================
-- This script creates tables that are used in the code but were missing 
-- from the database based on the provided screenshot.
-- ============================================================================

-- 1. Bet Cancels Table (投注取消记录)
CREATE TABLE IF NOT EXISTS bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    start_time BIGINT,
    end_time BIGINT,
    superceded_by VARCHAR(100),
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bet_cancels_event_id ON bet_cancels(event_id);
CREATE INDEX IF NOT EXISTS idx_bet_cancels_timestamp ON bet_cancels(timestamp);

-- 2. Rollback Bet Settlements Table (结算回滚记录)
CREATE TABLE IF NOT EXISTS rollback_bet_settlements (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rollback_bet_settlements_event_id ON rollback_bet_settlements(event_id);
CREATE INDEX IF NOT EXISTS idx_rollback_bet_settlements_timestamp ON rollback_bet_settlements(timestamp);

-- 3. Rollback Bet Cancels Table (取消回滚记录)
CREATE TABLE IF NOT EXISTS rollback_bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rollback_bet_cancels_event_id ON rollback_bet_cancels(event_id);
CREATE INDEX IF NOT EXISTS idx_rollback_bet_cancels_timestamp ON rollback_bet_cancels(timestamp);

-- 4. Mapping Outcomes Table (结果映射表)
CREATE TABLE IF NOT EXISTS mapping_outcomes (
    id SERIAL PRIMARY KEY,
    market_id VARCHAR(50) NOT NULL,
    outcome_id VARCHAR(50) NOT NULL,
    product_outcome_name TEXT,
    product_id INTEGER,
    sport_id VARCHAR(50),
    UNIQUE(market_id, outcome_id, product_id, sport_id)
);

CREATE INDEX IF NOT EXISTS idx_mapping_outcomes_market_id ON mapping_outcomes(market_id);
CREATE INDEX IF NOT EXISTS idx_mapping_outcomes_outcome_id ON mapping_outcomes(outcome_id);

-- 5. Outcomes Table (结果表 - 用于存储盘口结果)
CREATE TABLE IF NOT EXISTS outcomes (
    id SERIAL PRIMARY KEY,
    market_id INTEGER, -- REFERENCES markets(id) ON DELETE CASCADE, (Removed FK for simplicity)
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name VARCHAR(200),
    specifiers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_outcomes_market_id ON outcomes(market_id);
CREATE INDEX IF NOT EXISTS idx_outcomes_outcome_id ON outcomes(outcome_id);

-- 6. Players Table (球员信息)
CREATE TABLE IF NOT EXISTS players (
    player_id VARCHAR(50) PRIMARY KEY,
    player_name VARCHAR(200) NOT NULL,
    nationality VARCHAR(10),
    date_of_birth DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Completion Message
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✅ Missing tables created successfully!';
    RAISE NOTICE 'Total tables created: 6';
END $$;

