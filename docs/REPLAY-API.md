# Replay API 文档

## 概述

服务提供了4个Replay API端点,用于通过HTTP请求控制重放测试。

**前提条件**: 需要在环境变量中设置 `UOF_USERNAME` 和 `UOF_PASSWORD`

---

## API端点

### 1. 启动重放

**端点**: `POST /api/replay/start`

**描述**: 启动一个赛事的重放测试

**请求体**:
```json
{
  "event_id": "test:match:21797788",
  "speed": 20,
  "duration": 60,
  "node_id": 1,
  "max_delay": 10000,
  "use_replay_timestamp": true
}
```

**参数说明**:

| 参数 | 类型 | 必需 | 默认值 | 说明 |
|------|------|------|--------|------|
| `event_id` | string | ✅ | - | 赛事ID,例如 `test:match:21797788` |
| `speed` | int | ❌ | 20 | 重放速度倍数(1-100) |
| `duration` | int | ❌ | 0 | 运行时长(秒),0表示不自动停止 |
| `node_id` | int | ❌ | 1 | 节点ID,用于多会话隔离 |
| `max_delay` | int | ❌ | 10000 | 消息间最大延迟(毫秒) |
| `use_replay_timestamp` | bool | ❌ | false | 是否使用当前时间戳 |

**响应**:
```json
{
  "status": "accepted",
  "message": "Replay request accepted and processing",
  "event_id": "test:match:21797788",
  "speed": 20,
  "node_id": 1,
  "duration": 60,
  "time": 1761024000
}
```

**示例**:

```bash
# 基本用法
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 20,
    "duration": 60
  }'

# 高速测试(100倍速,30秒)
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 100,
    "duration": 30,
    "node_id": 1
  }'

# 慢速调试(1倍速,不自动停止)
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 1,
    "node_id": 1
  }'
```

---

### 2. 停止重放

**端点**: `POST /api/replay/stop`

**描述**: 停止当前正在运行的重放

**请求体**: 无

**响应**:
```json
{
  "status": "success",
  "message": "Replay stopped",
  "time": 1761024000
}
```

**示例**:

```bash
curl -X POST https://your-service.railway.app/api/replay/stop
```

---

### 3. 查看重放状态

**端点**: `GET /api/replay/status`

**描述**: 获取当前重放的状态

**响应**: XML格式

```xml
<?xml version="1.0" encoding="UTF-8"?>
<player_status status="PLAYING" last_msg_from_event="test:match:21797788"/>
```

**可能的状态**:
- `PLAYING` - 正在重放
- `STOPPED` - 已停止
- `SETTING_UP` - 正在准备中

**示例**:

```bash
curl https://your-service.railway.app/api/replay/status
```

---

### 4. 列出重放列表

**端点**: `GET /api/replay/list`

**描述**: 列出当前重放队列中的赛事

**响应**: XML格式

```xml
<?xml version="1.0" encoding="UTF-8"?>
<replay_events>
  <event id="test:match:21797788"/>
</replay_events>
```

**示例**:

```bash
curl https://your-service.railway.app/api/replay/list
```

---

## 使用流程

### 典型工作流

```bash
SERVICE_URL="https://your-service.railway.app"

# 1. 启动重放(20倍速,60秒)
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 20,
    "duration": 60
  }'

# 2. 查看状态
curl $SERVICE_URL/api/replay/status

# 3. 等待一段时间,查看数据
sleep 30

# 4. 检查统计
curl $SERVICE_URL/api/stats

# 5. 查看最新消息
curl "$SERVICE_URL/api/messages?limit=20"

# 6. 手动停止(如果没有设置duration)
curl -X POST $SERVICE_URL/api/replay/stop
```

---

## 推荐测试场景

### 场景1: 快速验证管道

```bash
# 100倍速,30秒快速测试
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 100,
    "duration": 30,
    "node_id": 1
  }'
```

### 场景2: 详细调试

```bash
# 1倍速,慢速观察每条消息
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 1,
    "node_id": 1
  }'
```

### 场景3: 测试特定场景

```bash
# 测试加时赛
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797805",
    "speed": 20,
    "duration": 60
  }'

# 测试点球大战
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797815",
    "speed": 20,
    "duration": 60
  }'
```

---

