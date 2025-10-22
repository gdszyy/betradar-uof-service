#!/bin/bash

# Live Data 连接测试脚本
# 用途: 自动订阅 bookable 比赛并测试 LD 连接

set -e

# 配置
SERVER_URL="${SERVER_URL:-http://localhost:8080}"
API_BASE="${SERVER_URL}/api"

echo "=========================================="
echo "Live Data 连接测试"
echo "=========================================="
echo "服务器: $SERVER_URL"
echo ""

# 1. 检查服务健康状态
echo "📊 1. 检查服务健康状态..."
HEALTH=$(curl -s "${API_BASE}/health")
echo "   $HEALTH"
echo ""

# 2. 获取服务器 IP
echo "🌐 2. 获取服务器公网 IP..."
IP_INFO=$(curl -s "${API_BASE}/ip")
echo "   $IP_INFO"
IP=$(echo $IP_INFO | grep -o '"ip":"[^"]*"' | cut -d'"' -f4)
echo "   📍 IP 地址: $IP"
echo ""

# 3. 自动订阅所有 bookable 比赛
echo "📝 3. 自动订阅所有 bookable 比赛..."
BOOKING_RESULT=$(curl -s -X POST "${API_BASE}/booking/auto")
echo "   $BOOKING_RESULT"
echo "   ⏳ 等待订阅完成 (10秒)..."
sleep 10
echo ""

# 4. 检查已订阅的比赛
echo "🔍 4. 检查已订阅的比赛..."
curl -s -X POST "${API_BASE}/monitor/trigger" > /dev/null
echo "   ✅ 监控报告已触发，请查看飞书通知"
echo "   ⏳ 等待报告生成 (5秒)..."
sleep 5
echo ""

# 5. 连接 Live Data
echo "🔌 5. 连接 Live Data 服务器..."
LD_CONNECT=$(curl -s -X POST "${API_BASE}/ld/connect")
echo "   $LD_CONNECT"
echo "   ⏳ 等待连接建立 (5秒)..."
sleep 5
echo ""

# 6. 检查 LD 连接状态
echo "📡 6. 检查 Live Data 连接状态..."
LD_STATUS=$(curl -s "${API_BASE}/ld/status")
echo "   $LD_STATUS"
echo ""

# 7. 检查已订阅的 LD 比赛
echo "📋 7. 检查 Live Data 已订阅比赛..."
LD_MATCHES=$(curl -s "${API_BASE}/ld/matches")
echo "   $LD_MATCHES"
echo ""

echo "=========================================="
echo "✅ 测试完成!"
echo "=========================================="
echo ""
echo "📱 请查看飞书通知获取详细报告:"
echo "   - 自动订阅报告"
echo "   - 比赛监控报告"
echo "   - Live Data 连接状态"
echo ""
echo "📊 查看数据库:"
echo "   - LD 事件: GET ${API_BASE}/ld/events"
echo "   - LD 比赛: GET ${API_BASE}/ld/matches"
echo ""

