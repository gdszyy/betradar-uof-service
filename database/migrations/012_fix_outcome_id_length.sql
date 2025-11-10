-- Migration 012: 修复 outcome_id 字段长度限制
-- 问题: outcome_id 的 VARCHAR(50) 长度不足以容纳某些长 URN 格式的 ID
-- 例如: sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333 (52字符)

-- 1. 修改 mapping_outcomes 表的 outcome_id 字段长度
ALTER TABLE mapping_outcomes 
ALTER COLUMN outcome_id TYPE VARCHAR(200);

-- 2. 修改 outcome_descriptions 表的 outcome_id 字段长度
ALTER TABLE outcome_descriptions 
ALTER COLUMN outcome_id TYPE VARCHAR(200);

-- 3. 同时修改 market_id 字段长度以防止类似问题
ALTER TABLE mapping_outcomes 
ALTER COLUMN market_id TYPE VARCHAR(200);

ALTER TABLE outcome_descriptions 
ALTER COLUMN market_id TYPE VARCHAR(200);

ALTER TABLE market_descriptions 
ALTER COLUMN market_id TYPE VARCHAR(200);

-- 完成
SELECT '✅ Migration 012: Fixed outcome_id and market_id length limits (50 -> 200)' AS status;
