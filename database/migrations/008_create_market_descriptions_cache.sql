-- Migration 008: 创建 Market Descriptions 缓存表
-- 用于本地持久化 market 和 outcome 的名称映射

-- 1. 创建 market_descriptions 表
CREATE TABLE IF NOT EXISTS market_descriptions (
    market_id VARCHAR(50) PRIMARY KEY,
    market_name TEXT NOT NULL,
    groups TEXT,
    specifiers JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_market_descriptions_updated_at ON market_descriptions(updated_at);

-- 2. 创建 outcome_descriptions 表
CREATE TABLE IF NOT EXISTS outcome_descriptions (
    id SERIAL PRIMARY KEY,
    market_id VARCHAR(50) NOT NULL,
    outcome_id VARCHAR(50) NOT NULL,
    outcome_name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(market_id, outcome_id)
);

CREATE INDEX IF NOT EXISTS idx_outcome_descriptions_market_id ON outcome_descriptions(market_id);
CREATE INDEX IF NOT EXISTS idx_outcome_descriptions_updated_at ON outcome_descriptions(updated_at);

-- 完成
SELECT '✅ Migration 008: Market descriptions cache tables created' AS status;
