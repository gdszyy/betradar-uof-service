# 实现总结：Categories、Tournaments 和 Events 接口

## 概述

根据 `docs/events_api_documentation.md` 和 `docs/league_api_documentation.md` 文档，我已经成功实现了以下三个接口：

1. **`/api/categories`** - 获取分类（Category）列表
2. **`/api/tournaments`** - 获取联赛（Tournament）列表
3. **`/api/events`** - 优化现有接口，支持 `market_id` 参数

---

## 1. `/api/categories` 接口

### 功能描述

获取所有追踪的分类列表，支持按体育类型筛选、分页和排序，并包含该分类下的比赛数量。

### 请求参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `sport_ids` | `string` | 无 | 否 | 体育类型 ID 列表（逗号分隔），例如：`sr:sport:1,sr:sport:2` |
| `page` | `int` | 1 | 否 | 页码 |
| `page_size` | `int` | 100 | 否 | 每页记录数（最大 500） |
| `sort` | `string` | `name_asc` | 否 | 排序方式：`name_asc`（按名称升序）、`name_desc`（按名称降序）、`match_count_asc`（按比赛数量升序）、`match_count_desc`（按比赛数量降序） |

### 响应示例

```json
{
  "success": true,
  "count": 10,
  "total": 50,
  "page": 1,
  "page_size": 100,
  "total_pages": 1,
  "sort": "name_asc",
  "data": [
    {
      "id": "sr:category:1",
      "name": "sr:category:1",
      "sport_id": "sr:sport:1",
      "match_count": 25
    }
  ]
}
```

### 实现细节

- **数据来源**：从 `tracked_events` 表的 `srn_id` 字段中提取 `category` 信息
- **srn_id 格式**：`sr:sport:1:category:1:tournament:100:match:12345`
- **提取逻辑**：使用 PostgreSQL 正则表达式 `regexp_match(srn_id, 'category:([0-9]+)')` 提取 category ID
- **比赛数量统计**：通过 `LEFT JOIN` 和 `COUNT` 统计每个 category 下的比赛数量

---

## 2. `/api/tournaments` 接口

### 功能描述

获取指定分类下的所有联赛列表，支持分页和排序，并包含该联赛下的比赛数量。

### 请求参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `category_id` | `string` | 无 | **是** | 分类 ID，例如：`sr:category:1` |
| `page` | `int` | 1 | 否 | 页码 |
| `page_size` | `int` | 100 | 否 | 每页记录数（最大 500） |
| `sort` | `string` | `name_asc` | 否 | 排序方式：`name_asc`（按名称升序）、`name_desc`（按名称降序）、`match_count_asc`（按比赛数量升序）、`match_count_desc`（按比赛数量降序） |

### 响应示例

```json
{
  "success": true,
  "count": 5,
  "total": 15,
  "page": 1,
  "page_size": 100,
  "total_pages": 1,
  "sort": "name_asc",
  "data": [
    {
      "id": "sr:tournament:100",
      "name": "sr:tournament:100",
      "category_id": "sr:category:1",
      "sport_id": "sr:sport:1",
      "match_count": 10
    }
  ]
}
```

### 实现细节

- **数据来源**：从 `tracked_events` 表的 `srn_id` 字段中提取 `tournament` 信息
- **srn_id 格式**：`sr:sport:1:category:1:tournament:100:match:12345`
- **提取逻辑**：使用 PostgreSQL 正则表达式 `regexp_match(srn_id, 'tournament:([0-9]+)')` 提取 tournament ID
- **过滤逻辑**：通过 `srn_id LIKE '%' || $1 || '%'` 过滤指定 category 下的 tournaments
- **比赛数量统计**：通过 `LEFT JOIN` 和 `COUNT` 统计每个 tournament 下的比赛数量

---

## 3. `/api/events` 接口优化

### 功能描述

在现有的 `/api/events` 接口基础上，新增 `market_id` 参数支持。当传入 `market_id` 参数时，返回对应的 markets 和 outcomes 数据；不传入时，不返回 markets 字段。

### 新增参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `market_id` | `string` | 无 | 否 | 盘口 ID 列表（逗号分隔），例如：`1,18,26`。ID 对应 `sr_market_id` 字段。 |

### 响应示例（传入 `market_id`）

```json
{
  "success": true,
  "count": 10,
  "total": 100,
  "page": 1,
  "page_size": 10,
  "total_pages": 10,
  "filters": {
    "market_id": "1,18"
  },
  "matches": [
    {
      "event_id": "sr:match:12345",
      "home_team_name": "Manchester United",
      "away_team_name": "Liverpool",
      "markets": [
        {
          "id": 1,
          "event_id": "sr:match:12345",
          "market_id": "1",
          "specifier": "",
          "status": "Active",
          "cashout_status": "Enabled",
          "bet_status": "Open",
          "outcomes": [
            {
              "id": 1,
              "market_id": 1,
              "outcome_id": "1",
              "odds": 2.50,
              "status": "Active",
              "bet_status": "Open"
            },
            {
              "id": 2,
              "market_id": 1,
              "outcome_id": "X",
              "odds": 3.20,
              "status": "Active",
              "bet_status": "Open"
            },
            {
              "id": 3,
              "market_id": 1,
              "outcome_id": "2",
              "odds": 2.80,
              "status": "Active",
              "bet_status": "Open"
            }
          ]
        }
      ]
    }
  ]
}
```

