# Betradar UOF Service - 联赛 (League) 接口文档

> **Base URL**: `https://betradar-uof-service-production.up.railway.app/api`
> 
> 本文档详细描述了所有与联赛信息查询相关的 API 接口，包括联赛列表、筛选和排序功能。

---

## 1. 获取联赛列表

**接口**: `/api/leagues`
**方法**: `GET`
**描述**: 获取所有追踪的联赛列表，支持按体育类型筛选和多种排序方式。

### 1.1 查询参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `sport_id` | `string` | 无 | 否 | **体育类型 ID**: 例如 `sr:sport:1` (足球)。如果提供，则只返回该体育类型下的联赛。 |
| `sort` | `string` | `popularity` | 否 | **排序字段**: 可选值包括: `popularity` (热门度), `name` (名称), `total_matches` (总比赛数), `live_matches` (Live 比赛数), `upcoming_matches` (即将开始的比赛数)。 |
| `order` | `string` | `desc` | 否 | **排序顺序**: 可选值包括: `asc` (升序), `desc` (降序)。 |

### 1.2 响应示例

```json
{
  "success": true,
  "count": 5,
  "sort_by": "popularity",
  "order": "desc",
  "leagues": [
    {
      "league_id": "sr:tournament:100",
      "league_name": "Premier League",
      "sport_id": "sr:sport:1",
      "category_id": "sr:category:1",
      "category_name": "England",
      "total_matches": 50,
      "live_matches": 3,
      "upcoming_matches": 10,
      "popularity": 95.5
    },
    {
      "league_id": "sr:tournament:200",
      "league_name": "La Liga",
      "sport_id": "sr:sport:1",
      "category_id": "sr:category:2",
      "category_name": "Spain",
      "total_matches": 45,
      "live_matches": 1,
      "upcoming_matches": 8,
      "popularity": 88.2
    }
    // ... 更多联赛
  ]
}
```

### 1.3 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `league_id` | `string` | 联赛唯一标识 (SR URN 格式)。 |
| `league_name` | `string` | 联赛名称 (已尝试提取英文名称)。 |
| `sport_id` | `string` | 体育类型 ID。 |
| `category_id` | `string` | 类别 ID (通常代表国家或地区)。 |
| `category_name` | `string` | 类别名称。 |
| `total_matches` | `int` | 当前追踪的总比赛数。 |
| `live_matches` | `int` | 当前正在进行的比赛数。 |
| `upcoming_matches` | `int` | 即将开始的比赛数。 |
| `popularity` | `float` | 联赛热门度评分 (0-100)，基于 Live 比赛数、总比赛数和即将开始的比赛数计算。 |

### 1.4 前端使用示例

```javascript
// 1. 获取所有联赛，按热门度降序排列
async function fetchAllLeagues() {
  const response = await fetch('/api/leagues');
  const data = await response.json();
  return data.leagues;
}

// 2. 获取足球 (sr:sport:1) 联赛，按 Live 比赛数升序排列
async function fetchFootballLeaguesByLiveCount() {
  const params = new URLSearchParams({
    sport_id: 'sr:sport:1',
    sort: 'live_matches',
    order: 'asc'
  });
  const response = await fetch(`/api/leagues?${params.toString()}`);
  const data = await response.json();
  return data.leagues;
}
```
