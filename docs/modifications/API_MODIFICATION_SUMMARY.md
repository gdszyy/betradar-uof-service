# GET /api/events/{eventId} 接口修改总结

## 概述

本文档详细说明了对 `betradar-uof-service` 项目的修改，为 `GET /api/events/{eventId}` 接口添加了完整的赛事详情、市场信息、specifier分组、TabID/ChipID和剩余specifier字典的支持。

## 修改内容

### 1. 新增文件：`web/event_detail_handler.go`

这个文件包含了新接口的完整实现，主要包括以下内容：

#### 数据结构定义

##### EventDetailResponse
赛事详情响应的顶级结构，包含：
- `event_id`: 赛事ID
- `srn_id`: SRN ID
- `sport_id`: 运动类型ID
- `status`: 赛事状态
- `schedule_time`: 计划时间
- `home_team_id`: 主队ID
- `home_team_name`: 主队名称
- `away_team_id`: 客队ID
- `away_team_name`: 客队名称
- `home_score`: 主队比分
- `away_score`: 客队比分
- `match_status`: 比赛状态
- `match_time`: 比赛时间
- `created_at`: 创建时间
- `updated_at`: 更新时间
- `markets`: 市场数组（按market_id分组）

##### MarketWithSpecifiersGroup
按market分组的盘口信息，包含：
- `market_id`: 市场ID
- `market_name`: 市场名称
- `market_type`: 市场类型
- `status`: 市场状态
- `producer_id`: 生产者ID
- `specifier_groups`: 按specifier分组的市场数组

##### SpecifierMarketGroup
按specifier分组的市场信息，包含：
- `specifier`: Specifier字符串（例如：`"periodnr=1|total=2.5"`）
- `specifier_dict`: Specifier字典（例如：`{"periodnr": "1", "total": "2.5"}`）
- `tab_id`: TabID（可选）
- `chip_id`: ChipID（可选）
- `remaining_specifiers`: 去除TabID/ChipID关联specifier后的剩余specifier字典
- `outcomes`: Outcome数组
- `updated_at`: 更新时间

##### OutcomeWithTabChip
Outcome信息，包含：
- `outcome_id`: Outcome ID
- `outcome_name`: Outcome名称
- `odds`: 赔率
- `active`: 是否活跃

#### 核心函数

##### handleGetEventDetail(w http.ResponseWriter, r *http.Request)
主处理函数，处理 `GET /api/events/{eventId}` 请求。

**流程：**
1. 从URL路径中提取eventId
2. 验证eventId不为空
3. 从数据库获取赛事基本信息
4. 获取赛事的所有市场及其specifier分组
5. 返回完整的JSON响应

**参数：**
- `eventId`: 赛事ID（例如：`sr:match:12345678`）

**返回：**
- 完整的赛事详情JSON

##### getEventDetailFromDB(eventId string) (*EventDetailResponse, error)
从数据库的 `tracked_events` 表中获取赛事基本信息。

**SQL查询：**
```sql
SELECT 
    event_id, srn_id, sport_id, status, schedule_time,
    home_team_id, home_team_name, away_team_id, away_team_name,
    home_score, away_score, match_status, match_time,
    created_at, updated_at
FROM tracked_events
WHERE event_id = $1
```

##### getEventMarketsWithSpecifiers(eventID string) ([]MarketWithSpecifiersGroup, error)
获取赛事的所有市场及其specifier分组。

**流程：**
1. 查询 `markets` 表获取所有市场
2. 对每个市场：
   - 获取其outcomes
   - 获取TabID和ChipID
   - 解析specifier为字典
   - 计算剩余specifier
3. 按market_id分组返回

**SQL查询：**
```sql
SELECT 
    m.id, m.sr_market_id, m.market_name, m.market_type, m.status,
    m.producer_id, m.specifiers, m.updated_at
FROM markets m
WHERE m.event_id = $1
ORDER BY m.sr_market_id, m.specifiers
```

##### getMarketOutcomes(marketID int) ([]OutcomeWithTabChip, error)
获取市场的所有outcomes。

**SQL查询：**
```sql
SELECT 
    outcome_id, outcome_name, odds_value, active
FROM odds
WHERE market_id = $1
ORDER BY outcome_id
```

##### getMarketTabChip(marketID int, eventID string) (*string, *string, error)
获取市场的TabID和ChipID。

**SQL查询：**
```sql
SELECT tab_id, chip_id
FROM markets
WHERE id = $1 AND event_id = $2
```

##### parseSpecifiers(specifiersStr string) map[string]string
将specifier字符串解析为字典。

