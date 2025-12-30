# Market/Outcome 名称获取修复方案

## 问题总结

当前 `betradar-uof-service` 项目中存在部分 Market 和 Outcome 名称为空的问题，经过代码审查和文档分析，主要原因如下:

1. **`market_descriptions_service.go` 中的模板替换不完整**: `GetMarketName` 和 `GetOutcomeName` 函数未实现对序数 (`{!X}`)、正负号 (`{+X}`, `{-X}`) 等特殊前缀的处理。

2. **Variant Market 处理流程存在缺陷**: 
   - `GetOutcomeName` 在检测到 `variant` specifier 时，没有立即触发 API 调用，而是依赖后台异步任务 `ProcessVariantMarkets`。
   - 后台任务只处理 `sr:` 开头的 variant，不处理 `pre:` 和 `liveodds:` 等产品特定的 variant。
   - 当前端请求时，如果后台任务尚未完成，名称仍然为空。

3. **对 `outcome_name_resolver.go` 的利用不充分**: 项目中已经存在一个独立的 `OutcomeNameResolver` 服务，它实现了完整的序数转换逻辑，但 `market_descriptions_service.go` 并未使用它。

## 修复方案

### 方案 1: 完善 `market_descriptions_service.go` 中的模板替换逻辑

**目标**: 在 `GetMarketName` 和 `GetOutcomeName` 函数中，补充对所有特殊前缀的处理。

**实施步骤**:

1. **引入序数转换函数**: 将 `outcome_name_resolver.go` 中的 `toOrdinal` 函数移至一个公共的 `utils` 包，或者在 `market_descriptions_service.go` 中重新实现。

2. **修改 `GetMarketName` 函数**:
   - 在现有的 specifiers 替换逻辑中，添加对 `{!X}` 的处理，将值转换为序数后再替换。
   - 添加对 `{+X}` 和 `{-X}` 的处理，确保正确添加符号。

3. **修改 `GetOutcomeName` 函数**:
   - 同样补充对 `{!X}`, `{+X}`, `{-X}` 的处理逻辑。

**代码示例** (伪代码):

```go
// 在 GetMarketName 和 GetOutcomeName 中的 specifiers 替换部分
if specifiers != "" {
    pairs := strings.Split(specifiers, "|")
    for _, pair := range pairs {
        parts := strings.Split(pair, "=")
        if len(parts) == 2 {
            key := parts[0]
            value := parts[1]
            
            // 基本替换
            name = strings.ReplaceAll(name, "{"+key+"}", value)
            
            // 序数替换 {!X}
            if strings.Contains(name, "{!"+key+"}") {
                ordinalValue := toOrdinal(value) // 需要实现 toOrdinal 函数
                name = strings.ReplaceAll(name, "{!"+key+"}", ordinalValue)
            }
            
            // 正号替换 {+X}
            if strings.Contains(name, "{+"+key+"}") {
                signedValue := formatWithSign(value, false) // 不取反
                name = strings.ReplaceAll(name, "{+"+key+"}", signedValue)
            }
            
            // 负号替换 {-X}
            if strings.Contains(name, "{-"+key+"}") {
                signedValue := formatWithSign(value, true) // 取反
                name = strings.ReplaceAll(name, "{-"+key+"}", signedValue)
            }
        }
    }
}
```

**优点**: 
- 修改范围小，风险低。
- 能立即解决大部分常规市场的名称显示问题。

**缺点**: 
- 不解决 Variant Market 的问题。

---

### 方案 2: 改进 Variant Market 的处理流程

**目标**: 确保在需要时能够实时获取 Variant Market 的名称，而不是依赖后台异步任务。

**实施步骤**:

1. **修改 `GetOutcomeName` 函数**:
   - 在检测到 `variant` specifier 且本地缓存中没有该 outcome 名称时，**同步调用** `fetchAndCacheVariant` 函数。
   - 为了避免阻塞过久，可以为 `fetchAndCacheVariant` 设置一个较短的超时时间 (例如 3 秒)。
   - 如果 API 调用失败或超时，记录警告日志，并返回一个默认值 (例如 `outcomeID`)。

