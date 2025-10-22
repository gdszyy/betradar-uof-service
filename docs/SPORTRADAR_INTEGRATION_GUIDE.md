# SportRadar 产品接入指南

**版本**: 1.0.x  
**目标**: 快速接入 SportRadar UOF 和 Live Data 产品  
**适用对象**: 开发人员

---

## 📋 目录

1. [产品概述](#产品概述)
2. [接入准备](#接入准备)
3. [UOF (Unified Odds Feed) 接入](#uof-unified-odds-feed-接入)
4. [Live Data 接入](#live-data-接入)
5. [数据关联](#数据关联)
6. [监控与告警](#监控与告警)
7. [常见问题](#常见问题)

---

## 产品概述

### UOF (Unified Odds Feed)
**用途**: 实时赔率数据  
**协议**: AMQP (RabbitMQ)  
**数据类型**:
- `odds_change` - 赔率变化
- `bet_stop` - 投注停止
- `bet_settlement` - 投注结算
- `bet_cancel` - 投注取消
- `fixture_change` - 赛程变更

### Live Data (LD)
**用途**: 实时比赛事件数据  
**协议**: Socket (SSL, Port 2017)  
**数据类型**:
- Match events (进球、红黄牌、角球等)
- Score updates (比分更新)
- Lineups (阵容信息)
- Match info (比赛基本信息)

### 产品互补关系

```
UOF (赔率数据) + Live Data (比赛事件) = 完整的体育数据解决方案
```

---

## 接入准备

### 1. 获取 SportRadar 账户

联系 SportRadar 销售团队获取:
- **UOF 凭证**: Username + Password
- **Live Data 凭证**: Username + Password (可能与 UOF 相同)
- **Bookmaker ID**: 您的博彩商标识符

### 2. 确认产品权限

确保您的账户已开通以下权限:
- ✅ UOF - Unified Odds Feed
- ✅ Live Data - Match Events
- ✅ API Access - REST API 访问

### 3. IP 白名单配置

**Live Data 需要 IP 白名单**:
1. 获取服务器出口 IP 地址
2. 联系 SportRadar 技术支持
3. 提供 IP 地址请求加入白名单
4. 等待确认 (通常 1-2 个工作日)

---

## UOF (Unified Odds Feed) 接入

### 第一步: 了解 UOF 架构

```
SportRadar AMQP → 您的服务 → 数据库 → 业务逻辑
```

**关键概念**:
- **Producer**: 数据生产者 (如 LiveOdds, Ctrl)
- **Event**: 比赛/赛事
- **Market**: 投注市场 (如 1X2, Over/Under)
- **Outcome**: 投注结果选项

### 第二步: 配置 AMQP 连接

**连接参数**:
```
Host: stgmq.betradar.com (集成环境)
      mq.betradar.com (生产环境)
Port: 5671 (SSL)
VHost: /unifiedfeed/<bookmaker_id>
Username: <your_username>
Password: <your_password>
Exchange: unifiedfeed
```

**本项目配置**:
```go
// services/amqp_consumer.go
func (ac *AMQPConsumer) Start() error {
    // 1. 建立连接
    conn, err := amqp.DialTLS(amqpURL, tlsConfig)
    
    // 2. 创建 Channel
    channel, err := conn.Channel()
    
    // 3. 声明队列 (服务器自动命名)
    queue, err := channel.QueueDeclare("", false, true, true, false, nil)
    
    // 4. 绑定路由键
    routingKeys := []string{
        "#.odds_change.#",
        "#.bet_stop.#",
        "#.bet_settlement.#",
        // ...
    }
    
    for _, key := range routingKeys {
        channel.QueueBind(queue.Name, key, "unifiedfeed", false, nil)
    }
    
    // 5. 开始消费
    msgs, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
}
```

### 第三步: 消息处理

**消息格式**: XML

**处理流程**:
1. 接收 AMQP 消息
2. 解析 XML
3. 提取关键字段
4. 存储到数据库
5. 通知业务层

**示例代码**:
```go
// services/amqp_consumer.go
func (ac *AMQPConsumer) handleMessage(msg amqp.Delivery) {
    // 1. 解析消息类型
    messageType := extractMessageType(msg.Body)
    
    // 2. 根据类型处理
    switch messageType {
    case "odds_change":
        ac.handleOddsChange(msg.Body)
    case "bet_stop":
        ac.handleBetStop(msg.Body)
    // ...
    }
    
    // 3. 确认消息
    msg.Ack(false)
}
```

### 第四步: 数据恢复 (Recovery)

**为什么需要恢复?**
- 服务重启时丢失的消息
- 网络中断期间的消息
- Producer 宕机恢复后的消息

**恢复类型**:

1. **全量恢复 (Full Recovery)**
   ```bash
   POST /v1/{product}/recovery/initiate_request
   ```
   用于首次启动或长时间离线

2. **事件恢复 (Event Recovery)**
   ```bash
   POST /v1/{product}/odds/events/{event_id}/initiate_request
   ```
   用于特定比赛的数据恢复

**本项目实现**:
```go
// services/recovery_manager.go
func (rm *RecoveryManager) InitiateFullRecovery(producerID int, after int64) error {
    url := fmt.Sprintf("%s/v1/%s/recovery/initiate_request?after=%d&request_id=%d",
        rm.apiBaseURL, rm.product, after, producerID)
    
    req, _ := http.NewRequest("POST", url, nil)
    req.Header.Set("x-access-token", rm.accessToken)
    
    resp, err := rm.client.Do(req)
    // ...
}
```

### 第五步: 比赛订阅 (Booking)

**为什么需要订阅?**
- 默认情况下不会收到任何赔率数据
- 必须主动订阅感兴趣的比赛
- 订阅后才会收到 `odds_change` 消息

**订阅方式**:

1. **查询可订阅比赛**
   ```bash
   GET /v1/sports/en/schedules/live/schedule.xml
   ```

2. **订阅比赛**
   ```bash
   POST /v1/liveodds/booking-calendar/events/{event_id}/book
   ```

**本项目实现**:
```go
// services/auto_booking.go
func (ab *AutoBookingService) BookAllLiveMatches() ([]string, error) {
    // 1. 查询 live 比赛
    matches, err := ab.fetchLiveMatches()
    
    // 2. 过滤 bookable 比赛
    bookableMatches := filterBookable(matches)
    
    // 3. 批量订阅
    for _, match := range bookableMatches {
        ab.bookMatch(match.ID)
    }
}
```

**API 使用**:
```bash
# 自动订阅所有 bookable 比赛
curl -X POST http://your-server:8080/api/booking/auto

# 订阅单个比赛
curl -X POST http://your-server:8080/api/booking/match/sr:match:12345678
```

---

## Live Data 接入

### 第一步: 了解 Live Data 架构

```
SportRadar Socket Server → SSL 连接 → 您的服务 → 数据库
```

**关键概念**:
- **Match**: 比赛
- **Event**: 比赛事件 (进球、红牌等)
- **Sequence Number**: 消息序列号 (用于检测丢失)
- **Data Source**: 数据来源 (BC/DC/iScout)

### 第二步: 配置 Socket 连接

**连接参数**:
```
Host: livedata.betradar.com
Port: 2017
Protocol: SSL/TLS
```

**本项目配置**:
```go
// services/ld_client.go
func (ldc *LDClient) Connect() error {
    // 1. 建立 TLS 连接
    conn, err := tls.Dial("tcp", "livedata.betradar.com:2017", &tls.Config{})
    
    // 2. 发送登录消息
    loginMsg := fmt.Sprintf(`<login>
<credential>
<loginname value="%s"/>
<password value="%s"/>
</credential>
</login>`, username, password)
    
    conn.Write([]byte(loginMsg))
    
    // 3. 等待登录响应
    // 4. 开始接收消息
}
```

### 第三步: 消息订阅

**订阅比赛**:
```xml
<match matchid="944423"/>
```

**取消订阅**:
```xml
<unmatch matchid="944423"/>
```

**本项目实现**:
```go
// services/ld_client.go
func (ldc *LDClient) SubscribeMatch(matchID string) error {
    msg := fmt.Sprintf(`<match matchid="%s"/>`, matchID)
    _, err := ldc.conn.Write([]byte(msg))
    return err
}
```

**API 使用**:
```bash
# 订阅比赛
curl -X POST -H "Content-Type: application/json" \
  -d '{"match_id": "sr:match:12345678"}' \
  http://your-server:8080/api/ld/subscribe

# 取消订阅
curl -X POST -H "Content-Type: application/json" \
  -d '{"match_id": "sr:match:12345678"}' \
  http://your-server:8080/api/ld/unsubscribe
```

### 第四步: 消息处理

**消息类型**:
1. **Match Info** - 比赛基本信息
2. **Event** - 比赛事件
3. **Lineup** - 阵容信息
4. **Score** - 比分更新

**处理流程**:
```go
// services/ld_event_handler.go
func (leh *LDEventHandler) HandleEvent(event *LDEvent) error {
    // 1. 存储原始 XML
    // 2. 解析事件数据
    // 3. 检查序列号连续性
    // 4. 存储到数据库
    // 5. 发送通知
}
```

### 第五步: 序列号管理

**为什么重要?**
- 检测消息丢失
- 保证数据完整性
- 触发数据恢复

**检查逻辑**:
```go
func checkSequenceContinuity(matchID string, currentSeq int64) error {
    lastSeq := getLastSequence(matchID)
    
    if currentSeq != lastSeq + 1 {
        gap := currentSeq - lastSeq - 1
        log.Printf("Sequence gap detected: %d messages missing", gap)
        // 触发告警或恢复
    }
}
```

---

## 数据关联

### UOF 与 Live Data 的关联

**关键字段**: `event_id` (UOF) ↔ `match_id` (Live Data)

**数据库设计**:
```sql
-- UOF 数据
CREATE TABLE odds_changes (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(50) NOT NULL,  -- sr:match:12345678
    market_id INTEGER,
    odds JSONB,
    timestamp TIMESTAMPTZ
);

-- Live Data 数据
CREATE TABLE livedata_events (
    id SERIAL PRIMARY KEY,
    match_id VARCHAR(50) NOT NULL,  -- sr:match:12345678
    event_type VARCHAR(50),
    event_data JSONB,
    timestamp TIMESTAMPTZ
);

-- 关联视图
CREATE VIEW v_match_complete_data AS
SELECT 
    te.event_id,
    COUNT(DISTINCT oc.id) as odds_change_count,
    COUNT(DISTINCT le.id) as livedata_event_count,
    MAX(oc.timestamp) as last_odds_update,
    MAX(le.timestamp) as last_event_update
FROM tracked_events te
LEFT JOIN odds_changes oc ON te.event_id = oc.event_id
LEFT JOIN livedata_events le ON te.event_id = le.match_id
GROUP BY te.event_id;
```

### 数据同步策略

1. **UOF 订阅** → 自动订阅 Live Data
2. **Live Data 事件** → 检查 UOF 订阅状态
3. **定期对账** → 确保数据一致性

---

## 监控与告警

### 飞书通知集成

**通知类型**:
- ✅ 服务启动
- ✅ 连接状态变化
- ✅ 数据恢复完成
- ✅ 错误告警
- ✅ 消息统计报告
- ✅ 比赛订阅报告

**配置**:
```bash
LARK_WEBHOOK_URL=https://open.larksuite.com/open-apis/bot/v2/hook/your-webhook-id
```

### 监控指标

1. **UOF 监控**
   - Producer 状态
   - 消息处理延迟
   - 订阅比赛数量
   - 恢复请求次数

2. **Live Data 监控**
   - 连接状态
   - 序列号间隙
   - 订阅比赛数量
   - 消息接收速率

### API 端点

```bash
# 健康检查
GET /api/health

# UOF 状态
GET /api/uof/status

# Live Data 状态
GET /api/ld/status

# 触发监控报告
POST /api/monitor/trigger

# 查看订阅的比赛
GET /api/ld/matches

# 查看接收到的事件
GET /api/ld/events?match_id=sr:match:12345678
```

---

## 常见问题

### Q1: 为什么收不到赔率数据?

**A**: 检查以下几点:
1. ✅ AMQP 连接是否正常
2. ✅ 是否订阅了比赛 (使用 Booking API)
3. ✅ 比赛是否正在进行 (只有 live 比赛才有实时赔率)
4. ✅ Producer 状态是否正常

**解决方案**:
```bash
# 1. 检查连接状态
curl http://your-server:8080/api/health

# 2. 自动订阅所有 live 比赛
curl -X POST http://your-server:8080/api/booking/auto

# 3. 触发监控报告
curl -X POST http://your-server:8080/api/monitor/trigger
```

### Q2: Live Data 连接失败?

**A**: 可能原因:
1. ❌ IP 地址未加入白名单
2. ❌ 用户名或密码错误
3. ❌ 网络防火墙阻止 2017 端口

**解决方案**:
1. 联系 SportRadar 技术支持确认 IP 白名单
2. 验证凭证是否正确
3. 检查防火墙规则

### Q3: 如何处理序列号间隙?

**A**: 系统会自动检测并记录序列号间隙。

**处理流程**:
1. 检测到间隙 → 记录日志
2. 发送飞书告警
3. 继续处理后续消息 (不阻塞)
4. 定期检查间隙统计

**查询间隙统计**:
```sql
SELECT match_id, gap_count, last_gap_detected
FROM livedata_sequence_tracker
WHERE gap_count > 0
ORDER BY last_gap_detected DESC;
```

### Q4: 如何进行数据恢复?

**A**: 使用 Recovery API

**全量恢复**:
```bash
# 恢复最近 3 小时的数据
POST /v1/liveodds/recovery/initiate_request?after=<timestamp>&request_id=1
```

**事件恢复**:
```bash
# 恢复特定比赛的数据
POST /v1/liveodds/odds/events/sr:match:12345678/initiate_request
```

**本项目 API**:
```bash
# 触发全量恢复
curl -X POST http://your-server:8080/api/recovery/full

# 触发事件恢复
curl -X POST http://your-server:8080/api/recovery/event/sr:match:12345678
```

### Q5: 如何优化性能?

**A**: 性能优化建议:

1. **数据库优化**
   - 添加索引 (event_id, timestamp)
   - 定期归档历史数据
   - 使用连接池

2. **消息处理优化**
   - 批量写入数据库
   - 异步处理非关键任务
   - 使用缓存减少数据库查询

3. **网络优化**
   - 使用持久连接
   - 启用消息压缩 (如支持)
   - 监控网络延迟

---

## 接入检查清单

### UOF 接入

- [ ] 获取 UOF 凭证
- [ ] 配置 AMQP 连接
- [ ] 测试连接和消息接收
- [ ] 实现消息处理逻辑
- [ ] 配置数据恢复
- [ ] 实现比赛订阅
- [ ] 配置监控和告警
- [ ] 压力测试

### Live Data 接入

- [ ] 获取 Live Data 凭证
- [ ] 配置服务器 IP 白名单
- [ ] 配置 Socket 连接
- [ ] 测试连接和消息接收
- [ ] 实现消息处理逻辑
- [ ] 实现序列号检查
- [ ] 实现比赛订阅
- [ ] 配置监控和告警
- [ ] 压力测试

### 数据关联

- [ ] 设计数据库关联结构
- [ ] 实现 UOF ↔ LD 数据关联
- [ ] 实现数据同步策略
- [ ] 测试数据一致性

### 监控与运维

- [ ] 配置飞书通知
- [ ] 实现健康检查
- [ ] 实现监控指标
- [ ] 配置日志收集
- [ ] 制定应急预案

---

## 技术支持

**SportRadar 技术支持**:
- 邮箱: support@sportradar.com
- 文档: https://docs.sportradar.com

**本项目文档**:
- [README.md](../README.md) - 项目概述
- [FEISHU-INTEGRATION.md](./FEISHU-INTEGRATION.md) - 飞书集成
- [LIVE-DATA-INTEGRATION.md](./LIVE-DATA-INTEGRATION.md) - Live Data 详细文档

---

**版本**: 1.0.x  
**最后更新**: 2025-10-22  
**维护者**: 项目开发团队

