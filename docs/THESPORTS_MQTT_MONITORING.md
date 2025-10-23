# The Sports MQTT 监控指南

**版本**: v1.0.7  
**更新日期**: 2025-10-23

---

## 📊 监控概述

本文档说明如何监控 The Sports MQTT 连接状态和消息接收情况。

---

## 🔍 日志监控

### 连接阶段日志

#### 成功连接
```
[TheSports] 🔌 Connecting to The Sports MQTT...
[TheSports] 📡 Subscribing to football/live/#...
[TheSports] ✅ Successfully subscribed to football/live/#
[TheSports] 📡 Subscribing to basketball/live/#...
[TheSports] ✅ Successfully subscribed to basketball/live/#
[TheSports] 📡 Subscribing to esports/live/# (experimental)...
[TheSports] ✅ Successfully subscribed to esports/live/#
[TheSports] ✅ Connected to The Sports MQTT successfully
```

#### 部分失败
```
[TheSports] 🔌 Connecting to The Sports MQTT...
[TheSports] 📡 Subscribing to football/live/#...
[TheSports] ✅ Successfully subscribed to football/live/#
[TheSports] 📡 Subscribing to basketball/live/#...
[TheSports] ❌ Failed to subscribe to basketball: not authorized
[TheSports] ℹ️  Basketball MQTT may not be available, continuing...
[TheSports] 📡 Subscribing to esports/live/# (experimental)...
[TheSports] ❌ Failed to subscribe to esports: topic not found
[TheSports] ℹ️  Esports MQTT may not be available, will use REST API only
[TheSports] ✅ Connected to The Sports MQTT successfully
```

### 消息接收日志

#### 足球消息
```
[TheSports] 📨 Received message on topic: football/live/12345 (1234 bytes)
[TheSports] 🏆 Sport type: football
[TheSports] 📝 Processing match data for match ID: 12345
[TheSports] 💾 Saved match to database: 12345
```

#### 篮球消息
```
[TheSports] 📨 Received message on topic: basketball/live/67890 (2345 bytes)
[TheSports] 🏆 Sport type: basketball
[TheSports] 📝 Processing basketball match data for match ID: 67890
[TheSports] 💾 Saved basketball match to database: 67890
```

#### 电竞消息 (实验性)
```
[TheSports] 📨 Received message on topic: esports/live/11111 (3456 bytes)
[TheSports] 🏆 Sport type: esports
[TheSports] 🎮 ESPORTS MESSAGE RECEIVED! Topic: esports/live/11111
[TheSports] 📝 Processing esports match data for match ID: 11111
[TheSports] 💾 Saved esports match to database: 11111
```

---

## 📈 监控指标

### 1. 连接状态监控

**API 端点**: `GET /api/thesports/status`

**响应示例**:
```json
{
  "connected": true,
  "subscriptions": {
    "football": "subscribed",
    "basketball": "subscribed",
    "esports": "failed"
  },
  "last_message_time": "2025-10-23T12:34:56Z",
  "message_count": {
    "football": 1234,
    "basketball": 567,
    "esports": 0
  }
}
```

### 2. 消息统计

**日志查询**:
```bash
# 查看所有 The Sports 日志
railway logs | grep "\[TheSports\]"

# 查看连接日志
railway logs | grep "\[TheSports\].*Connect"

# 查看消息接收日志
railway logs | grep "\[TheSports\].*Received message"

# 查看足球消息
railway logs | grep "\[TheSports\].*football"

# 查看篮球消息
railway logs | grep "\[TheSports\].*basketball"

# 查看电竞消息
railway logs | grep "\[TheSports\].*esports"

# 查看电竞消息 (特殊标记)
railway logs | grep "\[TheSports\].*ESPORTS MESSAGE"
```

### 3. 错误监控

**常见错误**:

#### 认证失败
```
[TheSports] ❌ Failed to connect: authentication failed
```
**解决方案**: 检查 `THESPORTS_USERNAME` 和 `THESPORTS_SECRET` 环境变量

#### 订阅失败
```
[TheSports] ❌ Failed to subscribe to basketball: not authorized
```
**解决方案**: 检查账户是否有篮球数据订阅权限

#### 主题不存在
```
[TheSports] ❌ Failed to subscribe to esports: topic not found
```
**解决方案**: 电竞 MQTT 可能不可用,使用 REST API 替代

---

## 🔧 故障排查

### 问题 1: 无法连接到 MQTT

**症状**:
```
[TheSports] ❌ Failed to connect: connection refused
```

**排查步骤**:
1. 检查网络连接
2. 检查防火墙设置
3. 验证 MQTT 服务器地址: `ssl://mq.thesports.com:443`
4. 检查 TLS 证书

