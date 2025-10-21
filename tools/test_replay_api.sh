#!/bin/bash

# Test Replay API Script
# 测试Replay API端点

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SERVICE_URL="${1:-https://betradar-uof-service-copy-production.up.railway.app}"

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Testing Replay API Endpoints         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
echo "Service URL: $SERVICE_URL"
echo ""

# 1. Test health endpoint first
echo -e "${YELLOW}1. Testing service health...${NC}"
HEALTH=$(curl -s "$SERVICE_URL/api/health")
echo "$HEALTH" | jq '.'
echo ""

# 2. Check if replay endpoint exists
echo -e "${YELLOW}2. Checking replay endpoint availability...${NC}"
REPLAY_STATUS_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL/api/replay/status")
if [ "$REPLAY_STATUS_CODE" = "404" ]; then
    echo -e "${RED}❌ Replay endpoints not found (404)${NC}"
    echo "   This might mean:"
    echo "   - Railway is still deploying the new code"
    echo "   - The deployment failed"
    echo "   - The code wasn't pushed correctly"
    echo ""
    echo "Please check Railway deployment logs and try again in a few minutes."
    exit 1
elif [ "$REPLAY_STATUS_CODE" = "503" ]; then
    echo -e "${YELLOW}⚠️  Replay client not configured (503)${NC}"
    echo "   Please ensure UOF_USERNAME and UOF_PASSWORD are set in Railway environment variables"
    exit 1
else
    echo -e "${GREEN}✅ Replay endpoint is available (HTTP $REPLAY_STATUS_CODE)${NC}"
fi
echo ""

# 3. Start replay test
echo -e "${YELLOW}3. Starting replay test...${NC}"
echo "   Event: test:match:21797788"
echo "   Speed: 50x"
echo "   Duration: 45 seconds"
echo ""

REPLAY_RESPONSE=$(curl -s -X POST "$SERVICE_URL/api/replay/start" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 50,
    "duration": 45,
    "node_id": 1,
    "use_replay_timestamp": true
  }')

echo "$REPLAY_RESPONSE" | jq '.'

if echo "$REPLAY_RESPONSE" | grep -q "accepted"; then
    echo -e "${GREEN}✅ Replay started successfully!${NC}"
else
    echo -e "${RED}❌ Failed to start replay${NC}"
    exit 1
fi
echo ""

# 4. Wait and monitor
echo -e "${YELLOW}4. Monitoring replay progress...${NC}"
echo "   (Will check every 10 seconds for 50 seconds)"
echo ""

for i in {1..5}; do
    echo "--- Check $i/5 ($(date +%H:%M:%S)) ---"
    
    # Get stats
    STATS=$(curl -s "$SERVICE_URL/api/stats")
    echo "Stats:"
    echo "$STATS" | jq '{total_messages, odds_changes, bet_stops, bet_settlements}'
    
    # Get recent messages count
    RECENT_COUNT=$(curl -s "$SERVICE_URL/api/messages?limit=100" | jq '.messages | length')
    echo "Recent messages: $RECENT_COUNT"
    
    echo ""
    
    if [ $i -lt 5 ]; then
        sleep 10
    fi
done

# 5. Final results
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Test Complete!                        ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""

echo "Final Statistics:"
curl -s "$SERVICE_URL/api/stats" | jq '.'
echo ""

echo "Recent Messages:"
curl -s "$SERVICE_URL/api/messages?limit=5" | jq '.messages[] | {type: .message_type, event: .event_id, time: .created_at}'
echo ""

echo -e "${BLUE}Next steps:${NC}"
echo "1. Check Railway logs for detailed message processing"
echo "2. Query database to verify odds_changes, bet_stops, bet_settlements"
echo "3. Open WebSocket UI: $SERVICE_URL/"
echo "4. Check recovery status: curl $SERVICE_URL/api/recovery/status"
echo ""

