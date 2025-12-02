# 市场卡片展示方案 - 优化指南

## 优化概述

本文档说明了对市场卡片展示方案的优化，包括：
1. **增量更新** - 仅在新市场添加时进行 tab/chip 映射
2. **异常处理** - 对无法映射的市场进行标记和记录
3. **审计日志** - 维护所有分配操作的历史记录

## 核心优化

### 1. 增量更新模式

#### 问题
原始方案每次都处理所有市场，即使大多数市场已经有 tab_id，这会浪费资源。

#### 解决方案
- **新增模式**：`AssignTabChipToNewMarkets()` - 仅处理 `tab_id IS NULL` 的市场
- **完整模式**：`AssignTabChipToAllMarkets()` - 处理所有市场（初始化时使用）

#### 使用场景
```bash
# 初始化：一次性处理所有市场
./bin/assign_tab_chip_incremental -db "$DATABASE_URL" -mode full

# 生产环境：仅处理新市场（推荐）
./bin/assign_tab_chip_incremental -db "$DATABASE_URL" -mode incremental
```

### 2. 异常处理和记录

#### 新增表：market_tab_chip_unmapped
用于记录无法映射的市场，便于定期维护。

**字段说明**：
| 字段 | 说明 | 示例 |
|------|------|------|
| market_id | 市场 ID | 12345 |
| event_id | 事件 ID | sr:match:123456 |
| market_type | 市场类型 | 1X2 |
| market_name | 市场名称 | Match Odds |
| groups | Groups 字符串 | "["live", "regular_play"]" |
| specifiers | Specifiers 字符串 | "quarternr=1,goalnr=2" |
| reason | 失败原因 | "unknown_group" |
| created_at | 创建时间 | 2025-12-02 10:00:00 |
| updated_at | 更新时间 | 2025-12-02 10:00:00 |

**查询示例**：
```sql
-- 查看所有未映射的市场
SELECT * FROM market_tab_chip_unmapped ORDER BY created_at DESC LIMIT 100;

-- 按原因统计
SELECT reason, COUNT(*) as count FROM market_tab_chip_unmapped GROUP BY reason;

-- 查看最近未映射的市场
SELECT * FROM market_tab_chip_unmapped WHERE created_at > NOW() - INTERVAL '1 day';
```

### 3. 审计日志

#### 新增表：market_tab_chip_assignment_log
记录所有 tab/chip 分配操作，便于追踪变更历史。

**字段说明**：
| 字段 | 说明 | 示例 |
|------|------|------|
| market_id | 市场 ID | 12345 |
| event_id | 事件 ID | sr:match:123456 |
| old_tab_id | 旧 Tab ID | regular_play |
| new_tab_id | 新 Tab ID | quarters |
| old_chip_id | 旧 Chip ID | regular_play_1 |
| new_chip_id | 新 Chip ID | quarters_quarternr_1 |
| assignment_type | 分配类型 | initial, update, correction |
| assigned_by | 分配者 | import_tool, api, manual |
| created_at | 创建时间 | 2025-12-02 10:00:00 |

**查询示例**：
```sql
-- 查看特定市场的分配历史
SELECT * FROM market_tab_chip_assignment_log WHERE market_id = 12345 ORDER BY created_at;

-- 查看今天的所有分配
SELECT * FROM market_tab_chip_assignment_log WHERE created_at > NOW() - INTERVAL '1 day';

-- 查看更新的市场
SELECT * FROM market_tab_chip_assignment_log WHERE assignment_type = 'update';
```

## 部署流程

### 初始化部署

```bash
# 1. 运行数据库迁移
psql $DATABASE_URL -f database/migrations/011_add_tab_chip_fields.sql

# 2. 导入配置（仅需一次）
./bin/import_tab_chip -db "$DATABASE_URL" -tabs final_tab_chip_config.csv -chips final_chip_enumeration.csv

# 3. 初始化分配（处理所有市场）
./bin/assign_tab_chip_incremental -db "$DATABASE_URL" -mode full
```

### 生产环境（定期运行）

```bash
# 仅处理新市场（推荐每小时或每天运行一次）
./bin/assign_tab_chip_incremental -db "$DATABASE_URL" -mode incremental
```

### 使用脚本

```bash
# 完整初始化脚本
bash scripts/import_all.sh

# 增量更新脚本（推荐用于生产环境）
bash scripts/import_incremental.sh
```

## 监控和维护

### 查看映射统计

```sql
-- 总体统计
SELECT 
    COUNT(*) as total_markets,
    COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) as mapped,
    COUNT(CASE WHEN tab_id IS NULL THEN 1 END) as unmapped,
    ROUND(100.0 * COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) / COUNT(*), 2) as mapped_percentage
FROM markets;

-- 按 Tab 统计
SELECT tab_id, COUNT(*) as count FROM markets WHERE tab_id IS NOT NULL GROUP BY tab_id ORDER BY count DESC;

-- 按 Event 统计
SELECT event_id, COUNT(*) as total, COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) as mapped 
FROM markets GROUP BY event_id ORDER BY total DESC LIMIT 20;
```

### 处理未映射市场

```sql
-- 查看未映射市场的原因分布
SELECT reason, COUNT(*) as count, COUNT(DISTINCT event_id) as event_count
FROM market_tab_chip_unmapped
GROUP BY reason
ORDER BY count DESC;

-- 查看特定原因的市场
SELECT m.*, u.reason
FROM markets m
JOIN market_tab_chip_unmapped u ON m.id = u.market_id
WHERE u.reason = 'unknown_group'
LIMIT 20;

-- 手动修正未映射市场
UPDATE markets 
SET tab_id = 'regular_play', chip_id = NULL
WHERE id IN (SELECT market_id FROM market_tab_chip_unmapped WHERE reason = 'unknown_group');
```

