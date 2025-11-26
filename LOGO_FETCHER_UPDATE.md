# Logo 获取服务更新说明

## 更新日期
2025-11-25

## 更新内容

### 三层递进式查询策略

为了提高 Logo 获取成功率，我们实施了一个**三层递进式查询策略**，对每个队伍名称按以下顺序进行查询：

#### 第一层：原始名称查询 (P0)
- **目的**：最精确匹配，避免任何清洗带来的风险
- **动作**：直接使用原始队伍名称查询 TheSportsDB API

#### 第二层：标准清洗查询 (P4)
- **目的**：解决虚拟比赛标识符的问题
- **动作**：应用 P4 清洗策略后再次查询
- **清洗规则**：
  1. 移除括号内容（如 `(Eros)`, `(Maverick)` 等玩家代号）
  2. 移除 `SRL` 后缀（Simulated Reality League）
  3. 移除 `U19`, `U21` 等青年队后缀
  4. 移除 `Womens`, `Youth`, `Reserves` 等后缀

#### 第三层：替代名称查询
- **目的**：解决队伍更名、常用简称等问题
- **动作**：查询预定义的替代名称映射表
- **当前映射**：
  - `Houston Christian Huskies` → `Houston Baptist`

### 代码变更

#### 新增函数

1. **`cleanTeamNameP4(name string) string`**
   - 实现 P4 清洗策略
   - 使用正则表达式移除虚拟比赛标识符

2. **`queryAPIWithName(teamName string) (string, error)`**
   - 底层 API 查询函数
   - 从原来的 `fetchLogoFromTheSportsDB` 重构而来

3. **`fetchLogoFromTheSportsDB(teamName string) (string, error)`** (重构)
   - 实现三层递进式查询策略
   - 依次尝试原始名称、清洗名称和替代名称

#### 新增变量

- **`alternativeTeamNames map[string]string`**
  - 替代名称映射表
  - 可以在未来轻松扩展

### 预期效果

- **提高成功率**：通过三层策略，预计可以额外获取 2-5% 的队伍 Logo
- **更好的日志**：每层查询都会记录详细日志，便于调试和监控
- **可扩展性**：替代名称映射表可以轻松添加新的映射关系

### 验证结果

根据测试，该策略已成功为以下队伍获取 Logo：
- `Houston Christian Huskies` → 通过替代名称 `Houston Baptist` 成功获取
- `FC Barcelona SRL` → 通过清洗后的 `FC Barcelona` 成功获取
- `Real Madrid (Eros)` → 通过清洗后的 `Real Madrid` 成功获取

### 未来改进

1. **动态映射表**：将替代名称映射表存储在数据库中，支持动态更新
2. **更多清洗策略**：根据实际运行数据，添加更多清洗规则
3. **多数据源**：集成其他 Logo 数据源，进一步提高覆盖率
