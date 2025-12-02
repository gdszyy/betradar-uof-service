# 市场卡片展示方案实现文档

## 概述

本文档描述了为 betradar-uof-service 项目实现市场卡片展示方案的完整实现。该方案为每个 market * specifier 记录添加 `tab_id` 和 `chip_id`，支持前端展示市场卡片的分层结构（Tab + Chip）。

## 核心概念

### Tab（一级导航）
- **定义**: 大分类，用户需要切换查看
- **数量**: 17 个
- **来源**: 
  - 基于 Groups（9 个）: 常规玩法、球员道具、角球、罚牌、射手、微盘口、组合玩法、上半场、下半场
  - 基于 Specifiers 聚合（8 个）: 分节、分时段、分盘、分局、分地图、分Frame、分Drive、分Over

### Chip（二级筛选）
- **定义**: 在 Tab 内的筛选维度
- **数量**: 每个 Tab 0-5 个 Chip Specifiers
- **作用**: 进一步细分 markets（如"分节"Tab 内筛选"第1节"、"第2节"）

## 实现架构

### 数据库模型

#### 1. 新增字段到 markets 表
```sql
ALTER TABLE markets
ADD COLUMN tab_id VARCHAR(50),
ADD COLUMN chip_id VARCHAR(200);
```

#### 2. 新建表

