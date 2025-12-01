# 热度评分 API 文档

## 概述

热度评分系统会自动在每天凌晨 2 点执行，计算比赛和联赛的热度评分。此外，也可以通过 API 手动触发评分计算。

## 自动执行

### 执行时间
- **频率**: 每天一次
- **时间**: 凌晨 2:00 AM（服务器时区）
- **首次执行**: 服务启动后 30 秒

### 执行内容
1. 计算所有比赛的热度评分（更新 `tracked_events.popularity_score`）
2. 计算所有联赛的热度评分（更新 `tournament_popularity_scores` 表）

### 日志监控

服务启动时会输出：
```
[PopularityScoring] ✅ Popularity scoring service started (daily at 2:00 AM)
[PopularityScoring] Next scoring scheduled at 2025-12-02 02:00:00 (in 23h 45m)
```

执行时会输出：
```
[PopularityScoring] 🔄 Daily popularity scoring triggered
[PopularityScoring] Event scoring completed: 1234 events updated
[PopularityScoring] Tournament scoring completed: 56 tournaments updated
[PopularityScoring] ✅ Daily scoring completed: 1234 events, 56 tournaments updated in 2.5s
```

## 手动触发（可选功能）

如果需要手动触发评分计算，可以添加以下 API 端点：

### POST /api/admin/popularity/calculate

手动触发热度评分计算。

**请求示例**:
```bash
curl -X POST https://betradar-uof-service-production.up.railway.app/api/admin/popularity/calculate \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

**响应示例**:
```json
{
  "success": true,
  "message": "Popularity scoring completed",
  "data": {
    "events_updated": 1234,
    "tournaments_updated": 56,
    "execution_time": "2.5s"
  }
}
```

**实现代码**（需要添加到 `web/admin_handlers.go`）:

```go
// handleCalculatePopularity 手动触发热度评分计算
func (s *Server) handleCalculatePopularity(w http.ResponseWriter, r *http.Request) {
    // 验证管理员权限
    // ... (添加认证逻辑)
    
    // 执行评分
    result := s.popularityService.ExecuteScoring()
    
    if result.Error != nil {
        http.Error(w, result.Error.Error(), http.StatusInternalServerError)
        return
    }
    
    response := map[string]interface{}{
        "success": true,
        "message": "Popularity scoring completed",
        "data": map[string]interface{}{
            "events_updated":      result.EventsUpdated,
            "tournaments_updated": result.TournamentsUpdated,
            "execution_time":      result.ExecutionTime.String(),
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## 查询热度评分结果

### 查询比赛热度评分

```sql
-- 查看热度最高的 10 场比赛
SELECT 
    event_id, 
    home_team, 
    away_team, 
    tournament_name,
    popularity_score 
FROM tracked_events 
WHERE popularity_score > 0
ORDER BY popularity_score DESC 
LIMIT 10;
```

### 查询联赛热度评分

```sql
-- 查看热度最高的 10 个联赛
SELECT 
    tournament_name, 
    match_count,
    final_popularity_score 
FROM tournament_popularity_scores 
ORDER BY final_popularity_score DESC 
LIMIT 10;
```

### 通过 API 查询

现有的 `/api/categories` 接口已经支持返回热度数据：

```bash
curl "https://betradar-uof-service-production.up.railway.app/api/categories"
```

响应中的 `top_tournaments` 字段包含热门联赛数据。

## 环境变量配置

### POPULARITY_SCRIPT_DIR
- **说明**: SQL 脚本所在目录路径
- **默认值**: 自动检测（`./scripts` 或 `/app/scripts`）
- **示例**: `POPULARITY_SCRIPT_DIR=/app/scripts`

### 执行时间配置
当前执行时间硬编码为凌晨 2 点。如需修改，可在 `services/popularity_scoring_service.go` 中调整：

```go
// 获取执行时间配置（默认凌晨 2 点）
executionHour := 2  // 修改为其他小时（0-23）
```

## 故障排查

### 问题：脚本未执行

**检查日志**:
```bash
# 查看服务日志
railway logs

# 搜索热度评分相关日志
railway logs | grep PopularityScoring
```

**可能原因**:
1. SQL 脚本文件未找到
2. 数据库连接失败
3. SQL 语法错误

### 问题：评分结果为 0

**检查数据**:
```sql
-- 检查是否有市场数据
SELECT COUNT(*) FROM markets;

-- 检查是否有比赛数据
SELECT COUNT(*) FROM tracked_events;
```

**可能原因**:
1. `markets` 表没有数据
2. `tracked_events` 表中的 `event_id` 与 `markets` 表无法关联

### 问题：服务启动失败

**检查错误**:
```bash
railway logs | grep "PopularityScoring.*Failed"
```

**可能原因**:
1. 数据库连接失败
2. 脚本文件路径错误
3. 权限不足

## 性能优化

### 执行时间
- 比赛评分: 通常 < 5 秒（取决于数据量）
- 联赛评分: 通常 < 3 秒（取决于联赛数量）

### 数据库负载
- 使用 CTE（公用表表达式）优化查询
- 使用 UPSERT（ON CONFLICT）避免重复插入
- 在事务中执行，保证数据一致性

### 建议
- 在低峰期执行（如凌晨 2-3 点）
- 监控执行时间，如超过 30 秒需优化
- 定期清理过期数据以提高性能

## 监控和告警

### 成功执行通知
服务会通过飞书（Lark）发送执行结果通知（如果配置了 `LARK_WEBHOOK`）。

### 失败告警
如果执行失败，会发送错误通知到飞书。

### 监控指标
- 执行频率: 每天一次
- 执行时长: < 10 秒
- 更新记录数: 比赛 > 1000，联赛 > 50

## 相关文档

- [热度评分算法说明](../scripts/README.md)
- [SQL 脚本详解](../scripts/)
- [数据库表结构](../database/migrations/)
