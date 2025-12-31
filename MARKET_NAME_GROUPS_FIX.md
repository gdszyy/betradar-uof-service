# Market Name 和 Market Groups 字段修复说明

## 问题描述
在UOF项目中，`market_name` 和 `market groups` 字段返回 `null` 的问题。

## 问题根源

### 1. Market Name 为 null 的原因
- **Sport Rader 的 odds_change 消息中不包含 market name**
  - odds_change XML 消息中的 `<market>` 元素只包含 `id`、`status`、`specifiers` 等属性
  - 不包含 `name` 属性
- **需要通过 Market Descriptions API 获取**
  - Market name 需要从 `/descriptions/{language}/markets.xml` API 获取
  - 或者从本地缓存的 `market_descriptions` 表中查询
- **原代码在插入时未填充**
  - `odds_parser.go` 的 `storeMarket` 函数在插入 market 时没有调用 `MarketDescriptionsService` 来获取 name
  - 虽然有后台更新逻辑，但初始插入时字段为 null

### 2. Market Groups 为 null 的原因
- **markets 表缺少 groups 字段**
  - 原始的 `markets` 表定义中没有 `groups` 字段
  - 即使 `market_descriptions` 表有 groups 信息，也无法存储到 `markets` 表
- **groups 信息的来源**
  - groups 信息在 Market Descriptions API 的响应中，格式为竖线分隔的字符串
  - 例如：`groups="all|score|regular_play"`

## 修复方案

### 1. 数据库层面修复

#### 添加 groups 字段到 markets 表
```sql
-- 文件: database/migrations/017_add_groups_to_markets.sql
ALTER TABLE markets ADD COLUMN IF NOT EXISTS groups TEXT;
CREATE INDEX IF NOT EXISTS idx_markets_groups ON markets(groups);

-- 从 market_descriptions 同步现有数据
UPDATE markets m
SET groups = md.groups, updated_at = CURRENT_TIMESTAMP
FROM market_descriptions md
WHERE m.sr_market_id = md.market_id
AND (m.groups IS NULL OR m.groups = '')
AND md.groups IS NOT NULL AND md.groups != '';
```

#### 更新表结构定义
- 修改 `database/database.go` 中的 CREATE TABLE 语句，添加 `groups TEXT` 字段
- 修改 `database/models_tab_chip.go` 中的 `MarketTabChipView` 结构体，添加 `Groups` 字段

### 2. 代码层面修复

#### 在 MarketDescriptionsService 中添加 GetMarketGroups 方法
```go
// GetMarketGroups 获取市场分组
func (s *MarketDescriptionsService) GetMarketGroups(marketID string) string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if market, ok := s.markets[marketID]; ok {
        return market.Groups
    }

    return ""
}
```

#### 修改 odds_parser.go 的 storeMarket 函数
在插入 market 时立即填充 `market_name` 和 `groups` 字段：

```go
func (p *OddsParser) storeMarket(tx *sql.Tx, eventID string, market MarketData, timestamp int64, productID int) error {
    // 获取 market name 和 groups
    var marketName, groups string
    if p.marketDescService != nil {
        // 查询 home_team_name 和 away_team_name 用于替换变量
        var homeTeam, awayTeam sql.NullString
        teamQuery := `SELECT home_team_name, away_team_name FROM events WHERE event_id = $1`
        tx.QueryRow(teamQuery, eventID).Scan(&homeTeam, &awayTeam)
        
        ctx := &ReplacementContext{
            HomeTeamName: homeTeam.String,
            AwayTeamName: awayTeam.String,
            Specifiers:   market.Specifiers,
        }
        marketName = p.marketDescService.GetMarketName(market.ID, market.Specifiers, ctx)
        groups = p.marketDescService.GetMarketGroups(market.ID)
    }
    
    // 插入时包含 market_name 和 groups
    marketQuery := `
        INSERT INTO markets (event_id, sr_market_id, market_type, market_name, groups, specifiers, status, producer_id, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
        ON CONFLICT (event_id, sr_market_id, specifiers) DO UPDATE
        SET status = EXCLUDED.status, 
            producer_id = EXCLUDED.producer_id,
            market_name = COALESCE(NULLIF(markets.market_name, ''), EXCLUDED.market_name),
            groups = COALESCE(NULLIF(markets.groups, ''), EXCLUDED.groups),
            updated_at = NOW()
        RETURNING id
    `
    
    // ... 执行查询
}
```

**关键改进点：**
1. 在插入前调用 `MarketDescriptionsService` 获取 market name 和 groups
2. 从 events 表查询 home_team_name 和 away_team_name，用于替换 market name 中的变量
3. 使用 `COALESCE(NULLIF(...), ...)` 确保只在字段为空时更新，避免覆盖已有数据

## 部署步骤

### 1. 运行数据库迁移
```bash
# 执行迁移脚本
psql -U your_user -d your_database -f database/migrations/017_add_groups_to_markets.sql
```

### 2. 重启服务
```bash
# 重新编译并重启服务
go build -o uof-service
./uof-service
```

### 3. 验证修复
```sql
-- 检查 groups 字段是否已添加
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'markets' AND column_name = 'groups';

-- 检查现有数据是否已同步
SELECT COUNT(*) FROM markets WHERE groups IS NOT NULL AND groups != '';

-- 查看示例数据
SELECT sr_market_id, market_name, groups, specifiers 
FROM markets 
WHERE groups IS NOT NULL 
LIMIT 10;
```

## 注意事项

1. **MarketDescriptionsService 必须先启动**
   - 确保在处理 odds_change 消息前，MarketDescriptionsService 已经加载了 market descriptions
   - 如果 service 未启动或数据未加载，market_name 和 groups 仍然会是空值

2. **异步更新作为兜底机制**
   - 保留 `market_descriptions_service.go` 中的 `UpdateMissingNames` 函数
   - 定期运行以填充遗漏的 market_name 和 groups

3. **性能考虑**
   - 在 storeMarket 中查询 events 表可能会增加一点开销
   - 但这是必要的，因为 market name 中可能包含 {$competitor1} 等变量
   - 如果性能成为问题，可以考虑在事务开始前批量查询 event 信息

4. **向后兼容**
   - 使用 `COALESCE` 确保不会覆盖已有的非空数据
   - 对于已存在的 market 记录，只在字段为空时才更新

## 测试建议

1. **单元测试**
   - 测试 `GetMarketGroups` 方法返回正确的 groups
   - 测试 `storeMarket` 正确填充 market_name 和 groups

2. **集成测试**
   - 接收真实的 odds_change 消息
   - 验证 markets 表中的 market_name 和 groups 字段已正确填充

3. **回归测试**
   - 确保修改不影响现有功能
   - 验证 market 的冲突处理逻辑仍然正常工作
