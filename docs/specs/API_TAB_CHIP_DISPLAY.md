# 市场卡片展示 API 文档

## 概述

本文档说明如何通过 API 获取市场的 Tab 和 Chip 信息，用于前端展示市场卡片。

## 数据结构

### 市场数据结构

```json
{
  "id": 227952,
  "event_id": 12345,
  "market_id": "sr:market:227952",
  "market_type": "totals",
  "tab_id": "regular_play",
  "chip_id": null,
  "specifiers": "total=4",
  "groups": null,
  "name": "总进球数"
}
```

### Tab 数据结构

```json
{
  "id": "quarters",
  "label": "分节",
  "type": "specifier",
  "order": 10
}
```

### Chip 数据结构

```json
{
  "id": "quarters_quarternr_1",
  "tab_id": "quarters",
  "specifier": "quarternr",
  "value": "1",
  "label": "第1节"
}
```

## API 端点

### 1. 获取事件的所有 Tab

**请求**：
```
GET /api/v1/events/{eventId}/tabs
```

**响应**：
```json
{
  "status": "success",
  "data": [
    {
      "id": "regular_play",
      "label": "常规玩法",
      "type": "group",
      "order": 0
    },
    {
      "id": "quarters",
      "label": "分节",
      "type": "specifier",
      "order": 10
    }
  ]
}
```

### 2. 获取 Tab 的所有 Chip

**请求**：
```
GET /api/v1/tabs/{tabId}/chips
```

**响应**：
```json
{
  "status": "success",
  "data": [
    {
      "id": "quarters_quarternr_1",
      "tab_id": "quarters",
      "specifier": "quarternr",
      "value": "1",
      "label": "第1节"
    },
    {
      "id": "quarters_quarternr_2",
      "tab_id": "quarters",
      "specifier": "quarternr",
      "value": "2",
      "label": "第2节"
    }
  ]
}
```

### 3. 获取事件的市场卡片数据

**请求**：
```
GET /api/v1/events/{eventId}/market-cards
```

**查询参数**：
- `tab_id` (optional): 按 Tab 过滤
- `chip_id` (optional): 按 Chip 过滤
- `limit` (optional, default=100): 返回数量限制
- `offset` (optional, default=0): 分页偏移

**响应**：
```json
{
  "status": "success",
  "data": {
    "tabs": [
      {
        "id": "regular_play",
        "label": "常规玩法",
        "type": "group",
        "order": 0,
        "market_count": 179124
      },
      {
        "id": "quarters",
        "label": "分节",
        "type": "specifier",
        "order": 10,
        "market_count": 3705,
        "chips": [
          {
            "id": "quarters_quarternr_1",
            "label": "第1节",
            "market_count": 1490
          },
          {
            "id": "quarters_quarternr_2",
            "label": "第2节",
            "market_count": 748
          }
        ]
      }
    ],
    "markets": [
      {
        "id": 227952,
        "event_id": 12345,
        "market_id": "sr:market:227952",
        "market_type": "totals",
        "tab_id": "regular_play",
        "chip_id": null,
        "name": "总进球数",
        "specifiers": "total=4"
      }
    ],
    "total": 182833,
    "limit": 100,
    "offset": 0
  }
}
```

### 4. 获取指定 Tab 和 Chip 的市场

**请求**：
```
GET /api/v1/events/{eventId}/markets?tab={tabId}&chip={chipId}
```

**响应**：
```json
{
  "status": "success",
  "data": [
    {
      "id": 552280,
      "event_id": 12345,
      "market_id": "sr:market:552280",
      "market_type": "quarters",
      "tab_id": "quarters",
      "chip_id": "quarters_quarternr_1",
      "name": "第1节总进球数",
      "specifiers": "quarternr=1|total=20.5"
    }
  ]
}
```

## 前端展示方案

### 方案一：两层卡片展示（推荐）

```
┌─────────────────────────────────────────┐
│  常规玩法 (regular_play)                 │
│  - 总进球数                              │
│  - 让球                                  │
│  - 大小球                                │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  分节 (quarters)                        │
│  ┌─────────────────────────────────────┐│
│  │ 第1节 (quarters_quarternr_1)        ││
│  │ - 第1节总进球数                      ││
│  │ - 第1节让球                          ││
│  └─────────────────────────────────────┘│
│  ┌─────────────────────────────────────┐│
│  │ 第2节 (quarters_quarternr_2)        ││
│  │ - 第2节总进球数                      ││
│  │ - 第2节让球                          ││
│  └─────────────────────────────────────┘│
└─────────────────────────────────────────┘
```

