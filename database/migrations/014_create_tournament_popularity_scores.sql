-- Migration: 014_create_tournament_popularity_scores.sql
-- Description: 创建联赛热度评分表，用于存储联赛的综合热度评分
-- Author: Manus AI
-- Date: 2025-12-01

-- 创建联赛热度评分表
CREATE TABLE IF NOT EXISTS tournament_popularity_scores (
    id SERIAL PRIMARY KEY,
    tournament_id VARCHAR(100) UNIQUE NOT NULL,      -- 联赛唯一标识符 (例如: sr:tournament:17)
    tournament_name VARCHAR(255) NOT NULL,           -- 联赛名称
    category_id VARCHAR(100),                        -- 分类 ID (例如: sr:category:1)
    sport_id VARCHAR(50),                            -- 体育项目 ID
    
    -- 热度评分相关字段
    match_count INTEGER DEFAULT 0,                   -- 当前赛事数量
    total_broadcasts INTEGER DEFAULT 0,              -- 总转播数量
    avg_attendance DECIMAL(10, 2) DEFAULT 0,         -- 平均上座率
    feature_match_count INTEGER DEFAULT 0,           -- 重点赛事数量
    sellout_count INTEGER DEFAULT 0,                 -- 售罄场次数量
    
    -- 综合热度评分 (0-100)
    final_popularity_score DECIMAL(5, 2) DEFAULT 0,  -- 最终热度评分
    
    -- 时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,  -- 创建时间
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP   -- 更新时间
);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_tournament_popularity_tournament_id ON tournament_popularity_scores(tournament_id);
CREATE INDEX IF NOT EXISTS idx_tournament_popularity_category_id ON tournament_popularity_scores(category_id);
CREATE INDEX IF NOT EXISTS idx_tournament_popularity_sport_id ON tournament_popularity_scores(sport_id);
CREATE INDEX IF NOT EXISTS idx_tournament_popularity_score ON tournament_popularity_scores(final_popularity_score DESC);
CREATE INDEX IF NOT EXISTS idx_tournament_popularity_match_count ON tournament_popularity_scores(match_count DESC);

-- 添加注释
COMMENT ON TABLE tournament_popularity_scores IS '联赛热度评分表，存储联赛的综合热度评分';
COMMENT ON COLUMN tournament_popularity_scores.tournament_id IS '联赛唯一标识符，来自 Sportradar';
COMMENT ON COLUMN tournament_popularity_scores.tournament_name IS '联赛名称';
COMMENT ON COLUMN tournament_popularity_scores.category_id IS '分类 ID (国家/地区)';
COMMENT ON COLUMN tournament_popularity_scores.match_count IS '当前该联赛的赛事数量';
COMMENT ON COLUMN tournament_popularity_scores.total_broadcasts IS '该联赛的总转播数量';
COMMENT ON COLUMN tournament_popularity_scores.avg_attendance IS '平均上座率';
COMMENT ON COLUMN tournament_popularity_scores.final_popularity_score IS '综合热度评分 (0-100)';
