# UOF Service API Documentation

**Version**: 2.0  
**Base URL**: `https://your-service.railway.app/api`  
**Last Updated**: 2025-10-24

---

## 目录

1. [认证](#认证)
2. [比赛查询 API](#比赛查询-api)
3. [订阅管理 API](#订阅管理-api)
4. [消息查询 API](#消息查询-api)
5. [监控 API](#监控-api)
6. [WebSocket API](#websocket-api)
7. [错误处理](#错误处理)
8. [示例代码](#示例代码)

---

## 认证

当前版本的 API **不需要认证**。所有端点都是公开的。

> ⚠️ **注意**: 生产环境建议添加认证机制。

---

## 比赛查询 API

### 1. 获取所有追踪的比赛

获取数据库中所有追踪的比赛列表。

**端点**: `GET /api/events`

**查询参数**:
- `limit` (可选): 返回数量限制,默认 50,最大 100
- `offset` (可选): 偏移量,默认 0

**响应示例**:
```json
{
  "success": true,
  "count": 25,
  "events": [
    {
      "event_id": "sr:match:12345",
      "srn_id": "srn:match:67890",
      "sport_id": "sr:sport:1",
      "status": "active",
      "schedule_time": "2025-10-24T15:00:00Z",
      "home_team_id": "sr:competitor:1001",
      "home_team_name": "Manchester United",
      "away_team_id": "sr:competitor:1002",
      "away_team_name": "Liverpool",
      "home_score": 1,
      "away_score": 0,
      "match_status": "6",
      "match_time": "23:15",
      "message_count": 45,
      "last_message_at": "2025-10-24T15:23:15Z",
      "created_at": "2025-10-24T14:00:00Z",
      "updated_at": "2025-10-24T15:23:15Z"
    }
  ]
}
```

---

### 2. 获取进行中的比赛

获取所有状态为 active 且有比分数据的比赛。

**端点**: `GET /api/matches/live`

**响应示例**:
```json
{
  "success": true,
  "count": 15,
  "matches": [
    {
      "event_id": "sr:match:12345",
      "home_team_name": "Manchester United",
      "away_team_name": "Liverpool",
      "home_score": 1,
      "away_score": 0,
      "match_status": "6",
      "match_time": "23:15",
      "schedule_time": "2025-10-24T15:00:00Z"
    }
  ]
}
```

---

### 3. 获取即将开始的比赛

获取未来指定小时内将要开始的比赛。

**端点**: `GET /api/matches/upcoming`

**查询参数**:
- `hours` (可选): 时间范围(小时),默认 24

**响应示例**:
```json
{
  "success": true,
  "count": 8,
  "hours": 24,
  "matches": [
    {
      "event_id": "sr:match:12346",
      "home_team_name": "Chelsea",
      "away_team_name": "Arsenal",
      "schedule_time": "2025-10-24T18:00:00Z",
      "status": "active"
    }
  ]
}
```

---

### 4. 按状态获取比赛

根据比赛状态筛选比赛。

**端点**: `GET /api/matches/status`

**查询参数**:
- `status` (必需): 比赛状态
  - `active` - 进行中
  - `ended` - 已结束
  - `scheduled` - 已安排

**响应示例**:
```json
{
  "success": true,
  "status": "active",
  "count": 12,
  "matches": [...]
}
```

---

### 5. 搜索比赛

根据关键词搜索比赛(支持球队名称和赛事 ID)。

**端点**: `GET /api/matches/search`

**查询参数**:
- `q` (必需): 搜索关键词

**响应示例**:
```json
{
  "success": true,
  "keyword": "Manchester",
  "count": 3,
  "matches": [
    {
      "event_id": "sr:match:12345",
      "home_team_name": "Manchester United",
      "away_team_name": "Liverpool",
      "home_score": 1,
      "away_score": 0
    }
  ]
}
```

---

### 6. 获取比赛详情

获取单个比赛的完整详细信息。

**端点**: `GET /api/matches/{event_id}`

**路径参数**:
- `event_id`: 赛事 ID (例如: `sr:match:12345`)

**响应示例**:
```json
{
  "success": true,
  "match": {
    "event_id": "sr:match:12345",
    "srn_id": "srn:match:67890",
    "sport_id": "sr:sport:1",
    "status": "active",
    "schedule_time": "2025-10-24T15:00:00Z",
    "home_team_id": "sr:competitor:1001",
    "home_team_name": "Manchester United",
    "away_team_id": "sr:competitor:1002",
    "away_team_name": "Liverpool",
    "home_score": 1,
    "away_score": 0,
    "match_status": "6",
    "match_time": "23:15",
    "message_count": 45,
    "last_message_at": "2025-10-24T15:23:15Z",
    "created_at": "2025-10-24T14:00:00Z",
    "updated_at": "2025-10-24T15:23:15Z"
  }
}
```

---

## 订阅管理 API

### 1. 获取已订阅的比赛

查询 Betradar 已订阅的比赛列表。

**端点**: `GET /api/booking/booked`

**响应示例**:
```json
{
  "success": true,
  "count": 20,
  "matches": [
    {
      "id": "sr:match:12345",
      "scheduled": "2025-10-24T15:00:00Z",
      "status": "not_started",
      "liveodds": "booked"
    }
  ]
}
```

---

### 2. 获取可订阅的比赛

查询当前可以订阅的比赛列表。

**端点**: `GET /api/booking/bookable`

**响应示例**:
```json
{
  "success": true,
  "total_live": 50,
  "bookable_count": 15,
  "matches": [
    {
      "id": "sr:match:12346",
      "scheduled": "2025-10-24T18:00:00Z",
      "status": "not_started",
      "liveodds": "bookable"
    }
  ]
}
```

---

### 3. 触发自动订阅

手动触发自动订阅流程,订阅所有可订阅的比赛。

**端点**: `POST /api/booking/trigger`

**响应示例**:
```json
{
  "success": true,
  "total_live": 50,
  "bookable": 15,
  "success_count": 14,
  "failed_count": 1,
  "verified_count": 14,
  "booked_matches": [
    "sr:match:12345",
    "sr:match:12346"
  ],
  "failed_matches": {
    "sr:match:12347": "status 400: Already booked"
  }
}
```

---

### 4. 订阅单个比赛

订阅指定的比赛。

**端点**: `POST /api/booking/match/{match_id}`

**路径参数**:
- `match_id`: 比赛 ID (例如: `sr:match:12345`)

**响应示例**:
```json
{
  "success": true,
  "message": "Match booked successfully",
  "match_id": "sr:match:12345"
}
```

---

## 消息查询 API

### 1. 获取消息列表

获取接收到的 UOF 消息列表。

**端点**: `GET /api/messages`

**查询参数**:
- `limit` (可选): 返回数量限制,默认 50,最大 100
- `offset` (可选): 偏移量,默认 0
- `type` (可选): 消息类型筛选

**响应示例**:
```json
{
  "success": true,
  "count": 50,
  "messages": [
    {
      "id": 12345,
      "message_type": "odds_change",
      "event_id": "sr:match:12345",
      "product_id": 1,
      "timestamp": 1729666800000,
      "received_at": "2025-10-24T15:23:15Z"
    }
  ]
}
```

---

### 2. 获取比赛的消息历史

获取指定比赛的所有消息。

**端点**: `GET /api/events/{event_id}/messages`

**路径参数**:
- `event_id`: 赛事 ID

**响应示例**:
```json
{
  "success": true,
  "event_id": "sr:match:12345",
  "count": 45,
  "messages": [
    {
      "id": 12345,
      "message_type": "odds_change",
      "timestamp": 1729666800000,
      "received_at": "2025-10-24T15:23:15Z"
    }
  ]
}
```

---

## 监控 API

### 1. 健康检查

检查服务是否正常运行。

**端点**: `GET /api/health`

**响应示例**:
```json
{
  "status": "ok",
  "time": 1729666800
}
```

---

### 2. 获取统计信息

获取服务的统计数据。

**端点**: `GET /api/stats`

**响应示例**:
```json
{
  "success": true,
  "stats": {
    "total_events": 150,
    "active_events": 25,
    "total_messages": 5420,
    "messages_by_type": {
      "odds_change": 3200,
      "bet_stop": 850,
      "bet_settlement": 920,
      "fixture": 450
    }
  }
}
```

---

### 3. 触发比赛监控

手动触发比赛监控,检查已订阅的比赛状态。

**端点**: `POST /api/monitor/trigger`

**响应示例**:
```json
{
  "success": true,
  "message": "Monitor triggered successfully"
}
```

---

### 4. 获取服务器 IP

获取服务器的公网 IP 地址。

**端点**: `GET /api/ip`

**响应示例**:
```json
{
  "success": true,
  "ip": "123.45.67.89",
  "source": "ipify"
}
```

---

## WebSocket API

### 连接

**端点**: `ws://your-service.railway.app/ws`

### 消息格式

WebSocket 会实时推送 UOF 消息,格式为 JSON:

```json
{
  "type": "odds_change",
  "event_id": "sr:match:12345",
  "product_id": 1,
  "timestamp": 1729666800000,
  "data": {
    "home_score": 1,
    "away_score": 0,
    "match_status": "6",
    "match_time": "23:15"
  },
  "xml": "<odds_change>...</odds_change>"
}
```

### 消息类型

- `odds_change` - 赔率变化
- `bet_stop` - 投注停止
- `bet_settlement` - 投注结算
- `fixture` - 赛程信息
- `fixture_change` - 赛程变更

### 连接示例

```javascript
const ws = new WebSocket('ws://your-service.railway.app/ws');

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
  
  if (message.type === 'odds_change') {
    // 更新赔率显示
    updateOdds(message.event_id, message.data);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket disconnected');
  // 5 秒后重连
  setTimeout(() => connectWebSocket(), 5000);
};
```

---

## 错误处理

### 错误响应格式

```json
{
  "success": false,
  "error": "Error message here"
}
```

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 示例代码

### JavaScript/TypeScript

```typescript
// 获取进行中的比赛
async function getLiveMatches() {
  const response = await fetch('/api/matches/live');
  const data = await response.json();
  
  if (data.success) {
    return data.matches;
  } else {
    throw new Error(data.error);
  }
}

// 搜索比赛
async function searchMatches(keyword: string) {
  const response = await fetch(`/api/matches/search?q=${encodeURIComponent(keyword)}`);
  const data = await response.json();
  return data.matches;
}

// 获取比赛详情
async function getMatchDetail(eventId: string) {
  const response = await fetch(`/api/matches/${eventId}`);
  const data = await response.json();
  return data.match;
}

// 触发自动订阅
async function triggerAutoBooking() {
  const response = await fetch('/api/booking/trigger', {
    method: 'POST',
  });
  const data = await response.json();
  return data;
}
```

---

### Python

```python
import requests

BASE_URL = "https://your-service.railway.app/api"

# 获取进行中的比赛
def get_live_matches():
    response = requests.get(f"{BASE_URL}/matches/live")
    data = response.json()
    
    if data['success']:
        return data['matches']
    else:
        raise Exception(data.get('error', 'Unknown error'))

# 搜索比赛
def search_matches(keyword):
    response = requests.get(f"{BASE_URL}/matches/search", params={'q': keyword})
    data = response.json()
    return data['matches']

# 获取比赛详情
def get_match_detail(event_id):
    response = requests.get(f"{BASE_URL}/matches/{event_id}")
    data = response.json()
    return data['match']

# 触发自动订阅
def trigger_auto_booking():
    response = requests.post(f"{BASE_URL}/booking/trigger")
    data = response.json()
    return data
```

---

### cURL

```bash
# 获取进行中的比赛
curl https://your-service.railway.app/api/matches/live

# 搜索比赛
curl "https://your-service.railway.app/api/matches/search?q=Manchester"

# 获取比赛详情
curl https://your-service.railway.app/api/matches/sr:match:12345

# 触发自动订阅
curl -X POST https://your-service.railway.app/api/booking/trigger

# 获取已订阅的比赛
curl https://your-service.railway.app/api/booking/booked
```

---

## 完整 API 端点列表

### 比赛查询

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/events` | 获取所有追踪的比赛 |
| GET | `/api/matches/live` | 获取进行中的比赛 |
| GET | `/api/matches/upcoming` | 获取即将开始的比赛 |
| GET | `/api/matches/status` | 按状态获取比赛 |
| GET | `/api/matches/search` | 搜索比赛 |
| GET | `/api/matches/{event_id}` | 获取比赛详情 |

### 订阅管理

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/booking/booked` | 获取已订阅的比赛 |
| GET | `/api/booking/bookable` | 获取可订阅的比赛 |
| POST | `/api/booking/trigger` | 触发自动订阅 |
| POST | `/api/booking/match/{match_id}` | 订阅单个比赛 |

### 消息查询

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/messages` | 获取消息列表 |
| GET | `/api/events/{event_id}/messages` | 获取比赛的消息历史 |

### 监控

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/stats` | 获取统计信息 |
| POST | `/api/monitor/trigger` | 触发比赛监控 |
| GET | `/api/ip` | 获取服务器 IP |

### WebSocket

| 端点 | 描述 |
|------|------|
| `/ws` | WebSocket 连接 |

---

## 更新日志

### v2.0 (2025-10-24)
- ✅ 添加启动时自动订阅功能
- ✅ 添加订阅验证和查询接口
- ✅ 添加前端比赛查询 API
- ✅ 修复 XML 解析 bug
- ✅ 移除 LD 和 TheSports 模块

### v1.0 (2025-10-16)
- ✅ 初始版本发布
- ✅ UOF AMQP 消费者
- ✅ WebSocket 实时推送
- ✅ 基础 API 端点

---

## 支持

如有问题或建议,请:
1. 查看 [FRONTEND_DATA_FLOW.md](./FRONTEND_DATA_FLOW.md) 了解前端集成
2. 查看 [DATABASE_MIGRATION_GUIDE.md](./DATABASE_MIGRATION_GUIDE.md) 了解数据库迁移
3. 提交 GitHub Issue

---

**文档版本**: 2.0  
**最后更新**: 2025-10-24  
**维护者**: UOF Service Team