## 验证结果

### 1. 查看服务日志

在Railway Dashboard → Logs中应该看到:

```
🎬 Starting replay via API: event=test:match:21797788, speed=20x, node_id=1
✅ Replay started successfully: test:match:21797788
⏱️  Replay will run for 60 seconds
🛑 Replay stopped after 60 seconds
```

### 2. 查询数据库

```bash
# 使用psql或API查询
curl "$SERVICE_URL/api/stats"

# 应该看到:
{
  "total_messages": 增加,
  "odds_changes": 有数据,
  "bet_stops": 有数据,
  "bet_settlements": 有数据
}
```

### 3. 查看WebSocket UI

打开 `https://your-service.railway.app/` 应该能看到实时消息流。

---

## 错误处理

### 错误: "Replay client not configured"

**原因**: 环境变量未设置

**解决**:
```bash
# 在Railway中设置环境变量
UOF_USERNAME=your_username
UOF_PASSWORD=your_password
```

### 错误: "event_id is required"

**原因**: 请求体缺少event_id

**解决**:
```bash
# 确保请求体包含event_id
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id": "test:match:21797788"}'
```

### 错误: "Failed to add event"

**原因**: 
- 赛事ID不存在
- API凭证无效
- 赛事不满足重放条件(必须48小时前结束)

**解决**: 使用推荐的测试赛事ID

---

## 与其他API的集成

### 完整测试流程

```bash
SERVICE_URL="https://your-service.railway.app"

echo "1. 获取初始统计"
curl "$SERVICE_URL/api/stats"

echo "2. 启动重放"
curl -X POST $SERVICE_URL/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "test:match:21797788",
    "speed": 50,
    "duration": 45
  }'

echo "3. 等待重放运行"
sleep 50

echo "4. 获取最终统计"
curl "$SERVICE_URL/api/stats"

echo "5. 查看最新消息"
curl "$SERVICE_URL/api/messages?limit=10"

echo "6. 查看跟踪的赛事"
curl "$SERVICE_URL/api/events"
```

---

## 监控脚本

创建一个监控脚本 `monitor_replay.sh`:

```bash
#!/bin/bash

SERVICE_URL="$1"
INTERVAL="${2:-5}"

if [ -z "$SERVICE_URL" ]; then
    echo "Usage: $0 <service_url> [interval_seconds]"
    exit 1
fi

echo "Monitoring replay at $SERVICE_URL (interval: ${INTERVAL}s)"
echo "Press Ctrl+C to stop"
echo ""

while true; do
    clear
    echo "=== Replay Monitor ==="
    echo "Time: $(date)"
    echo ""
    
    echo "--- Replay Status ---"
    curl -s "$SERVICE_URL/api/replay/status" | head -n 5
    echo ""
    
    echo "--- Service Stats ---"
    curl -s "$SERVICE_URL/api/stats" | jq '.'
    echo ""
    
    sleep $INTERVAL
done
```

使用:
```bash
chmod +x monitor_replay.sh
./monitor_replay.sh https://your-service.railway.app 5
```

---

## 最佳实践

### 1. 开发测试
- 使用 `speed=100`, `duration=30` 快速验证
- 使用 `node_id=1` 避免冲突

### 2. 调试问题
- 使用 `speed=1` 慢速观察
- 不设置 `duration`,手动控制停止

### 3. 性能测试
- 使用 `speed=50`, `duration=120`
- 监控数据库和内存使用

### 4. 自动化测试
- 使用 `duration` 自动停止
- 通过API验证结果

---

## 相关文档

- **完整测试指南**: `docs/REPLAY-TESTING-GUIDE.md`
- **Replay功能总结**: `REPLAY-FEATURE-SUMMARY.md`
- **Replay Server基础**: `docs/REPLAY-SERVER.md`

---

## 快速参考

```bash
# 启动重放(推荐设置)
curl -X POST https://your-service.railway.app/api/replay/start \
  -H "Content-Type: application/json" \
  -d '{"event_id":"test:match:21797788","speed":20,"duration":60}'

# 停止重放
curl -X POST https://your-service.railway.app/api/replay/stop

# 查看状态
curl https://your-service.railway.app/api/replay/status

# 查看统计
curl https://your-service.railway.app/api/stats
```