**示例：**
- 输入：`"periodnr=1|total=2.5"`
- 输出：`{"periodnr": "1", "total": "2.5"}`

##### getRemainingSpecifiers(specifierDict map[string]string, tabID *string, chipID *string) map[string]string
获取去除TabID/ChipID关联specifier后的剩余specifier。

**逻辑：**
1. 复制原specifier字典
2. 根据TabID从字典中删除关联的specifier
3. 根据ChipID从字典中删除关联的specifier
4. 返回剩余的specifier字典

**TabID和ChipID关联的specifier映射：**
```
TabID关联：
- "innings" -> ["inningnr"]
- "sets" -> ["setnr"]
- "maps" -> ["mapnr"]
- "quarters" -> ["quarternr"]
- "periods" -> ["periodnr"]
- "frames" -> ["framenr"]
- "overs" -> ["overnr"]
- "drives" -> ["drivenr"]
- "1st_half" -> ["goalnr"]
- "2nd_half" -> ["goalnr"]
- "corners" -> ["cornernr"]

ChipID关联：
- "innings" -> ["inningnr"]
- "sets" -> ["setnr"]
- "maps" -> ["mapnr"]
- "quarters" -> ["quarternr"]
- "periods" -> ["periodnr"]
- "frames" -> ["framenr"]
- "overs" -> ["overnr"]
- "drives" -> ["drivenr"]
```

##### extractEventIDFromPath(path string, prefix string) string
从URL路径中提取eventId。

**示例：**
- 输入：`path="/api/events/sr:match:12345678"`, `prefix="/api/events/"`
- 输出：`"sr:match:12345678"`

### 2. 修改文件：`web/server.go`

在路由注册部分添加了新的路由：

```go
// 赛事详情API - 包含所有市场、specifier和outcomes
api.HandleFunc("/events/{eventId}", s.handleGetEventDetail).Methods("GET")
```

这条路由被添加在第172行，位于市场查询API之后。

## API使用说明

### 请求

**方法：** GET

**URL：** `/api/events/{eventId}`

**参数：**
- `eventId` (路径参数，必需): 赛事ID，格式为 `sr:match:数字` 或 `sr:stage:数字`

**示例：**
```bash
curl -X GET "https://betradar-uof-service-production.up.railway.app/api/events/sr:match:12345678"
```

### 响应

**状态码：**
- `200 OK`: 成功
- `400 Bad Request`: 缺少或无效的eventId
- `404 Not Found`: 赛事不存在
- `500 Internal Server Error`: 服务器错误

**响应格式：** JSON

**响应示例：**
```json
{
  "event_id": "sr:match:12345678",
  "srn_id": "sr:match:12345678",
  "sport_id": "sr:sport:1",
  "status": "live",
  "schedule_time": "2025-12-04T10:00:00Z",
  "home_team_id": "sr:competitor:1",
  "home_team_name": "Team A",
  "away_team_id": "sr:competitor:2",
  "away_team_name": "Team B",
  "home_score": 2,
  "away_score": 1,
  "match_status": "live",
  "match_time": "45:30",
  "created_at": "2025-12-04T09:00:00Z",
  "updated_at": "2025-12-04T10:15:00Z",
  "markets": [
    {
      "market_id": "1",
      "market_name": "Match Winner",
      "market_type": "regular",
      "status": "active",
      "producer_id": 1,
      "specifier_groups": [
        {
          "specifier": "periodnr=1|total=2.5",
          "specifier_dict": {
            "periodnr": "1",
            "total": "2.5"
          },
          "tab_id": "periods",
          "chip_id": "1",
          "remaining_specifiers": {
            "total": "2.5"
          },
          "outcomes": [
            {
              "outcome_id": "1",
              "outcome_name": "Home Win",
              "odds": 1.95,
              "active": true
            },
            {
              "outcome_id": "2",
              "outcome_name": "Draw",
              "odds": 3.50,
              "active": true
            },
            {
              "outcome_id": "3",
              "outcome_name": "Away Win",
              "odds": 4.20,
              "active": true
            }
          ],
          "updated_at": "2025-12-04T10:15:00Z"
        }
      ]
    }
  ]
}
```

## 数据库要求

该接口依赖于以下数据库表：

