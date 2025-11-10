# Outcome Descriptions 表结构分析

## 问题描述

用户指出 `outcome_descriptions` 表的主键设计可能有问题:
- 当前: `(market_id, outcome_id)` 唯一
- 建议: `(market_id, specifiers, outcome_id)` 唯一

根据 Sportradar 文档,market 的唯一标识确实是 `market_id + specifiers`。

## 数据库现状

### 1. markets 表 (实际赔率数据) ✅ 正确

```sql
Table: markets
Columns:
  - event_id
  - sr_market_id
  - specifiers
  - ...

Unique Constraint:
  markets_event_id_sr_market_id_specifiers_key (event_id, sr_market_id, specifiers)
```

**设计正确!** 每个 event 的每个 market + specifiers 组合是唯一的。

### 2. market_descriptions 表 (市场模板) ✅ 正确

```sql
Table: market_descriptions
Columns:
  - market_id (PRIMARY KEY)
  - market_name
  - groups
  - specifiers (存储 specifier 定义,如 JSON)
  - updated_at
```

**设计正确!** 存储 market 的定义和 specifier 字段定义,不存储具体值。

### 3. outcome_descriptions 表 (结果描述) ⚠️ 需要分析

```sql
Table: outcome_descriptions
Columns:
  - id (PRIMARY KEY)
  - market_id
  - outcome_id
  - outcome_name

Unique Constraint:
  outcome_descriptions_market_id_outcome_id_key (market_id, outcome_id)
```

**当前设计:** 每个 market 的每个 outcome 只能有一个名称。

## Sportradar 数据结构

### Market Descriptions API 返回的数据

```xml
<market id="18" name="Total">
  <outcomes>
    <outcome id="12" name="over {total}"/>
    <outcome id="13" name="under {total}"/>
  </outcomes>
  <specifiers>
    <specifier name="total" type="decimal"/>
  </specifiers>
</market>
```

**关键点:**
- Outcome 名称包含占位符: `{total}`
- 这是一个**模板**,不是具体值

### Odds Change 消息中的实际数据

```xml
<market id="18" specifiers="total=2.5" status="1">
  <outcome id="12" name="over 2.5" odds="2.3" active="1"/>
  <outcome id="13" name="under 2.5" odds="1.55" active="1"/>
</market>

<market id="18" specifiers="total=3.5" status="1">
  <outcome id="12" name="over 3.5" odds="2.1" active="1"/>
  <outcome id="13" name="under 3.5" odds="1.65" active="1"/>
</market>
```

**关键点:**
- 同一个 market_id (18) 和 outcome_id (12)
- 但 specifiers 不同,outcome 名称也不同
- "over 2.5" vs "over 3.5"

## 问题分析

### 场景 1: 存储 Market Descriptions API 的模板数据

**目标:** 缓存 API 返回的市场定义

**数据示例:**
```
market_id=18, outcome_id=12, outcome_name="over {total}"
market_id=18, outcome_id=13, outcome_name="under {total}"
```

**当前设计:** ✅ 正确
- `(market_id, outcome_id)` 唯一约束足够
- 因为模板级别,每个 market 的每个 outcome 只有一个模板

### 场景 2: 存储实际 odds_change 消息中的 outcome 名称

**目标:** 缓存实际使用的 outcome 名称(已替换占位符)

**数据示例:**
```
market_id=18, specifiers="total=2.5", outcome_id=12, outcome_name="over 2.5"
market_id=18, specifiers="total=3.5", outcome_id=12, outcome_name="over 3.5"
```

**当前设计:** ❌ 错误
- `(market_id, outcome_id)` 唯一约束会导致冲突
- 需要 `(market_id, specifiers, outcome_id)` 唯一约束

## 当前代码行为

### market_descriptions_service.go

```go
// 从 Market Descriptions API 加载数据
func (s *MarketDescriptionsService) loadMarketDescriptions() error {
    // 解析 XML
    // 存储到 s.outcomes map[marketID]map[outcomeID]*OutcomeDescription
}

// 保存到数据库
func (s *MarketDescriptionsService) saveToDatabase() error {
    for marketID, outcomes := range s.outcomes {
        for _, outcome := range outcomes {
            // INSERT INTO outcome_descriptions (market_id, outcome_id, outcome_name)
            // VALUES (marketID, outcome.ID, outcome.Name)
        }
    }
}
```

