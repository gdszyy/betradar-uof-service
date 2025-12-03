-- Migration 016: 为 outcome_descriptions 表添加 variant 相关字段
-- 用于支持可变盘口（Variant Market）的结果描述缓存
-- 这是修复 "Outcome name not found" 异常的关键迁移

-- 1. 添加 is_variant 字段
ALTER TABLE outcome_descriptions 
ADD COLUMN IF NOT EXISTS is_variant BOOLEAN DEFAULT FALSE;

-- 2. 添加 variant_urn 字段
ALTER TABLE outcome_descriptions 
ADD COLUMN IF NOT EXISTS variant_urn VARCHAR(500);

-- 3. 为 is_variant 字段添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_outcome_descriptions_is_variant ON outcome_descriptions(is_variant) WHERE is_variant = TRUE;

-- 4. 为 variant_urn 字段添加索引
CREATE INDEX IF NOT EXISTS idx_outcome_descriptions_variant_urn ON outcome_descriptions(variant_urn);

-- 完成
SELECT '✅ Migration 016: Added variant fields to outcome_descriptions table' AS status;
