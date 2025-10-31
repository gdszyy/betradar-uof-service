-- 数据库迁移脚本: 将所有表的 market_id 字段重命名为 sr_market_id
-- 目的: 区分 Sportradar Market ID 和 Database Primary Key

-- 1. bet_settlements 表
ALTER TABLE IF EXISTS bet_settlements RENAME COLUMN market_id TO sr_market_id;

-- 2. bet_cancels 表
ALTER TABLE IF EXISTS bet_cancels RENAME COLUMN market_id TO sr_market_id;

-- 3. rollback_bet_settlements 表
ALTER TABLE IF EXISTS rollback_bet_settlements RENAME COLUMN market_id TO sr_market_id;

-- 4. rollback_bet_cancels 表
ALTER TABLE IF EXISTS rollback_bet_cancels RENAME COLUMN market_id TO sr_market_id;

-- 5. outcomes 表 (如果有 market_id 字段)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'outcomes' AND column_name = 'market_id'
    ) THEN
        ALTER TABLE outcomes RENAME COLUMN market_id TO sr_market_id;
    END IF;
END $$;

-- 验证修改
SELECT 
    table_name, 
    column_name, 
    data_type 
FROM information_schema.columns 
WHERE column_name IN ('market_id', 'sr_market_id')
    AND table_schema = 'public'
ORDER BY table_name, column_name;
