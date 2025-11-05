-- Migration 010: Add popularity-related fields to tracked_events table
-- These fields are used to calculate match popularity/hotness

ALTER TABLE tracked_events
ADD COLUMN IF NOT EXISTS attendance INTEGER,                    -- 到场人数
ADD COLUMN IF NOT EXISTS sellout BOOLEAN DEFAULT FALSE,         -- 是否售罄
ADD COLUMN IF NOT EXISTS feature_match BOOLEAN DEFAULT FALSE,   -- 是否焦点赛
ADD COLUMN IF NOT EXISTS live_video_available BOOLEAN DEFAULT FALSE,  -- 是否提供直播
ADD COLUMN IF NOT EXISTS live_data_available BOOLEAN DEFAULT FALSE,   -- 是否提供实时数据
ADD COLUMN IF NOT EXISTS broadcasts_count INTEGER DEFAULT 0,    -- 转播平台数量
ADD COLUMN IF NOT EXISTS broadcasts_data JSONB,                 -- 转播详细信息（JSON）
ADD COLUMN IF NOT EXISTS social_links JSONB,                    -- 社交媒体链接（JSON）
ADD COLUMN IF NOT EXISTS ticket_url TEXT,                       -- 售票链接
ADD COLUMN IF NOT EXISTS popularity_score DECIMAL(5,2) DEFAULT 0.0;  -- 热门度评分（0-100）

-- 创建索引以优化热门比赛查询
CREATE INDEX IF NOT EXISTS idx_tracked_events_popularity_score ON tracked_events(popularity_score DESC);
CREATE INDEX IF NOT EXISTS idx_tracked_events_feature_match ON tracked_events(feature_match) WHERE feature_match = TRUE;
CREATE INDEX IF NOT EXISTS idx_tracked_events_sellout ON tracked_events(sellout) WHERE sellout = TRUE;
CREATE INDEX IF NOT EXISTS idx_tracked_events_broadcasts_count ON tracked_events(broadcasts_count DESC);

-- 添加注释
COMMENT ON COLUMN tracked_events.attendance IS '到场人数';
COMMENT ON COLUMN tracked_events.sellout IS '是否售罄';
COMMENT ON COLUMN tracked_events.feature_match IS '是否标记为焦点赛';
COMMENT ON COLUMN tracked_events.live_video_available IS '是否提供直播视频';
COMMENT ON COLUMN tracked_events.live_data_available IS '是否提供实时数据';
COMMENT ON COLUMN tracked_events.broadcasts_count IS '转播平台数量';
COMMENT ON COLUMN tracked_events.broadcasts_data IS '转播详细信息（平台、类型、开始时间等）';
COMMENT ON COLUMN tracked_events.social_links IS '社交媒体链接（官网、Twitter、Facebook等）';
COMMENT ON COLUMN tracked_events.ticket_url IS '售票链接';
COMMENT ON COLUMN tracked_events.popularity_score IS '热门度评分（0-100），基于多个因素计算';

