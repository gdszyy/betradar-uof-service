#!/bin/bash

# API测试脚本
# 用法: ./test_api.sh <base_url>
# 例如: ./test_api.sh https://your-service.railway.app

BASE_URL="${1:-http://localhost:8080}"

echo "=== Testing Betradar UOF Service API ==="
echo "Base URL: $BASE_URL"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local name="$1"
    local endpoint="$2"
    local method="${3:-GET}"
    
    echo -n "Testing $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "202" ]; then
        echo -e "${GREEN}✓ OK${NC} (HTTP $http_code)"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        echo -e "${RED}✗ FAILED${NC} (HTTP $http_code)"
        echo "$body"
    fi
    echo ""
}

# 1. Health Check
test_endpoint "Health Check" "/api/health"

# 2. Get Messages
test_endpoint "Get Messages (last 10)" "/api/messages?limit=10"

# 3. Get Tracked Events
test_endpoint "Get Tracked Events" "/api/events"

# 4. Get Stats
test_endpoint "Get Stats" "/api/stats"

# 5. Get Recovery Status
test_endpoint "Get Recovery Status" "/api/recovery/status?limit=10"

# 6. WebSocket Test (just check if endpoint exists)
echo -n "Testing WebSocket endpoint... "
ws_response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/ws")
if [ "$ws_response" = "400" ] || [ "$ws_response" = "426" ]; then
    echo -e "${GREEN}✓ OK${NC} (WebSocket endpoint exists, HTTP $ws_response)"
else
    echo -e "${YELLOW}? UNKNOWN${NC} (HTTP $ws_response)"
fi
echo ""

# 7. Static Files
echo -n "Testing Static Files... "
static_response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/")
if [ "$static_response" = "200" ] || [ "$static_response" = "304" ]; then
    echo -e "${GREEN}✓ OK${NC} (HTTP $static_response)"
else
    echo -e "${YELLOW}? UNKNOWN${NC} (HTTP $static_response)"
fi
echo ""

echo "=== API Test Complete ==="
echo ""
echo "Summary:"
echo "- All core endpoints tested"
echo "- Check output above for any failures"
echo "- WebSocket requires browser or ws client for full test"
echo ""
echo "Next steps:"
echo "1. Open $BASE_URL in browser to test WebSocket UI"
echo "2. Monitor /api/recovery/status to track recovery completion"
echo "3. Check /api/messages to see new messages with correct types"

