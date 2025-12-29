# 前端博彩页面数据流设计

## 概述

本文档描述了前端博彩页面如何与 UOF 服务进行数据交互,实现实时比赛数据展示和更新。

---

## 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                     前端博彩页面                              │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  比赛列表    │  │  比赛详情    │  │  实时赔率    │      │
│  │  组件        │  │  组件        │  │  组件        │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │              │
│         └──────────────────┼──────────────────┘              │
│                            │                                 │
│                    ┌───────▼────────┐                        │
│                    │  数据管理层     │                        │
│                    │  (Vuex/Pinia)  │                        │
│                    └───────┬────────┘                        │
│                            │                                 │
│         ┌──────────────────┼──────────────────┐             │
│         │                  │                  │             │
│    ┌────▼─────┐      ┌────▼─────┐      ┌────▼─────┐       │
│    │ HTTP API │      │WebSocket │      │ 轮询定时器│       │
│    └────┬─────┘      └────┬─────┘      └────┬─────┘       │
└─────────┼──────────────────┼──────────────────┼─────────────┘
          │                  │                  │
          │                  │                  │
┌─────────▼──────────────────▼──────────────────▼─────────────┐
│                    UOF 后端服务                               │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  REST API    │  │  WebSocket   │  │  数据库      │      │
│  │  /api/*      │  │  /ws         │  │  PostgreSQL  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
└───────────────────────────────────────────────────────────────┘
                            │
                            │
                    ┌───────▼────────┐
                    │  Betradar UOF  │
                    │  AMQP Stream   │
                    └────────────────┘
```

---

## 数据流类型

### 1. 初始化数据加载 (HTTP API)

**场景**: 页面首次加载或刷新

**流程**:
```
前端页面加载
  ↓
调用 GET /api/events (获取所有追踪的比赛)
  ↓
调用 GET /api/booking/booked (获取已订阅的比赛)
  ↓
渲染比赛列表
  ↓
建立 WebSocket 连接 (接收实时更新)
```

**API 端点**:
- `GET /api/events` - 获取所有追踪的比赛
- `GET /api/booking/booked` - 获取已订阅的比赛
- `GET /api/booking/bookable` - 获取可订阅的比赛
- `GET /api/stats` - 获取统计信息

---

### 2. 实时数据更新 (WebSocket)

**场景**: 比赛进行中,实时接收赔率变化、比分更新

**流程**:
```
WebSocket 连接建立
  ↓
订阅特定比赛或全局消息
  ↓
接收实时消息 (odds_change, bet_stop, bet_settlement)
  ↓
更新前端状态
  ↓
触发 UI 重新渲染
```

**WebSocket 消息格式**:
```json
{
  "type": "odds_change",
  "event_id": "sr:match:12345",
  "timestamp": 1729666800000,
  "data": {
    "home_score": 1,
    "away_score": 0,
    "match_status": "6",
    "match_time": "23:15"
  }
}
```

---

### 3. 轮询更新 (HTTP API)

**场景**: 作为 WebSocket 的备用方案,或用于低频更新的数据

**流程**:
```
设置定时器 (例如每 5 秒)
  ↓
调用 GET /api/events (获取最新比赛数据)
  ↓
比较数据变化
  ↓
更新前端状态
```

**推荐配置**:
- WebSocket 可用: 不使用轮询
- WebSocket 不可用: 每 5-10 秒轮询一次
- 比赛详情页: 每 3-5 秒轮询一次
- 比赛列表页: 每 10-30 秒轮询一次

---

## 前端实现建议

### 技术栈

**推荐**: Vue 3 + TypeScript + Pinia + Axios

```typescript
// 数据结构定义
interface Match {
  event_id: string;
  srn_id?: string;
  home_team_name: string;
  away_team_name: string;
  home_score: number;
  away_score: number;
  match_status: string;
  match_time: string;
  schedule_time: string;
  status: string;
}

interface OddsChangeMessage {
  type: 'odds_change';
  event_id: string;
  timestamp: number;
  data: {
    home_score: number;
    away_score: number;
    match_status: string;
    match_time: string;
  };
}
```

---

### 数据管理 (Pinia Store)

```typescript
// stores/matches.ts
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useMatchesStore = defineStore('matches', () => {
  // 状态
  const matches = ref<Map<string, Match>>(new Map());
  const loading = ref(false);
  const error = ref<string | null>(null);
  
  // WebSocket 连接
  const ws = ref<WebSocket | null>(null);
  const wsConnected = ref(false);
  
  // 计算属性
  const liveMatches = computed(() => {
    return Array.from(matches.value.values())
      .filter(m => m.status === 'active');
  });
  
  const endedMatches = computed(() => {
    return Array.from(matches.value.values())
      .filter(m => m.status === 'ended');
  });
  
  // 初始化加载
  async function fetchMatches() {
    loading.value = true;
    error.value = null;
    
    try {
      const response = await fetch('/api/events');
      const data = await response.json();
      
      // 更新 matches Map
      data.events.forEach((match: Match) => {
        matches.value.set(match.event_id, match);
      });
    } catch (err) {
      error.value = err.message;
    } finally {
      loading.value = false;
    }
  }
  
  // 建立 WebSocket 连接
  function connectWebSocket() {
    const wsUrl = `ws://${window.location.host}/ws`;
    ws.value = new WebSocket(wsUrl);
    
    ws.value.onopen = () => {
      console.log('[WebSocket] Connected');
      wsConnected.value = true;
    };
    
    ws.value.onmessage = (event) => {
      const message = JSON.parse(event.data);
      handleWebSocketMessage(message);
    };
    
    ws.value.onerror = (error) => {
      console.error('[WebSocket] Error:', error);
      wsConnected.value = false;
    };
    
    ws.value.onclose = () => {
      console.log('[WebSocket] Disconnected');
      wsConnected.value = false;
      
      // 5 秒后重连
      setTimeout(() => {
        connectWebSocket();
      }, 5000);
    };
  }
  
  // 处理 WebSocket 消息
  function handleWebSocketMessage(message: OddsChangeMessage) {
    if (message.type === 'odds_change') {
      const match = matches.value.get(message.event_id);
      if (match) {
        // 更新比赛数据
        match.home_score = message.data.home_score;
        match.away_score = message.data.away_score;
        match.match_status = message.data.match_status;
        match.match_time = message.data.match_time;
        
        // 触发响应式更新
        matches.value.set(message.event_id, { ...match });
      }
    }
  }
  
  // 更新单个比赛
  function updateMatch(eventId: string, updates: Partial<Match>) {
    const match = matches.value.get(eventId);
    if (match) {
      matches.value.set(eventId, { ...match, ...updates });
    }
  }
  
  return {
    matches,
    loading,
    error,
    wsConnected,
    liveMatches,
    endedMatches,
    fetchMatches,
    connectWebSocket,
    updateMatch,
  };
});
```

---

### 组件示例

#### 比赛列表组件

```vue
<template>
  <div class="matches-list">
    <div class="header">
      <h2>直播比赛</h2>
      <div class="status">
        <span :class="{ 'connected': wsConnected, 'disconnected': !wsConnected }">
          {{ wsConnected ? '● 实时连接' : '○ 已断开' }}
        </span>
      </div>
    </div>
    
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    
    <div v-else class="matches">
      <MatchCard
        v-for="match in liveMatches"
        :key="match.event_id"
        :match="match"
        @click="goToMatchDetail(match.event_id)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue';
import { useMatchesStore } from '@/stores/matches';
import MatchCard from './MatchCard.vue';

const matchesStore = useMatchesStore();
const { liveMatches, loading, error, wsConnected } = matchesStore;

onMounted(async () => {
  // 1. 加载初始数据
  await matchesStore.fetchMatches();
  
  // 2. 建立 WebSocket 连接
  matchesStore.connectWebSocket();
});

onUnmounted(() => {
  // 断开 WebSocket
  if (matchesStore.ws) {
    matchesStore.ws.close();
  }
});

function goToMatchDetail(eventId: string) {
  // 跳转到比赛详情页
  router.push(`/match/${eventId}`);
}
</script>
```

#### 比赛卡片组件

```vue
<template>
  <div class="match-card" :class="{ 'live': match.status === 'active' }">
    <div class="match-header">
      <span class="match-time">{{ formatMatchTime(match.schedule_time) }}</span>
      <span v-if="match.match_time" class="live-time">{{ match.match_time }}</span>
    </div>
    
    <div class="teams">
      <div class="team home">
        <span class="team-name">{{ match.home_team_name }}</span>
        <span class="score">{{ match.home_score }}</span>
      </div>
      
      <div class="separator">VS</div>
      
      <div class="team away">
        <span class="team-name">{{ match.away_team_name }}</span>
        <span class="score">{{ match.away_score }}</span>
      </div>
    </div>
    
    <div class="match-status">
      <span :class="`status-${match.match_status}`">
        {{ getMatchStatusText(match.match_status) }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue';
import type { Match } from '@/types';

defineProps<{
  match: Match;
}>();

function formatMatchTime(timestamp: string) {
  if (!timestamp) return '';
  const date = new Date(timestamp);
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
}

function getMatchStatusText(status: string) {
  const statusMap: Record<string, string> = {
    '0': '未开始',
    '1': '进行中',
    '2': '暂停',
    '3': '已结束',
    '4': '延期',
    '5': '取消',
    '6': '上半场',
    '7': '中场',
    '8': '下半场',
  };
  return statusMap[status] || '未知';
}
</script>

<style scoped>
.match-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  cursor: pointer;
  transition: all 0.3s;
}

.match-card:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  transform: translateY(-2px);
}

.match-card.live {
  border-color: #ff4444;
  background: linear-gradient(135deg, #fff 0%, #fff5f5 100%);
}

.teams {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 12px 0;
}

.team {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 1;
}

.team-name {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 8px;
}

.score {
  font-size: 24px;
  font-weight: bold;
  color: #333;
}

.separator {
  font-size: 12px;
  color: #999;
  margin: 0 16px;
}

.live-time {
  color: #ff4444;
  font-weight: bold;
  animation: blink 1.5s infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
```

---

## API 端点总结

### 比赛数据

| 方法 | 端点 | 描述 | 返回 |
|------|------|------|------|
| GET | `/api/events` | 获取所有追踪的比赛 | 比赛列表 |
| GET | `/api/events/{event_id}/messages` | 获取比赛的消息历史 | 消息列表 |
| GET | `/api/stats` | 获取统计信息 | 统计数据 |

### 订阅管理

| 方法 | 端点 | 描述 | 返回 |
|------|------|------|------|
| GET | `/api/booking/booked` | 获取已订阅的比赛 | 已订阅列表 |
| GET | `/api/booking/bookable` | 获取可订阅的比赛 | 可订阅列表 |
| POST | `/api/booking/trigger` | 触发自动订阅 | 订阅结果 |
| POST | `/api/booking/match/{match_id}` | 订阅单个比赛 | 成功/失败 |

### WebSocket

| 端点 | 描述 | 消息类型 |
|------|------|----------|
| `/ws` | WebSocket 连接 | odds_change, bet_stop, bet_settlement, fixture |

---

## 数据更新策略

### 推荐配置

| 场景 | 主要方式 | 备用方式 | 更新频率 |
|------|----------|----------|----------|
| 比赛列表页 | WebSocket | HTTP 轮询 | 实时 / 30秒 |
| 比赛详情页 | WebSocket | HTTP 轮询 | 实时 / 5秒 |
| 赔率展示 | WebSocket | HTTP 轮询 | 实时 / 3秒 |
| 统计数据 | HTTP 轮询 | - | 60秒 |

### 优化建议

1. **使用 WebSocket 作为主要方式**
   - 低延迟,实时性好
   - 减少服务器负载
   - 更好的用户体验

2. **HTTP 轮询作为备用**
   - WebSocket 连接失败时启用
   - 兼容性更好
   - 实现简单

3. **智能更新**
   - 页面不可见时降低更新频率
   - 使用 `document.visibilityState` 检测
   - 节省带宽和电量

4. **数据缓存**
   - 使用 Map 存储比赛数据
   - 只更新变化的字段
   - 减少不必要的渲染

---

## 性能优化

### 前端优化

1. **虚拟滚动**
   - 比赛数量超过 50 场时使用
   - 推荐库: `vue-virtual-scroller`

2. **防抖和节流**
   - WebSocket 消息处理使用节流
   - 搜索输入使用防抖

3. **懒加载**
   - 比赛详情按需加载
   - 图片懒加载

### 后端优化

1. **数据库索引**
   - `event_id` 索引
   - `status` 索引
   - `schedule_time` 索引

2. **缓存策略**
   - Redis 缓存热门比赛数据
   - 5-10 秒过期时间

3. **分页**
   - 比赛列表分页
   - 每页 20-50 条

---

## 错误处理

### WebSocket 断线重连

```typescript
function connectWebSocket() {
  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;
  
  function connect() {
    ws.value = new WebSocket(wsUrl);
    
    ws.value.onclose = () => {
      wsConnected.value = false;
      
      if (reconnectAttempts < maxReconnectAttempts) {
        reconnectAttempts++;
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
        console.log(`[WebSocket] Reconnecting in ${delay}ms...`);
        setTimeout(connect, delay);
      } else {
        console.error('[WebSocket] Max reconnect attempts reached');
        // 切换到 HTTP 轮询
        startPolling();
      }
    };
  }
  
  connect();
}
```

### API 请求失败

```typescript
async function fetchMatches() {
  try {
    const response = await fetch('/api/events');
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const data = await response.json();
    return data;
  } catch (err) {
    console.error('[API] Failed to fetch matches:', err);
    // 显示错误提示
    showErrorToast('获取比赛数据失败,请稍后重试');
    return null;
  }
}
```

---

## 安全考虑

1. **CORS 配置**
   - 后端已配置允许所有来源
   - 生产环境应限制特定域名

2. **WebSocket 认证**
   - 当前无认证
   - 如需认证,可在连接时传递 token

3. **数据验证**
   - 前端验证 API 返回的数据格式
   - 防止 XSS 攻击

---

## 总结

本数据流设计提供了:

1. ✅ **实时性** - WebSocket 推送实时数据
2. ✅ **可靠性** - HTTP 轮询作为备用方案
3. ✅ **可扩展性** - 清晰的数据结构和 API 设计
4. ✅ **性能** - 优化的更新策略和缓存机制
5. ✅ **用户体验** - 流畅的 UI 更新和错误处理

前端开发者可以根据此设计快速实现博彩页面的数据交互功能。

