#!/bin/bash

# 获取可重放的赛事列表
# 只有48小时以前的赛事才能重放

echo "🔍 Finding replayable events (>48 hours old)..."
echo ""

# 从数据库获取48小时以前的赛事
export DATABASE_URL="${DATABASE_URL:-postgresql://postgres:vhxGYSEpQHGpsaGyyfMuhNzUHotWuLed@interchange.proxy.rlwy.net:29295/railway}"

# 使用psql查询
psql "$DATABASE_URL" -c "
SELECT 
    event_id,
    sport,
    tournament,
    start_time,
    NOW() - start_time AS age
FROM tracked_events
WHERE start_time < NOW() - INTERVAL '48 hours'
ORDER BY start_time DESC
LIMIT 20;
" 2>/dev/null

echo ""
echo "💡 Tip: Use these event IDs for replay testing"
echo "   They are older than 48 hours and should be replayable"
echo ""
echo "Example:"
echo "  curl -X POST https://your-service.railway.app/api/replay/start \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"event_id\":\"sr:match:XXXXX\",\"speed\":20,\"duration\":60}'"

