-- Migration: 013_create_teams_table.sql
-- Description: 创建队伍表，用于存储队伍信息和 Logo URL
-- Author: Manus AI
-- Date: 2025-11-25

-- 创建队伍表
CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    team_id VARCHAR(100) UNIQUE NOT NULL,           -- 队伍唯一标识符 (例如: sr:competitor:12345)
    team_name VARCHAR(255) NOT NULL,                -- 队伍名称
    sport_id VARCHAR(50),                           -- 体育项目 ID
    sport_name VARCHAR(100),                        -- 体育项目名称
    category_id VARCHAR(50),                        -- 类别 ID (例如: 国家)
    category_name VARCHAR(200),                     -- 类别名称
    logo_url VARCHAR(500),                          -- 队伍 Logo URL
    logo_fetched BOOLEAN DEFAULT FALSE,             -- Logo 是否已获取
    logo_fetch_attempted_at TIMESTAMP,              -- Logo 获取尝试时间
    logo_fetch_retry_count INTEGER DEFAULT 0,       -- Logo 获取重试次数
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS teams_team_id_idx ON teams(team_id);
CREATE INDEX IF NOT EXISTS teams_team_name_idx ON teams(team_name);
CREATE INDEX IF NOT EXISTS teams_sport_id_idx ON teams(sport_id);
CREATE INDEX IF NOT EXISTS teams_logo_fetched_idx ON teams(logo_fetched);

-- 添加注释
COMMENT ON TABLE teams IS '队伍信息表，存储队伍基本信息和 Logo URL';
COMMENT ON COLUMN teams.team_id IS '队伍唯一标识符，来自 Sportradar';
COMMENT ON COLUMN teams.team_name IS '队伍名称';
COMMENT ON COLUMN teams.logo_url IS '队伍 Logo 的 URL 地址';
COMMENT ON COLUMN teams.logo_fetched IS '标识 Logo 是否已成功获取';
COMMENT ON COLUMN teams.logo_fetch_attempted_at IS '最后一次尝试获取 Logo 的时间';
COMMENT ON COLUMN teams.logo_fetch_retry_count IS 'Logo 获取失败后的重试次数';
