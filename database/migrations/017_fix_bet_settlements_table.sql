-- Migration 017: 修复 bet_settlements 表结构和约束
-- 问题: ON CONFLICT 子句引用的列没有 UNIQUE 约束
-- 解决: 添加缺失的字段和 UNIQUE 约束

BEGIN;

-- 1. 添加缺失的字段（如果不存在）
ALTER TABLE bet_settlements 
ADD COLUMN IF NOT EXISTS producer_id INTEGER;

ALTER TABLE bet_settlements 
ADD COLUMN IF NOT EXISTS sr_market_id VARCHAR(200);

ALTER TABLE bet_settlements 
ADD COLUMN IF NOT EXISTS specifiers TEXT;

ALTER TABLE bet_settlements 
ADD COLUMN IF NOT EXISTS outcome_id VARCHAR(200);

ALTER TABLE bet_settlements 
ADD COLUMN IF NOT EXISTS result INTEGER;

ALTER TABLE bet_settlements 
ADD COLUMN IF NOT EXISTS void_factor DECIMAL(5, 4);

ALTER TABLE bet_settlements 
ADD COLUMN IF NOT EXISTS dead_heat_factor DECIMAL(5, 4);

-- 2. 添加 UNIQUE 约束（如果不存在）
-- 首先检查是否已存在约束，如果不存在则添加
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE table_name = 'bet_settlements' 
        AND constraint_type = 'UNIQUE'
        AND constraint_name = 'unique_bet_settlement'
    ) THEN
        ALTER TABLE bet_settlements 
        ADD CONSTRAINT unique_bet_settlement 
        UNIQUE (event_id, sr_market_id, specifiers, outcome_id, producer_id);
    END IF;
END $$;

-- 3. 添加索引
CREATE INDEX IF NOT EXISTS idx_bet_settlements_market_id ON bet_settlements(sr_market_id);
CREATE INDEX IF NOT EXISTS idx_bet_settlements_outcome_id ON bet_settlements(outcome_id);
CREATE INDEX IF NOT EXISTS idx_bet_settlements_producer_id ON bet_settlements(producer_id);

COMMIT;

-- 完成
SELECT '✅ Migration 017: Fixed bet_settlements table structure and constraints' AS status;
