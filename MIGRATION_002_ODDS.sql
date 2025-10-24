-- 快速执行脚本: 添加盘口和赔率表
-- 执行方式: 复制粘贴到 Railway PostgreSQL Query 窗口

-- 1. 创建盘口表
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

-- 2. 创建赔率表
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

-- 3. 创建赔率历史表
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

-- 4. 添加注释
COMMENT ON TABLE markets IS '盘口表,存储比赛的所有盘口';
COMMENT ON TABLE odds IS '赔率表,存储当前最新的赔率';
COMMENT ON TABLE odds_history IS '赔率历史表,追踪赔率变化';

-- 5. 验证表创建成功
SELECT 
    'markets' as table_name, 
    COUNT(*) as row_count 
FROM markets
UNION ALL
SELECT 
    'odds' as table_name, 
    COUNT(*) as row_count 
FROM odds
UNION ALL
SELECT 
    'odds_history' as table_name, 
    COUNT(*) as row_count 
FROM odds_history;

-- 预期结果: 3 行,每个表的行数(初始为 0)

