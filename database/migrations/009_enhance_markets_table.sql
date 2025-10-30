-- Migration 009: 增强 markets 表结构
-- 为 markets 表添加 home_team_name 和 away_team_name 字段

ALTER TABLE markets
ADD COLUMN IF NOT EXISTS home_team_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS away_team_name VARCHAR(255);

-- 完成
SELECT '✅ Migration 009: markets table enhanced' AS status;
