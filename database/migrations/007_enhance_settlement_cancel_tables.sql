-- Migration 007: 增强 bet_settlements, bet_cancels 和 rollback 表结构
-- 添加详细字段以支持完整的结算和取消逻辑

-- ============================================================================
-- 1. 增强 bet_settlements 表
-- ============================================================================

-- 检查表是否存在,如果不存在则创建
CREATE TABLE IF NOT EXISTS bet_settlements (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 添加新字段 (如果不存在)
DO $$ 
BEGIN
    -- 添加 market_id
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='bet_settlements' AND column_name='market_id') THEN
        ALTER TABLE bet_settlements ADD COLUMN market_id VARCHAR(50);
    END IF;
    
    -- 添加 specifiers
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='bet_settlements' AND column_name='specifiers') THEN
        ALTER TABLE bet_settlements ADD COLUMN specifiers VARCHAR(200);
    END IF;
    
    -- 添加 outcome_id
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='bet_settlements' AND column_name='outcome_id') THEN
        ALTER TABLE bet_settlements ADD COLUMN outcome_id VARCHAR(50);
    END IF;
    
    -- 添加 result
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='bet_settlements' AND column_name='result') THEN
        ALTER TABLE bet_settlements ADD COLUMN result INTEGER;
    END IF;
    
    -- 添加 certainty
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='bet_settlements' AND column_name='certainty') THEN
        ALTER TABLE bet_settlements ADD COLUMN certainty INTEGER;
    END IF;
    
    -- 添加 void_factor
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='bet_settlements' AND column_name='void_factor') THEN
        ALTER TABLE bet_settlements ADD COLUMN void_factor DECIMAL(5,4);
    END IF;
    
    -- 添加 dead_heat_factor
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='bet_settlements' AND column_name='dead_heat_factor') THEN
        ALTER TABLE bet_settlements ADD COLUMN dead_heat_factor DECIMAL(5,4);
    END IF;
    
    -- 重命名 product_id 为 producer_id (如果需要)
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name='bet_settlements' AND column_name='product_id') 
       AND NOT EXISTS (SELECT 1 FROM information_schema.columns 
                       WHERE table_name='bet_settlements' AND column_name='producer_id') THEN
        ALTER TABLE bet_settlements RENAME COLUMN product_id TO producer_id;
    END IF;
END $$;

-- 删除旧的唯一约束 (如果存在)
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'bet_settlements_event_id_product_id_timestamp_key') THEN
        ALTER TABLE bet_settlements DROP CONSTRAINT bet_settlements_event_id_product_id_timestamp_key;
    END IF;
END $$;

-- 创建新的唯一约束
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'bet_settlements_unique_key') THEN
        ALTER TABLE bet_settlements 
        ADD CONSTRAINT bet_settlements_unique_key 
        UNIQUE (event_id, market_id, specifiers, outcome_id, producer_id);
    END IF;
END $$;

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_bet_settlements_event_id ON bet_settlements(event_id);
CREATE INDEX IF NOT EXISTS idx_bet_settlements_market ON bet_settlements(event_id, market_id);

-- ============================================================================
-- 2. 创建 bet_cancels 表
-- ============================================================================

CREATE TABLE IF NOT EXISTS bet_cancels (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    market_id VARCHAR(50),
    specifiers VARCHAR(200),
    void_reason INTEGER,
    start_time BIGINT,
    end_time BIGINT,
    superceded_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT bet_cancels_unique_key UNIQUE (event_id, market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_bet_cancels_event_id ON bet_cancels(event_id);
CREATE INDEX IF NOT EXISTS idx_bet_cancels_market ON bet_cancels(event_id, market_id);

-- ============================================================================
-- 3. 创建 rollback_bet_settlements 表
-- ============================================================================

CREATE TABLE IF NOT EXISTS rollback_bet_settlements (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    market_id VARCHAR(50),
    specifiers VARCHAR(200),
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT rollback_bet_settlements_unique_key UNIQUE (event_id, market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_rollback_settlements_event_id ON rollback_bet_settlements(event_id);
CREATE INDEX IF NOT EXISTS idx_rollback_settlements_market ON rollback_bet_settlements(event_id, market_id);

-- ============================================================================
-- 4. 创建 rollback_bet_cancels 表
-- ============================================================================

CREATE TABLE IF NOT EXISTS rollback_bet_cancels (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    market_id VARCHAR(50),
    specifiers VARCHAR(200),
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT rollback_bet_cancels_unique_key UNIQUE (event_id, market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_rollback_cancels_event_id ON rollback_bet_cancels(event_id);
CREATE INDEX IF NOT EXISTS idx_rollback_cancels_market ON rollback_bet_cancels(event_id, market_id);

-- ============================================================================
-- 完成
-- ============================================================================

-- 输出迁移完成信息
DO $$ 
BEGIN
    RAISE NOTICE 'Migration 007 completed successfully';
    RAISE NOTICE '- Enhanced bet_settlements table with detailed fields';
    RAISE NOTICE '- Created bet_cancels table';
    RAISE NOTICE '- Created rollback_bet_settlements table';
    RAISE NOTICE '- Created rollback_bet_cancels table';
END $$;

