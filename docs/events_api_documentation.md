# Betradar UOF Service - 赛事 (Events) 接口文档

> **Base URL**: `https://betradar-uof-service-production.up.railway.app/api`
> 
> 本文档详细描述了所有与赛事信息查询相关的 API 接口，包括比赛列表、详情、以及各种筛选和分页功能。

---

## 1. 综合赛事列表 (支持筛选和分页)

**接口**: `/api/events`
**方法**: `GET`
**描述**: 获取所有追踪的比赛列表，支持丰富的筛选、排序和分页功能。这是获取比赛列表最灵活的接口。

### 1.1 查询参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `page` | `int` | 1 | 否 | 页码。 |
| `page_size` | `int` | 100 | 否 | 每页记录数 (最大 500)。 |
| `is_live` | `bool` | 无 | 否 | **是否直播中**: `true` (进行中且有比分/时间), `false` (非直播), `无` (全部)。 |
| `status` | `string` | 无 | 否 | **比赛状态**: `active`, `ended`, `scheduled` 等。如果指定，则覆盖 `is_live` 的部分逻辑。 |
| `include_ended` | `bool` | `false` | 否 | **包含已结束比赛**: `true` (包含), `false` (排除)。仅在未指定 `status` 时生效。 |
| `sport_id` | `string` | 无 | 否 | **体育类型 ID**: 支持逗号分隔的多个 ID (例如: `1,2,3`)。ID 对应 `sr:sport:{ID}`。 |
| `start_time_from` | `string` | 无 | 否 | **开赛时间从**: ISO 8601 格式 (`YYYY-MM-DDTHH:MM:SSZ`) 或 `YYYY-MM-DD`。左闭区间。 |
| `start_time_to` | `string` | 无 | 否 | **开赛时间到**: ISO 8601 格式 (`YYYY-MM-DDTHH:MM:SSZ`) 或 `YYYY-MM-DD`。右闭区间。 |
| `market_id` | `string` | 无 | 否 | **盘口 ID**: 支持逗号分隔的多个 ID (例如: `1,18,26`)。ID 对应 `sr_market_id`。 |
| `team_id` | `string` | 无 | 否 | **队伍 ID**: 支持逗号分隔的多个 ID。匹配主队或客队 ID。 |
| `team_name` | `string` | 无 | 否 | **队伍名称**: 模糊搜索主队或客队名称 (不区分大小写)。 |
| `league_id` | `string` | 无 | 否 | **联赛 ID**: 支持逗号分隔的多个 ID。通过 `srn_id` 模糊匹配。 |
| `search` | `string` | 无 | 否 | **通用搜索**: 模糊搜索 `event_id`、主队名称或客队名称。 |
| `popular` | `bool` | 无 | 否 | **是否热门**: `true` (热门比赛), `false` (非热门比赛), `无` (全部)。热门定义: 焦点赛 OR 售罄 OR 转播数 > 0 OR 热门度评分 > 50。 |
| `min_popularity` | `float` | 0 | 否 | **最小热门度评分**: 0-100 之间的浮点数。 |
| `sort_by` | `string` | `time` | 否 | **排序字段**: `popularity` (按热门度降序), `time` (按开赛时间降序, 默认)。 |

### 1.2 响应示例

```json
{
  "success": true,
  "count": 50,
  "total": 120,
  "page": 1,
  "page_size": 50,
  "total_pages": 3,
  "filters": {
    "is_live": true,
    "sport_id": "1"
  },
  "matches": [
    {
      "event_id": "sr:match:12345",
      "srn_id": "SRN123456",
      "sport_id": "sr:sport:1",
      "status": "active",
      "schedule_time": "2025-10-24T10:00:00Z",
      "home_team_name": "Manchester United",
      "away_team_name": "Liverpool",
      "home_score": 1,
      "away_score": 0,
      "match_status": "40",
      "match_time": "65:23",
      "popularity_score": 85.5,
      "feature_match": true,
      "live_video_available": true,
      "live_data_available": true,
      "broadcasts_count": 3,
      "created_at": "2025-10-24T10:00:00Z",
      "updated_at": "2025-10-24T11:05:23Z"
    }
  ]
}
```

### 1.3 响应字段说明 (新增/补充)

| 字段 | 类型 | 说明 |
|------|------|------|
| `total` | `int` | 满足筛选条件的总记录数。 |
| `page` | `int` | 当前页码。 |
| `page_size` | `int` | 当前页大小。 |
| `total_pages` | `int` | 总页数。 |
| `filters` | `object` | 当前请求使用的筛选参数。 |
| `popularity_score` | `float` | 比赛热门度评分 (0-100)。 |
| `feature_match` | `bool` | 是否为焦点比赛。 |
| `live_video_available` | `bool` | 是否有直播视频。 |
| `live_data_available` | `bool` | 是否有实时数据。 |

---

## 2. 实时比赛列表

**接口**: `/api/matches/live`
**方法**: `GET`
**描述**: 获取所有正在进行的比赛列表。

### 2.1 查询参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `page` | `int` | 1 | 否 | 页码。 |
| `page_size` | `int` | 100 | 否 | 每页记录数 (最大 500)。 |

