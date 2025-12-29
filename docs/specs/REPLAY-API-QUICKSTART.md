# 🚀 Replay API 快速开始

## 概述

现在您可以通过HTTP API直接触发Replay测试,无需手动运行脚本!

---

## ✅ 前提条件

### 1. 确认环境变量已设置

在Railway项目中,确保已设置:

```
UOF_USERNAME=your_betradar_username
UOF_PASSWORD=your_betradar_password
```

**设置方法**:
1. 打开 Railway Dashboard
2. 选择您的项目
3. 点击 "Variables" 标签
4. 添加上述两个变量

### 2. 等待部署完成

推送代码后,Railway需要2-3分钟重新部署。您可以在 **Deployments** 标签查看部署状态。

---

## 🎬 快速测试

### 最简单的方式

```bash
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":20,"duration":60}'
```

**就这么简单!** 🎉

---

## 📋 API端点

### 1. 启动重放

```bash
POST /api/replay/start
```

**请求示例**:
```bash
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 50,
    "duration": 45,
    "node_id": 1
  }'
```

**响应**:
```json
{
  "status": "accepted",
  "message": "Replay request accepted and processing",
  "event_id": "test:match:21797788",
  "speed": 50,
  "node_id": 1,
  "duration": 45,
  "time": 1761025000
}
```

### 2. 停止重放

```bash
POST /api/replay/stop
```

**请求示例**:
```bash
curl -X POST https://your-service.railway.app/api/replay/stop
```

### 3. 查看状态

```bash
GET /api/replay/status
```

**请求示例**:
```bash
curl https://your-service.railway.app/api/replay/status
```

### 4. 列出队列

```bash
GET /api/replay/list
```

**请求示例**:
```bash
curl https://your-service.railway.app/api/replay/list
```

---

## 🎯 推荐测试场景

### 场景1: 快速验证(推荐)

```bash
# 50倍速,45秒,自动停止
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 50,
    "duration": 45
  }'
```

**预期结果**:
- 45秒后自动停止
- 收到大量 odds_change, bet_stop, bet_settlement 消息
- 数据库表有数据

### 场景2: 超快速测试

```bash
# 100倍速,30秒
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 100,
    "duration": 30
  }'
```

### 场景3: 慢速调试

```bash
# 1倍速,不自动停止
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 1
  }'

# 手动停止
curl -X POST https://your-service.railway.app/api/replay/stop
```

---

## 📊 验证结果

### 1. 查看统计

```bash
curl https://your-service.railway.app/api/stats
```

**应该看到**:
```json
{
  "total_messages": 增加,
  "odds_changes": 有数据(100+),
  "bet_stops": 有数据(20+),
  "bet_settlements": 有数据(15+)
}
```

### 2. 查看最新消息

```bash
curl "https://your-service.railway.app/api/messages?limit=10"
```

**应该看到**:
- `odds_change` 消息
- `bet_stop` 消息
- `bet_settlement` 消息

### 3. 查看Railway日志

在Railway Dashboard → Deployments → Logs中:

```
🎬 Starting replay via API: event=test:match:21797788, speed=50x, node_id=1
✅ Replay started successfully: test:match:21797788
⏱️  Replay will run for 45 seconds
🛑 Replay stopped after 45 seconds
```

### 4. 查询数据库

```sql
-- 检查赔率变化
SELECT COUNT(*) FROM odds_changes 
WHERE created_at > NOW() - INTERVAL '5 minutes';

-- 查看最新的赔率变化
SELECT event_id, market_count, market_status, created_at
FROM odds_changes
ORDER BY created_at DESC
LIMIT 10;
```

---

## 🔄 完整测试流程

### 使用Shell脚本

