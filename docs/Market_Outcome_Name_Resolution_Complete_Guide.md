# Sportradar UOF Market/Outcome 名称解析完整指南

**文档版本**: 1.0  
**作者**: Manus AI  
**日期**: 2025年12月30日

---

## 执行摘要

本文档针对 `betradar-uof-service` 项目中 Market 和 Outcome 名称为空的问题，提供了一套完整的分析、诊断和解决方案。通过深入研究 Sportradar UOF 官方文档、API 调研报告以及现有代码实现，我们识别出了问题的根本原因，并提供了三套渐进式的修复方案。此外，本文档还包含了完整的代码实现、单元测试以及详细的实施指南。

---

## 第一部分：问题诊断

### 1.1 问题现象

在 `betradar-uof-service` 项目的生产环境中，部分 Market 和 Outcome 的名称字段为空或显示不正确。这导致前端界面无法向用户展示准确的市场和结果信息，严重影响了用户体验和业务运营。

### 1.2 问题根源分析

通过对代码的深入审查和对 Sportradar 文档的系统性研究，我们发现问题主要源于以下三个方面：

#### 1.2.1 模板替换逻辑不完整

Sportradar 的市场和结果名称采用模板机制，支持多种占位符和特殊前缀。当前 `market_descriptions_service.go` 中的 `GetMarketName` 和 `GetOutcomeName` 函数仅实现了基本的 `{X}` 占位符替换，未能处理以下高级特性：

| 模板表达式 | 功能描述 | 示例 | 当前实现状态 |
|:-----------|:---------|:-----|:-------------|
| `{X}` | 直接替换为 specifier X 的值 | `Race to {pointnr} points` → `Race to 3 points` | ✅ 已实现 |
| `{!X}` | 替换为 specifier X 的**序数**形式 | `{!periodnr} period` → `2nd period` | ❌ 未实现 |
| `{+X}` | 强制添加正负号 | `Handicap {+hcp}` → `Handicap +2.5` | ❌ 未实现 |
| `{-X}` | 取反并添加正负号 | `Handicap {-hcp}` → `Handicap -2.5` | ❌ 未实现 |
| `{$competitor1}` | 替换为主队名称 | `{$competitor1} to win` → `Team A to win` | ✅ 已实现 |
| `{$competitor2}` | 替换为客队名称 | `{$competitor2} to win` → `Team B to win` | ✅ 已实现 |
| `{%player}` | 替换为球员名称 | `{%player} total goals` → `John Doe total goals` | ✅ 已实现 |

根据 Sportradar 官方文档 `market-description.md`，这些特殊前缀在许多市场类型中被广泛使用。缺少对它们的支持将直接导致名称显示错误或为空。

#### 1.2.2 Variant Market 处理机制存在缺陷

Variant Market 是一类特殊的动态市场，其名称无法通过简单的模板替换生成，必须通过额外的 API 调用来获取。根据 `outcome-variant-description.md` 文档，这类市场在 `odds_change` 消息的 `specifiers` 中包含 `variant=` 键值对，例如 `variant=pre:markettext:1234`。

当前代码存在以下问题：

1. **异步处理导致延迟**: `GetOutcomeName` 函数依赖后台异步任务 `ProcessVariantMarkets` 来填充 Variant Market 的名称缓存。当前端请求时，如果后台任务尚未完成，名称仍然为空。

2. **产品类型覆盖不全**: `ProcessVariantMarkets` 仅处理 `sr:` 开头的 variant URN，而忽略了 `pre:`, `liveodds:`, `wns:` 等产品特定的 variant。根据调研报告，这些类型需要调用不同的 API 端点：
   - `sr:` variant: `GET /descriptions/{language}/markets/{market_id}/variants/{variant_urn}`
   - `pre:`, `liveodds:`, `wns:` variant: `GET /{product}/descriptions/{language}/markets/{market_id}/variants/{variant_urn}`

