-- 添加 SRN ID mapping 字段到 ld_matches 表
ALTER TABLE ld_matches ADD COLUMN IF NOT EXISTS srn_id VARCHAR(200);
ALTER TABLE ld_matches ADD COLUMN IF NOT EXISTS schedule_time TIMESTAMP;
ALTER TABLE ld_matches ADD COLUMN IF NOT EXISTS home_team_id VARCHAR(50);
ALTER TABLE ld_matches ADD COLUMN IF NOT EXISTS away_team_id VARCHAR(50);

-- 添加索引
CREATE INDEX IF NOT EXISTS idx_ld_matches_srn_id ON ld_matches(srn_id);
CREATE INDEX IF NOT EXISTS idx_ld_matches_schedule_time ON ld_matches(schedule_time);

-- 添加 SRN ID mapping 字段到 tracked_events 表
ALTER TABLE tracked_events ADD COLUMN IF NOT EXISTS srn_id VARCHAR(200);
CREATE INDEX IF NOT EXISTS idx_tracked_events_srn_id ON tracked_events(srn_id);

