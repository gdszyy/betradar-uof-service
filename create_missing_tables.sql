-- 创建缺失的表和修改现有表结构

-- 1. 修改 bet_settlements 表,添加详细字段
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS sr_market_id VARCHAR(50);
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS specifiers TEXT;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS void_factor FLOAT;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS outcome_id VARCHAR(50);
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS result INTEGER;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS dead_heat_factor FLOAT;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS producer_id INTEGER;

-- 添加唯一约束
CREATE UNIQUE INDEX IF NOT EXISTS idx_bet_settlements_unique 
ON bet_settlements (event_id, sr_market_id, specifiers, outcome_id, producer_id);

-- 2. 创建 bet_cancels 表
CREATE TABLE IF NOT EXISTS bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    sr_market_id VARCHAR(50) NOT NULL,
    specifiers TEXT,
    void_reason INTEGER,
    start_time BIGINT,
    end_time BIGINT,
    superceded_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, sr_market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_bet_cancels_event_id ON bet_cancels(event_id);

-- 3. 创建 rollback_bet_settlements 表
CREATE TABLE IF NOT EXISTS rollback_bet_settlements (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    sr_market_id VARCHAR(50) NOT NULL,
    specifiers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, sr_market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_rollback_bet_settlements_event_id ON rollback_bet_settlements(event_id);

-- 4. 创建 rollback_bet_cancels 表
CREATE TABLE IF NOT EXISTS rollback_bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    sr_market_id VARCHAR(50) NOT NULL,
    specifiers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, sr_market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_rollback_bet_cancels_event_id ON rollback_bet_cancels(event_id);

-- 验证表结构
SELECT 
    table_name, 
    column_name, 
    data_type 
FROM information_schema.columns 
WHERE table_name IN ('bet_settlements', 'bet_cancels', 'rollback_bet_settlements', 'rollback_bet_cancels')
    AND table_schema = 'public'
ORDER BY table_name, ordinal_position;