### 响应示例（不传入 `market_id`）

```json
{
  "success": true,
  "count": 10,
  "total": 100,
  "page": 1,
  "page_size": 10,
  "total_pages": 10,
  "filters": {},
  "matches": [
    {
      "event_id": "sr:match:12345",
      "home_team_name": "Manchester United",
      "away_team_name": "Liverpool"
      // 没有 markets 字段
    }
  ]
}
```

### 实现细节

- **新增字段**：在 `EnhancedMatchDetail` 结构体中添加 `Markets []MarketInfo` 字段，使用 `json:"markets,omitempty"` 标签，当 markets 为空时不返回该字段
- **数据加载**：当 `market_id` 参数存在时，调用 `getMarketsForEvent` 函数加载 markets 和 outcomes 数据
- **查询逻辑**：
  - 从 `markets` 表中查询指定 `event_id` 和 `sr_market_id` 的 markets
  - 通过 `LEFT JOIN outcomes` 表加载对应的 outcomes 数据
  - 使用 `map` 结构避免重复，并保持 markets 和 outcomes 的嵌套关系

---

## 代码文件

### 新增文件

1. **`web/categories_tournaments_handler.go`**
   - 实现 `handleGetCategories` 函数（处理 `/api/categories` 请求）
   - 实现 `handleGetTournaments` 函数（处理 `/api/tournaments` 请求）
   - 定义 `CategoryInfo` 和 `TournamentInfo` 结构体

2. **`web/markets_handler.go`**
   - 实现 `getMarketsForEvent` 函数（获取指定赛事和盘口类型的 markets 和 outcomes 数据）
   - 定义 `MarketInfo` 和 `OutcomeInfo` 结构体

### 修改文件

1. **`web/match_mapper.go`**
   - 在 `EnhancedMatchDetail` 结构体中添加 `Markets []MarketInfo` 字段

2. **`web/events_filter_handler.go`**
   - 在 `handleGetEventsWithFilters` 函数中添加 markets 数据加载逻辑（第 103-112 行）

3. **`web/server.go`**
   - 注册 `/api/categories` 和 `/api/tournaments` 路由（第 187-188 行）

---

## 部署说明

### 1. 自动部署

代码已推送到 GitHub `main` 分支，Railway 会自动检测并部署新版本。

### 2. 测试接口

部署完成后，可以使用以下命令测试接口：

```bash
# 测试 /api/categories 接口
curl "https://betradar-uof-service-production.up.railway.app/api/categories?sport_ids=sr:sport:1&page=1&page_size=10&sort=match_count_desc"

# 测试 /api/tournaments 接口
curl "https://betradar-uof-service-production.up.railway.app/api/tournaments?category_id=sr:category:1&page=1&page_size=10&sort=match_count_desc"

# 测试 /api/events 接口（不传入 market_id）
curl "https://betradar-uof-service-production.up.railway.app/api/events?page=1&page_size=10"

# 测试 /api/events 接口（传入 market_id）
curl "https://betradar-uof-service-production.up.railway.app/api/events?page=1&page_size=10&market_id=1,18"
```

---

## 注意事项

### 1. 数据依赖

- **Categories 和 Tournaments** 数据是从 `tracked_events` 表的 `srn_id` 字段中动态提取的，不需要额外的数据库表。
- **srn_id 格式**：`sr:sport:1:category:1:tournament:100:match:12345`
- 如果 `srn_id` 字段为空或格式不正确，可能导致 categories 和 tournaments 数据为空。

### 2. 性能优化

- **分页**：所有接口都支持分页，避免一次性加载过多数据。
- **索引**：建议在 `tracked_events` 表的 `srn_id` 字段上创建索引，以提高查询性能。
- **缓存**：`/api/events` 接口已经使用了缓存机制，可以减少数据库查询次数。

### 3. 数据准确性

- **Category 和 Tournament 名称**：目前直接使用 ID 作为名称（例如：`sr:category:1`），如果需要显示真实的名称（例如：`England`、`Premier League`），需要调用 Sportradar API 获取静态数据。
- **比赛数量统计**：基于 `tracked_events` 表中的数据，只统计已追踪的比赛，不包括未追踪的比赛。

---

## 总结

我已经成功实现了以下功能：

✅ **`/api/categories`** 接口：支持按体育类型筛选、分页和排序，并返回比赛数量。

✅ **`/api/tournaments`** 接口：支持按分类ID筛选、分页和排序，并返回比赛数量。

✅ **`/api/events`** 接口优化：支持 `market_id` 参数，按需返回 markets 和 outcomes 数据。

所有代码已提交并推送到 GitHub `main` 分支，Railway 会自动部署新版本。

---

**Commit 信息**:
```
feat: add /api/categories and /api/tournaments endpoints, optimize /api/events with market_id filter

- Add /api/categories endpoint: supports filtering by sport_ids, pagination, and sorting
- Add /api/tournaments endpoint: supports filtering by category_id, pagination, and sorting
- Optimize /api/events endpoint: add market_id parameter to return markets and outcomes data
- Add MarketInfo and OutcomeInfo structures for market data
- Extract category and tournament data from srn_id field in tracked_events table
```

**Commit ID**: `161729c`
