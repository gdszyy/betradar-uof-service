# Match Subscription Monitoring

## 概述

Match监控功能用于查询和分析已订阅的比赛(booked matches),帮助诊断为什么没有收到odds_change消息。

---

## 为什么需要监控Match订阅?

### Betradar消息发送规则

| 消息类型 | 发送条件 |
|---------|---------|
| `fixture_change` | 所有可用比赛 (booked=0 和 booked=1) |
| `alive` | 心跳消息,总是发送 |
| `snapshot_complete` | 恢复完成,总是发送 |
| **`odds_change`** | **只有订阅的比赛** (booked=1) |
| `bet_stop` | 只有订阅的比赛 (booked=1) |
| `bet_settlement` | 只有订阅的比赛 (booked=1) |

### 症状诊断

**如果您**:
- ✅ 接收 fixture_change
- ✅ 接收 alive
- ✅ 恢复成功完成
- ❌ 不接收 odds_change

**最可能的原因**: 没有订阅任何比赛 (booked=0)

---

## 使用方法

### 方法1: 使用命令行工具 (推荐)

```bash
cd /home/ubuntu/uof-go-service

# 设置环境变量
export BETRADAR_ACCESS_TOKEN="your_access_token"
export BETRADAR_MESSAGING_HOST="stgmq.betradar.com:5671"
export BETRADAR_API_BASE_URL="https://api.betradar.com"

# 运行检查
go run tools/check_booked_matches.go
```

### 方法2: 编译后运行

```bash
cd /home/ubuntu/uof-go-service

# 编译
go build -o check_booked tools/check_booked_matches.go

# 运行
./check_booked
```

---

## 输出示例

### 场景1: 有订阅的比赛

```
🔍 Checking booked matches...
Connecting to: stgmq.betradar.com:5671
Bookmaker ID: 12345
Virtual Host: /bookmaker_12345
✅ Connected to AMQP
📋 Querying booked matches (back: 6h, forward: 24h)...
📤 Sending Match List request:
<?xml version="1.0" encoding="UTF-8"?>
<matchlist hoursback="6" hoursforward="24" includeavailable="yes">
</matchlist>
⏳ Waiting for response...
📥 Received response (15234 bytes)

═══════════════════════════════════════════════════════════
📊 BOOKED MATCHES ANALYSIS
═══════════════════════════════════════════════════════════
📈 Summary:
  Total matches: 150
  Booked matches: 25
    - Pre-match (NOT_STARTED): 18
    - Live (other status): 7
  Available but not booked: 125

🎯 Booked Matches:
Match ID             Status          Home                           Away                           Start Time
------------------------------------------------------------------------------------------------------------------------
sr:match:12345678    LIVE            Manchester United              Liverpool                      2025-10-21T15:00:00Z
sr:match:23456789    LIVE            Barcelona                      Real Madrid                    2025-10-21T14:30:00Z
sr:match:34567890    NOT_STARTED     Bayern Munich                  Borussia Dortmund              2025-10-21T18:00:00Z
...

⚠️  NOTE: No live matches currently.
   Odds_change messages are typically sent for live matches.
   Pre-match odds updates are less frequent.
═══════════════════════════════════════════════════════════
```

### 场景2: 没有订阅的比赛

```
═══════════════════════════════════════════════════════════
📊 BOOKED MATCHES ANALYSIS
═══════════════════════════════════════════════════════════
📈 Summary:
  Total matches: 150
  Booked matches: 0
    - Pre-match (NOT_STARTED): 0
    - Live (other status): 0
  Available but not booked: 150

⚠️  WARNING: No booked matches found!
   This explains why you're not receiving odds_change messages.
   You need to subscribe to matches to receive odds updates.

💡 TIP: There are 150 available matches you can book.
   Use bookmatch command to subscribe to matches.
═══════════════════════════════════════════════════════════
```

---

## Match List 请求格式

### XML请求

```xml
<?xml version="1.0" encoding="UTF-8"?>
<matchlist hoursback="6" hoursforward="24" includeavailable="yes">
  <!-- 可选: 只查询特定运动 -->
  <sport sportid="1"/>  <!-- 1 = 足球 -->
</matchlist>
```

