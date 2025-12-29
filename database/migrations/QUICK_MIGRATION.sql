-- 快速数据库迁移脚本
-- 为 tracked_events 表添加所有必需字段
-- 可以安全地重复执行 (使用 IF NOT EXISTS)

-- 添加 SRN ID mapping
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS srn_id VARCHAR(200);

-- 添加比赛时间
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS schedule_time TIMESTAMP;

-- 添加主队信息
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_team_id VARCHAR(50);
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_team_name VARCHAR(200);

-- 添加客队信息
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_team_id VARCHAR(50);
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_team_name VARCHAR(200);

-- 添加比分
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS home_score INTEGER DEFAULT 0;
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS away_score INTEGER DEFAULT 0;

-- 添加比赛状态
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS match_status VARCHAR(50);
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS match_time VARCHAR(20);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_tracked_events_srn_id ON tracked_events(srn_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_schedule_time ON tracked_events(schedule_time);
CREATE INDEX IF NOT EXISTS idx_tracked_events_home_team_id ON tracked_events(home_team_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_away_team_id ON tracked_events(away_team_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_match_status ON tracked_events(match_status);

-- 验证字段已添加
SELECT 
    column_name, 
    data_type, 
    is_nullable,
    column_default
FROM information_schema.columns
WHERE table_name = 'tracked_events'
AND column_name IN (
    'srn_id', 'schedule_time', 
    'home_team_id', 'home_team_name',
    'away_team_id', 'away_team_name',
    'home_score', 'away_score',
    'match_status', 'match_time'
)
ORDER BY column_name;

