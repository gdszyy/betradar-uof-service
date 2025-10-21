#!/bin/bash

# Quick Replay Test Script
# 快速重放测试脚本 - 用于验证管道功能

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Betradar UOF Quick Replay Test       ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# 检查凭证
if [ -z "$UOF_USERNAME" ] || [ -z "$UOF_PASSWORD" ]; then
    echo -e "${RED}❌ 错误: 需要设置环境变量${NC}"
    echo ""
    echo "请设置以下环境变量:"
    echo "  export UOF_USERNAME=\"your_username\""
    echo "  export UOF_PASSWORD=\"your_password\""
    echo ""
    echo "可选(用于监控):"
    echo "  export DATABASE_URL=\"postgresql://...\""
    echo ""
    exit 1
fi

echo -e "${GREEN}✅ 凭证已设置${NC}"
echo ""

# 推荐的测试赛事
echo -e "${YELLOW}📋 推荐测试赛事:${NC}"
echo ""
echo "1. test:match:21797788 - 足球(VAR场景) - 丰富的赔率变化"
echo "2. test:match:21797805 - 足球(加时赛)"
echo "3. test:match:21797815 - 足球(点球大战)"
echo "4. test:match:21797802 - 网球(5盘制抢十)"
echo ""

# 默认使用足球VAR场景
EVENT_ID="${1:-test:match:21797788}"
SPEED="${2:-20}"
DURATION="${3:-45}"

echo -e "${BLUE}🎬 测试配置:${NC}"
echo "  赛事ID: $EVENT_ID"
echo "  速度: ${SPEED}x"
echo "  时长: ${DURATION}秒"
echo ""

read -p "按Enter开始测试,或Ctrl+C取消..."
echo ""

# 运行重放
echo -e "${YELLOW}▶️  启动重放测试...${NC}"
echo ""

./replay_event.sh "$EVENT_ID" "$SPEED" "$DURATION" 1

echo ""
echo -e "${GREEN}✅ 重放测试完成!${NC}"
echo ""

# 如果有数据库连接,显示统计
if [ -n "$DATABASE_URL" ]; then
    echo -e "${YELLOW}📊 数据库统计:${NC}"
    echo ""
    
    # 消息类型分布
    echo "消息类型分布(最近5分钟):"
    psql "$DATABASE_URL" -t -c "
        SELECT 
            COALESCE(message_type, '(empty)') as type,
            COUNT(*) as count
        FROM uof_messages
        WHERE created_at > NOW() - INTERVAL '5 minutes'
        GROUP BY message_type
        ORDER BY count DESC;
    " 2>/dev/null || echo "  (无法连接数据库)"
    
    echo ""
    
    # 专门表统计
    echo "专门表统计:"
    for table in odds_changes bet_stops bet_settlements tracked_events; do
        count=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null || echo "N/A")
        echo "  $table: $count"
    done
    
    echo ""
fi

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🎉 测试完成!${NC}"
echo ""
echo "下一步:"
echo "1. 检查服务日志查看消息处理详情"
echo "2. 查询数据库验证数据存储:"
echo "   SELECT * FROM odds_changes LIMIT 10;"
echo "3. 打开WebSocket UI查看实时消息"
echo "4. 使用API查看统计: curl \$SERVICE_URL/api/stats"
echo ""
echo "如需测试其他赛事:"
echo "  ./quick_replay_test.sh test:match:21797805 20 45"
echo ""

