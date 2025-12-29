-- 性能优化索引脚本
-- 用于提高事件筛选 API 的查询性能

-- 1. tracked_events 表的索引
-- 状态索引 (用于 is_live 和 status 筛选)
CREATE INDEX IF NOT EXISTS idx_tracked_events_status ON tracked_events(status);

-- 体育类型索引
CREATE INDEX IF NOT EXISTS idx_tracked_events_sport_id ON tracked_events(sport_id);

-- 开赛时间索引 (用于时间范围筛选)
CREATE INDEX IF NOT EXISTS idx_tracked_events_schedule_time ON tracked_events(schedule_time);

-- 队伍 ID 索引
CREATE INDEX IF NOT EXISTS idx_tracked_events_home_team_id ON tracked_events(home_team_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_away_team_id ON tracked_events(away_team_id);

-- 队伍名称索引 (用于 ILIKE 搜索)
CREATE INDEX IF NOT EXISTS idx_tracked_events_home_team_name ON tracked_events(home_team_name);
CREATE INDEX IF NOT EXISTS idx_tracked_events_away_team_name ON tracked_events(away_team_name);

-- 赛事 ID 索引 (用于搜索)
CREATE INDEX IF NOT EXISTS idx_tracked_events_event_id ON tracked_events(event_id);

-- 联赛 ID 索引 (从 srn_id 提取)
CREATE INDEX IF NOT EXISTS idx_tracked_events_srn_id ON tracked_events(srn_id);

-- 复合索引 (用于常见的组合查询)
-- Live 比赛查询 (status + match_status)
CREATE INDEX IF NOT EXISTS idx_tracked_events_live ON tracked_events(status, match_status) WHERE match_status IS NOT NULL;

-- 体育类型 + 时间范围查询
CREATE INDEX IF NOT EXISTS idx_tracked_events_sport_schedule ON tracked_events(sport_id, schedule_time);

-- 状态 + 时间范围查询
CREATE INDEX IF NOT EXISTS idx_tracked_events_status_schedule ON tracked_events(status, schedule_time);

-- 2. markets 表的索引
-- 赛事 ID 索引 (用于 JOIN)
CREATE INDEX IF NOT EXISTS idx_markets_event_id ON markets(event_id);

-- Sportradar Market ID 索引
CREATE INDEX IF NOT EXISTS idx_markets_sr_market_id ON markets(sr_market_id);

-- 盘口组索引 (用于 LIKE 搜索)
CREATE INDEX IF NOT EXISTS idx_markets_groups ON markets(groups);

-- 复合索引 (event_id + sr_market_id)
CREATE INDEX IF NOT EXISTS idx_markets_event_sr_market ON markets(event_id, sr_market_id);

-- 3. 查看索引创建结果
SELECT 
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
AND tablename IN ('tracked_events', 'markets')
ORDER BY tablename, indexname;