**market_tabs** - Tab 配置表
```sql
CREATE TABLE market_tabs (
    id VARCHAR(50) PRIMARY KEY,
    label VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,  -- 'group' or 'specifier_aggregate'
    market_count INTEGER DEFAULT 0,
    chip_specifiers TEXT,
    group_name VARCHAR(50),
    primary_specifier VARCHAR(50),
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**market_chips** - Chip 配置表
```sql
CREATE TABLE market_chips (
    id VARCHAR(200) PRIMARY KEY,
    tab_id VARCHAR(50) NOT NULL REFERENCES market_tabs(id),
    specifier VARCHAR(50),
    value VARCHAR(100),
    label VARCHAR(200) NOT NULL,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**market_tab_chip_mapping** - Market 与 Tab/Chip 的映射表
```sql
CREATE TABLE market_tab_chip_mapping (
    id SERIAL PRIMARY KEY,
    market_id INTEGER NOT NULL REFERENCES markets(id),
    event_id VARCHAR(100) NOT NULL,
    tab_id VARCHAR(50) NOT NULL REFERENCES market_tabs(id),
    chip_id VARCHAR(200),
    specifier_name VARCHAR(50),
    specifier_value VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(market_id, tab_id, chip_id)
);
```

### 核心服务

**MarketTabChipService** (`services/market_tab_chip_service.go`)

主要功能：
- `AssignTabChipToMarket()` - 为单个市场分配 tab_id 和 chip_id
- `AssignTabChipToAllMarkets()` - 为所有市场分配 tab_id 和 chip_id
- `GetMarketsByTabChip()` - 按 tab 和 chip 查询市场
- `GetTabsForEvent()` - 获取事件的所有 Tab
- `GetChipsForTab()` - 获取 Tab 的所有 Chip

### REST API 端点

**MarketTabChipHandler** (`handlers/market_tab_chip_handler.go`)

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/v1/events/{eventId}/tabs` | 获取事件的所有 Tab |
| GET | `/api/v1/events/{eventId}/markets?tab={tabId}&chip={chipId}` | 按 Tab/Chip 查询市场 |
| GET | `/api/v1/tabs/{tabId}/chips` | 获取 Tab 的所有 Chip |
| GET | `/api/v1/events/{eventId}/market-cards` | 获取完整的市场卡片数据 |
| GET | `/api/v1/health` | 健康检查 |

## 实现步骤

### 1. 数据库迁移

```bash
# 运行迁移脚本
psql $DATABASE_URL -f database/migrations/011_add_tab_chip_fields.sql
```

### 2. 导入 Tab 和 Chip 配置

```bash
# 使用导入脚本
./scripts/import_all.sh
```

或手动运行：

```bash
# 构建导入工具
go build -o bin/import_tab_chip ./cmd/import_tab_chip/main.go

# 导入配置
./bin/import_tab_chip \
  -db "$DATABASE_URL" \
  -tabs final_tab_chip_config.csv \
  -chips final_chip_enumeration.csv
```

### 3. 为市场分配 Tab/Chip

```bash
# 构建分配工具
go build -o bin/assign_tab_chip ./cmd/assign_tab_chip/main.go

# 执行分配
./bin/assign_tab_chip -db "$DATABASE_URL"
```

### 4. 集成到主应用

在主应用中注册 API 处理器：

```go
import (
    "betradar-uof-service/handlers"
    "betradar-uof-service/services"
)

// 在路由初始化中
tabChipService := services.NewMarketTabChipService(db)
tabChipHandler := handlers.NewMarketTabChipHandler(tabChipService)

// 注册路由
router.HandleFunc("/api/v1/events/{eventId}/tabs", tabChipHandler.GetTabsForEvent).Methods("GET")
router.HandleFunc("/api/v1/events/{eventId}/markets", tabChipHandler.GetMarketsByTabChip).Methods("GET")
router.HandleFunc("/api/v1/tabs/{tabId}/chips", tabChipHandler.GetChipsForTab).Methods("GET")
router.HandleFunc("/api/v1/events/{eventId}/market-cards", tabChipHandler.GetMarketCardData).Methods("GET")
router.HandleFunc("/api/v1/health", tabChipHandler.HealthCheck).Methods("GET")
```

## 文件结构

```
betradar-uof-service/
├── database/
│   ├── migrations/
│   │   └── 011_add_tab_chip_fields.sql      # 数据库迁移脚本
│   ├── models_tab_chip.go                    # Tab/Chip 数据模型
│   └── ...
├── services/
│   ├── market_tab_chip_service.go            # Tab/Chip 业务逻辑
│   └── ...
├── handlers/
│   ├── market_tab_chip_handler.go            # REST API 处理器
│   └── ...
├── cmd/
│   ├── import_tab_chip/
│   │   └── main.go                           # 导入工具
│   ├── assign_tab_chip/
│   │   └── main.go                           # 分配工具
│   └── ...
├── scripts/
│   └── import_all.sh                         # 完整导入脚本
├── final_tab_chip_config.csv                 # Tab 配置数据
├── final_chip_enumeration.csv                # Chip 配置数据
└── MARKET_TAB_CHIP_IMPLEMENTATION.md         # 本文档
```

## Tab 配置详情

### Tab 列表（17 个）

| 优先级 | Tab ID | Tab 名称 | 类型 | Chip Specifiers |
|--------|--------|---------|------|-----------------|
| 1 | `regular_play` | 常规玩法 | group | 无 |
| 2 | `innings` | 分局 | specifier_aggregate | inningnr, dismissalnr, deliverynr |
| 3 | `player_props` | 球员道具 | group | count, appearancenr |
| 4 | `micro_market` | 微盘口 | group | pitchnr, playnr, pointnr |
| 5 | `sets` | 分盘 | specifier_aggregate | setnr, gamenr, legnr, endnr |
| 6 | `maps` | 分地图 | specifier_aggregate | mapnr, roundnr |
| 7 | `bookings` | 罚牌 | group | 无 |
| 8 | `corners` | 角球 | group | cornernr |
| 9 | `1st_half` | 上半场 | group | goalnr, pointnr |
| 10 | `quarters` | 分节 | specifier_aggregate | quarternr |
| 11 | `combo` | 组合玩法 | group | 无 |
| 12 | `2nd_half` | 下半场 | group | goalnr |
| 13 | `periods` | 分时段 | specifier_aggregate | periodnr |
| 14 | `frames` | 分Frame | specifier_aggregate | framenr |
| 15 | `scorers` | 射手 | group | goalnr |
| 16 | `overs` | 分Over | specifier_aggregate | overnr |
| 17 | `drives` | 分Drive | specifier_aggregate | drivenr |

## 数据流程

### 1. 导入流程

```
CSV 文件
  ↓
import_tab_chip 工具
  ↓
market_tabs 表 + market_chips 表
```

### 2. 分配流程

```
markets 表（未分配）
  ↓
market_tab_chip_service.AssignTabChipToMarket()
  ↓
确定 tab_id（基于 groups 或 specifiers）
  ↓
确定 chip_id（基于 tab_id 和 specifiers）
  ↓
更新 markets 表（tab_id, chip_id）
  ↓
记录到 market_tab_chip_mapping 表
```

### 3. 查询流程

```
客户端请求
  ↓
GET /api/v1/events/{eventId}/tabs
  ↓
MarketTabChipHandler.GetTabsForEvent()
  ↓
MarketTabChipService.GetTabsForEvent()
  ↓
查询 market_tabs 表 + 统计 markets 表
  ↓
返回 JSON 响应
```

## API 使用示例

### 获取事件的所有 Tab

```bash
curl -X GET "http://localhost:8080/api/v1/events/sr:match:12345/tabs"
```

响应：
```json
{
  "event_id": "sr:match:12345",
  "tabs": [
    {
      "id": "regular_play",
      "label": "常规玩法",
      "type": "group",
      "market_count": 198,
      "chip_specifiers": "",
      "display_order": 0
    },
    {
      "id": "quarters",
      "label": "分节",
      "type": "specifier_aggregate",
      "market_count": 22,
      "chip_specifiers": "quarternr",
      "display_order": 9
    }
  ],
  "count": 2
}
```

### 按 Tab 和 Chip 查询市场

```bash
curl -X GET "http://localhost:8080/api/v1/events/sr:match:12345/markets?tab=quarters&chip=quarters_quarternr_1"
```

响应：
```json
{
  "event_id": "sr:match:12345",
  "tab_id": "quarters",
  "chip_id": "quarters_quarternr_1",
  "markets": [
    {
      "market_id": 235,
      "event_id": "sr:match:12345",
      "sr_market_id": "1:235",
      "market_type": "Quarter Winner",
      "market_name": "1st Quarter Winner",
      "specifiers": "quarternr=1",
      "status": "active",
      "tab_id": "quarters",
      "chip_id": "quarters_quarternr_1",
      "tab_label": "分节",
      "tab_type": "specifier_aggregate",
      "chip_label": "第1节",
      "chip_specifier": "quarternr",
      "chip_value": "1"
    }
  ],
  "count": 1
}
```

### 获取 Tab 的所有 Chip

```bash
curl -X GET "http://localhost:8080/api/v1/tabs/quarters/chips"
```

响应：
```json
{
  "tab_id": "quarters",
  "chips": [
    {
      "id": "quarters_quarternr_1",
      "tab_id": "quarters",
      "specifier": "quarternr",
      "value": "1",
      "label": "第1节",
      "display_order": 0
    },
    {
      "id": "quarters_quarternr_2",
      "tab_id": "quarters",
      "specifier": "quarternr",
      "value": "2",
      "label": "第2节",
      "display_order": 1
    }
  ],
  "count": 4
}
```

### 获取完整市场卡片数据

```bash
curl -X GET "http://localhost:8080/api/v1/events/sr:match:12345/market-cards"
```

响应：
```json
{
  "event_id": "sr:match:12345",
  "tabs": [...],
  "markets": {
    "regular_play": [...],
    "quarters": [...],
    "player_props": [...]
  },
  "chips": {
    "quarters": [...],
    "1st_half": [...]
  },
  "tab_count": 5
}
```

## 性能优化

### 索引

已在以下字段创建索引以优化查询性能：

```sql
-- markets 表
CREATE INDEX idx_markets_tab_id ON markets(tab_id);
CREATE INDEX idx_markets_chip_id ON markets(chip_id);
CREATE INDEX idx_markets_event_tab_chip ON markets(event_id, tab_id, chip_id);

-- market_tabs 表
CREATE INDEX idx_market_tabs_type ON market_tabs(type);
CREATE INDEX idx_market_tabs_display_order ON market_tabs(display_order);

-- market_chips 表
CREATE INDEX idx_market_chips_tab_id ON market_chips(tab_id);
CREATE INDEX idx_market_chips_specifier ON market_chips(specifier, value);

-- market_tab_chip_mapping 表
CREATE INDEX idx_mapping_market_id ON market_tab_chip_mapping(market_id);
CREATE INDEX idx_mapping_event_id ON market_tab_chip_mapping(event_id);
CREATE INDEX idx_mapping_tab_id ON market_tab_chip_mapping(tab_id);
CREATE INDEX idx_mapping_event_tab ON market_tab_chip_mapping(event_id, tab_id);
```

### 缓存表

- `market_groups_cache` - 缓存市场的 groups，避免重复解析
- `market_specifiers_cache` - 缓存解析后的 specifiers JSON

## 故障排查

### 问题：市场没有被分配 tab_id

**原因**: 市场的 groups 和 specifiers 都不匹配任何 Tab 配置

**解决方案**:
1. 检查市场的 groups 和 specifiers
2. 更新 `determineTabID()` 方法中的映射规则
3. 重新运行分配工具

### 问题：Chip 显示为空

**原因**: Tab 没有配置 chip_specifiers，或市场没有对应的 specifier

**解决方案**:
1. 检查 market_chips 表中是否有该 Tab 的 Chip 数据
2. 确认市场的 specifiers 包含 chip_specifiers 中的字段
3. 运行导入脚本重新导入 Chip 配置

### 问题：查询性能缓慢

**原因**: 缺少索引或查询优化不足

**解决方案**:
1. 检查索引是否已创建
2. 使用 `EXPLAIN ANALYZE` 分析查询计划
3. 考虑使用物化视图缓存常用查询

## 部署到 Railway

### 1. 准备配置文件

将 CSV 文件上传到项目根目录：
- `final_tab_chip_config.csv`
- `final_chip_enumeration.csv`

### 2. 运行迁移和导入

```bash
# 在 Railway 环境中
./scripts/import_all.sh
```

### 3. 更新主应用

在 `main.go` 中集成 API 处理器：

```go
// 初始化 Tab/Chip 服务
tabChipService := services.NewMarketTabChipService(db)
tabChipHandler := handlers.NewMarketTabChipHandler(tabChipService)

// 注册路由
router.HandleFunc("/api/v1/events/{eventId}/tabs", tabChipHandler.GetTabsForEvent).Methods("GET")
// ... 其他路由
```

### 4. 部署

```bash
git add .
git commit -m "Add market tab/chip display implementation"
git push origin main
```

Railway 将自动部署新版本。

## 监控和维护

### 监控指标

```sql
-- 检查分配进度
SELECT 
  COUNT(*) as total_markets,
  COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) as markets_with_tab,
  COUNT(CASE WHEN chip_id IS NOT NULL THEN 1 END) as markets_with_chip,
  ROUND(100.0 * COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) / COUNT(*), 2) as tab_assignment_percentage
FROM markets;

-- 检查 Tab 分布
SELECT tab_id, COUNT(*) as market_count
FROM markets
WHERE tab_id IS NOT NULL
GROUP BY tab_id
ORDER BY market_count DESC;

-- 检查 Chip 分布
SELECT chip_id, COUNT(*) as market_count
FROM markets
WHERE chip_id IS NOT NULL
GROUP BY chip_id
ORDER BY market_count DESC;
```

### 日志

所有导入和分配操作都会输出详细的日志，便于故障排查。

## 总结

本实现方案为 betradar-uof-service 提供了完整的市场卡片展示功能，包括：

1. **数据库模型** - 支持 Tab 和 Chip 的存储和查询
2. **业务逻辑** - 自动为市场分配 Tab 和 Chip
3. **REST API** - 提供灵活的查询接口
4. **导入工具** - 便捷的配置导入流程
5. **性能优化** - 索引和缓存支持

该方案可以直接用于前端和后端开发，支持响应式设计和多种展示方案。