### 参数说明

| 参数 | 说明 | 示例 |
|------|------|------|
| `hoursback` | 查询过去N小时的比赛 | 6 |
| `hoursforward` | 查询未来N小时的比赛 | 24 |
| `includeavailable` | 包含可订阅但未订阅的比赛 | "yes" |
| `sport.sportid` | 运动ID (可选) | 1=足球, 2=篮球 |

### XML响应

```xml
<?xml version="1.0" encoding="UTF-8"?>
<matchlist>
  <match 
    id="sr:match:12345678" 
    booked="1" 
    sportid="1" 
    startdate="2025-10-21T15:00:00Z">
    <status id="1" name="LIVE"/>
    <hometeam name="Manchester United"/>
    <awayteam name="Liverpool"/>
  </match>
  
  <match 
    id="sr:match:23456789" 
    booked="0" 
    sportid="1" 
    startdate="2025-10-21T16:00:00Z">
    <status id="0" name="NOT_STARTED"/>
    <hometeam name="Arsenal"/>
    <awayteam name="Chelsea"/>
  </match>
</matchlist>
```

### 关键字段

| 字段 | 值 | 说明 |
|------|---|------|
| `booked` | "1" | 已订阅 → 会收到odds_change |
| `booked` | "0" | 未订阅 → 不会收到odds_change |
| `status.name` | "NOT_STARTED" | Pre-match比赛 |
| `status.name` | "LIVE" | Live比赛 (odds更新频繁) |
| `status.name` | 其他 | Live比赛的不同状态 |

---

## 订阅比赛

### 方法1: 通过AMQP发送订阅请求

```xml
<?xml version="1.0" encoding="UTF-8"?>
<bookmatch matchid="sr:match:12345678"/>
```

### 方法2: 通过Betradar管理界面

1. 登录 https://developer.sportradar.com/
2. 进入Match Management
3. 选择要订阅的比赛
4. 点击"Book"按钮

### 方法3: 联系Sportradar

如果您的账户类型不支持自助订阅,联系Sportradar技术支持。

---

## 故障排查

### 问题1: 连接失败

```
Failed to connect to AMQP: dial tcp: lookup stgmq.betradar.com: no such host
```

**解决**: 检查网络连接和MESSAGING_HOST配置

### 问题2: 认证失败

```
Failed to connect to AMQP: Exception (403) Reason: "ACCESS_REFUSED"
```

**解决**: 检查ACCESS_TOKEN是否正确

### 问题3: 超时

```
timeout waiting for response
```

**可能原因**:
1. Match List请求的routing key不正确
2. AMQP服务器不支持Match List查询
3. 需要使用不同的通信方式

**解决**: 联系Sportradar确认Match List API的使用方式

---

## 定期监控

### 在服务中集成

可以在AMQP Consumer中集成Match Monitor,定期查询订阅状态:

```go
// 在AMQPConsumer.Start()中
monitor := services.NewMatchMonitor(cfg, channel)

// 每小时查询一次
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()
    
    for range ticker.C {
        response, err := monitor.QueryBookedMatches(6, 24)
        if err != nil {
            log.Printf("Failed to query booked matches: %v", err)
            continue
        }
        monitor.AnalyzeBookedMatches(response)
    }
}()
```

---

## 参考文档

- [Match List - Sportradar Docs](https://docs.sportradar.com/live-data/introduction/system-communication/xml-messages-sent-from-the-client-system/match-list)
- [Book Match - Sportradar Docs](https://docs.sportradar.com/live-data/introduction/system-communication/xml-messages-sent-from-the-client-system/book-match)

---

## 总结

### 使用场景

1. **诊断**: 为什么没有收到odds_change消息?
2. **监控**: 当前订阅了哪些比赛?
3. **分析**: 有多少live比赛正在进行?

### 关键指标

- **Booked matches**: 订阅的比赛数量
- **Live matches**: 进行中的比赛数量
- **Available matches**: 可订阅但未订阅的比赛

### 建议

- 至少订阅一些比赛才能接收odds_change
- Live比赛的odds更新频率远高于pre-match
- 定期监控订阅状态,确保服务正常工作

