-- ============================================================================
-- Betradar UOF Service - Complete Database Initialization Script
-- ============================================================================
-- This script creates ALL tables required by the service
-- Run this after database wipe or on fresh deployment
-- ============================================================================

-- ============================================================================
-- Core Tables
-- ============================================================================

-- 1. UOF Messages Table (原始消息存储)
CREATE TABLE IF NOT EXISTS uof_messages (
    id BIGSERIAL PRIMARY KEY,
    message_type VARCHAR(50) NOT NULL,
    event_id VARCHAR(100),
    product_id INTEGER,
    sport_id VARCHAR(50),
    routing_key VARCHAR(200),
    xml_content TEXT NOT NULL,
    timestamp BIGINT,
    received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_uof_messages_type ON uof_messages(message_type);
CREATE INDEX IF NOT EXISTS idx_uof_messages_event_id ON uof_messages(event_id);
CREATE INDEX IF NOT EXISTS idx_uof_messages_received_at ON uof_messages(received_at);
CREATE INDEX IF NOT EXISTS idx_uof_messages_timestamp ON uof_messages(timestamp);

-- 2. Tracked Events Table (跟踪的赛事)
CREATE TABLE IF NOT EXISTS tracked_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) UNIQUE NOT NULL,
    srn_id VARCHAR(200),
    sport_id VARCHAR(50),
    sport VARCHAR(50),
    status VARCHAR(50) DEFAULT 'active',
    schedule_time TIMESTAMP,
    home_team_id VARCHAR(100),
    home_team_name VARCHAR(200),
    away_team_id VARCHAR(100),
    away_team_name VARCHAR(200),
    home_score INTEGER,
    away_score INTEGER,
    match_status VARCHAR(50),
    match_time VARCHAR(50),
    message_count INTEGER DEFAULT 0,
    last_message_at TIMESTAMP,
    subscribed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Popularity fields (Migration 010)
    attendance INTEGER,
    sellout BOOLEAN DEFAULT FALSE,
    feature_match BOOLEAN DEFAULT FALSE,
    live_video_available BOOLEAN DEFAULT FALSE,
    live_data_available BOOLEAN DEFAULT FALSE,
    broadcasts_count INTEGER DEFAULT 0,
    broadcasts_data JSONB,
    social_links JSONB,
    ticket_url TEXT,
    popularity_score DECIMAL(5,2) DEFAULT 0.0
);