**解决方案**:
```bash
# 测试网络连接
curl -v https://mq.thesports.com

# 检查环境变量
railway variables | grep THESPORTS
```

---

### 问题 2: 订阅成功但没有消息

**症状**:
```
[TheSports] ✅ Successfully subscribed to football/live/#
# 但之后没有任何消息日志
```

**排查步骤**:
1. 检查是否有正在进行的比赛
2. 验证订阅的 topic 是否正确
3. 检查消息处理器是否正常

**解决方案**:
```bash
# 查看最近的消息
railway logs | grep "Received message" | tail -20

# 手动触发测试
curl -X POST https://your-app.railway.app/api/thesports/connect
```

---

### 问题 3: 电竞消息未收到

**症状**:
```
[TheSports] ✅ Successfully subscribed to esports/live/#
# 但从未看到 "ESPORTS MESSAGE RECEIVED" 日志
```

**可能原因**:
1. The Sports API 可能不提供电竞 MQTT 实时数据
2. 电竞使用不同的 topic 格式
3. 账户没有电竞 MQTT 权限

**验证方法**:
```bash
# 检查是否收到任何电竞消息
railway logs | grep "ESPORTS MESSAGE"

# 如果没有,使用 REST API 获取电竞数据
curl https://your-app.railway.app/api/thesports/esports/today
```

---

## 📊 实时监控命令

### Railway 实时日志
```bash
# 实时查看所有日志
railway logs --follow

# 实时查看 The Sports 日志
railway logs --follow | grep "\[TheSports\]"

# 实时查看消息接收
railway logs --follow | grep "Received message"

# 实时查看电竞消息
railway logs --follow | grep "ESPORTS MESSAGE"
```

### 本地开发监控
```bash
# 启动服务并查看日志
go run main.go 2>&1 | grep "\[TheSports\]"

# 只看连接日志
go run main.go 2>&1 | grep "\[TheSports\].*Connect"

# 只看消息日志
go run main.go 2>&1 | grep "\[TheSports\].*Received"
```

---

## 📝 日志级别说明

### 图标含义

| 图标 | 含义 | 级别 |
|------|------|------|
| 🔌 | 连接操作 | INFO |
| 📡 | 订阅操作 | INFO |
| ✅ | 成功 | INFO |
| ❌ | 失败 | ERROR |
| ℹ️ | 提示信息 | INFO |
| 📨 | 消息接收 | DEBUG |
| 🏆 | 运动类型 | DEBUG |
| 🎮 | 电竞消息 | INFO |
| 📝 | 数据处理 | DEBUG |
| 💾 | 数据保存 | DEBUG |
| ⚠️ | 警告 | WARN |

---

## 🎯 监控最佳实践

### 1. 定期检查连接状态
```bash
# 每小时检查一次
*/60 * * * * curl https://your-app.railway.app/api/thesports/status
```

### 2. 监控消息接收频率
```bash
# 统计最近1小时的消息数
railway logs --since 1h | grep "Received message" | wc -l
```

### 3. 设置告警

**飞书告警** (已集成):
- 连接成功时发送通知
- 连接失败时发送告警
- 长时间无消息时发送警告

### 4. 数据验证

**定期验证数据完整性**:
```sql
-- 检查最近1小时的数据量
SELECT COUNT(*) FROM ld_matches 
WHERE updated_at > NOW() - INTERVAL '1 hour';

-- 检查各运动类型的数据量
SELECT sport_type, COUNT(*) 
FROM ld_matches 
GROUP BY sport_type;
```

---

## 🔍 调试技巧

### 1. 增加日志详细度

修改 `services/thesports_client.go`:
```go
// 在 handleMessage 开头添加
log.Printf("[TheSports] 🐛 DEBUG: Full payload: %s", string(payload))
```

### 2. 测试特定 Topic

```go
// 手动订阅测试
c.mqttClient.Subscribe("esports/live/12345", 1)
```

### 3. 模拟消息

```go
// 手动触发消息处理
c.handleMessage("esports/live/test", []byte(`{"test": "data"}`))
```

---

## 📚 相关文档

- [The Sports SDK 文档](../thesports/README.md)
- [MQTT WebSocket 文档](../thesports/MQTT_WEBSOCKET.md)
- [篮球和电竞支持](../thesports/BASKETBALL_ESPORTS.md)
- [API 文档](API.md) (待创建)

---

## 🆘 获取帮助

如果遇到问题:

1. 查看本文档的故障排查部分
2. 检查 Railway 日志
3. 查看 The Sports API 文档
4. 联系 The Sports 技术支持

---

**文档版本**: v1.0.7  
**最后更新**: 2025-10-23

