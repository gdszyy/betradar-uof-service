# Specifiers 数据结构修改说明

## 修改内容

将 `/api/events` 接口返回的 market 数据中的 `specifiers` 字段从**字典（map）**改为**列表（array）**。

## 修改的文件

- `web/enhanced_events_handler.go`

## 数据结构变化

### 修改前（字典结构）

```json
{
    "18": {
        "sr_market_id": "18",
        "market_name": "Total",
        "specifiers": {
            "total=150.5": {
                "specifier": "total=150.5",
                "status": "1",
                "producer_id": 3,
                "outcomes": [
                    {
                        "outcome_id": "12",
                        "name": "",
                        "outcome_name": "over 150.5",
                        "odds": 1.61,
                        "probability": 0.6227,
                        "active": true
                    }
                ],
                "updated_at": "2025-12-08T04:53:19.349527Z"
            },
            "total=155.5": {
                "specifier": "total=155.5",
                "status": "0",
                "producer_id": 3,
                "outcomes": [],
                "updated_at": "2025-12-08T04:53:11.116702Z"
            }
        }
    }
}
```

### 修改后（列表结构）

```json
{
    "18": {
        "sr_market_id": "18",
        "market_name": "Total",
        "specifiers": [
            {
                "specifier": "total=150.5",
                "status": "1",
                "producer_id": 3,
                "outcomes": [
                    {
                        "outcome_id": "12",
                        "name": "",
                        "outcome_name": "over 150.5",
                        "odds": 1.61,
                        "probability": 0.6227,
                        "active": true
                    }
                ],
                "updated_at": "2025-12-08T04:53:19.349527Z"
            },
            {
                "specifier": "total=155.5",
                "status": "0",
                "producer_id": 3,
                "outcomes": [],
                "updated_at": "2025-12-08T04:53:11.116702Z"
            }
        ]
    }
}
```

## 代码修改详情

### 1. 修改 `MarketGroup` 结构体定义

**修改前：**
```go
type MarketGroup struct {
    MarketID   string                     `json:"sr_market_id"`
    MarketName string                     `json:"market_name"`
    Specifiers map[string]SpecifierGroup `json:"specifiers"`
}
```

**修改后：**
```go
type MarketGroup struct {
    MarketID   string            `json:"sr_market_id"`
    MarketName string            `json:"market_name"`
    Specifiers []SpecifierGroup `json:"specifiers"`
}
```

### 2. 修改 `getEventMarketsGrouped` 函数

**修改前：**
```go
// 初始化为 map
Specifiers: make(map[string]SpecifierGroup),

// 使用 key 添加
specifierKey := specifierStr
if specifierKey == "" {
    specifierKey = "default"
}
marketGroup.Specifiers[specifierKey] = SpecifierGroup{...}
```

**修改后：**
```go
// 初始化为空 slice
Specifiers: []SpecifierGroup{},

// 直接 append 到列表
marketGroup.Specifiers = append(marketGroup.Specifiers, SpecifierGroup{...})
```

## 影响范围

- **受影响的接口：** `/api/events`
- **不受影响的接口：** `/api/events/{event_id}/markets` (该接口返回的是不同的数据结构)

## 优势

1. **更符合 RESTful 规范：** 列表结构更适合表示集合数据
2. **保持顺序：** 列表可以保持 specifiers 的原始顺序（按数据库查询顺序）
3. **易于遍历：** 前端可以直接使用 `map()` 或 `forEach()` 遍历
4. **避免 key 冲突：** 不需要处理空 specifier 的特殊 key（如 "default"）

## 注意事项

- 前端代码需要相应调整，从 `markets[marketId].specifiers[specifierKey]` 改为遍历 `markets[marketId].specifiers` 数组
- 每个 `SpecifierGroup` 对象中仍然包含 `specifier` 字段，可以用于识别具体的 specifier 值
