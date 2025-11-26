# API 接口修改说明

**修改日期**: 2025-11-26  
**修改人**: Manus AI  
**版本**: v1.1.0

---

## 概述

本次修改优化了三个前端 API 接口，提升了数据查询的灵活性和性能：

1. **`/api/categories`** - 获取分类数据（优化）
2. **`/api/tournaments`** - 获取联赛数据（优化）
3. **`/api/events`** - 获取比赛数据（新增 market 过滤功能）

---

## 1. 获取分类数据 `/api/categories`

### 接口说明

获取指定体育类型下的所有分类（Category）信息，并包含每个分类下的比赛数量。

### 请求方式

```
GET /api/categories
```

### 请求参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `sport_ids` | string | 否 | - | 体育类型列表，逗号分隔（例如：`sr:sport:1,sr:sport:2`） |
| `page` | integer | 否 | 1 | 分页页数 |
| `page_size` | integer | 否 | 20 | 每页大小（最大 100） |
| `sort` | string | 否 | `name` | 排序方式：`name`（按字母）、`match_count_asc`（比赛数升序）、`match_count_desc`（比赛数降序） |

### 响应示例

```json
{
  "success": true,
  "data": [
    {
      "category_id": "sr:category:1",
      "category_name": "England",
      "sport_id": "sr:sport:1",
      "match_count": 45
    },
    {
      "category_id": "sr:category:4",
      "category_name": "Spain",
      "sport_id": "sr:sport:1",
      "match_count": 32
    }
  ],
  "page": 1,
  "page_size": 20,
  "count": 2,
  "total": 15,
  "total_pages": 1
}
```

### 修改内容

- ✅ 支持多个体育类型筛选（`sport_ids` 参数）
- ✅ 支持分页（`page` 和 `page_size` 参数）
- ✅ 支持多种排序方式（`sort` 参数）
- ✅ 返回每个分类下的比赛数量（`match_count` 字段）
- ✅ 数据来源从 `teams` 表改为 `tracked_events` 表，确保数据准确性

### 使用示例

```bash
# 获取足球的所有分类，按比赛数量降序排列
curl "http://localhost:8080/api/categories?sport_ids=sr:sport:1&sort=match_count_desc&page_size=10"

# 获取多个体育类型的分类
curl "http://localhost:8080/api/categories?sport_ids=sr:sport:1,sr:sport:2&page=1&page_size=20"
```

---

## 2. 获取联赛数据 `/api/tournaments`

### 接口说明

获取指定分类（Category）下的所有联赛（Tournament）信息，并包含每个联赛下的比赛数量。

### 请求方式

```
GET /api/tournaments
```

### 请求参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `category_id` | string | **是** | - | 分类 ID（例如：`sr:category:1`） |
| `page` | integer | 否 | 1 | 分页页数 |
| `page_size` | integer | 否 | 20 | 每页大小（最大 100） |
| `sort` | string | 否 | `name` | 排序方式：`name`（按字母）、`match_count_asc`（比赛数升序）、`match_count_desc`（比赛数降序） |

### 响应示例

```json
{
  "success": true,
  "data": [
    {
      "tournament_id": "sr:tournament:17",
      "tournament_name": "Premier League",
      "category_id": "sr:category:1",
      "sport_id": "sr:sport:1",
      "match_count": 20
    },
    {
      "tournament_id": "sr:tournament:34",
      "tournament_name": "FA Cup",
      "category_id": "sr:category:1",
      "sport_id": "sr:sport:1",
      "match_count": 8
    }
  ],
  "page": 1,
  "page_size": 20,
  "count": 2,
  "total": 5,
  "total_pages": 1
}
```

### 修改内容

- ✅ 支持按分类筛选（`category_id` 参数，必填）
- ✅ 支持分页（`page` 和 `page_size` 参数）
- ✅ 支持多种排序方式（`sort` 参数）
- ✅ 返回每个联赛下的比赛数量（`match_count` 字段）
- ✅ 数据来源从 `teams` 表改为 `tracked_events` 表，确保数据准确性

### 使用示例

```bash
# 获取英格兰的所有联赛，按比赛数量降序排列
curl "http://localhost:8080/api/tournaments?category_id=sr:category:1&sort=match_count_desc&page_size=10"

# 获取第二页数据
curl "http://localhost:8080/api/tournaments?category_id=sr:category:1&page=2&page_size=20"
```

---

## 3. 获取比赛数据 `/api/events`

### 接口说明

获取比赛数据，支持按需返回指定的 market（盘口）信息，避免返回所有 market 造成的性能问题。

### 请求方式

