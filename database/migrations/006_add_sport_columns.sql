-- 006_add_sport_columns.sql
-- 添加 sport 和 sport_name 列到 tracked_events 表

-- 添加 sport 列（运动名称，如 "ESport Counter-Strike"）
ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS sport VARCHAR(100);

-- 添加 sport_name 列（运动名称的别名，与 sport 相同）
ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS sport_name VARCHAR(100);

-- 添加 home_team_id 列（如果不存在）
ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS home_team_id VARCHAR(100);

-- 添加 home_team_name 列（如果不存在）
ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS home_team_name VARCHAR(255);

-- 添加 away_team_id 列（如果不存在）
ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS away_team_id VARCHAR(100);

-- 添加 away_team_name 列（如果不存在）
ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS away_team_name VARCHAR(255);

-- 添加索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_tracked_events_sport_id ON tracked_events(sport_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_home_team_id ON tracked_events(home_team_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_away_team_id ON tracked_events(away_team_id);

-- 注释
COMMENT ON COLUMN tracked_events.sport IS 'Sport name from Fixture API (e.g., ESport Counter-Strike, Soccer)';
COMMENT ON COLUMN tracked_events.sport_name IS 'Sport name alias (same as sport)';
COMMENT ON COLUMN tracked_events.home_team_id IS 'Home team ID (e.g., sr:competitor:235852)';
COMMENT ON COLUMN tracked_events.home_team_name IS 'Home team name (e.g., Team Liquid)';
COMMENT ON COLUMN tracked_events.away_team_id IS 'Away team ID (e.g., sr:competitor:1045391)';
COMMENT ON COLUMN tracked_events.away_team_name IS 'Away team name (e.g., Betboom Team)';

