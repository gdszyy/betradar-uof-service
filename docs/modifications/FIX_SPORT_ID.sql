-- 修复 tracked_events 表中的 sport_id
-- 通过解析 event_id 中的 sport 信息来更新

-- 方法 1: 如果有 fixture 消息记录,从中提取 sport_id
-- (这需要 fixture 消息中包含 sport 信息)

-- 方法 2: 根据常见的 event_id 模式设置默认值
-- 大多数足球比赛的 event_id 格式为 sr:match:xxxxx
-- 可以暂时设置为足球 (sr:sport:1)

UPDATE tracked_events 
SET sport_id = 'sr:sport:1'
WHERE sport_id IS NULL 
  AND event_id LIKE 'sr:match:%';

-- 验证更新结果
SELECT 
    sport_id,
    COUNT(*) as count
FROM tracked_events
GROUP BY sport_id
ORDER BY count DESC;

-- 查看更新后的数据
SELECT 
    event_id,
    sport_id,
    home_team_name,
    away_team_name,
    status
FROM tracked_events
LIMIT 10;

