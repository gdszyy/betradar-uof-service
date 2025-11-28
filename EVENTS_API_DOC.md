# API文档：GET /api/events

本文档详细说明了`betradar-uof-service`项目中`GET /api/events`接口的功能、参数和返回数据结构。

## 1. 接口概述

`GET /api/events`接口用于获取增强的赛事信息列表，包含完整的比赛数据和实时盘口赔率。该接口支持多种过滤条件和分页，方便前端进行灵活的数据查询和展示。

## 2. 请求

### 2.1. URL

```
GET /api/events
```

### 2.2. 查询参数

该接口支持以下查询参数，所有参数均为可选：

| 参数 | 类型 | 描述 | 示例 |
|---|---|---|---|
| `status` | string | 按比赛状态过滤。`live`表示滚球，`not_started`表示未开赛。 | `live` |
| `subscribed` | boolean | 按是否已订阅过滤。`true`或`false`。 | `true` |
| `sport_id` | string | 按体育类型ID过滤。 | `1` (足球) |
| `search` | string | 搜索查询。可精确匹配`event_id`或模糊匹配主客队名称。 | `Real Madrid` |
| `producer` | integer | 按Producer ID过滤。 | `1` |
| `is_live` | boolean | 只返回滚球中的比赛。`true`或`false`。 | `true` |
| `is_ended` | boolean | 按比赛是否结束过滤。`true`表示已结束，`false`表示未结束。 | `false` |
| `has_markets` | boolean | 只返回有盘口数据的比赛。`true`或`false`。 | `true` |
| `market_ids` | string | 按Market ID过滤，多个ID用逗号分隔。 | `1,16,18` |
| `page` | integer | 页码，默认为`1`。 | `2` |
| `page_size` | integer | 每页数量，默认为`100`，最大为`500`。 | `50` |

### 2.3. 排序规则

接口默认按照以下规则排序：

1. **主排序**：按最后更新时间（`last_update`）降序排列，最新更新的比赛排在前面
2. **次排序**：当更新时间相同时，按`event_id`排序

其中`last_update`取以下两者中的较新值：
- 盘口数据的最后更新时间（`markets.updated_at`）
- 比赛消息的最后接收时间（`tracked_events.last_message_at`）

这样可以确保有最新盘口变化或消息更新的比赛优先展示。

## 3. 响应

接口返回一个JSON对象，包含赛事列表和总数。

### 3.1. 成功响应示例

```json
{
  "success": true,
  "count": 1,
  "events": [
    {
      "event_id": "sr:match:12345",
      "srn_id": "srn:match:12345",
      "sport_id": "1",
      "status": "live",
      "schedule_time": "2025-11-28T20:00:00Z",
      "home_team_id": "sr:competitor:1",
      "home_team_name": "Real Madrid",
      "away_team_id": "sr:competitor:2",
      "away_team_name": "FC Barcelona",
      "home_score": 1,
      "away_score": 0,
      "match_status": "2nd_half",
      "match_time": "65:00",
      "message_count": 120,
      "last_message_at": "2025-11-28T21:10:00Z",
      "subscribed": true,
      "created_at": "2025-11-28T19:00:00Z",
      "updated_at": "2025-11-28T21:10:05Z",
      "sport": "Soccer",
      "sport_name": "足球",
      "match_status_mapped": "2nd Half",
      "match_status_name": "下半场",
      "match_time_mapped": "65:00",
      "home_team_id_mapped": "1",
      "away_team_id_mapped": "2",
      "is_live": true,
      "is_ended": false,
      "markets": [
        {
          "sr_market_id": "1",
          "market_name": "1X2",
          "specifiers": "",
          "status": "active",
          "producer_id": 1,
          "outcomes": [
            {
              "outcome_id": "1",
              "name": "Real Madrid",
              "outcome_name": "主胜",
              "odds": 1.5,
              "probability": 0.65,
              "active": true
            }
          ],
          "outcomes_count": 3,
          "updated_at": "2025-11-28T21:09:30Z"
        }
      ]
    }
  ]
}
```

### 3.2. 数据结构

#### `EnhancedEvent` 对象

| 字段 | 类型 | 描述 |
|---|---|---|
| `event_id` | string | 赛事唯一ID |
| `sport_id` | string | 体育类型ID |
| `status` | string | 赛事状态 (e.g., `live`, `not_started`, `ended`) |
| `home_team_name` | string | 主队名称 |
| `away_team_name` | string | 客队名称 |
| `home_score` | integer | 主队得分 |
| `away_score` | integer | 客队得分 |
| `match_status` | string | 比赛内部状态 (e.g., `2nd_half`) |
| `is_live` | boolean | 是否为滚球 |
| `is_ended` | boolean | 是否已结束 |
| `markets` | array | 盘口信息数组，见`MarketInfo` |

#### `MarketInfo` 对象

| 字段 | 类型 | 描述 |
|---|---|---|
| `sr_market_id` | string | Sportradar市场ID |
| `market_name` | string | 市场名称 (e.g., "1X2", "Handicap") |
| `specifiers` | string | 盘口参数 (e.g., `hcp=-1.5`) |
| `status` | string | 盘口状态 (e.g., `active`, `suspended`) |
| `outcomes` | array | 赔率项数组，见`OutcomeInfo` |

#### `OutcomeInfo` 对象

| 字段 | 类型 | 描述 |
|---|---|---|
| `outcome_id` | string | 赔率项ID |
| `name` | string | 赔率项名称 (e.g., "Real Madrid") |
| `outcome_name` | string | 翻译后的赔率项名称 (e.g., "主胜") |
| `odds` | float | 赔率 |
| `active` | boolean | 是否可投注 |

## 4. 使用示例

### 4.1. 获取所有滚球中的足球比赛

```bash
curl -X GET "http://<your-service-address>/api/events?sport_id=1&is_live=true"
```

### 4.2. 搜索包含"Barcelona"的比赛并获取1x2和让球盘

```bash
curl -X GET "http://<your-service-address>/api/events?search=Barcelona&market_ids=1,16"
```

### 4.3. 获取第二页已订阅的比赛

```bash
curl -X GET "http://<your-service-address>/api/events?subscribed=true&page=2&page_size=20"
```

## 5. 注意事项

- **性能**: 谨慎使用不带过滤条件的请求，尤其是在生产环境中，以避免返回大量数据导致性能问题。
- **分页**: 对于大量数据，请务必使用`page`和`page_size`参数进行分页查询。
- **数据映射**: `sport_name`, `match_status_name`等字段是由后端根据ID映射的，方便前端直接在前端直接显示。