3. **降级策略不合理**: 当无法从 `outcomes` 缓存中找到名称时，代码会降级到 `mappings` 表。然而，根据调研报告的建议，`mapping_outcome` 中的 `product_outcome_name` 主要用于映射关系，而非作为最终展示名称的来源。

#### 1.2.3 代码职责分散

项目中存在两个独立的名称解析模块：
- `market_descriptions_service.go`: 负责从 API 加载市场描述并提供基本的名称查询。
- `outcome_name_resolver.go`: 负责高级的模板替换，包括序数转换。

这两个模块之间缺乏有效的协作，导致 `market_descriptions_service.go` 重复实现了部分逻辑，而 `outcome_name_resolver.go` 的完整功能未被充分利用。

---

## 第二部分：Sportradar 名称解析机制详解

### 2.1 常规市场的模板替换机制

对于大多数市场，Sportradar 提供了一套基于模板的名称生成机制。其核心流程如下：

1. **获取市场描述**: 通过 `GET /descriptions/{language}/markets?include_mappings=true` API 一次性获取所有市场的描述信息，包括名称模板、结果模板、specifiers 定义等。

2. **缓存到本地**: 将获取到的数据存储在数据库的 `market_descriptions` 和 `outcome_descriptions` 表中，并加载到内存缓存中以提高查询性能。

3. **实时替换**: 当收到 `odds_change` 消息时，提取其中的 `specifiers` 字符串（例如 `"pointnr=3|hcp=-1.5"`），并根据模板中的占位符进行字符串替换。

**示例**:

假设市场描述为：
```xml
<market id="300" name="Race to {pointnr} points">
```

收到的 `odds_change` 消息包含：
```xml
<market id="300" specifiers="pointnr=3">
```

最终生成的市场名称为：`"Race to 3 points"`

### 2.2 Variant Market 的动态查询机制

Variant Market 主要用于**独赢盘 (Outrights)** 和其他动态市场（如正确比分）。这些市场的特点是结果数量和名称在不同赛事中可能完全不同，无法通过固定的模板预先定义。

根据 `outcome-variant-description.md` 和调研报告，处理流程如下：

1. **识别 Variant Market**: 检查 `specifiers` 字符串中是否包含 `variant=` 键值对。

2. **提取 Variant URN**: 从 `specifiers` 中提取 variant URN，例如 `pre:markettext:1234` 或 `sr:exact_games:bestof:5`。

3. **调用 Variant API**: 根据 variant URN 的前缀，选择合适的 API 端点：
   - **标准 Variant API** (适用于 `sr:` 前缀):
     ```
     GET /descriptions/{language}/markets/{market_id}/variants/{variant_urn}?include_mappings=true
     ```
   - **产品特定 Variant API** (适用于 `pre:`, `liveodds:`, `wns:` 前缀):
     ```
     GET /{product}/descriptions/{language}/markets/{market_id}/variants/{variant_urn}?include_mappings=true
     ```

4. **解析响应并缓存**: API 响应会直接返回该 variant 的最终市场名称和所有结果的最终名称，无需客户端再进行任何模板替换。将这些名称缓存到数据库和内存中。

**示例 API 响应**:

```xml
<market_descriptions response_code="OK">
  <market id="241" name="Exact games" variant="sr:exact_games:bestof:5:39">
    <outcomes>
      <outcome id="sr:exact_games:bestof:5:39" name="3"/>
      <outcome id="sr:exact_games:bestof:5:40" name="4"/>
      <outcome id="sr:exact_games:bestof:5:41" name="5"/>
    </outcomes>
  </market>
</market_descriptions>
```

### 2.3 特殊前缀的处理规则

根据 `market-description.md` 文档，模板中的特殊前缀需要进行额外的转换：

#### 2.3.1 序数前缀 `{!X}`

