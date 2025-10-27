-- 003_fix_schedule_time_nullable.sql
-- 确保 schedule_time 字段允许 NULL 值
-- 修复冷启动存储失败的问题

-- 1. 确保 schedule_time 允许 NULL
ALTER TABLE tracked_events ALTER COLUMN schedule_time DROP NOT NULL;

-- 2. 验证修改
-- 执行后可以运行以下查询验证:
-- SELECT column_name, is_nullable, data_type 
-- FROM information_schema.columns 
-- WHERE table_name = 'tracked_events' 
-- AND column_name = 'schedule_time';
-- 预期结果: is_nullable = 'YES'

COMMENT ON COLUMN tracked_events.schedule_time IS 'Scheduled match start time from fixture (nullable - some matches may not have scheduled time)';

