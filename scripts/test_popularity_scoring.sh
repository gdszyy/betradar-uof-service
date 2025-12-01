#!/bin/bash
# 测试热度评分脚本
# 用法: ./test_popularity_scoring.sh [DATABASE_URL]

set -e

# 数据库连接字符串
DB_URL="${1:-$DATABASE_URL}"

if [ -z "$DB_URL" ]; then
    echo "❌ Error: DATABASE_URL not provided"
    echo "Usage: $0 <database_url>"
    echo "   or: DATABASE_URL=<url> $0"
    exit 1
fi

echo "🔍 Testing Popularity Scoring Scripts"
echo "======================================"
echo ""

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 1. 测试比赛热度评分脚本
echo "📊 Testing Event Popularity Scoring..."
if psql "$DB_URL" -f "$SCRIPT_DIR/calculate_event_popularity.sql" > /dev/null 2>&1; then
    echo "✅ Event popularity scoring script executed successfully"
    
    # 查询结果统计
    UPDATED_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM tracked_events WHERE popularity_score > 0;" | tr -d ' ')
    echo "   📈 Events with popularity score: $UPDATED_COUNT"
    
    # 查询得分分布
    echo "   📊 Score distribution:"
    psql "$DB_URL" -c "
        SELECT 
            popularity_score,
            COUNT(*) as count
        FROM tracked_events
        WHERE popularity_score > 0
        GROUP BY popularity_score
        ORDER BY popularity_score DESC
        LIMIT 10;
    " | head -15
else
    echo "❌ Event popularity scoring script failed"
    exit 1
fi

echo ""

# 2. 测试联赛热度评分脚本
echo "🏆 Testing Tournament Popularity Scoring..."
if psql "$DB_URL" -f "$SCRIPT_DIR/calculate_tournament_popularity.sql" > /dev/null 2>&1; then
    echo "✅ Tournament popularity scoring script executed successfully"
    
    # 查询结果统计
    TOURNAMENT_COUNT=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM tournament_popularity_scores;" | tr -d ' ')
    echo "   📈 Tournaments scored: $TOURNAMENT_COUNT"
    
    # 查询 Top 5 联赛
    echo "   🏆 Top 5 Tournaments:"
    psql "$DB_URL" -c "
        SELECT 
            tournament_name,
            match_count,
            final_popularity_score
        FROM tournament_popularity_scores
        ORDER BY final_popularity_score DESC
        LIMIT 5;
    " | head -10
else
    echo "❌ Tournament popularity scoring script failed"
    exit 1
fi

echo ""
echo "======================================"
echo "✅ All tests passed!"
echo ""
echo "💡 Next steps:"
echo "   1. Deploy the updated service to Railway"
echo "   2. The service will automatically run scoring daily at 2:00 AM"
echo "   3. You can also trigger manual scoring via API (if implemented)"
