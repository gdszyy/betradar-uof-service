# UOF Service 快速开始指南

本指南帮助您快速了解和使用 Betradar UOF Service。

---

## 🎯 服务概述

**Betradar UOF Service** 是一个实时体育赛事数据服务,通过 Betradar Unified Odds Feed (UOF) 获取:
- 📊 实时比赛数据
- 🎲 赔率变化
- ⚽ 比分更新
- 📅 赛程信息

---

## 🚀 核心功能

### 1. 启动时自动订阅 ✨ NEW

服务启动后会自动:
1. 查询所有直播比赛
2. 筛选可订阅的比赛
3. 自动订阅所有可订阅比赛
4. 验证订阅状态
5. 发送飞书通知

**日志示例**:
```
[StartupBooking] 🚀 Starting automatic booking on service startup...
[StartupBooking] 📊 Found 50 live matches
[StartupBooking] 🎯 Found 15 bookable matches
[StartupBooking] 📝 Booking 15 matches...
[StartupBooking] ✅ Successfully booked sr:match:12345
[StartupBooking] 🔍 Verifying subscriptions...
[StartupBooking] ✅ Verified 14 subscriptions
[StartupBooking] 📈 Booking completed: 14 success, 1 failed out of 15 bookable
```

---

### 2. 实时数据流

**AMQP 消息接收** → **解析存储** → **WebSocket 推送** → **前端实时更新**

支持的消息类型:
- `odds_change` - 赔率变化
- `bet_stop` - 投注停止
- `bet_settlement` - 投注结算
- `fixture` - 赛程信息
- `fixture_change` - 赛程变更

---

### 3. 前端 API

提供完整的 REST API 供前端调用:

| API | 用途 |
|-----|------|
| `/api/matches/live` | 获取进行中的比赛 |
| `/api/matches/upcoming` | 获取即将开始的比赛 |
| `/api/matches/search` | 搜索比赛 |
| `/api/matches/{id}` | 获取比赛详情 |
| `/api/booking/booked` | 查看已订阅的比赛 |
| `/api/booking/trigger` | 手动触发自动订阅 |

---

## 📦 部署状态

### Railway 自动部署

代码推送到 GitHub 后,Railway 会自动:
1. 检测代码变更
2. 构建 Docker 镜像
3. 部署新版本
4. 重启服务

**当前版本**: `20fff3f` (2025-10-24)

---

## 🔧 快速测试

### 1. 检查服务状态

```bash
curl https://your-service.railway.app/api/health
```

预期响应:
```json
{
  "status": "ok",
  "time": 1729666800
}
```

---

### 2. 查看进行中的比赛

```bash
curl https://your-service.railway.app/api/matches/live
```

预期响应:
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
      "match_time": "23:15"
    }
  ]
}
```

---

### 3. 查看已订阅的比赛

```bash
curl https://your-service.railway.app/api/booking/booked
```

预期响应:
```json
{
  "success": true,
  "count": 20,
  "matches": [...]
}
```

---

### 4. 手动触发自动订阅

```bash
curl -X POST https://your-service.railway.app/api/booking/trigger
```

预期响应:
```json
{
  "success": true,
  "total_live": 50,
  "bookable": 15,
  "success_count": 14,
  "failed_count": 1,
  "verified_count": 14
}
```

---

### 5. 搜索比赛

```bash
curl "https://your-service.railway.app/api/matches/search?q=Manchester"
```

预期响应:
```json
{
  "success": true,
  "keyword": "Manchester",
  "count": 3,
  "matches": [...]
}
```

---

## 🌐 WebSocket 连接

### JavaScript 示例

```javascript
const ws = new WebSocket('ws://your-service.railway.app/ws');