```bash
#!/bin/bash

SERVICE_URL="https://your-service.railway.app"

echo "1. 获取初始统计"
curl -s "$SERVICE_URL/api/stats" | jq '.'

echo ""
echo "2. 启动重放测试"
curl -X POST "$SERVICE_URL/api/replay/start" \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 50,
    "duration": 45
  }' | jq '.'

echo ""
echo "3. 等待50秒..."
sleep 50

echo ""
echo "4. 查看最终统计"
curl -s "$SERVICE_URL/api/stats" | jq '.'

echo ""
echo "5. 查看最新消息"
curl -s "$SERVICE_URL/api/messages?limit=5" | jq '.messages[] | {type: .message_type, event: .event_id}'
```

### 使用提供的测试脚本

```bash
# 已经创建了完整的测试脚本
./test_replay_api.sh https://your-service.railway.app
```

---

## 🐛 故障排查

### 问题1: 404 Not Found

**原因**: Railway还在部署新代码

**解决**:
1. 检查Railway Dashboard → Deployments
2. 等待部署完成(通常2-3分钟)
3. 确认最新的commit已部署

### 问题2: 503 Service Unavailable

**原因**: 环境变量未设置

**解决**:
1. 在Railway中设置 `UOF_USERNAME` 和 `UOF_PASSWORD`
2. 重新部署服务

### 问题3: 没有收到消息

**原因**: 服务可能还连接到生产AMQP服务器

**解决**:
- 重放消息会发送到 `global.replaymq.betradar.com`
- 您的服务需要连接到Replay服务器才能接收消息
- 参考 `docs/REPLAY-TESTING-GUIDE.md` 配置Replay模式

### 问题4: odds_changes表为空

**原因**: 可能选择的赛事没有赔率变化

**解决**: 使用推荐的测试赛事
```bash
# 这个赛事保证有丰富的赔率变化
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":50,"duration":45}'
```

---

## 📚 推荐测试赛事

| 赛事ID | 描述 | 特点 |
|--------|------|------|
| `test:match:21797788` | ⭐ 足球VAR | 丰富的赔率变化,推荐! |
| `test:match:21797805` | 足球加时赛 | 测试加时赛消息 |
| `test:match:21797815` | 足球点球 | 测试点球大战 |
| `test:match:21797802` | 网球5盘 | 测试网球规则 |

**完整列表**: https://docs.sportradar.com/uof/replay-server/uof-example-replays

---

## 💡 使用技巧

### 1. 快速验证管道

```bash
# 100倍速,30秒快速测试
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":100,"duration":30}'
```

### 2. 详细调试

```bash
# 1倍速,观察每条消息
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":1}'
```

### 3. 监控实时数据

```bash
# 在一个终端启动重放
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":20,"duration":60}'

# 在另一个终端监控
watch -n 5 "curl -s $SERVICE_URL/api/stats | jq '.'"
```

### 4. 测试不同场景

```bash
# 测试加时赛
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797805","speed":20,"duration":60}'

# 测试点球大战
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797815","speed":20,"duration":60}'
```

---

## 🎉 总结

### 新功能

✅ **4个新API端点** - 完全控制Replay测试  
✅ **自动化测试** - 无需手动运行脚本  
✅ **异步执行** - 立即返回,后台运行  
✅ **自动停止** - 设置duration自动停止  
✅ **完整参数** - 支持所有Replay参数  

### 使用流程

1. **设置环境变量** (Railway)
2. **等待部署完成** (2-3分钟)
3. **调用API启动测试**
4. **查看结果** (日志、数据库、API)

### 一键测试

```bash
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":50,"duration":45}'
```

**就这么简单!** 🚀

---

## 📖 相关文档

- **完整API文档**: `docs/REPLAY-API.md`
- **详细测试指南**: `docs/REPLAY-TESTING-GUIDE.md`
- **功能总结**: `REPLAY-FEATURE-SUMMARY.md`
- **Replay基础**: `docs/REPLAY-SERVER.md`

---

## 🆘 需要帮助?

1. 查看 Railway 部署日志
2. 运行测试脚本: `./test_replay_api.sh`
3. 查看完整文档: `docs/REPLAY-API.md`
4. 检查环境变量是否正确设置

**准备好了吗?** 现在就开始测试! 🎬

