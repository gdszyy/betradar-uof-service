-- 🔥 HOTFIX: 修复 outcome_id 字段长度限制
-- 
-- 问题描述:
-- mapping_outcomes 和 outcome_descriptions 表中的 outcome_id 字段
-- 定义为 VARCHAR(50),但实际数据中存在超过50字符的 outcome_id
-- 例如: sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333 (52字符)
--
-- 错误信息:
-- pq: value too long for type character varying(50)
--
-- 解决方案:
-- 将相关字段长度从 VARCHAR(50) 扩展到 VARCHAR(200)

BEGIN;

-- 1. 修改 mapping_outcomes 表
ALTER TABLE mapping_outcomes 
ALTER COLUMN outcome_id TYPE VARCHAR(200);

ALTER TABLE mapping_outcomes 
ALTER COLUMN market_id TYPE VARCHAR(200);

-- 2. 修改 outcome_descriptions 表
ALTER TABLE outcome_descriptions 
ALTER COLUMN outcome_id TYPE VARCHAR(200);

ALTER TABLE outcome_descriptions 
ALTER COLUMN market_id TYPE VARCHAR(200);

-- 3. 修改 market_descriptions 表
ALTER TABLE market_descriptions 
ALTER COLUMN market_id TYPE VARCHAR(200);

COMMIT;

-- 验证修改
SELECT 
    table_name, 
    column_name, 
    data_type, 
    character_maximum_length 
FROM information_schema.columns 
WHERE table_name IN ('mapping_outcomes', 'outcome_descriptions', 'market_descriptions')
  AND column_name IN ('market_id', 'outcome_id')
ORDER BY table_name, column_name;

SELECT '✅ HOTFIX 完成: outcome_id 和 market_id 字段长度已扩展至 VARCHAR(200)' AS status;