### 2.2 响应示例

与 `/api/events` 接口的 `matches` 数组结构相同，但额外包含分页信息。

```json
{
  "success": true,
  "count": 15,
  "total": 15,
  "page": 1,
  "page_size": 100,
  "total_pages": 1,
  "matches": [
    // ... MatchDetail 结构
  ]
}
```

---

## 3. 即将开始的比赛列表

**接口**: `/api/matches/upcoming`
**方法**: `GET`
**描述**: 获取未来指定时间内即将开始的比赛。

### 3.1 查询参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `hours` | `int` | 24 | 否 | 未来多少小时内的比赛。 |
| `page` | `int` | 1 | 否 | 页码。 |
| `page_size` | `int` | 100 | 否 | 每页记录数 (最大 500)。 |

### 3.2 响应示例

```json
{
  "success": true,
  "count": 8,
  "total": 8,
  "page": 1,
  "page_size": 100,
  "total_pages": 1,
  "hours": 24,
  "matches": [
    // ... MatchDetail 结构
  ]
}
```

---

## 4. 按状态筛选比赛 (简单)

**接口**: `/api/matches/status`
**方法**: `GET`
**描述**: 根据比赛状态筛选比赛列表。

### 4.1 查询参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `status` | `string` | `active` | 是 | 比赛状态: `active`, `ended`, `scheduled`。 |

### 4.2 响应示例

```json
{
  "success": true,
  "status": "active",
  "count": 10,
  "matches": [
    // ... MatchDetail 结构
  ]
}
```

---

## 5. 搜索比赛 (简单)

**接口**: `/api/matches/search`
**方法**: `GET`
**描述**: 根据关键词搜索比赛(支持球队名称、比赛 ID)。

### 5.1 查询参数

| 参数 | 类型 | 默认值 | 必填 | 说明 |
|------|------|--------|------|------|
| `q` | `string` | 无 | 是 | 搜索关键词。 |

### 5.2 响应示例

```json
{
  "success": true,
  "query": "Manchester",
  "count": 3,
  "matches": [
    // ... MatchDetail 结构
  ]
}
```

---

## 6. 获取比赛详情

**接口**: `/api/matches/{event_id}`
**方法**: `GET`
**描述**: 获取单个比赛的详细信息。

### 6.1 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `event_id` | `string` | 比赛 ID (例如: `sr:match:12345`)。 |

### 6.2 响应示例

```json
{
  "success": true,
  "match": {
    "event_id": "sr:match:12345",
    "srn_id": "SRN123456",
    "sport_id": "sr:sport:1",
    "status": "active",
    "schedule_time": "2025-10-24T10:00:00Z",
    "home_team_id": "sr:competitor:1001",
    "home_team_name": "Manchester United",
    "away_team_id": "sr:competitor:1002",
    "away_team_name": "Liverpool",
    "home_score": 1,
    "away_score": 0,
    "match_status": "40",
    "match_time": "65:23",
    "message_count": 150,
    "last_message_at": "2025-10-24T11:05:23Z",
    "popularity_score": 85.5,
    "feature_match": true
  }
}
```

---

## 附录：MatchDetail 结构 (比赛详情)

| 字段 | 类型 | 说明 |
|------|------|------|
| `event_id` | `string` | 比赛唯一标识 (SR URN 格式)。 |
| `srn_id` | `string` | SRN ID (可选)。 |
| `sport_id` | `string` | 运动类型 ID (例如: `sr:sport:1` = 足球)。 |
| `status` | `string` | 比赛状态: `active`, `ended`, `scheduled`。 |
| `schedule_time` | `string` | 比赛计划时间 (ISO 8601)。 |
| `home_team_id` | `string` | 主队 ID。 |
| `home_team_name` | `string` | 主队名称。 |
| `away_team_id` | `string` | 客队 ID。 |
| `away_team_name` | `string` | 客队名称。 |
| `home_score` | `int` | 主队比分。 |
| `away_score` | `int` | 客队比分。 |
| `match_status` | `string` | SR 比赛状态码 (例如: `20`=上半场, `40`=下半场)。 |
| `match_time` | `string` | 比赛时间 (分:秒)。 |
| `message_count` | `int` | 接收到的消息总数。 |
| `last_message_at` | `string` | 最后一条消息接收时间 (ISO 8601)。 |
| `created_at` | `string` | 记录创建时间 (ISO 8601)。 |
| `updated_at` | `string` | 记录更新时间 (ISO 8601)。 |
| `attendance` | `int` | 到场人数 (热门度相关)。 |
| `sellout` | `bool` | 是否售罄 (热门度相关)。 |
| `feature_match` | `bool` | 是否焦点赛 (热门度相关)。 |
| `live_video_available` | `bool` | 是否提供直播 (热门度相关)。 |
| `live_data_available` | `bool` | 是否提供实时数据 (热门度相关)。 |
| `broadcasts_count` | `int` | 转播平台数量 (热门度相关)。 |
| `popularity_score` | `float` | 热门度评分 (0-100)。 |
