-- 添加 producer_id 字段到 markets 表
-- 用于区分 pre-match (producer=3) 和 live (producer=1) 的赔率

ALTER TABLE markets ADD COLUMN IF NOT EXISTS producer_id INTEGER;

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_markets_producer_id ON markets(producer_id);
CREATE INDEX IF NOT EXISTS idx_markets_event_producer ON markets(event_id, producer_id);

-- 注释
COMMENT ON COLUMN markets.producer_id IS 'UOF Producer ID: 1=Live, 3=Pre-match';

