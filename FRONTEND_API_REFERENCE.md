# 前端 API 接口文档

> **Betradar UOF Service - 前端开发者参考手册**
> 
> 版本: v2.0  
> 更新时间: 2025-10-24  
> Base URL: `https://your-service.railway.app/api`

---

## 📋 目录

- [认证](#认证)
- [比赛查询 API](#比赛查询-api)
- [盘口赔率 API](#盘口赔率-api)
- [订阅管理 API](#订阅管理-api)
- [消息查询 API](#消息查询-api)
- [系统监控 API](#系统监控-api)
- [WebSocket 实时推送](#websocket-实时推送)
- [错误处理](#错误处理)
- [前端集成示例](#前端集成示例)

---

## 认证

当前版本 **无需认证**,所有 API 端点都是公开的。

未来版本可能会添加 API Key 或 JWT 认证。

---

## 比赛查询 API

### 1. 获取进行中的比赛

获取所有正在进行的比赛列表。

```http
GET /api/matches/live
```

#### 响应示例

```json
{
  "success": true,
  "count": 15,
  "matches": [
    {
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
      "created_at": "2025-10-24T10:00:00Z",
      "updated_at": "2025-10-24T11:05:23Z"
    }
  ]
}
```

#### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `event_id` | string | 比赛唯一标识 (SR URN 格式) |
| `srn_id` | string | SRN ID (可选) |
| `sport_id` | string | 运动类型 ID (sr:sport:1 = 足球) |
| `status` | string | 比赛状态: `active`, `ended`, `scheduled` |
| `schedule_time` | string | 比赛计划时间 (ISO 8601) |
| `home_team_name` | string | 主队名称 |
| `away_team_name` | string | 客队名称 |
| `home_score` | int | 主队比分 |
| `away_score` | int | 客队比分 |
| `match_status` | string | SR 比赛状态码 (20=上半场, 40=下半场, 100=点球) |
| `match_time` | string | 比赛时间 (分:秒) |

#### 前端使用

```javascript
async function fetchLiveMatches() {
  const response = await fetch('/api/matches/live');
  const data = await response.json();
  
  if (data.success) {
    return data.matches;
  }
  throw new Error('Failed to fetch live matches');
}
```

---

### 2. 获取即将开始的比赛

获取未来指定时间内即将开始的比赛。

```http
GET /api/matches/upcoming?hours=24
```

#### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `hours` | int | 24 | 未来多少小时内的比赛 |

#### 响应示例

```json
{
  "success": true,
  "count": 8,
  "hours": 24,
  "matches": [
    {
      "event_id": "sr:match:67890",
      "sport_id": "sr:sport:1",
      "status": "scheduled",
      "schedule_time": "2025-10-24T15:00:00Z",
      "home_team_name": "Arsenal",
      "away_team_name": "Chelsea",
      "home_score": null,
      "away_score": null,
      "match_status": "0",
      "match_time": null
    }
  ]
}
```

#### 前端使用

```javascript
// 获取未来 12 小时的比赛
async function fetchUpcomingMatches(hours = 12) {
  const response = await fetch(`/api/matches/upcoming?hours=${hours}`);
  const data = await response.json();
  return data.matches;
}
```

---

### 3. 按状态筛选比赛

根据比赛状态筛选比赛列表。

```http
GET /api/matches/status?status=active
```

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `status` | string | 是 | 比赛状态: `active`, `ended`, `scheduled` |

#### 响应示例

```json
{
  "success": true,
  "status": "active",
  "count": 10,
  "matches": [...]
}
```

#### 前端使用

```javascript
// 获取已结束的比赛
async function fetchEndedMatches() {
  const response = await fetch('/api/matches/status?status=ended');
  const data = await response.json();
  return data.matches;
}
```

---

### 4. 搜索比赛

根据关键词搜索比赛(支持球队名称、比赛 ID)。

```http
GET /api/matches/search?q=Manchester
```

#### 查询参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `q` | string | 是 | 搜索关键词 |

#### 响应示例

```json
{
  "success": true,
  "query": "Manchester",
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

#### 前端使用

```javascript
// 搜索比赛
async function searchMatches(keyword) {
  const response = await fetch(`/api/matches/search?q=${encodeURIComponent(keyword)}`);
  const data = await response.json();
  return data.matches;
}
```

---

### 5. 获取比赛详情

获取单个比赛的详细信息。

```http
GET /api/matches/{event_id}
```

#### 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `event_id` | string | 比赛 ID (例如: sr:match:12345) |

#### 响应示例

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
    "last_message_at": "2025-10-24T11:05:23Z"
  }
}
```

#### 前端使用

```javascript
async function fetchMatchDetail(eventId) {
  const response = await fetch(`/api/matches/${eventId}`);
  const data = await response.json();
  
  if (data.success) {
    return data.match;
  }
  throw new Error('Match not found');
}
```

---

### 6. 获取所有追踪的比赛

获取所有正在追踪的比赛(包括进行中、已结束、即将开始)。

```http
GET /api/events
```

#### 响应示例

```json
{
  "success": true,
  "count": 50,
  "events": [...]
}
```

---

## 盘口赔率 API

### 1. 获取所有已订阅比赛的盘口和赔率

获取所有进行中比赛的完整盘口和赔率数据。

```http
GET /api/odds/all
```

#### 响应示例

```json
{
  "success": true,
  "count": 10,
  "events": [
    {
      "event_id": "sr:match:12345",
      "markets_count": 5,
      "markets": [
        {
          "market_id": "1",
          "market_type": "1x2",
          "market_name": "胜平负",
          "specifiers": "",
          "status": "active",
          "odds": [
            {
              "outcome_id": "1",
              "outcome_name": "Home",
              "odds_value": 2.50,
              "probability": 0.4000,
              "active": true,
              "timestamp": 1698765432000,
              "updated_at": "2025-10-24T11:05:23Z"
            },
            {
              "outcome_id": "X",
              "outcome_name": "Draw",
              "odds_value": 3.20,
              "probability": 0.3125,
              "active": true,
              "timestamp": 1698765432000,
              "updated_at": "2025-10-24T11:05:23Z"
            },
            {
              "outcome_id": "2",
              "outcome_name": "Away",
              "odds_value": 3.00,
              "probability": 0.3333,
              "active": true,
              "timestamp": 1698765432000,
              "updated_at": "2025-10-24T11:05:23Z"
            }
          ],
          "updated_at": "2025-10-24T11:05:23Z"
        },
        {
          "market_id": "18",
          "market_type": "handicap",
          "market_name": "让球",
          "specifiers": "hcp=-1",
          "status": "active",
          "odds": [
            {
              "outcome_id": "1",
              "outcome_name": "Home",
              "odds_value": 1.85,
              "probability": 0.5405,
              "active": true,
              "timestamp": 1698765432000,
              "updated_at": "2025-10-24T11:05:23Z"
            },
            {
              "outcome_id": "2",
              "outcome_name": "Away",
              "odds_value": 1.95,
              "probability": 0.5128,
              "active": true,
              "timestamp": 1698765432000,
              "updated_at": "2025-10-24T11:05:23Z"
            }
          ],
          "updated_at": "2025-10-24T11:05:23Z"
        }
      ]
    }
  ]
}
```

#### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `market_id` | string | 盘口 ID (1=胜平负, 18=让球, 26=大小球) |
| `market_type` | string | 盘口类型: `1x2`, `handicap`, `totals` |
| `market_name` | string | 盘口中文名称 |
| `specifiers` | string | 盘口参数 (例如: hcp=-1 表示让1球) |
| `outcome_id` | string | 结果 ID (1=主队, X=平局, 2=客队) |
| `outcome_name` | string | 结果名称 |
| `odds_value` | float | 赔率值 |
| `probability` | float | 隐含概率 (0-1) |
| `active` | bool | 是否可投注 |
| `timestamp` | int64 | 赔率更新时间戳(毫秒) |

#### 前端使用

```javascript
async function fetchAllOdds() {
  const response = await fetch('/api/odds/all');
  const data = await response.json();
  
  if (data.success) {
    return data.events;
  }
  throw new Error('Failed to fetch odds');
}

// 使用示例
const events = await fetchAllOdds();
events.forEach(event => {
  console.log(`比赛: ${event.event_id}`);
  event.markets.forEach(market => {
    console.log(`  ${market.market_name}:`);
    market.odds.forEach(odd => {
      console.log(`    ${odd.outcome_name}: ${odd.odds_value}`);
    });
  });
});
```

---

### 2. 获取单个比赛的所有盘口

获取指定比赛的所有盘口列表。

```http
GET /api/odds/{event_id}/markets
```

#### 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `event_id` | string | 比赛 ID |

#### 响应示例

```json
{
  "success": true,
  "event_id": "sr:match:12345",
  "count": 5,
  "markets": [
    {
      "id": 1,
      "market_id": "1",
      "market_type": "1x2",
      "market_name": "胜平负",
      "specifiers": "",
      "status": "active",
      "odds_count": 3,
      "updated_at": "2025-10-24T11:05:23Z"
    },
    {
      "id": 2,
      "market_id": "18",
      "market_type": "handicap",
      "market_name": "让球",
      "specifiers": "hcp=-1",
      "status": "active",
      "odds_count": 2,
      "updated_at": "2025-10-24T11:05:23Z"
    }
  ]
}
```

#### 前端使用

```javascript
async function fetchEventMarkets(eventId) {
  const response = await fetch(`/api/odds/${eventId}/markets`);
  const data = await response.json();
  return data.markets;
}
```

---

### 3. 获取单个盘口的当前赔率

获取指定盘口的当前赔率详情。

```http
GET /api/odds/{event_id}/{market_id}
```

#### 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `event_id` | string | 比赛 ID |
| `market_id` | string | 盘口 ID (例如: 1, 18, 26) |

#### 响应示例

```json
{
  "success": true,
  "event_id": "sr:match:12345",
  "market_id": "1",
  "count": 3,
  "odds": [
    {
      "outcome_id": "1",
      "outcome_name": "Home",
      "odds_value": 2.50,
      "probability": 0.4000,
      "active": true,
      "timestamp": 1698765432000,
      "updated_at": "2025-10-24T11:05:23Z"
    },
    {
      "outcome_id": "X",
      "outcome_name": "Draw",
      "odds_value": 3.20,
      "probability": 0.3125,
      "active": true,
      "timestamp": 1698765432000,
      "updated_at": "2025-10-24T11:05:23Z"
    },
    {
      "outcome_id": "2",
      "outcome_name": "Away",
      "odds_value": 3.00,
      "probability": 0.3333,
      "active": true,
      "timestamp": 1698765432000,
      "updated_at": "2025-10-24T11:05:23Z"
    }
  ]
}
```

#### 前端使用

```javascript
async function fetchMarketOdds(eventId, marketId) {
  const response = await fetch(`/api/odds/${eventId}/${marketId}`);
  const data = await response.json();
  return data.odds;
}

// 获取胜平负赔率
const odds1x2 = await fetchMarketOdds('sr:match:12345', '1');
```

---

### 4. 获取赔率变化历史

获取指定盘口结果的赔率变化历史。

```http
GET /api/odds/{event_id}/{market_id}/{outcome_id}/history?limit=50
```

#### 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `event_id` | string | 比赛 ID |
| `market_id` | string | 盘口 ID |
| `outcome_id` | string | 结果 ID (1=主队, X=平局, 2=客队) |

#### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `limit` | int | 50 | 返回的历史记录数量 (最大 200) |

#### 响应示例

```json
{
  "success": true,
  "event_id": "sr:match:12345",
  "market_id": "1",
  "outcome_id": "1",
  "count": 10,
  "history": [
    {
      "odds_value": 2.50,
      "probability": 0.4000,
      "change_type": "up",
      "timestamp": 1698765432000,
      "created_at": "2025-10-24T11:05:23Z"
    },
    {
      "odds_value": 2.45,
      "probability": 0.4082,
      "change_type": "down",
      "timestamp": 1698765400000,
      "created_at": "2025-10-24T11:04:50Z"
    },
    {
      "odds_value": 2.48,
      "probability": 0.4032,
      "change_type": "new",
      "timestamp": 1698765000000,
      "created_at": "2025-10-24T11:00:00Z"
    }
  ]
}
```

#### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `change_type` | string | 变化类型: `up`(上升), `down`(下降), `new`(新增) |
| `timestamp` | int64 | 赔率变化时间戳(毫秒) |

#### 前端使用

```javascript
async function fetchOddsHistory(eventId, marketId, outcomeId, limit = 50) {
  const response = await fetch(
    `/api/odds/${eventId}/${marketId}/${outcomeId}/history?limit=${limit}`
  );
  const data = await response.json();
  return data.history;
}

// 获取主队胜赔率的历史变化
const history = await fetchOddsHistory('sr:match:12345', '1', '1', 100);

// 绘制赔率趋势图
const chartData = history.map(h => ({
  time: new Date(h.created_at),
  odds: h.odds_value,
  changeType: h.change_type
}));
```

---

## 订阅管理 API

### 1. 获取已订阅的比赛

查询当前已订阅的所有比赛。

```http
GET /api/booking/booked
```

#### 响应示例

```json
{
  "success": true,
  "count": 15,
  "matches": [
    {
      "id": "sr:match:12345",
      "scheduled": "2025-10-24T10:00:00Z",
      "status": "live",
      "liveodds": "booked"
    }
  ]
}
```

---

### 2. 获取可订阅的比赛

查询当前可以订阅的比赛列表。

```http
GET /api/booking/bookable
```

#### 响应示例

```json
{
  "success": true,
  "count": 25,
  "matches": [
    {
      "id": "sr:match:67890",
      "scheduled": "2025-10-24T15:00:00Z",
      "status": "not_started",
      "liveodds": "bookable"
    }
  ]
}
```

---

### 3. 手动触发自动订阅

手动触发自动订阅流程(查询并订阅所有可订阅的比赛)。

```http
POST /api/booking/trigger
```

#### 响应示例

```json
{
  "success": true,
  "message": "Booking triggered successfully",
  "booked": 10,
  "failed": 1
}
```

---

### 4. 订阅单个比赛

订阅指定的单个比赛。

```http
POST /api/booking/match/{match_id}
```

#### 路径参数

| 参数 | 类型 | 说明 |
|------|------|------|
| `match_id` | string | 比赛 ID |

#### 响应示例

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

获取最近接收的消息列表。

```http
GET /api/messages?limit=100
```

#### 查询参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `limit` | int | 100 | 返回的消息数量 |

#### 响应示例

```json
{
  "success": true,
  "count": 100,
  "messages": [
    {
      "id": 1,
      "message_type": "odds_change",
      "event_id": "sr:match:12345",
      "product_id": 1,
      "timestamp": 1698765432000,
      "received_at": "2025-10-24T11:05:23Z"
    }
  ]
}
```

---

### 2. 获取比赛的消息历史

获取指定比赛的所有消息历史。

```http
GET /api/events/{event_id}/messages
```

#### 响应示例

```json
{
  "success": true,
  "event_id": "sr:match:12345",
  "count": 150,
  "messages": [...]
}
```

---

## 系统监控 API

### 1. 健康检查

检查服务是否正常运行。

```http
GET /api/health
```

#### 响应示例

```json
{
  "status": "ok",
  "timestamp": "2025-10-24T11:05:23Z",
  "uptime": "5h 30m"
}
```

---

### 2. 获取统计信息

获取服务的统计信息。

```http
GET /api/stats
```

#### 响应示例

```json
{
  "success": true,
  "total_messages": 15000,
  "total_events": 50,
  "active_events": 15,
  "message_types": {
    "odds_change": 10000,
    "fixture": 50,
    "bet_stop": 500
  }
}
```

---

### 3. 手动触发监控

手动触发比赛监控检查。

```http
POST /api/monitor/trigger
```

#### 响应示例

```json
{
  "success": true,
  "message": "Monitor triggered successfully"
}
```

---

### 4. 获取服务器 IP

获取服务器的公网 IP 地址。

```http
GET /api/ip
```

#### 响应示例

```json
{
  "ip": "203.0.113.42"
}
```

---

## WebSocket 实时推送

### 连接

```javascript
const ws = new WebSocket('ws://your-service.railway.app/ws');
```

### 消息格式

```json
{
  "type": "odds_change",
  "timestamp": 1698765432000,
  "data": {
    "event_id": "sr:match:12345",
    "home_score": 1,
    "away_score": 0,
    "match_status": "40",
    "markets": [...]
  }
}
```

### 使用示例

```javascript
const ws = new WebSocket('ws://your-service.railway.app/ws');

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch (message.type) {
    case 'odds_change':
      updateOdds(message.data);
      break;
    case 'fixture':
      updateFixture(message.data);
      break;
    case 'bet_stop':
      handleBetStop(message.data);
      break;
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket disconnected');
  // 重连逻辑
  setTimeout(() => {
    connectWebSocket();
  }, 5000);
};
```

---

## 错误处理

### 错误响应格式

```json
{
  "success": false,
  "error": "Match not found",
  "code": 404
}
```

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

### 前端错误处理示例

```javascript
async function fetchWithErrorHandling(url) {
  try {
    const response = await fetch(url);
    
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Unknown error');
    }
    
    return data;
  } catch (error) {
    console.error('API Error:', error);
    throw error;
  }
}
```

---

## 前端集成示例

### Vue 3 + Composition API

```vue
<template>
  <div class="matches">
    <h2>进行中的比赛</h2>
    
    <div v-if="loading">加载中...</div>
    <div v-else-if="error">{{ error }}</div>
    
    <div v-else class="match-list">
      <div v-for="match in matches" :key="match.event_id" class="match-card">
        <div class="teams">
          <span class="home">{{ match.home_team_name }}</span>
          <span class="score">{{ match.home_score }} - {{ match.away_score }}</span>
          <span class="away">{{ match.away_team_name }}</span>
        </div>
        
        <div class="match-info">
          <span class="status">{{ getMatchStatus(match.match_status) }}</span>
          <span class="time">{{ match.match_time }}</span>
        </div>
        
        <div class="odds" v-if="match.odds">
          <div v-for="odd in match.odds" :key="odd.outcome_id" class="odd-item">
            <span class="outcome">{{ odd.outcome_name }}</span>
            <span class="value">{{ odd.odds_value }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';

const matches = ref([]);
const loading = ref(true);
const error = ref(null);
let intervalId = null;

// 获取进行中的比赛
async function fetchLiveMatches() {
  try {
    const response = await fetch('/api/matches/live');
    const data = await response.json();
    
    if (data.success) {
      matches.value = data.matches;
      
      // 获取每个比赛的赔率
      for (const match of matches.value) {
        await fetchMatchOdds(match.event_id);
      }
    }
  } catch (err) {
    error.value = err.message;
  } finally {
    loading.value = false;
  }
}

// 获取比赛赔率
async function fetchMatchOdds(eventId) {
  try {
    const response = await fetch(`/api/odds/${eventId}/1`); // 获取胜平负赔率
    const data = await response.json();
    
    if (data.success) {
      const match = matches.value.find(m => m.event_id === eventId);
      if (match) {
        match.odds = data.odds;
      }
    }
  } catch (err) {
    console.error('Failed to fetch odds:', err);
  }
}

// 获取比赛状态文本
function getMatchStatus(statusCode) {
  const statusMap = {
    '20': '上半场',
    '30': '中场休息',
    '40': '下半场',
    '100': '点球大战'
  };
  return statusMap[statusCode] || '进行中';
}

onMounted(() => {
  fetchLiveMatches();
  
  // 每 5 秒刷新一次
  intervalId = setInterval(fetchLiveMatches, 5000);
});

onUnmounted(() => {
  if (intervalId) {
    clearInterval(intervalId);
  }
});
</script>

<style scoped>
.match-card {
  border: 1px solid #ddd;
  padding: 16px;
  margin-bottom: 12px;
  border-radius: 8px;
}

.teams {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.score {
  font-size: 24px;
  font-weight: bold;
}

.odds {
  display: flex;
  gap: 12px;
  margin-top: 12px;
}

.odd-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 16px;
  background: #f5f5f5;
  border-radius: 4px;
}

.odd-item .value {
  font-size: 18px;
  font-weight: bold;
  color: #1890ff;
}
</style>
```

---

### React + Hooks

```jsx
import React, { useState, useEffect } from 'react';

function LiveMatches() {
  const [matches, setMatches] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // 获取进行中的比赛
  const fetchLiveMatches = async () => {
    try {
      const response = await fetch('/api/matches/live');
      const data = await response.json();
      
      if (data.success) {
        setMatches(data.matches);
        
        // 获取每个比赛的赔率
        for (const match of data.matches) {
          await fetchMatchOdds(match.event_id);
        }
      }
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  // 获取比赛赔率
  const fetchMatchOdds = async (eventId) => {
    try {
      const response = await fetch(`/api/odds/${eventId}/1`);
      const data = await response.json();
      
      if (data.success) {
        setMatches(prevMatches =>
          prevMatches.map(match =>
            match.event_id === eventId
              ? { ...match, odds: data.odds }
              : match
          )
        );
      }
    } catch (err) {
      console.error('Failed to fetch odds:', err);
    }
  };

  useEffect(() => {
    fetchLiveMatches();
    
    // 每 5 秒刷新一次
    const intervalId = setInterval(fetchLiveMatches, 5000);
    
    return () => clearInterval(intervalId);
  }, []);

  if (loading) return <div>加载中...</div>;
  if (error) return <div>错误: {error}</div>;

  return (
    <div className="matches">
      <h2>进行中的比赛</h2>
      
      <div className="match-list">
        {matches.map(match => (
          <div key={match.event_id} className="match-card">
            <div className="teams">
              <span className="home">{match.home_team_name}</span>
              <span className="score">
                {match.home_score} - {match.away_score}
              </span>
              <span className="away">{match.away_team_name}</span>
            </div>
            
            <div className="match-info">
              <span className="status">{getMatchStatus(match.match_status)}</span>
              <span className="time">{match.match_time}</span>
            </div>
            
            {match.odds && (
              <div className="odds">
                {match.odds.map(odd => (
                  <div key={odd.outcome_id} className="odd-item">
                    <span className="outcome">{odd.outcome_name}</span>
                    <span className="value">{odd.odds_value}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function getMatchStatus(statusCode) {
  const statusMap = {
    '20': '上半场',
    '30': '中场休息',
    '40': '下半场',
    '100': '点球大战'
  };
  return statusMap[statusCode] || '进行中';
}

export default LiveMatches;
```

---

### 原生 JavaScript

```javascript
class MatchService {
  constructor(baseUrl = '/api') {
    this.baseUrl = baseUrl;
  }

  // 获取进行中的比赛
  async getLiveMatches() {
    const response = await fetch(`${this.baseUrl}/matches/live`);
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Failed to fetch live matches');
    }
    
    return data.matches;
  }

  // 获取比赛赔率
  async getMatchOdds(eventId, marketId = '1') {
    const response = await fetch(`${this.baseUrl}/odds/${eventId}/${marketId}`);
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Failed to fetch odds');
    }
    
    return data.odds;
  }

  // 获取赔率历史
  async getOddsHistory(eventId, marketId, outcomeId, limit = 50) {
    const response = await fetch(
      `${this.baseUrl}/odds/${eventId}/${marketId}/${outcomeId}/history?limit=${limit}`
    );
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Failed to fetch odds history');
    }
    
    return data.history;
  }

  // 搜索比赛
  async searchMatches(query) {
    const response = await fetch(
      `${this.baseUrl}/matches/search?q=${encodeURIComponent(query)}`
    );
    const data = await response.json();
    
    if (!data.success) {
      throw new Error(data.error || 'Failed to search matches');
    }
    
    return data.matches;
  }
}

// 使用示例
const matchService = new MatchService();

// 获取进行中的比赛
const liveMatches = await matchService.getLiveMatches();
console.log('Live matches:', liveMatches);

// 获取赔率
const odds = await matchService.getMatchOdds('sr:match:12345');
console.log('Odds:', odds);

// 搜索比赛
const searchResults = await matchService.searchMatches('Manchester');
console.log('Search results:', searchResults);
```

---

## 常见问题

### Q1: 如何实时更新赔率?

**方案 1: 轮询** (推荐用于简单场景)
```javascript
setInterval(async () => {
  const odds = await fetchAllOdds();
  updateUI(odds);
}, 5000); // 每 5 秒刷新
```

**方案 2: WebSocket** (推荐用于实时性要求高的场景)
```javascript
const ws = new WebSocket('ws://your-service.railway.app/ws');
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  if (message.type === 'odds_change') {
    updateOdds(message.data);
  }
};
```

---

### Q2: 如何处理大量比赛数据?

使用分页或虚拟滚动:

```javascript
// 分批加载
async function loadMatchesInBatches(batchSize = 10) {
  const allMatches = await fetchLiveMatches();
  
  for (let i = 0; i < allMatches.length; i += batchSize) {
    const batch = allMatches.slice(i, i + batchSize);
    renderMatches(batch);
    await new Promise(resolve => setTimeout(resolve, 100));
  }
}
```

---

### Q3: 如何优化 API 调用性能?

**使用缓存**:
```javascript
class CachedMatchService {
  constructor() {
    this.cache = new Map();
    this.cacheDuration = 5000; // 5 秒缓存
  }

  async getLiveMatches() {
    const cacheKey = 'live_matches';
    const cached = this.cache.get(cacheKey);
    
    if (cached && Date.now() - cached.timestamp < this.cacheDuration) {
      return cached.data;
    }
    
    const data = await fetch('/api/matches/live').then(r => r.json());
    this.cache.set(cacheKey, { data: data.matches, timestamp: Date.now() });
    
    return data.matches;
  }
}
```

---

## 更新日志

### v2.0 (2025-10-24)

- ✅ 新增盘口赔率 API (4 个端点)
- ✅ 新增赔率历史追踪
- ✅ 新增 SR 数据映射
- ✅ 新增订阅管理 API

### v1.0 (2025-10-23)

- ✅ 初始版本
- ✅ 比赛查询 API
- ✅ WebSocket 实时推送

---

## 支持

如有问题,请联系:
- **文档**: https://github.com/gdszyy/betradar-uof-service
- **问题反馈**: https://help.manus.im

---

**文档版本**: v2.0  
**最后更新**: 2025-10-24  
**维护者**: Betradar UOF Service Team