```
GET /api/events
```

### 请求参数

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `status` | string | 否 | - | 比赛状态（例如：`live`、`not_started`） |
| `subscribed` | string | 否 | - | 是否已订阅（`true`/`false`） |
| `sport_id` | string | 否 | - | 体育类型 ID（例如：`sr:sport:1`） |
| `search` | string | 否 | - | 搜索关键词（支持 event_id 精确匹配或队伍名称模糊匹配） |
| `producer` | string | 否 | - | Producer ID（例如：`1` 表示 Live Odds） |
| `is_live` | string | 否 | - | 是否为直播比赛（`true`/`false`） |
| `is_ended` | string | 否 | - | 是否已结束（`true`/`false`） |
| `has_markets` | string | 否 | - | 是否有盘口数据（`true`/`false`） |
| **`market_ids`** | **string** | **否** | **-** | **盘口 ID 列表，逗号分隔（例如：`1,18,226`）** |
| `page` | integer | 否 | 1 | 分页页数 |
| `page_size` | integer | 否 | 100 | 每页大小（最大 500） |

### 响应示例

```json
{
  "success": true,
  "count": 2,
  "events": [
    {
      "event_id": "sr:match:12345678",
      "sport_id": "sr:sport:1",
      "status": "live",
      "home_team_name": "Manchester United",
      "away_team_name": "Liverpool",
      "home_score": 1,
      "away_score": 1,
      "markets": [
        {
          "sr_market_id": "1",
          "market_name": "1X2",
          "status": "active",
          "producer_id": 1,
          "outcomes": [
            {
              "outcome_id": "1",
              "outcome_name": "Manchester United",
              "odds": 2.50,
              "active": true
            },
            {
              "outcome_id": "X",
              "outcome_name": "Draw",
              "odds": 3.20,
              "active": true
            },
            {
              "outcome_id": "2",
              "outcome_name": "Liverpool",
              "odds": 2.80,
              "active": true
            }
          ],
          "outcomes_count": 3
        },
        {
          "sr_market_id": "18",
          "market_name": "Total Goals",
          "specifiers": "total=2.5",
          "status": "active",
          "producer_id": 1,
          "outcomes": [
            {
              "outcome_id": "over",
              "outcome_name": "Over 2.5",
              "odds": 1.85,
              "active": true
            },
            {
              "outcome_id": "under",
              "outcome_name": "Under 2.5",
              "odds": 1.95,
              "active": true
            }
          ],
          "outcomes_count": 2
        }
      ]
    }
  ]
}
```

### 修改内容

- ✅ **新增 `market_ids` 参数**：支持按需返回指定的 market 信息
- ✅ 当 `market_ids` 为空时，返回所有 market（保持向后兼容）
- ✅ 当 `market_ids` 有值时，只返回指定的 market（例如：`market_ids=1,18,226`）
- ✅ 优化查询性能，避免返回大量不需要的 market 数据

### 使用示例

```bash
# 获取所有直播比赛，只返回 1X2 和 Total Goals 盘口
curl "http://localhost:8080/api/events?is_live=true&market_ids=1,18"

# 获取足球比赛，只返回 1X2、Asian Handicap 和 Total Goals 盘口
curl "http://localhost:8080/api/events?sport_id=sr:sport:1&market_ids=1,16,18&page_size=50"

# 获取所有比赛的所有盘口（不传 market_ids 参数）
curl "http://localhost:8080/api/events?is_live=true"
```

---

## 常用 Market ID 参考

以下是一些常用的 Sportradar Market ID：

| Market ID | Market Name | 说明 |
|-----------|-------------|------|
| `1` | 1X2 | 主胜/平局/客胜 |
| `16` | Asian Handicap | 亚洲让球 |
| `18` | Total Goals | 总进球数 |
| `226` | Both Teams to Score | 双方都进球 |
| `8` | Correct Score | 正确比分 |
| `10` | Double Chance | 双重机会 |
| `14` | Odd/Even | 单/双 |

---

## 技术实现细节

### 数据来源优化

**修改前**：
- `/api/categories` 和 `/api/tournaments` 从 `teams` 表查询数据
- 问题：`teams` 表没有 `tournament_id` 字段，无法返回联赛数据

**修改后**：
- 从 `tracked_events` 表查询数据
- 优点：`tracked_events` 表包含完整的 `category_id`、`category_name`、`tournament_id`、`tournament_name` 字段
- 结果：可以准确返回分类和联赛数据，并统计每个分类/联赛下的比赛数量

### Market 过滤实现

**修改前**：
- `/api/events` 总是返回所有 market 数据
- 问题：当比赛有大量 market 时，响应数据过大，影响性能