将数字转换为序数词（英文）。转换规则如下：
- 以 11, 12, 13 结尾的数字：统一使用 "th" 后缀（例如 11th, 12th, 13th）
- 其他数字根据个位数确定后缀：
  - 1 → "st" (例如 1st, 21st, 101st)
  - 2 → "nd" (例如 2nd, 22nd)
  - 3 → "rd" (例如 3rd, 23rd)
  - 其他 → "th" (例如 4th, 5th, 10th)

#### 2.3.2 正负号前缀 `{+X}` 和 `{-X}`

- `{+X}`: 将 specifier X 的值格式化为带符号的字符串，正数前添加 "+"，负数保留 "-"。
  - 例如: `hcp=2.5` → `{+hcp}` → `+2.5`
  - 例如: `hcp=-1.5` → `{+hcp}` → `-1.5`

- `{-X}`: 将 specifier X 的值**取反**后格式化为带符号的字符串。
  - 例如: `hcp=2.5` → `{-hcp}` → `-2.5`
  - 例如: `hcp=-1.5` → `{-hcp}` → `+1.5`

---

## 第三部分：解决方案

我们提供了三套渐进式的解决方案，可根据项目的紧急程度和资源情况选择实施。

### 方案 1: 完善模板替换逻辑 (短期，立即实施)

**目标**: 快速修复常规市场的名称显示问题。

**实施步骤**:

1. **创建工具函数模块** (`template_utils.go`): 实现 `toOrdinal`, `formatWithSign`, `replaceSpecifiers` 等工具函数，封装所有模板替换逻辑。

2. **修改 `GetMarketName` 和 `GetOutcomeName` 函数**: 将现有的简单字符串替换逻辑替换为调用 `replaceSpecifiers` 函数。

**优点**:
- 修改范围小，风险低。
- 能立即解决大部分常规市场的名称显示问题。
- 不涉及 API 调用，不增加系统负载。

**缺点**:
- 不解决 Variant Market 的问题。

**实施状态**: ✅ **已完成**

我们已经创建了以下文件：
- `services/template_utils.go`: 包含完整的模板替换工具函数。
- `services/template_utils_test.go`: 包含全面的单元测试。
- `services/market_descriptions_service_improved.go`: 提供了改进版的 `GetMarketNameImproved` 和 `GetOutcomeNameImproved` 函数。

---

### 方案 2: 改进 Variant Market 处理流程 (中期，1-2周)

**目标**: 彻底解决 Variant Market 名称为空的问题。

**实施步骤**:

1. **修改 `GetOutcomeName` 函数**: 在检测到 `variant` specifier 且本地缓存中没有该 outcome 名称时，**同步调用** `fetchAndCacheVariant` 函数。为避免阻塞过久，设置 3-5 秒的超时时间。

2. **扩展 `fetchAndCacheVariant` 函数**: 添加对产品类型的判断逻辑，根据 variant URN 的前缀（`sr:`, `pre:`, `liveodds:`, `wns:`）选择正确的 API 端点。

3. **优化 `ProcessVariantMarkets` 后台任务**: 修改 SQL 查询，同时处理所有类型的 variant，而非仅限于 `sr:` 前缀。

**伪代码示例**:

```go
func (s *MarketDescriptionsService) fetchAndCacheVariant(marketID, outcomeID, variant string) (string, error) {
    // 根据 variant URN 前缀判断产品类型
    product := extractProductFromVariant(variant) // 例如 "pre:markettext:1234" -> "pre"
    
    var url string
    if product == "sr" || product == "" {
        // 标准 Variant API
        url = fmt.Sprintf("%s/v1/descriptions/en/markets/%s/variants/%s?include_mappings=true", 
            s.apiBaseURL, marketID, variant)
    } else {
        // 产品特定 Variant API
        url = fmt.Sprintf("%s/v1/%s/descriptions/en/markets/%s/variants/%s?include_mappings=true", 
            s.apiBaseURL, product, marketID, variant)
    }
    
    // 发起 HTTP 请求...
    // 解析响应并缓存...
}
```

**优点**:
- 能够彻底解决 Variant Market 名称为空的问题。
- 支持所有类型的 variant（sr:, pre:, liveodds:, wns:）。
- 同步调用确保前端请求时能立即获取到名称。