### 定期维护任务

**每日任务**：
```bash
# 运行增量更新
bash scripts/import_incremental.sh

# 检查未映射市场数量
psql $DATABASE_URL -c "SELECT COUNT(*) FROM market_tab_chip_unmapped;"
```

**每周任务**：
```bash
# 查看未映射市场的原因分布
psql $DATABASE_URL -c "SELECT reason, COUNT(*) FROM market_tab_chip_unmapped GROUP BY reason;"

# 检查分配日志
psql $DATABASE_URL -c "SELECT assignment_type, COUNT(*) FROM market_tab_chip_assignment_log WHERE created_at > NOW() - INTERVAL '7 days' GROUP BY assignment_type;"
```

**每月任务**：
```bash
# 清理旧的未映射记录（可选）
DELETE FROM market_tab_chip_unmapped WHERE updated_at < NOW() - INTERVAL '30 days';

# 生成月度报告
psql $DATABASE_URL << EOF
SELECT 
    DATE_TRUNC('day', created_at) as date,
    COUNT(*) as assignments,
    COUNT(CASE WHEN assignment_type = 'initial' THEN 1 END) as initial,
    COUNT(CASE WHEN assignment_type = 'update' THEN 1 END) as updates
FROM market_tab_chip_assignment_log
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY date DESC;
EOF
```

## 性能优化

### 索引优化

已创建的索引：
```sql
-- markets 表
CREATE INDEX idx_markets_tab_id ON markets(tab_id);
CREATE INDEX idx_markets_chip_id ON markets(chip_id);
CREATE INDEX idx_markets_event_tab_chip ON markets(event_id, tab_id, chip_id);

-- unmapped 表
CREATE INDEX idx_unmapped_market_id ON market_tab_chip_unmapped(market_id);
CREATE INDEX idx_unmapped_event_id ON market_tab_chip_unmapped(event_id);
CREATE INDEX idx_unmapped_reason ON market_tab_chip_unmapped(reason);
CREATE INDEX idx_unmapped_created_at ON market_tab_chip_unmapped(created_at);

-- assignment_log 表
CREATE INDEX idx_assignment_log_market_id ON market_tab_chip_assignment_log(market_id);
CREATE INDEX idx_assignment_log_event_id ON market_tab_chip_assignment_log(event_id);
CREATE INDEX idx_assignment_log_created_at ON market_tab_chip_assignment_log(created_at);
```

### 查询优化建议

```sql
-- 使用视图查询映射状态
SELECT * FROM market_tab_chip_view WHERE mapping_status = 'unmapped' LIMIT 100;

-- 使用视图查看未映射摘要
SELECT * FROM market_unmapped_summary;

-- 分页查询大量数据
SELECT * FROM markets WHERE tab_id IS NOT NULL ORDER BY id LIMIT 1000 OFFSET 0;
```

## API 集成

### 查询未映射市场

```go
// 获取未映射市场列表
unmapped, err := service.GetUnmappedMarkets(100)
if err != nil {
    log.Fatal(err)
}

for _, market := range unmapped {
    fmt.Printf("Market %d: %s (reason: %s)\n", 
        market["market_id"], 
        market["market_name"], 
        market["reason"])
}
```

### 查询未映射摘要

```go
// 获取未映射市场的原因分布
summary, err := service.GetUnmappedSummary()
if err != nil {
    log.Fatal(err)
}

for _, item := range summary {
    fmt.Printf("%s: %d markets (%d events)\n",
        item["reason"],
        item["count"],
        item["event_count"])
}
```

## 故障排查

### 问题：未映射市场过多

**原因**：
- 新的市场类型或 specifier 未在映射规则中定义
- Groups 或 specifiers 格式不正确

**解决方案**：
1. 查看未映射市场的原因分布
2. 分析失败的市场数据
3. 更新映射规则
4. 重新运行完整分配

```bash
# 查看失败原因
psql $DATABASE_URL -c "SELECT reason, COUNT(*) FROM market_tab_chip_unmapped GROUP BY reason;"

# 查看具体失败的市场
psql $DATABASE_URL -c "SELECT * FROM market_tab_chip_unmapped WHERE reason = 'unknown_group' LIMIT 10;"

# 重新运行完整分配
./bin/assign_tab_chip_incremental -db "$DATABASE_URL" -mode full
```

### 问题：分配性能缓慢

**原因**：
- 市场数量过多
- 数据库连接不足
- 索引缺失

**解决方案**：
1. 确认索引已创建
2. 增加数据库连接池大小
3. 考虑分批处理

```bash
# 检查索引
psql $DATABASE_URL -c "SELECT * FROM pg_indexes WHERE tablename IN ('markets', 'market_tab_chip_unmapped');"

# 分析查询性能
psql $DATABASE_URL -c "EXPLAIN ANALYZE SELECT COUNT(*) FROM markets WHERE tab_id IS NULL;"
```

## 总结

优化后的方案提供了：
- ✅ **高效的增量更新** - 仅处理新市场
- ✅ **完整的异常处理** - 记录和追踪未映射市场
- ✅ **详细的审计日志** - 维护所有操作历史
- ✅ **灵活的维护工具** - 便于定期维护和修正

这些优化使系统更加高效、可靠和易于维护。