2. **扩展 `ProcessVariantMarkets` 后台任务**:
   - 修改 SQL 查询，同时处理 `sr:`, `pre:`, `liveodds:`, `wns:` 等所有类型的 variant。
   - 对于 `pre:`, `liveodds:`, `wns:` 类型的 variant，调用 **Variant Market Description Direct API**:
     ```
     GET /{product}/descriptions/{language}/markets/{market_id}/variants/{variant_urn}
     ```
     其中 `{product}` 从 variant URN 的前缀中提取 (例如 `pre:markettext:1234` -> `product=pre`)。

3. **优化 `fetchAndCacheVariant` 函数**:
   - 添加对产品类型的判断，根据 variant URN 的前缀选择正确的 API 端点。
   - 添加缓存机制，避免对同一 `variant_urn` 的重复请求。

**代码示例** (伪代码):

```go
func (s *MarketDescriptionsService) GetOutcomeName(marketID, outcomeID, specifiers string, ctx *ReplacementContext) string {
    // ... 现有的查找逻辑 ...
    
    // 如果在 outcomes 中找不到，检查是否是 variant market
    if strings.Contains(specifiers, "variant=") {
        variantURN := s.extractVariantURN(specifiers)
        if variantURN != "" {
            // 尝试从缓存中获取
            s.mu.RLock()
            if outcomes, ok := s.outcomes[marketID]; ok {
                if outcome, ok := outcomes[outcomeID]; ok {
                    s.mu.RUnlock()
                    return outcome.Name
                }
            }
            s.mu.RUnlock()
            
            // 缓存中没有，同步调用 API
            logger.Printf("[MarketDescService] Variant outcome not cached, fetching synchronously: marketID=%s, outcomeID=%s, variant=%s", marketID, outcomeID, variantURN)
            name, err := s.fetchAndCacheVariant(marketID, outcomeID, variantURN)
            if err != nil {
                logger.Printf("[MarketDescService] ⚠️  Failed to fetch variant synchronously: %v", err)
                return outcomeID // 降级返回 ID
            }
            return name
        }
    }
    
    // ... 其他降级逻辑 ...
}
```

**优点**: 
- 能够彻底解决 Variant Market 名称为空的问题。
- 支持所有类型的 variant (sr:, pre:, liveodds:, wns:)。

**缺点**: 
- 同步 API 调用可能增加响应延迟 (首次请求)。
- 需要更复杂的错误处理和超时控制。

---

### 方案 3: 统一使用 `OutcomeNameResolver` (推荐)

**目标**: 将所有名称解析逻辑集中到 `OutcomeNameResolver` 中，避免代码重复，提高可维护性。

**实施步骤**:

1. **重构 `OutcomeNameResolver`**:
   - 将其改造为一个更通用的服务，能够处理所有类型的名称解析，包括常规模板替换和 Variant Market 查询。
   - 在 `OutcomeNameResolver` 中注入 `MarketDescriptionsService`，以便访问缓存和 API。

2. **修改 `market_descriptions_service.go`**:
   - `GetMarketName` 和 `GetOutcomeName` 函数内部调用 `OutcomeNameResolver` 的相应方法。
   - 保留 `fetchAndCacheVariant` 等底层函数，供 `OutcomeNameResolver` 使用。

3. **完善 `OutcomeNameResolver` 的功能**:
   - 补充对 `{+X}`, `{-X}` 等其他特殊前缀的处理 (当前只实现了 `{!X}`)。
   - 添加对 Variant Market 的处理逻辑。

**优点**: 
- 代码结构更清晰，职责分离明确。
- 便于单元测试和后续维护。
- 避免了 `market_descriptions_service.go` 和 `outcome_name_resolver.go` 中的重复逻辑。

**缺点**: 
- 需要较大的代码重构工作量。

---

## 推荐实施顺序

1. **短期 (立即实施)**: 采用 **方案 1**，快速修复常规市场的名称显示问题。
2. **中期 (1-2 周)**: 采用 **方案 2**，完善 Variant Market 的处理流程。
3. **长期 (可选)**: 采用 **方案 3**，进行代码重构，提高系统的可维护性。

## 测试建议

1. **单元测试**: 为 `toOrdinal`, `formatWithSign` 等工具函数编写单元测试。
2. **集成测试**: 
   - 测试包含各种 specifiers 的市场名称解析 (包括 `{!X}`, `{+X}`, `{-X}`)。
   - 测试 Variant Market 的名称获取 (包括 `sr:`, `pre:`, `liveodds:` 等类型)。
3. **回归测试**: 确保修改后，现有的名称解析功能不受影响。

---

**文档作者**: Manus AI  
**日期**: 2025年12月30日