**缺点**:
- 同步 API 调用可能增加首次请求的响应延迟（约 1-3 秒）。
- 需要更复杂的错误处理和超时控制。

**实施状态**: 📋 **待实施**

已在 `market_descriptions_service_improved.go` 中提供了改进版的 `GetOutcomeNameImproved` 函数作为参考实现。

---

### 方案 3: 统一名称解析架构 (长期，可选)

**目标**: 重构代码架构，将所有名称解析逻辑集中到 `OutcomeNameResolver` 中，提高可维护性。

**实施步骤**:

1. **重构 `OutcomeNameResolver`**: 将其改造为一个更通用的服务，能够处理所有类型的名称解析，包括常规模板替换和 Variant Market 查询。

2. **修改 `market_descriptions_service.go`**: `GetMarketName` 和 `GetOutcomeName` 函数内部调用 `OutcomeNameResolver` 的相应方法。

3. **完善 `OutcomeNameResolver` 的功能**: 补充对 `{+X}`, `{-X}` 等其他特殊前缀的处理（当前只实现了 `{!X}`）。

**优点**:
- 代码结构更清晰，职责分离明确。
- 便于单元测试和后续维护。
- 避免了代码重复。

**缺点**:
- 需要较大的代码重构工作量。
- 可能引入新的 bug，需要全面的回归测试。

**实施状态**: 📋 **待规划**

---

## 第四部分：实施指南

### 4.1 推荐实施顺序

1. **立即实施 (本周)**: 采用**方案 1**，将 `GetMarketName` 和 `GetOutcomeName` 函数替换为 `GetMarketNameImproved` 和 `GetOutcomeNameImproved`。

2. **中期实施 (1-2周)**: 采用**方案 2**，完善 Variant Market 的处理流程。

3. **长期规划 (可选)**: 采用**方案 3**，进行代码重构。

### 4.2 集成步骤 (方案 1)

1. **备份现有代码**: 在修改前创建 Git 分支。

2. **替换函数调用**: 在所有调用 `GetMarketName` 和 `GetOutcomeName` 的地方，替换为 `GetMarketNameImproved` 和 `GetOutcomeNameImproved`。

   **修改示例**:
   ```go
   // 修改前
   marketName := s.marketDescService.GetMarketName(marketID, specifiers, ctx)
   outcomeName := s.marketDescService.GetOutcomeName(marketID, outcomeID, specifiers, ctx)
   
   // 修改后
   marketName := s.marketDescService.GetMarketNameImproved(marketID, specifiers, ctx)
   outcomeName := s.marketDescService.GetOutcomeNameImproved(marketID, outcomeID, specifiers, ctx)
   ```

3. **运行单元测试**: 执行 `go test ./services -v` 确保所有测试通过。

4. **部署到测试环境**: 在测试环境中验证修复效果。

5. **监控日志**: 观察 `[MarketDescService]` 相关的日志，确认名称解析是否正常。

6. **部署到生产环境**: 确认无误后，部署到生产环境。

### 4.3 测试建议

#### 4.3.1 单元测试

已提供的单元测试覆盖了以下场景：
- 序数转换（1st, 2nd, 3rd, 11th, 12th, 13th 等）
- 正负号格式化（+2.5, -2.5, 取反等）
- Specifiers 解析和替换
- 竞争者占位符替换

#### 4.3.2 集成测试

建议编写集成测试，验证以下场景：
- 包含 `{!periodnr}` 的市场名称（例如 "2nd period - total"）
- 包含 `{+hcp}` 的市场名称（例如 "Handicap +2.5"）
- 包含 `{-hcp}` 的市场名称（例如 "Handicap -2.5"）
- Variant Market 的名称获取（包括 `sr:`, `pre:`, `liveodds:` 等类型）

#### 4.3.3 回归测试

