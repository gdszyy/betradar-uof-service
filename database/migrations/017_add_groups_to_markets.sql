-- 添加 groups 字段到 markets 表
-- 用于存储市场分组信息（如 "all|score|regular_play"）

ALTER TABLE markets ADD COLUMN IF NOT EXISTS groups TEXT;

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_markets_groups ON markets(groups);

-- 从 market_descriptions 同步现有数据
UPDATE markets m
SET groups = md.groups, updated_at = CURRENT_TIMESTAMP
FROM market_descriptions md
WHERE m.sr_market_id = md.market_id
AND (m.groups IS NULL OR m.groups = '')
AND md.groups IS NOT NULL AND md.groups != '';
