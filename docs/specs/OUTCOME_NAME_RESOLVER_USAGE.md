# Outcome Name Resolver 使用指南

## 概述

`OutcomeNameResolver` 服务用于将 outcome 名称模板(如 "over {total}")转换为实际的 outcome 名称(如 "over 2.5")。

## 为什么需要这个服务?

根据 Sportradar UOF 的设计:

1. **Market Descriptions API** 返回的是**模板**:
   ```xml
   <outcome id="12" name="over {total}"/>
   ```

2. **Odds Change 消息**中包含具体的 specifiers:
   ```xml
   <market id="18" specifiers="total=2.5">
     <outcome id="12" odds="2.3"/>
   </market>
   ```

3. 需要将模板 + specifiers 组合成实际名称:
   ```
   "over {total}" + "total=2.5" = "over 2.5"
   ```

## 数据库设计说明

### outcome_descriptions 表

**用途:** 存储 outcome 名称**模板**

**结构:**
```sql
CREATE TABLE outcome_descriptions (
    id SERIAL PRIMARY KEY,
    market_id VARCHAR(200) NOT NULL,
    outcome_id VARCHAR(200) NOT NULL,
    outcome_name TEXT NOT NULL,
    UNIQUE (market_id, outcome_id)
);
```

**数据示例:**
```
market_id | outcome_id | outcome_name
----------|------------|-------------
18        | 12         | over {total}
18        | 13         | under {total}
```

**为什么不需要 specifiers 字段?**
- 因为存储的是**模板**,不是具体值
- 每个 market 的每个 outcome 只有一个模板
- 具体值在运行时通过 `OutcomeNameResolver` 动态生成

## 使用方法

### 1. 初始化服务

```go
// 在 main.go 或服务初始化时
marketDescService := services.NewMarketDescriptionsService(
    cfg.UOFAPIToken,
    cfg.APIBaseURL,
    db,
    playersService,
)

// 加载 market descriptions
if err := marketDescService.LoadAndCache(); err != nil {
    log.Fatalf("Failed to load market descriptions: %v", err)
}

// 创建 OutcomeNameResolver
outcomeResolver := services.NewOutcomeNameResolver(marketDescService)
```

### 2. 解析单个 Outcome 名称

```go
// 从 odds_change 消息中获取的数据
marketID := "18"
outcomeID := "12"
specifiers := "total=2.5"

// 解析 outcome 名称
name, err := outcomeResolver.ResolveOutcomeName(marketID, outcomeID, specifiers)
if err != nil {
    log.Printf("Failed to resolve outcome name: %v", err)
    name = outcomeID // fallback to ID
}

fmt.Println(name) // 输出: "over 2.5"
```

### 3. 批量解析 Outcome 名称

```go
// 一个 market 的所有 outcomes
marketID := "18"
outcomeIDs := []string{"12", "13"}
specifiers := "total=2.5"

// 批量解析
names, err := outcomeResolver.BatchResolveOutcomeNames(marketID, outcomeIDs, specifiers)
if err != nil {
    log.Printf("Failed to resolve outcome names: %v", err)
}

// names = map[string]string{
//     "12": "over 2.5",
//     "13": "under 2.5",
// }
```

### 4. 解析 Market 名称

```go
marketID := "447"
specifiers := "periodnr=1|total=2.5"
homeTeam := "Manchester United"
awayTeam := "Liverpool"

name, err := outcomeResolver.ResolveMarketName(
    marketID,
    specifiers,
    homeTeam,
    awayTeam,
)

// 输出: "1st period - Manchester United total"
```

## 集成到现有代码

### 在 OddsChangeParser 中使用

```go
type OddsChangeParser struct {
    db              *sql.DB
    outcomeResolver *OutcomeNameResolver // 添加这个字段
}

func (p *OddsChangeParser) ParseOddsChange(xmlContent string) error {
    // ... 解析 XML ...

    for _, market := range oddsChange.Markets {
        marketID := market.ID
        specifiers := market.Specifiers

        for _, outcome := range market.Outcomes {
            outcomeID := outcome.ID

            // 解析 outcome 名称
            outcomeName, err := p.outcomeResolver.ResolveOutcomeName(
                marketID,
                outcomeID,
                specifiers,
            )
            if err != nil {
                log.Printf("Failed to resolve outcome name: %v", err)
                outcomeName = outcomeID // fallback
            }

            // 保存到数据库或返回给前端
            // ...
        }
    }
}
```

### 在 API 响应中使用

```go
// GET /api/markets/:event_id
func GetMarkets(c *gin.Context) {
    eventID := c.Param("event_id")

    // 从数据库查询 markets
    markets, err := db.Query(`
        SELECT sr_market_id, specifiers, home_team_name, away_team_name
        FROM markets
        WHERE event_id = $1
    `, eventID)

    var response []MarketResponse
    for markets.Next() {
        var m Market
        markets.Scan(&m.MarketID, &m.Specifiers, &m.HomeTeam, &m.AwayTeam)

        // 解析 market 名称
        marketName, _ := outcomeResolver.ResolveMarketName(
            m.MarketID,
            m.Specifiers,
            m.HomeTeam,
            m.AwayTeam,
        )

        response = append(response, MarketResponse{
            MarketID:   m.MarketID,
            MarketName: marketName,
            Specifiers: m.Specifiers,
        })
    }

    c.JSON(200, response)
}
```