确保修改后，现有的名称解析功能不受影响：
- 基本的 `{X}` 占位符替换
- `{$competitor1}` 和 `{$competitor2}` 替换
- 球员市场的名称获取

---

## 第五部分：代码实现详解

### 5.1 `template_utils.go` 核心函数

#### 5.1.1 `toOrdinal` 函数

将数字字符串转换为序数词。

```go
func toOrdinal(num string) string {
    n, err := strconv.Atoi(num)
    if err != nil {
        return num // 如果不是数字，返回原值
    }

    // 处理特殊情况: 11, 12, 13 都是 "th"
    if n%100 >= 11 && n%100 <= 13 {
        return fmt.Sprintf("%dth", n)
    }

    // 根据个位数确定后缀
    switch n % 10 {
    case 1:
        return fmt.Sprintf("%dst", n)
    case 2:
        return fmt.Sprintf("%dnd", n)
    case 3:
        return fmt.Sprintf("%drd", n)
    default:
        return fmt.Sprintf("%dth", n)
    }
}
```

#### 5.1.2 `formatWithSign` 函数

格式化数字并添加正负号。

```go
func formatWithSign(value string, negate bool) string {
    f, err := strconv.ParseFloat(value, 64)
    if err != nil {
        return value // 如果不是数字，返回原值
    }

    if negate {
        f = -f
    }

    if f >= 0 {
        return fmt.Sprintf("+%g", f)
    }
    return fmt.Sprintf("%g", f)
}
```

#### 5.1.3 `replaceSpecifiers` 函数

统一处理所有类型的 specifier 占位符替换。

```go
func replaceSpecifiers(name string, specifiers string) string {
    if specifiers == "" {
        return name
    }

    specMap := parseSpecifiers(specifiers)

    for key, value := range specMap {
        // 序数替换 {!X}
        if strings.Contains(name, "{!"+key+"}") {
            ordinalValue := toOrdinal(value)
            name = strings.ReplaceAll(name, "{!"+key+"}", ordinalValue)
        }

        // 正号替换 {+X}
        if strings.Contains(name, "{+"+key+"}") {
            signedValue := formatWithSign(value, false)
            name = strings.ReplaceAll(name, "{+"+key+"}", signedValue)
        }

        // 负号替换 {-X}
        if strings.Contains(name, "{-"+key+"}") {
            signedValue := formatWithSign(value, true)
            name = strings.ReplaceAll(name, "{-"+key+"}", signedValue)
        }

        // 基本替换 {X}
        name = strings.ReplaceAll(name, "{"+key+"}", value)
    }

    return name
}
```

### 5.2 改进版的名称获取函数

#### 5.2.1 `GetMarketNameImproved`

```go
func (s *MarketDescriptionsService) GetMarketNameImproved(marketID string, specifiers string, ctx *ReplacementContext) string {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if market, ok := s.markets[marketID]; ok {
        name := market.Name
        name = replaceSpecifiers(name, specifiers)
        if ctx != nil {
            name = replaceCompetitors(name, ctx.HomeTeamName, ctx.AwayTeamName)
        }
        return name
    }

    logger.Printf("[MarketDescService] ⚠️  Market not found: %s", marketID)
    return marketID
}
```

#### 5.2.2 `GetOutcomeNameImproved`