**修改后**：
- 新增 `getEventMarketsWithFilters` 函数
- 支持通过 `market_ids` 参数过滤 market
- SQL 查询优化：`WHERE sr_market_id IN ($1, $2, ...)`
- 保持向后兼容：不传 `market_ids` 时返回所有 market

---

## 测试建议

### 1. 测试 Categories API

```bash
# 测试基本查询
curl "http://localhost:8080/api/categories"

# 测试体育类型筛选
curl "http://localhost:8080/api/categories?sport_ids=sr:sport:1"

# 测试排序
curl "http://localhost:8080/api/categories?sort=match_count_desc"

# 测试分页
curl "http://localhost:8080/api/categories?page=1&page_size=10"
```

### 2. 测试 Tournaments API

```bash
# 测试基本查询（需要替换为实际的 category_id）
curl "http://localhost:8080/api/tournaments?category_id=sr:category:1"

# 测试排序
curl "http://localhost:8080/api/tournaments?category_id=sr:category:1&sort=match_count_desc"

# 测试分页
curl "http://localhost:8080/api/tournaments?category_id=sr:category:1&page=1&page_size=10"
```

### 3. 测试 Events API

```bash
# 测试返回所有 market
curl "http://localhost:8080/api/events?is_live=true"

# 测试只返回指定 market
curl "http://localhost:8080/api/events?is_live=true&market_ids=1,18"

# 测试多个 market
curl "http://localhost:8080/api/events?is_live=true&market_ids=1,16,18,226"

# 测试空 market_ids（应该返回所有 market）
curl "http://localhost:8080/api/events?is_live=true&market_ids="
```

---

## 部署说明

### 本地测试

```bash
# 1. 编译项目
cd /home/ubuntu/betradar-uof-service
go build -o uof-service

# 2. 运行服务
./uof-service
```

### Railway 部署

```bash
# 1. 提交代码到 Git
git add .
git commit -m "feat: optimize categories, tournaments, and events API"
git push origin main

# 2. Railway 会自动检测到代码变更并重新部署
# 3. 等待部署完成后测试新接口
```

---

## 向后兼容性

所有修改都保持了向后兼容性：

- ✅ `/api/categories` 和 `/api/tournaments` 的原有参数仍然有效
- ✅ `/api/events` 在不传 `market_ids` 参数时，行为与修改前一致（返回所有 market）
- ✅ 响应数据结构未发生变化，只是数据来源和查询逻辑优化

---

## 性能优化

### 查询优化

- 使用 `DISTINCT` 去重，避免重复数据
- 添加 `GROUP BY` 聚合，统计比赛数量
- 使用索引加速查询（`tracked_events` 表已有相关索引）

### Market 过滤优化

- 通过 `IN` 子句过滤 market，避免返回所有 market
- 减少数据传输量，提升响应速度
- 前端可以根据需要只请求必要的 market

---

## 常见问题

### Q1: 为什么 `/api/categories` 返回的数据为空？

**A**: 请确保数据库中的 `tracked_events` 表有数据，并且 `category_id` 字段不为空。可以通过以下 SQL 检查：

```sql
SELECT DISTINCT category_id, category_name FROM tracked_events WHERE category_id IS NOT NULL;
```

### Q2: 为什么 `/api/tournaments` 返回 400 错误？

**A**: 请确保传入了 `category_id` 参数，该参数是必填的。例如：

```bash
curl "http://localhost:8080/api/tournaments?category_id=sr:category:1"
```

### Q3: `/api/events` 的 `market_ids` 参数支持哪些格式？

**A**: 支持逗号分隔的 market ID 列表，例如：
- `market_ids=1` - 单个 market
- `market_ids=1,18` - 多个 market
- `market_ids=1,16,18,226` - 更多 market

### Q4: 如何获取所有可用的 Market ID？

**A**: 可以通过以下 SQL 查询：

```sql
SELECT DISTINCT sr_market_id, market_name FROM markets ORDER BY sr_market_id;
```

或者调用 `/api/market-descriptions/status` 接口查看所有 market 描述。

---

## 总结

本次修改优化了三个核心 API 接口，主要改进包括：

1. **数据准确性**：从 `tracked_events` 表查询数据，确保分类和联赛数据的准确性
2. **灵活性**：支持多种筛选和排序方式，满足前端不同的展示需求
3. **性能优化**：通过 `market_ids` 参数按需返回 market 数据，减少数据传输量
4. **向后兼容**：保持原有接口行为，不影响现有功能

如有任何问题或建议，请联系开发团队。