CREATE INDEX IF NOT EXISTS idx_tracked_events_event_id ON tracked_events(event_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_srn_id ON tracked_events(srn_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_sport_id ON tracked_events(sport_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_status ON tracked_events(status);
CREATE INDEX IF NOT EXISTS idx_tracked_events_schedule_time ON tracked_events(schedule_time);
CREATE INDEX IF NOT EXISTS idx_tracked_events_subscribed ON tracked_events(subscribed);
CREATE INDEX IF NOT EXISTS idx_tracked_events_popularity_score ON tracked_events(popularity_score DESC);
CREATE INDEX IF NOT EXISTS idx_tracked_events_feature_match ON tracked_events(feature_match) WHERE feature_match = TRUE;
CREATE INDEX IF NOT EXISTS idx_tracked_events_sellout ON tracked_events(sellout) WHERE sellout = TRUE;
CREATE INDEX IF NOT EXISTS idx_tracked_events_broadcasts_count ON tracked_events(broadcasts_count DESC);

-- ============================================================================
-- Odds & Markets Tables
-- ============================================================================

-- 3. Markets Table (盘口表)
CREATE TABLE IF NOT EXISTS markets (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    sr_market_id VARCHAR(200) NOT NULL,
    market_type VARCHAR(100),
    market_name VARCHAR(200),
    specifiers TEXT,
    status VARCHAR(50),
    producer_id INTEGER,
    home_team_name VARCHAR(200),
    away_team_name VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, sr_market_id, specifiers)
);

CREATE INDEX IF NOT EXISTS idx_markets_event_id ON markets(event_id);
CREATE INDEX IF NOT EXISTS idx_markets_sr_market_id ON markets(sr_market_id);
CREATE INDEX IF NOT EXISTS idx_markets_market_type ON markets(market_type);
CREATE INDEX IF NOT EXISTS idx_markets_status ON markets(status);
CREATE INDEX IF NOT EXISTS idx_markets_producer_id ON markets(producer_id);

-- 4. Odds Table (赔率表)
CREATE TABLE IF NOT EXISTS odds (
    id SERIAL PRIMARY KEY,
    market_id INTEGER REFERENCES markets(id) ON DELETE CASCADE,
    event_id VARCHAR(100) NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name VARCHAR(200),
    odds_value DECIMAL(10, 2),
    probability DECIMAL(5, 4),
    active BOOLEAN DEFAULT true,
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(market_id, outcome_id)
);

CREATE INDEX IF NOT EXISTS idx_odds_market_id ON odds(market_id);
CREATE INDEX IF NOT EXISTS idx_odds_event_id ON odds(event_id);
CREATE INDEX IF NOT EXISTS idx_odds_outcome_id ON odds(outcome_id);
CREATE INDEX IF NOT EXISTS idx_odds_timestamp ON odds(timestamp);

-- 5. Odds History Table (赔率历史表)
CREATE TABLE IF NOT EXISTS odds_history (
    id SERIAL PRIMARY KEY,
    market_id INTEGER REFERENCES markets(id) ON DELETE CASCADE,
    event_id VARCHAR(100) NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name VARCHAR(200),
    odds_value DECIMAL(10, 2),
    probability DECIMAL(5, 4),
    change_type VARCHAR(20),
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_odds_history_market_id ON odds_history(market_id);
CREATE INDEX IF NOT EXISTS idx_odds_history_event_id ON odds_history(event_id);
CREATE INDEX IF NOT EXISTS idx_odds_history_outcome_id ON odds_history(outcome_id);
CREATE INDEX IF NOT EXISTS idx_odds_history_timestamp ON odds_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_odds_history_created_at ON odds_history(created_at);

-- 6. Odds Changes Table (赔率变化记录)
CREATE TABLE IF NOT EXISTS odds_changes (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    markets_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_odds_changes_event_id ON odds_changes(event_id);
CREATE INDEX IF NOT EXISTS idx_odds_changes_timestamp ON odds_changes(timestamp);

-- ============================================================================
-- Bet Settlement & Cancel Tables
-- ============================================================================

-- 8. Bet Stops Table (投注停止记录)
CREATE TABLE IF NOT EXISTS bet_stops (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    groups VARCHAR(200),
    reason VARCHAR(200),
    market_count INTEGER DEFAULT 0,
    market_status VARCHAR(50),
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bet_stops_event_id ON bet_stops(event_id);
CREATE INDEX IF NOT EXISTS idx_bet_stops_timestamp ON bet_stops(timestamp);

-- 9. Bet Settlements Table (投注结算记录)
CREATE TABLE IF NOT EXISTS bet_settlements (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    certainty INTEGER,
    market_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bet_settlements_event_id ON bet_settlements(event_id);
CREATE INDEX IF NOT EXISTS idx_bet_settlements_timestamp ON bet_settlements(timestamp);

-- 10. Bet Cancels Table (投注取消记录)
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

-- 11. Rollback Bet Settlements Table (结算回滚记录)
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

-- 12. Rollback Bet Cancels Table (取消回滚记录)
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

-- ============================================================================
-- Producer & Recovery Tables
-- ============================================================================

-- 13. Producer Status Table (生产者状态)
CREATE TABLE IF NOT EXISTS producer_status (
    product_id INTEGER PRIMARY KEY,
    status VARCHAR(20) NOT NULL DEFAULT 'online',
    last_alive BIGINT,
    subscribed BOOLEAN DEFAULT FALSE,
    recovery_id BIGINT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_producer_status_status ON producer_status(status);
CREATE INDEX IF NOT EXISTS idx_producer_status_subscribed ON producer_status(subscribed);

-- 14. Recovery Status Table (恢复状态)
CREATE TABLE IF NOT EXISTS recovery_status (
    id BIGSERIAL PRIMARY KEY,
    request_id BIGINT NOT NULL,
    product_id INTEGER NOT NULL,
    node_id INTEGER,
    status VARCHAR(50) DEFAULT 'pending',
    timestamp BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recovery_status_request_id ON recovery_status(request_id);
CREATE INDEX IF NOT EXISTS idx_recovery_status_product_id ON recovery_status(product_id);
CREATE INDEX IF NOT EXISTS idx_recovery_status_status ON recovery_status(status);

-- ============================================================================
-- Market Descriptions & Mappings Tables
-- ============================================================================

-- 15. Market Descriptions Table (盘口描述缓存)
CREATE TABLE IF NOT EXISTS market_descriptions (
    market_id VARCHAR(200) PRIMARY KEY,
    market_name TEXT NOT NULL,
    groups TEXT,
    specifiers TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 16. Outcome Descriptions Table (结果描述缓存)
CREATE TABLE IF NOT EXISTS outcome_descriptions (
    id SERIAL PRIMARY KEY,
    market_id VARCHAR(200) NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name TEXT NOT NULL,
    UNIQUE(market_id, outcome_id)
);

CREATE INDEX IF NOT EXISTS idx_outcome_descriptions_market_id ON outcome_descriptions(market_id);

-- 17. Mapping Outcomes Table (结果映射表)
CREATE TABLE IF NOT EXISTS mapping_outcomes (
    id SERIAL PRIMARY KEY,
    market_id VARCHAR(200) NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    product_outcome_name TEXT,
    product_id INTEGER,
    sport_id VARCHAR(50),
    UNIQUE(market_id, outcome_id, product_id, sport_id)
);

CREATE INDEX IF NOT EXISTS idx_mapping_outcomes_market_id ON mapping_outcomes(market_id);
CREATE INDEX IF NOT EXISTS idx_mapping_outcomes_outcome_id ON mapping_outcomes(outcome_id);

-- ============================================================================
-- Static Data Tables
-- ============================================================================

-- 18. Sports Table (体育类型)
CREATE TABLE IF NOT EXISTS sports (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 19. Categories Table (分类/国家)
CREATE TABLE IF NOT EXISTS categories (
    id VARCHAR(50) PRIMARY KEY,
    sport_id VARCHAR(50) REFERENCES sports(id),
    name VARCHAR(200) NOT NULL,
    country_code VARCHAR(10),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_categories_sport_id ON categories(sport_id);

-- 20. Tournaments Table (联赛/锦标赛)
CREATE TABLE IF NOT EXISTS tournaments (
    id VARCHAR(50) PRIMARY KEY,
    sport_id VARCHAR(50) REFERENCES sports(id),
    category_id VARCHAR(50) REFERENCES categories(id),
    name VARCHAR(200) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tournaments_sport_id ON tournaments(sport_id);
CREATE INDEX IF NOT EXISTS idx_tournaments_category_id ON tournaments(category_id);

-- 21. Void Reasons Table (作废原因)
CREATE TABLE IF NOT EXISTS void_reasons (
    id INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 22. Betstop Reasons Table (停止投注原因)
CREATE TABLE IF NOT EXISTS betstop_reasons (
    id INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 23. Players Table (球员信息)
CREATE TABLE IF NOT EXISTS players (
    player_id VARCHAR(50) PRIMARY KEY,
    player_name VARCHAR(200) NOT NULL,
    nationality VARCHAR(10),
    date_of_birth DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- Comments
-- ============================================================================

COMMENT ON TABLE uof_messages IS 'UOF原始消息存储';
COMMENT ON TABLE tracked_events IS '跟踪的赛事信息';
COMMENT ON TABLE markets IS '盘口表,存储比赛的所有盘口';
COMMENT ON TABLE odds IS '赔率表,存储当前最新的赔率';
COMMENT ON TABLE odds_history IS '赔率历史表,追踪赔率变化';
COMMENT ON TABLE bet_stops IS '投注停止记录';
COMMENT ON TABLE bet_settlements IS '投注结算记录';
COMMENT ON TABLE bet_cancels IS '投注取消记录';
COMMENT ON TABLE producer_status IS '生产者状态监控';
COMMENT ON TABLE market_descriptions IS '盘口描述缓存';
COMMENT ON TABLE outcome_descriptions IS '结果描述缓存';

-- ============================================================================
-- Completion Message
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✅ All tables created successfully!';
    RAISE NOTICE 'Total tables: 23';
    RAISE NOTICE 'Database is ready for Betradar UOF Service';
END $$;