### 方案二：单层卡片展示

```
┌─────────────────────────────────────────┐
│  常规玩法                                │
│  [总进球数] [让球] [大小球]              │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  分节 > 第1节                            │
│  [第1节总进球数] [第1节让球]             │
└─────────────────────────────────────────┘
```

## 数据分布

### 按 Tab 分布

| Tab ID | 市场数 | 占比 | Chips |
|--------|--------|------|-------|
| regular_play | 179,124 | 97.97% | 无 |
| quarters | 3,705 | 2.03% | 4 个 |
| sets | 4 | 0.002% | 2 个 |

### 按 Chip 分布（仅 quarters tab）

| Chip ID | 市场数 | 占比 |
|---------|--------|------|
| quarters_quarternr_1 | 1,490 | 40.2% |
| quarters_quarternr_2 | 748 | 20.2% |
| quarters_quarternr_3 | 740 | 20.0% |
| quarters_quarternr_4 | 727 | 19.6% |

## 前端处理逻辑

### 获取市场卡片数据

```javascript
// 1. 获取事件的所有 Tab
const tabs = await fetch('/api/v1/events/{eventId}/tabs').then(r => r.json());

// 2. 对于每个 Tab，获取其 Chips（如果有）
for (const tab of tabs) {
  if (tab.type === 'specifier') {
    const chips = await fetch(`/api/v1/tabs/${tab.id}/chips`).then(r => r.json());
    tab.chips = chips;
  }
}

// 3. 获取市场数据
const markets = await fetch(`/api/v1/events/{eventId}/markets`).then(r => r.json());

// 4. 按 Tab 和 Chip 分组市场
const groupedMarkets = {};
for (const market of markets) {
  const key = `${market.tab_id}:${market.chip_id || 'null'}`;
  if (!groupedMarkets[key]) {
    groupedMarkets[key] = [];
  }
  groupedMarkets[key].push(market);
}
```

### 渲染市场卡片

```javascript
// 对于每个 Tab
for (const tab of tabs) {
  console.log(`<Tab: ${tab.label}>`);
  
  if (tab.chips && tab.chips.length > 0) {
    // 有 Chips 的 Tab：显示两层
    for (const chip of tab.chips) {
      console.log(`  <Chip: ${chip.label}>`);
      
      const markets = groupedMarkets[`${tab.id}:${chip.id}`] || [];
      for (const market of markets) {
        console.log(`    <Market: ${market.name}>`);
      }
    }
  } else {
    // 无 Chips 的 Tab：显示单层
    const markets = groupedMarkets[`${tab.id}:null`] || [];
    for (const market of markets) {
      console.log(`  <Market: ${market.name}>`);
    }
  }
}
```

## 性能优化

### 缓存策略

1. **Tab 和 Chip 配置**
   - 缓存时间：24 小时
   - 变化频率：低（配置级别）

2. **市场数据**
   - 缓存时间：1 小时
   - 变化频率：中（市场级别）

### 查询优化

1. **使用分页**
   ```
   GET /api/v1/events/{eventId}/markets?limit=100&offset=0
   ```

2. **使用过滤**
   ```
   GET /api/v1/events/{eventId}/markets?tab=quarters&chip=quarters_quarternr_1
   ```

3. **批量查询**
   ```
   GET /api/v1/events/{eventId}/market-cards
   ```

## 错误处理

### 常见错误

| 错误码 | 说明 | 处理方式 |
|--------|------|--------|
| 404 | 事件或 Tab 不存在 | 显示空列表 |
| 500 | 服务器错误 | 重试或显示错误信息 |

### 降级方案

如果 API 不可用，前端应该：
1. 显示缓存的数据
2. 显示简化的市场列表（不分 Tab 和 Chip）
3. 显示错误提示

## 相关文件

- `MARKET_TAB_CHIP_IMPLEMENTATION.md` - 详细实现
- `GROUPS_AND_TABS_GUIDE.md` - Groups 和 Tabs 指南
- `scripts/assign_all_chip_ids.py` - Chip ID 分配脚本
- `handlers/market_tab_chip_handler.go` - API 处理器
