# Betradar UOF Service - /api/events 接口返回结果 Market Group 分析

> **Base URL**: `https://betradar-uof-service-production.up.railway.app/api`
> 
> 本文档分析了 `/api/events` 接口返回的比赛数据结构，以确定其中是否包含盘口组（Market Group）信息。

---

## 1. 结论

`/api/events` 接口返回的比赛列表（`matches` 数组中的每个元素）**不包含** Market Group 信息。

### 1.1 详细分析

`/api/events` 接口返回的比赛数据结构基于 `MatchDetail` 和 `EnhancedMatchDetail` 结构体。

#### 1.1.1 `MatchDetail` 结构体 (原始数据库字段)

该结构体主要包含比赛的基本信息、比分、状态和热门度相关字段，**不包含**任何盘口或盘口组信息。

```go
type MatchDetail struct {
    EventID        string    `json:"event_id"`
    // ... 比赛基本信息
    HomeScore      *int      `json:"home_score"`
    AwayScore      *int      `json:"away_score"`
    // ... 状态和时间
    // ... 热门度相关字段 (Attendance, Sellout, PopularityScore 等)
}
```

#### 1.1.2 `EnhancedMatchDetail` 结构体 (增强后的返回字段)

该结构体在 `MatchDetail` 的基础上，通过 `MapMatchDetail` 函数添加了映射后的字段（如 `SportName`, `MatchStatusMapped` 等），以方便前端使用。

```go
type EnhancedMatchDetail struct {
    // ... 原始字段
    // ... 映射后的字段
    Sport              string `json:"sport"`
    MatchStatusMapped  string `json:"match_status_mapped"`
    // ...
}
```

通过对 `EnhancedMatchDetail` 结构体和 `MapMatchDetail` 函数的分析，可以确认：

- **比赛详情中不包含 Market Group 字段。**
- `/api/events` 接口返回的是比赛的**元数据**和**状态**，它**不包含**该比赛的任何盘口（Market）或赔率（Odds）信息。

### 1.2 如何获取 Market Group 信息

如果前端需要获取比赛的盘口信息（包括 Market Group），则需要调用专门的盘口 API。

虽然 `/api/events` 接口的返回结果中不包含 Market Group，但项目中的 `/api/odds/all` 接口返回的结构中包含详细的盘口信息，其中可能包含 Market Group 的相关信息（例如，通过盘口 ID 或类型进行分组）。

**推荐做法：**

1.  **获取比赛列表**: 调用 `/api/events` 获取比赛 ID 列表。
2.  **获取盘口信息**: 调用 `/api/odds/{event_id}/markets` 获取单个比赛的盘口列表，然后前端可以根据盘口 ID 或类型自行进行分组（Group）。

---

## 2. 补充：`/api/events` 接口返回结构概览

`/api/events` 接口的返回结构如下：

```json
{
  "success": true,
  "count": 50,
  "total": 100,
  "page": 1,
  "page_size": 50,
  "total_pages": 2,
  "filters": {
    // ... 筛选参数的反射
  },
  "matches": [
    {
      "event_id": "sr:match:12345",
      "sport_id": "sr:sport:1",
      "status": "active",
      "home_team_name": "Manchester United",
      "away_team_name": "Liverpool",
      "is_live": true,
      "sport": "football",
      // ... 其他比赛元数据
    }
    // ...
  ]
}
```

**总结**: `/api/events` 接口返回的比赛对象中，**不包含** Market Group 字段。Market Group 相关的逻辑仅存在于筛选参数（且目前未实现）和专门的盘口 API 中。
