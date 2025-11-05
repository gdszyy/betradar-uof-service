-- Migration 011: Fix missing columns in tracked_events table
-- This script adds all columns that should be present in tracked_events but might be missing due to incomplete migrations.

ALTER TABLE tracked_events
ADD COLUMN IF NOT EXISTS srn_id VARCHAR(200),
ADD COLUMN IF NOT EXISTS sport VARCHAR(50),
ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active',
ADD COLUMN IF NOT EXISTS schedule_time TIMESTAMP,
ADD COLUMN IF NOT EXISTS home_team_id VARCHAR(100),
ADD COLUMN IF NOT EXISTS home_team_name VARCHAR(200),
ADD COLUMN IF NOT EXISTS away_team_id VARCHAR(100),
ADD COLUMN IF NOT EXISTS away_team_name VARCHAR(200),
ADD COLUMN IF NOT EXISTS home_score INTEGER,
ADD COLUMN IF NOT EXISTS away_score INTEGER,
ADD COLUMN IF NOT EXISTS match_status VARCHAR(50),
ADD COLUMN IF NOT EXISTS match_time VARCHAR(50),
ADD COLUMN IF NOT EXISTS message_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMP,
ADD COLUMN IF NOT EXISTS subscribed BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

-- Popularity fields (from Migration 010)
ADD COLUMN IF NOT EXISTS attendance INTEGER,
ADD COLUMN IF NOT EXISTS sellout BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS feature_match BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS live_video_available BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS live_data_available BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS broadcasts_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS broadcasts_data JSONB,
ADD COLUMN IF NOT EXISTS social_links JSONB,
ADD COLUMN IF NOT EXISTS ticket_url TEXT,
ADD COLUMN IF NOT EXISTS popularity_score DECIMAL(5,2) DEFAULT 0.0;

-- Recreate all necessary indexes for tracked_events
CREATE UNIQUE INDEX IF NOT EXISTS idx_tracked_events_event_id ON tracked_events(event_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_srn_id ON tracked_events(srn_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_sport_id ON tracked_events(sport_id);
CREATE INDEX IF NOT EXISTS idx_tracked_events_status ON tracked_events(status);
CREATE INDEX IF NOT EXISTS idx_tracked_events_schedule_time ON tracked_events(schedule_time);
CREATE INDEX IF NOT EXISTS idx_tracked_events_subscribed ON tracked_events(subscribed);
CREATE INDEX IF NOT EXISTS idx_tracked_events_popularity_score ON tracked_events(popularity_score DESC);
CREATE INDEX IF NOT EXISTS idx_tracked_events_feature_match ON tracked_events(feature_match) WHERE feature_match = TRUE;
CREATE INDEX IF NOT EXISTS idx_tracked_events_sellout ON tracked_events(sellout) WHERE sellout = TRUE;
CREATE INDEX IF NOT EXISTS idx_tracked_events_broadcasts_count ON tracked_events(broadcasts_count DESC);

-- Add comments for clarity
COMMENT ON COLUMN tracked_events.srn_id IS 'Sportradar Number ID';
COMMENT ON COLUMN tracked_events.schedule_time IS '比赛开始时间';
COMMENT ON COLUMN tracked_events.subscribed IS '是否已订阅';
COMMENT ON COLUMN tracked_events.attendance IS '到场人数';
COMMENT ON COLUMN tracked_events.sellout IS '是否售罄';
COMMENT ON COLUMN tracked_events.feature_match IS '是否标记为焦点赛';
COMMENT ON COLUMN tracked_events.live_video_available IS '是否提供直播视频';
COMMENT ON COLUMN tracked_events.live_data_available IS '是否提供实时数据';
COMMENT ON COLUMN tracked_events.broadcasts_count IS '转播平台数量';
COMMENT ON COLUMN tracked_events.popularity_score IS '热门度评分（0-100）';