### tracked_events 表
存储赛事基本信息
```sql
CREATE TABLE tracked_events (
    event_id VARCHAR(255) PRIMARY KEY,
    srn_id VARCHAR(255),
    sport_id VARCHAR(255),
    status VARCHAR(50),
    schedule_time TIMESTAMP,
    home_team_id VARCHAR(255),
    home_team_name VARCHAR(255),
    away_team_id VARCHAR(255),
    away_team_name VARCHAR(255),
    home_score INT,
    away_score INT,
    match_status VARCHAR(50),
    match_time VARCHAR(50),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### markets 表
存储市场信息
```sql
CREATE TABLE markets (
    id INT PRIMARY KEY,
    event_id VARCHAR(255),
    sr_market_id VARCHAR(255),
    market_name VARCHAR(255),
    market_type VARCHAR(50),
    status VARCHAR(50),
    producer_id INT,
    specifiers TEXT,
    tab_id VARCHAR(255),
    chip_id VARCHAR(255),
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES tracked_events(event_id)
);
```

### odds 表
存储outcome信息
```sql
CREATE TABLE odds (
    id INT PRIMARY KEY,
    market_id INT,
    outcome_id VARCHAR(255),
    outcome_name VARCHAR(255),
    odds_value DECIMAL(10, 2),
    active BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    FOREIGN KEY (market_id) REFERENCES markets(id)
);
```

## 部署步骤

1. **备份现有代码**
   ```bash
   git commit -m "Backup before adding event detail API"
   ```

2. **应用修改**
   - 将 `web/event_detail_handler.go` 添加到项目
   - 修改 `web/server.go` 添加新路由

3. **编译和测试**
   ```bash
   go build -o betradar-uof-service
   ```

4. **部署到Railway**
   ```bash
   git push origin main
   ```
   Railway会自动检测到代码变更并重新部署。

5. **验证部署**
   ```bash
   curl -X GET "https://betradar-uof-service-production.up.railway.app/api/events/sr:match:12345678"
   ```

## 性能考虑

1. **数据库查询优化**
   - 使用了索引查询（event_id, sr_market_id）
   - 避免了N+1查询问题（在单个循环中获取所有相关数据）

2. **缓存建议**
   - 可以考虑为频繁访问的eventId添加缓存
   - 建议缓存时间：5-10分钟

3. **分页建议**
   - 如果市场数量很多，可以考虑添加分页参数
   - 建议每页返回最多100个市场

## 错误处理

接口包含以下错误处理：

1. **缺少eventId**
   - 状态码：400
   - 消息：`"Missing required parameter: eventId"`

2. **无效的eventId格式**
   - 状态码：400
   - 消息：`"Invalid eventId format"`

3. **赛事不存在**
   - 状态码：404
   - 消息：`"Event not found"`

4. **数据库错误**
   - 状态码：500
   - 消息：`"Internal server error"`

## 扩展功能建议

1. **添加过滤参数**
   - 按市场类型过滤
   - 按状态过滤
   - 按specifier值过滤

2. **添加排序参数**
   - 按market_id排序
   - 按更新时间排序
   - 按市场名称排序

3. **添加分页**
   - `page` 参数
   - `page_size` 参数

4. **添加缓存**
   - 使用Redis缓存热点数据
   - 设置合理的过期时间

5. **添加性能监控**
   - 记录查询时间
   - 监控数据库连接
   - 记录错误日志

## 常见问题

### Q: 为什么specifier_dict和specifier_groups都需要？
**A:** 
- `specifier`: 原始的specifier字符串，用于数据库查询和缓存
- `specifier_dict`: 解析后的字典，便于前端使用
- `specifier_groups`: 按specifier分组，便于按specifier展示数据

### Q: remaining_specifiers的用途是什么？
**A:** 
- 用于显示该市场除了TabID/ChipID关联的specifier外，还有哪些其他specifier
- 例如，如果TabID="periods"，则会去除"periodnr"，剩余的specifier（如"total"）会显示在remaining_specifiers中

### Q: 如何判断一个市场是否有TabID或ChipID？
**A:** 
- 检查 `tab_id` 和 `chip_id` 字段是否为null
- 如果为null，则表示该市场没有分配TabID或ChipID

## 相关文档

- [SportRader UOF文档 - Market Description](https://docs.sportradar.com/uof/data-and-features/markets-and-outcomes/market-description)
- [SportRader UOF文档 - Specifiers](https://docs.sportradar.com/uof/introduction/key-concepts/specifiers)
- [项目README](./README.md)

## 版本历史

- **v1.0** (2025-12-04): 初始实现
  - 添加了GET /api/events/{eventId}接口
  - 支持marketName、specifier分组、outcomeName、TabID和ChipID
  - 支持剩余specifier字典

## 联系方式

如有问题或建议，请提交Issue或Pull Request。
