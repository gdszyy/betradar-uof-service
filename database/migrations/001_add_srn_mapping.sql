-- 为 tracked_events 表添加比赛详细信息字段
-- 这些字段用于存储从 UOF 消息中解析的比赛数据

-- 添加 SRN ID mapping 字段
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS srn_id VARCHAR(200);

-- 添加比赛时间字段
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS schedule_time TIMESTAMP;

-- 添加主队信息字段
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_team_id VARCHAR(50);
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_team_name VARCHAR(200);

-- 添加客队信息字段
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_team_id VARCHAR(50);
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_team_name VARCHAR(200);

-- 添加比分字段
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_score INTEGER DEFAULT 0;
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_score INTEGER DEFAULT 0;

-- 添加比赛状态字段
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS match_status VARCHAR(50);
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS match_time VARCHAR(20);

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_tracked_events_srn_id ON tracked_events(srn_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_schedule_time ON tracked_events(schedule_time);
CREATE INDEX IF NOT EXISTS idx_tracked_events_home_team_id ON tracked_events(home_team_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_away_team_id ON tracked_events(away_team_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_match_status ON tracked_events(match_status);

-- 添加注释
COMMENT ON COLUMN tracked_events.srn_id IS 'SportRadar SRN ID mapping';
COMMENT ON COLUMN tracked_events.schedule_time IS 'Scheduled match start time from fixture';
COMMENT ON COLUMN tracked_events.home_team_id IS 'Home team SportRadar ID';
COMMENT ON COLUMN tracked_events.home_team_name IS 'Home team name';
COMMENT ON COLUMN tracked_events.away_team_id IS 'Away team SportRadar ID';
COMMENT ON COLUMN tracked_events.away_team_name IS 'Away team name';
COMMENT ON COLUMN tracked_events.home_score IS 'Home team score';
COMMENT ON COLUMN tracked_events.away_score IS 'Away team score';
COMMENT ON COLUMN tracked_events.match_status IS 'Match status code';
COMMENT ON COLUMN tracked_events.match_time IS 'Current match time (e.g., 45:00)';

