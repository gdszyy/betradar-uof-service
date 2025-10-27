-- 添加 subscribed 字段到 tracked_events 表
-- 用于标记比赛是否已订阅

ALTER TABLE tracked_events 
ADD COLUMN IF NOT EXISTS subscribed BOOLEAN DEFAULT false;

-- 创建索引以提高查询性能
CREATE INDEX IF NOT EXISTS idx_tracked_events_subscribed 
ON tracked_events(subscribed);

-- 更新说明
-- subscribed = true: 比赛已订阅,会接收实时消息
-- subscribed = false: 比赛未订阅,只有基本信息

