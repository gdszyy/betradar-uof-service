# 热度评分脚本使用指南

## 概述

本目录包含用于计算比赛和联赛热度评分的 SQL 脚本。这些脚本基于市场数量、赛事特征等多个维度计算热度分数。

## 脚本列表

### 1. `calculate_event_popularity.sql`
**功能**: 计算并更新每场比赛的热度评分

**更新字段**: `tracked_events.popularity_score`

**评分规则**:
- 基于市场数量的单维度评分
- 评分范围: 1-10 分（整数）
- 市场数量越多，热度评分越高

**评分阈值**:
| 市场数量 | 热度评分 |
|---------|---------|
| > 400   | 10      |
| > 300   | 9       |
| > 200   | 8       |
| > 150   | 7       |
| > 100   | 6       |
| > 50    | 5       |
| > 30    | 4       |
| > 10    | 3       |
| > 5     | 2       |
| ≤ 5     | 1       |

### 2. `calculate_tournament_popularity.sql`
**功能**: 计算并存储每个联赛的热度评分

**更新表**: `tournament_popularity_scores`

**评分模型**: 二维加权
- 联赛层级评分 (50%)
- 市场深度评分 (50%)

**评分范围**: 1-10 分（小数，保留两位）

**层级评分规则**:
- World Cup / Olympics: 10 分
- Champions League: 9 分
- Premier League / NBA / UEFA: 8 分
- International / Europa League / Serie A / La Liga / Bundesliga: 7 分
- 其他根据平均市场数量: 4-6 分

**市场深度评分规则**:
| 平均市场数 | 评分 |
|-----------|------|
| > 300     | 10   |
| > 200     | 9    |
| > 150     | 8    |
| > 100     | 7    |
| > 50      | 6    |
| > 30      | 5    |
| > 20      | 4    |
| > 10      | 3    |
| > 5       | 2    |
| ≤ 5       | 1    |

## 使用方法

### 手动执行

#### 方式一：使用 psql 命令行
```bash
# 计算比赛热度评分
psql $DATABASE_URL -f scripts/calculate_event_popularity.sql

# 计算联赛热度评分
psql $DATABASE_URL -f scripts/calculate_tournament_popularity.sql
```

#### 方式二：使用 PostgreSQL 客户端
```sql
-- 在 PostgreSQL 客户端中执行
\i scripts/calculate_event_popularity.sql
\i scripts/calculate_tournament_popularity.sql
```

### 定期执行（推荐）

建议每天执行一次以保持数据最新。可以使用以下方式：

#### 方式一：使用 cron（Linux/Mac）
```bash
# 编辑 crontab
crontab -e

# 添加以下行（每天凌晨 2 点执行）
0 2 * * * psql $DATABASE_URL -f /path/to/scripts/calculate_event_popularity.sql
5 2 * * * psql $DATABASE_URL -f /path/to/scripts/calculate_tournament_popularity.sql
```

#### 方式二：集成到 Go 服务中
在 `main.go` 中添加定时任务：

```go
// 启动热度评分定时任务
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        // 执行比赛热度评分
        if err := executePopularityScoring(); err != nil {
            logger.Errorf("[Popularity] Failed to calculate scores: %v", err)
        }
    }
}()
```

## 验证结果

### 查看比赛热度评分
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

-- 查看热度评分分布
SELECT 
    popularity_score,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM tracked_events
WHERE popularity_score > 0
GROUP BY popularity_score
ORDER BY popularity_score DESC;
```

### 查看联赛热度评分
```sql
-- 查看热度最高的 10 个联赛
SELECT 
    tournament_name, 
    match_count,
    final_popularity_score 
FROM tournament_popularity_scores 
ORDER BY final_popularity_score DESC 
LIMIT 10;

-- 查看各热度等级的联赛数量
SELECT 
    CASE 
        WHEN final_popularity_score >= 9.0 THEN 'S级 (9.0-10.0)'
        WHEN final_popularity_score >= 8.0 THEN 'A级 (8.0-8.9)'
        WHEN final_popularity_score >= 7.0 THEN 'B级 (7.0-7.9)'
        WHEN final_popularity_score >= 6.0 THEN 'C级 (6.0-6.9)'
        ELSE 'D级 (<6.0)'
    END as grade,
    COUNT(*) as count
FROM tournament_popularity_scores
GROUP BY grade
ORDER BY grade;
```

## 依赖关系

这些脚本依赖以下数据库表：
- `tracked_events` - 比赛数据
- `markets` - 市场数据
- `tournament_popularity_scores` - 联赛热度评分表（由脚本自动创建/更新）

## 注意事项

1. **执行顺序**: 建议先执行 `calculate_event_popularity.sql`，再执行 `calculate_tournament_popularity.sql`
2. **数据依赖**: 确保 `tracked_events` 和 `markets` 表有足够的数据
3. **性能考虑**: 在数据量大时，脚本执行可能需要几分钟，建议在低峰期执行
4. **数据库连接**: 确保有足够的数据库连接权限和资源

## 故障排查

### 问题：脚本执行失败
**解决方案**:
1. 检查数据库连接是否正常
2. 确认表结构是否完整（运行数据库迁移）
3. 查看错误日志获取详细信息

### 问题：评分结果全为 0
**解决方案**:
1. 检查 `markets` 表是否有数据
2. 确认 `tracked_events` 表中的 `event_id` 与 `markets` 表中的 `event_id` 能够关联
3. 手动执行查询验证数据

### 问题：联赛热度评分表为空
**解决方案**:
1. 确认 `tracked_events` 表中有 `tournament_id` 不为空的记录
2. 检查 `markets` 表是否有对应的市场数据
3. 手动执行脚本中的 CTE 查询逐步排查

## 更新历史

- **v1.2** (2025-12-01): 适配 betradar-uof-service 项目，更新字段映射
- **v1.1** (2025-11-27): 优化评分算法，增加更多联赛层级判断
- **v1.0** (2025-11-27): 初始版本，基础热度评分功能