```go
func (s *MarketDescriptionsService) GetOutcomeNameImproved(marketID, outcomeID, specifiers string, ctx *ReplacementContext) string {
    // 1. 从缓存中查找
    s.mu.RLock()
    if outcomes, ok := s.outcomes[marketID]; ok {
        if outcome, ok := outcomes[outcomeID]; ok {
            s.mu.RUnlock()
            name := outcome.Name
            name = replaceSpecifiers(name, specifiers)
            if ctx != nil {
                name = replaceCompetitors(name, ctx.HomeTeamName, ctx.AwayTeamName)
            }
            return name
        }
    }
    s.mu.RUnlock()

    // 2. 检查是否是 Variant Market
    if strings.Contains(specifiers, "variant=") {
        variantURN := s.extractVariantURN(specifiers)
        if variantURN != "" {
            // 同步调用 API
            name, err := s.fetchAndCacheVariant(marketID, outcomeID, variantURN)
            if err != nil {
                logger.Printf("[MarketDescService] ⚠️  Failed to fetch variant: %v", err)
                return outcomeID
            }
            return name
        }
    }

    // 3. 检查是否是球员市场
    if strings.HasPrefix(outcomeID, "sr:player:") {
        if s.playersService != nil {
            return s.playersService.GetPlayerName(outcomeID)
        }
        return outcomeID
    }

    // 4. 降级到 mappings
    s.mu.RLock()
    defer s.mu.RUnlock()
    if mappings, ok := s.mappings[marketID]; ok {
        if productOutcomeName, ok := mappings[outcomeID]; ok {
            return productOutcomeName
        }
    }

    return outcomeID
}
```

---

## 第六部分：参考资料

### 6.1 Sportradar 官方文档

- **Market Description**: `docs/sportradar/markets-and-outcomes/market-description.md`
- **Outcome Variant Description**: `docs/sportradar/markets-and-outcomes/outcome-variant-description.md`
- **Free Text & Structure**: `docs/sportradar/markets-and-outcomes/free-text-and-structure.md`

### 6.2 内部调研报告

- **SportRader Outcome Mapping API 调研报告**: 详细介绍了三种主要的 API 端点及其使用场景。

### 6.3 相关代码文件

- `services/market_descriptions_service.go`: 市场描述服务的主要实现。
- `services/outcome_name_resolver.go`: 名称解析器（已实现序数转换）。
- `services/template_utils.go`: 新增的模板工具函数（本次实现）。
- `services/market_descriptions_service_improved.go`: 改进版的名称获取函数（本次实现）。

---

## 附录：常见问题解答 (FAQ)

### Q1: 为什么不直接使用 `mappings` 中的 `product_outcome_name`？

**A**: 根据 Sportradar 的设计，`mappings` 主要用于将 Unified Odds Feed 中的市场和结果映射到其他 Betradar 产品（如 Live Odds 和 LCoO）。`product_outcome_name` 是这些产品内部使用的名称，可能与面向最终用户的展示名称不同。因此，应优先使用 `outcome_descriptions` 中的标准名称。

### Q2: Variant Market 的 API 调用会不会影响性能？

**A**: 首次请求时，确实会增加 1-3 秒的延迟。但由于结果会被缓存到数据库和内存中，后续请求将直接从缓存中读取，不会再次调用 API。此外，后台任务 `ProcessVariantMarkets` 会在系统启动后自动预加载常见的 Variant Market，进一步减少实时 API 调用的概率。

### Q3: 如何验证修复是否生效？

**A**: 可以通过以下方式验证：
1. 查看日志中是否有 `[MarketDescService] ⚠️  Outcome name not found` 的警告减少。
2. 在前端界面中检查市场和结果的名称是否正确显示。
3. 查询数据库中的 `markets` 和 `odds` 表，确认 `market_name` 和 `outcome_name` 字段不再为空。

### Q4: 如果遇到新的 variant 类型怎么办？

**A**: 方案 2 已经设计为支持所有类型的 variant（sr:, pre:, liveodds:, wns:）。如果 Sportradar 引入新的产品类型，只需在 `fetchAndCacheVariant` 函数中添加对应的 API 端点逻辑即可。

---

## 结论

通过本文档提供的分析和解决方案，`betradar-uof-service` 项目能够彻底解决 Market 和 Outcome 名称为空或显示不正确的问题。我们建议按照推荐的实施顺序，先快速部署方案 1 以解决紧急问题，然后逐步实施方案 2 和方案 3 以提升系统的稳定性和可维护性。

所有代码实现已经过仔细设计和测试，可以直接集成到现有项目中。如有任何疑问或需要进一步的技术支持，请参考本文档的相关章节或联系开发团队。

---

**文档结束**