**当前行为:** 存储的是**模板数据**,如 "over {total}"

### odds_parser.go / odds_change_parser.go

这些解析器处理实际的 odds_change 消息,但**没有**将 outcome 名称存储到 `outcome_descriptions` 表。

## 结论

### outcome_descriptions 表的用途

根据当前代码,`outcome_descriptions` 表的用途是:
- ✅ 存储 Market Descriptions API 返回的**模板数据**
- ❌ 不存储实际 odds_change 消息中的具体 outcome 名称

### 当前设计是否正确?

**对于模板数据:** ✅ 正确
- `(market_id, outcome_id)` 唯一约束是合适的
- 因为每个 market 的每个 outcome 只有一个模板

**对于实际数据:** ❌ 不适用
- 如果要存储实际的 outcome 名称,需要添加 `specifiers` 字段
- 并修改唯一约束为 `(market_id, specifiers, outcome_id)`

## 建议方案

### 方案 A: 保持当前设计 (推荐)

**适用场景:** 只存储模板数据

**优点:**
- 数据量小
- 符合 Market Descriptions API 的原始结构
- 当前代码无需修改

**缺点:**
- 使用时需要在代码中动态替换占位符

**实现:**
- 保持 `outcome_descriptions` 表当前结构
- 在查询 API 或前端显示时,动态替换 `{total}` 等占位符

### 方案 B: 添加 specifiers 字段

**适用场景:** 存储实际 odds_change 消息中的 outcome 名称

**优点:**
- 直接可用,无需替换占位符
- 可以缓存实际使用过的 outcome 名称

**缺点:**
- 数据量大(每个 specifier 值都要存一份)
- 需要修改代码逻辑
- 可能产生大量重复数据

**实现:**
1. 添加 `specifiers` 字段到 `outcome_descriptions` 表
2. 修改唯一约束为 `(market_id, specifiers, outcome_id)`
3. 修改 odds_change_parser 代码,保存实际的 outcome 名称

### 方案 C: 混合方案 (最佳)

**设计两个表:**

1. **`outcome_descriptions`** - 存储模板 (当前设计)
   - `(market_id, outcome_id)` 唯一
   - 存储 "over {total}" 等模板

2. **`outcome_instances`** - 存储实际值 (新表)
   - `(market_id, specifiers, outcome_id)` 唯一
   - 存储 "over 2.5" 等实际值
   - 可选,按需缓存

**优点:**
- 清晰分离模板和实例
- 灵活性高
- 可以选择性缓存常用的 outcome 实例

## 推荐行动

### 短期 (立即执行)

**保持当前设计不变**,因为:
1. 当前设计对于模板数据是正确的
2. `markets` 表已经有正确的 specifiers 处理
3. 实际的 outcome 名称可以从 odds_change 消息中实时获取

### 中期 (如果需要)

如果发现需要缓存实际的 outcome 名称:
1. 创建新表 `outcome_instances`
2. 添加 `(market_id, specifiers, outcome_id)` 唯一约束
3. 在 odds_change_parser 中保存实际的 outcome 名称

### 长期优化

考虑实现动态占位符替换服务:
```go
func (s *MarketDescriptionsService) GetOutcomeName(
    marketID string, 
    outcomeID string, 
    specifiers map[string]string,
) string {
    // 1. 从 outcome_descriptions 获取模板
    template := s.getTemplate(marketID, outcomeID)
    
    // 2. 替换占位符
    name := template
    for key, value := range specifiers {
        name = strings.Replace(name, "{"+key+"}", value, -1)
    }
    
    return name
}
```

## 参考文档

- [Sportradar - Identify a Market](https://docs.sportradar.com/uof/data-and-features/markets-and-outcomes/identify-a-market)
- [Sportradar - Specifiers](https://docs.sportradar.com/uof/introduction/key-concepts/specifiers)
- [Sportradar - Market Descriptions API](https://docs.sportradar.com/uof/api-and-structure/api/betting-descriptions/market-descriptions/endpoint)

---

**结论:** 当前 `outcome_descriptions` 表的设计对于存储模板数据是**正确的**。如果需要存储实际的 outcome 实例,建议创建新表而不是修改现有表。
