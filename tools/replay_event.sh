#!/bin/bash

# Replay Event Script
# 用法: ./replay_event.sh <event_id> [speed] [duration]
# 例如: ./replay_event.sh sr:match:12345 10 60

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 参数
EVENT_ID="${1}"
SPEED="${2:-10}"
DURATION="${3:-60}"
NODE_ID="${4:-1}"

if [ -z "$EVENT_ID" ]; then
    echo -e "${RED}❌ Error: Event ID is required${NC}"
    echo "Usage: $0 <event_id> [speed] [duration] [node_id]"
    echo "Example: $0 sr:match:12345 10 60 1"
    exit 1
fi

# 检查环境变量
if [ -z "$UOF_USERNAME" ] || [ -z "$UOF_PASSWORD" ]; then
    echo -e "${RED}❌ Error: UOF_USERNAME and UOF_PASSWORD environment variables are required${NC}"
    exit 1
fi

API_BASE="https://api.betradar.com/v1"
AUTH="$UOF_USERNAME:$UOF_PASSWORD"

echo -e "${GREEN}🎬 Betradar UOF Replay Test${NC}"
echo "================================"
echo "Event ID: $EVENT_ID"
echo "Speed: ${SPEED}x"
echo "Duration: ${DURATION}s"
echo "Node ID: $NODE_ID"
echo ""

# 1. 重置重放列表
echo -e "${YELLOW}🔄 Resetting replay list...${NC}"
curl -s -u "$AUTH" -X POST "$API_BASE/replay/reset" > /dev/null || echo "   (may be already empty)"
echo -e "${GREEN}✅ Reset complete${NC}"
echo ""

# 2. 添加赛事
echo -e "${YELLOW}➕ Adding event to replay list...${NC}"
RESPONSE=$(curl -s -u "$AUTH" -X PUT "$API_BASE/replay/events/$EVENT_ID")
echo "$RESPONSE"
echo -e "${GREEN}✅ Event added${NC}"
echo ""

# 3. 开始重放
echo -e "${YELLOW}▶️  Starting replay...${NC}"
PLAY_URL="$API_BASE/replay/play?speed=$SPEED&max_delay=10000&node_id=$NODE_ID&use_replay_timestamp=true"
RESPONSE=$(curl -s -u "$AUTH" -X POST "$PLAY_URL")
echo "$RESPONSE"
echo -e "${GREEN}✅ Replay started${NC}"
echo ""

# 4. 等待设置完成
echo -e "${YELLOW}⏳ Waiting for replay to be ready...${NC}"
for i in {1..15}; do
    sleep 2
    STATUS=$(curl -s -u "$AUTH" "$API_BASE/replay/status")
    if ! echo "$STATUS" | grep -q "SETTING_UP"; then
        echo -e "${GREEN}✅ Replay is ready!${NC}"
        break
    fi
    echo "   Still setting up... ($i/15)"
done
echo ""

# 5. 显示状态
echo -e "${YELLOW}📊 Current status:${NC}"
curl -s -u "$AUTH" "$API_BASE/replay/status" | sed 's/^/   /'
echo ""
echo ""

# 6. 监控
echo -e "${GREEN}⏱️  Replay is running for ${DURATION} seconds...${NC}"
echo "   Check your service logs to see incoming messages:"
echo "   - Railway: https://railway.app → Your Project → Deployments → Logs"
echo "   - Local: Check console output"
echo ""

# 倒计时
for ((i=$DURATION; i>0; i-=10)); do
    echo "   $i seconds remaining..."
    sleep 10
done

echo ""
echo -e "${GREEN}⏰ Duration completed${NC}"
echo ""

# 7. 停止重放
echo -e "${YELLOW}🛑 Stopping replay...${NC}"
curl -s -u "$AUTH" -X POST "$API_BASE/replay/stop" > /dev/null
echo -e "${GREEN}✅ Replay stopped${NC}"
echo ""

# 8. 最终状态
echo -e "${YELLOW}📊 Final status:${NC}"
curl -s -u "$AUTH" "$API_BASE/replay/status" | sed 's/^/   /'
echo ""
echo ""

echo -e "${GREEN}✅ Test completed!${NC}"
echo ""
echo "Next steps:"
echo "1. Check your service logs for message processing details"
echo "2. Query the database to verify data was stored:"
echo "   SELECT message_type, COUNT(*) FROM uof_messages GROUP BY message_type;"
echo "3. Check specialized tables:"
echo "   SELECT COUNT(*) FROM odds_changes;"
echo "   SELECT COUNT(*) FROM bet_stops;"
echo "   SELECT COUNT(*) FROM bet_settlements;"
echo "4. Open WebSocket UI to see real-time messages"