ws.onopen = () => {
  console.log('✅ WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('📨 Received:', message.type, message.event_id);
  
  if (message.type === 'odds_change') {
    console.log(`⚽ Score: ${message.data.home_score}-${message.data.away_score}`);
  }
};

ws.onerror = (error) => {
  console.error('❌ WebSocket error:', error);
};

ws.onclose = () => {
  console.log('🔌 WebSocket disconnected');
};
```

---

## 📊 数据库查询

### 查看追踪的比赛

```sql
SELECT 
    event_id,
    home_team_name,
    away_team_name,
    home_score,
    away_score,
    match_status,
    match_time,
    schedule_time
FROM tracked_events
WHERE status = 'active'
ORDER BY schedule_time DESC
LIMIT 20;
```

---

### 查看消息统计

```sql
SELECT 
    message_type,
    COUNT(*) as count
FROM messages
GROUP BY message_type
ORDER BY count DESC;
```

---

## 🎨 前端集成

### Vue 3 + Pinia 示例

```typescript
// stores/matches.ts
import { defineStore } from 'pinia';
import { ref } from 'vue';

export const useMatchesStore = defineStore('matches', () => {
  const matches = ref([]);
  const loading = ref(false);
  
  async function fetchLiveMatches() {
    loading.value = true;
    try {
      const response = await fetch('/api/matches/live');
      const data = await response.json();
      matches.value = data.matches;
    } catch (error) {
      console.error('Failed to fetch matches:', error);
    } finally {
      loading.value = false;
    }
  }
  
  return {
    matches,
    loading,
    fetchLiveMatches,
  };
});
```

```vue
<!-- components/MatchList.vue -->
<template>
  <div class="match-list">
    <h2>进行中的比赛</h2>
    <div v-if="loading">加载中...</div>
    <div v-else>
      <div v-for="match in matches" :key="match.event_id" class="match-card">
        <div class="teams">
          <span>{{ match.home_team_name }}</span>
          <span class="score">{{ match.home_score }} - {{ match.away_score }}</span>
          <span>{{ match.away_team_name }}</span>
        </div>
        <div class="time">{{ match.match_time }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue';
import { useMatchesStore } from '@/stores/matches';

const matchesStore = useMatchesStore();
const { matches, loading } = matchesStore;

onMounted(() => {
  matchesStore.fetchLiveMatches();
});
</script>
```

---

## 📝 日志监控

### Railway 日志

在 Railway 控制台查看实时日志:

**启动日志**:
```
Starting Betradar UOF Service...
Database connected and migrated
AMQP consumer started
Web server started on port 8080
Match monitor started (hourly)
Service is running. Press Ctrl+C to stop.
All data is sourced from UOF (Unified Odds Feed)
```

**自动订阅日志**:
```
[StartupBooking] 🚀 Starting automatic booking on service startup...
[StartupBooking] 📊 Found 50 live matches
[StartupBooking] 🎯 Found 15 bookable matches
[StartupBooking] ✅ Startup booking completed: 14/15 successful
```

**解析器日志**:
```
[FixtureParser] Parsing fixture for event: sr:match:12345
[FixtureParser] Stored fixture data for event sr:match:12345: home=Team A, away=Team B
[OddsChangeParser] Parsing odds_change for event: sr:match:12345
[OddsChangeParser] Extracted from sport_event_status: score=1-0, status=3, match_status=6
[OddsChangeParser] Stored odds_change data for event sr:match:12345: 1-0, status=6
```

---

## 🔍 故障排查

### 问题 1: 服务无法启动

**检查**:
1. Railway 日志中是否有错误
2. 环境变量是否正确配置
3. 数据库连接是否正常

**解决**:
```bash
# 检查环境变量
railway variables

# 查看日志
railway logs

# 重启服务
railway restart
```

---

### 问题 2: 没有收到消息

**检查**:
1. AMQP 连接是否建立
2. 是否有已订阅的比赛
3. 路由键配置是否正确

**解决**:
```bash
# 查看已订阅的比赛
curl https://your-service.railway.app/api/booking/booked

# 触发自动订阅
curl -X POST https://your-service.railway.app/api/booking/trigger

# 查看消息统计
curl https://your-service.railway.app/api/stats
```

---

### 问题 3: 数据库字段不存在

**错误**: `column "schedule_time" does not exist`

**解决**: 执行数据库迁移

```bash
# 方法 1: Railway CLI
railway run go run cmd/migrate/main.go

# 方法 2: Web Console
# 在 Railway PostgreSQL 服务的 Query 界面执行 QUICK_MIGRATION.sql
```

详见: [DATABASE_MIGRATION_GUIDE.md](./DATABASE_MIGRATION_GUIDE.md)

---

## 📚 相关文档

| 文档 | 描述 |
|------|------|
| [API_DOCUMENTATION.md](./API_DOCUMENTATION.md) | 完整的 API 文档 |
| [FRONTEND_DATA_FLOW.md](./FRONTEND_DATA_FLOW.md) | 前端数据流设计 |
| [DATABASE_MIGRATION_GUIDE.md](./DATABASE_MIGRATION_GUIDE.md) | 数据库迁移指南 |
| [UOF_FIX_REPORT.md](./UOF_FIX_REPORT.md) | 修复报告 |

---

## 🎯 下一步

1. ✅ **验证服务运行** - 使用健康检查 API
2. ✅ **测试自动订阅** - 查看启动日志和飞书通知
3. ✅ **查询比赛数据** - 使用比赛查询 API
4. ✅ **集成前端** - 参考 FRONTEND_DATA_FLOW.md
5. ✅ **监控日志** - 在 Railway 控制台查看实时日志

---

## 💡 最佳实践

### 1. 数据更新策略

- **主要方式**: WebSocket 实时推送
- **备用方式**: HTTP 轮询 (5-10 秒间隔)
- **页面不可见时**: 降低更新频率或暂停

### 2. 错误处理

- WebSocket 断线自动重连 (指数退避)
- API 请求失败重试 (最多 3 次)
- 显示友好的错误提示

### 3. 性能优化

- 使用虚拟滚动 (比赛数量 > 50)
- 防抖和节流 (搜索、WebSocket 消息)
- 懒加载图片和详情数据

### 4. 安全考虑

- 生产环境限制 CORS 来源
- WebSocket 添加认证 (如需要)
- 前端验证 API 返回数据

---

## 🆘 获取帮助

遇到问题?
1. 查看相关文档
2. 检查 Railway 日志
3. 提交 GitHub Issue
4. 联系开发团队

---

## 🎉 总结

您现在已经了解了:
- ✅ 服务的核心功能
- ✅ 如何测试 API
- ✅ 如何连接 WebSocket
- ✅ 如何集成前端
- ✅ 如何排查问题

开始构建您的博彩页面吧! 🚀

---

**版本**: 2.0  
**更新时间**: 2025-10-24  
**维护者**: UOF Service Team