## 支持的占位符类型

### 1. 普通占位符 `{xxx}`

从 specifiers 中获取值并替换

**示例:**
- 模板: `"over {total}"`
- Specifiers: `"total=2.5"`
- 结果: `"over 2.5"`

### 2. 序数词占位符 `{!xxx}`

从 specifiers 中获取数字并转换为序数词

**示例:**
- 模板: `"{!periodnr} period"`
- Specifiers: `"periodnr=1"`
- 结果: `"1st period"`

### 3. 队伍占位符 `{$competitor1}` `{$competitor2}`

需要额外提供队伍名称

**示例:**
- 模板: `"{$competitor1} total"`
- 队伍名称: `"Manchester United"`
- 结果: `"Manchester United total"`

### 4. 球员占位符 `{$player}`

需要额外提供球员名称(未来实现)

## Specifiers 格式

Specifiers 是一个字符串,格式为 `key=value|key2=value2`

**示例:**
```
"total=2.5"
"hcp=1.5"
"periodnr=1|total=2.5"
"goalnr=2"
```

## 错误处理

### 1. Market 不存在

```go
name, err := resolver.ResolveOutcomeName("999", "12", "total=2.5")
// err: "market 999 not found"
```

**处理方式:** 使用 outcome ID 作为 fallback

### 2. Outcome 不存在

```go
name, err := resolver.ResolveOutcomeName("18", "999", "total=2.5")
// err: "outcome 999 not found in market 18"
```

**处理方式:** 使用 outcome ID 作为 fallback

### 3. Specifiers 格式错误

如果 specifiers 格式错误,`parseSpecifiers` 会返回空 map,占位符不会被替换。

**示例:**
- 模板: `"over {total}"`
- Specifiers: `"invalid"` (格式错误)
- 结果: `"over {total}"` (保持原样)

## 性能考虑

### 1. 缓存

`MarketDescriptionsService` 已经在内存中缓存了所有 market descriptions,查询非常快。

### 2. 批量处理

对于同一个 market 的多个 outcomes,使用 `BatchResolveOutcomeNames` 可以减少重复的模板查询。

### 3. 懒加载

只在需要显示 outcome 名称时才调用 resolver,不需要预先生成所有可能的组合。

## 测试

### 单元测试示例

```go
func TestResolveOutcomeName(t *testing.T) {
    // 创建 mock MarketDescriptionsService
    mockService := &MockMarketDescriptionsService{
        outcomes: map[string]map[string]*OutcomeDescription{
            "18": {
                "12": {ID: "12", Name: "over {total}"},
                "13": {ID: "13", Name: "under {total}"},
            },
        },
    }

    resolver := NewOutcomeNameResolver(mockService)

    tests := []struct {
        marketID   string
        outcomeID  string
        specifiers string
        expected   string
    }{
        {"18", "12", "total=2.5", "over 2.5"},
        {"18", "13", "total=3.5", "under 3.5"},
        {"18", "12", "", "over {total}"}, // 没有 specifiers
    }

    for _, tt := range tests {
        result, err := resolver.ResolveOutcomeName(
            tt.marketID,
            tt.outcomeID,
            tt.specifiers,
        )
        assert.NoError(t, err)
        assert.Equal(t, tt.expected, result)
    }
}
```

## 常见问题

### Q1: 为什么不在数据库中存储所有可能的 outcome 名称?

**A:** 因为可能的组合数量太大:
- Market 18 (Total Goals) 可能有 total=0.5, 1.5, 2.5, 3.5, ..., 10.5
- 每个 total 值有 2 个 outcomes (over/under)
- 一个 market 就可能有 20+ 个组合
- 数据库会变得非常大

使用模板 + 动态替换的方式更高效。

### Q2: 如果需要缓存实际的 outcome 名称怎么办?

**A:** 可以在应用层添加缓存:

```go
type OutcomeNameCache struct {
    cache map[string]string // key: "marketID|specifiers|outcomeID"
    mu    sync.RWMutex
}

func (c *OutcomeNameCache) Get(marketID, specifiers, outcomeID string) (string, bool) {
    key := fmt.Sprintf("%s|%s|%s", marketID, specifiers, outcomeID)
    c.mu.RLock()
    defer c.mu.RUnlock()
    name, exists := c.cache[key]
    return name, exists
}
```

### Q3: 占位符替换失败怎么办?

**A:** 代码会保持占位符原样,如 "over {total}"。调用方应该检查结果中是否还包含 `{` 和 `}`,如果有则说明替换失败。

## 参考文档

- [Sportradar - Identify a Market](https://docs.sportradar.com/uof/data-and-features/markets-and-outcomes/identify-a-market)
- [Sportradar - Specifiers](https://docs.sportradar.com/uof/introduction/key-concepts/specifiers)
- [数据库设计分析](./ANALYSIS_OUTCOME_DESCRIPTIONS_SCHEMA.md)

---

**总结:** `OutcomeNameResolver` 提供了一个清晰、高效的方式来处理 outcome 名称的动态生成,无需修改数据库 schema,保持了数据的简洁性和灵活性。
