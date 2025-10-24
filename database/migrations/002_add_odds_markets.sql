-- 002_add_odds_markets.sql
-- 添加盘口和赔率表

-- 盘口表 (markets)
CREATE TABLE IF NOT EXISTS markets (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    market_id VARCHAR(200) NOT NULL,
    market_type VARCHAR(100),
    market_name VARCHAR(200),
    specifiers TEXT,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, market_id, specifiers)
);

CREATE INDEX IF NOT EXISTS idx_markets_event_id ON markets(event_id);
CREATE INDEX IF NOT EXISTS idx_markets_market_type ON markets(market_type);
CREATE INDEX IF NOT EXISTS idx_markets_status ON markets(status);

-- 赔率表 (odds)
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

-- 赔率历史表 (odds_history)
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

-- 添加注释
COMMENT ON TABLE markets IS '盘口表,存储比赛的所有盘口';
COMMENT ON TABLE odds IS '赔率表,存储当前最新的赔率';
COMMENT ON TABLE odds_history IS '赔率历史表,追踪赔率变化';

COMMENT ON COLUMN markets.market_id IS 'Betradar 盘口 ID';
COMMENT ON COLUMN markets.market_type IS '盘口类型,如 1x2, handicap, totals';
COMMENT ON COLUMN markets.specifiers IS '盘口参数,如让球数、大小球数';
COMMENT ON COLUMN odds.odds_value IS '赔率值';
COMMENT ON COLUMN odds.probability IS '隐含概率';
COMMENT ON COLUMN odds_history.change_type IS '变化类型: up, down, new';

