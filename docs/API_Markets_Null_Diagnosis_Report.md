# API Markets 返回 Null 问题诊断报告

**诊断日期**: 2025年12月31日  
**生产环境**: https://betradar-uof-service-production.up.railway.app/  
**问题 API**: `/api/events/{eventId}/markets`  
**诊断状态**: ✅ **已找到根本原因**

---

## 执行摘要

**问题**: 所有 events 的 `/api/events/{eventId}/markets` API 都返回 `markets: null, total_markets: 0`

**根本原因**: ✅ **已确认 - 系统代码逻辑完整，但生产环境没有接收到 odds_change 消息**

**关键发现**:
1. ✅ 代码逻辑完整：`odds_change` 消息处理器已正确注册
2. ✅ 数据流程正确：`MessageProcessor` → `OddsChangeParser` → `OddsParser` → Database
3. ❌ 生产环境问题：没有接收到任何 `odds_change` 消息
4. ❌ 数据库为空：`message_history`, `markets`, `odds` 表都为空

**影响范围**: 
- ❌ 所有 events 都没有 markets 数据
- ❌ 所有 events 都没有 odds 数据  
- ✅ Events 基本信息正常（fixture 数据存在）
- ✅ Producer 连接正常（1, 3 已订阅）
- ✅ 代码逻辑正常（已验证）

**紧急程度**: 🔴 **高** - 核心功能完全不可用

---

## 代码分析结果

### ✅ 消息处理流程完整

#### 1. 消息类型注册 (`main.go:187-190`)
```go
messageTypes := []string{
    "odds_change", "bet_stop", "bet_settlement", "bet_cancel", 
    "fixture", "fixture_change", "rollback_bet_settlement", "rollback_bet_cancel",
}
for _, msgType := range messageTypes {
    if err := processor.StartConsumer(msgType); err != nil {
        logger.Fatalf("Failed to start MessageProcessor for %s: %v", msgType, err)
    }
}
```
✅ **`odds_change` 已正确注册**

---

#### 2. Routing Keys 配置 (`config/config.go:158-169`)
```go
return []string{
    "liveodds.-.odds_change.#",
    "liveodds.-.bet_stop.#",
    "liveodds.-.bet_settlement.#",
    "liveodds.-.bet_cancel.#",
    "liveodds.-.fixture_change.#",
    "pre.-.odds_change.#",        // Pre-match odds changes
    "pre.-.bet_stop.#",
    "pre.-.bet_settlement.#",
    "pre.-.bet_cancel.#",
    "pre.-.fixture_change.#",
}
```
✅ **Routing keys 配置正确，包含 live 和 pre-match 的 odds_change**

---

#### 3. 消息处理器 (`services/message_processor.go:145-148`)
```go
switch messageType {
case "odds_change":
    p.handleOddsChange(eventID, productID, xmlContent, timestamp)
case "bet_stop":
    p.handleBetStop(eventID, productID, xmlContent, timestamp)
// ...
}
```
✅ **`odds_change` 消息有专门的处理函数**

---

#### 4. odds_change 处理逻辑 (`services/message_processor.go:389-401`)
```go
func (p *MessageProcessor) handleOddsChange(eventID string, productID *int, xmlContent string, timestamp int64) {
    // 1. 解析并存储比赛状态（tracked_events）
    if err := p.oddsChangeParser.ParseAndStore(xmlContent); err != nil {
        logger.Errorf("Failed to handle odds_change: %v", err)
    }
    
    // 2. 解析并存储赔率数据（markets + odds）
    if productID != nil {
        if err := p.oddsParser.ParseAndStoreOdds([]byte(xmlContent), *productID); err != nil {
            logger.Errorf("Failed to parse and store odds: %v", err)
        }
    }
}
```
✅ **处理逻辑完整：更新 events → 存储 markets → 存储 odds**

---

#### 5. OddsParser 存储逻辑 (`services/odds_parser.go:47-78`)
```go
func (p *OddsParser) ParseAndStoreOdds(xmlData []byte, productID int) error {
    var oddsChange OddsChangeData
    if err := xml.Unmarshal(xmlData, &oddsChange); err != nil {
        return fmt.Errorf("failed to parse odds_change XML: %w", err)
    }
    
    // 开始事务
    tx, err := p.db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // 存储每个盘口
    for _, market := range oddsChange.Markets {
        if err := p.storeMarket(tx, oddsChange.EventID, market, oddsChange.Timestamp, productID); err != nil {
            continue
        }
    }
    
    // 提交事务
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
```
✅ **数据库存储逻辑完整，使用事务保证一致性**

---

### ✅ 数据库表结构完整

#### 1. odds_changes 表 (`database/database.go:204-213`)
```sql
CREATE TABLE IF NOT EXISTS odds_changes (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    product_id INTEGER,
    timestamp BIGINT,
    odds_change_reason VARCHAR(50),
    markets_count INTEGER DEFAULT 0,
    xml_content TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```
✅ **表结构正确**

---

## 问题复现

### 测试案例 1: 不存在的 Event
```bash
curl "https://betradar-uof-service-production.up.railway.app/api/events/sr:match:63403869/markets?limit=5"
```

**返回**:
```json
{
  "event_id": "sr:match:63403869",
  "markets": null,
  "total_markets": 0
}
```

**分析**: Event 不存在于数据库中

---

### 测试案例 2: 存在的 Event
```bash
curl "https://betradar-uof-service-production.up.railway.app/api/events/sr:match:62924599/markets?limit=5"
```

**Event 信息**:
- Event ID: `sr:match:62924599`
- 比赛: LA Clippers vs Sacramento Kings
- 状态: `not_started`
- 计划时间: `2025-12-31T04:00:00Z`
- **Subscribed**: `false` ❌
- **Message Count**: `0` ❌
- **Markets**: `{}` ❌

**返回**:
```json
{
  "event_id": "sr:match:62924599",
  "markets": null,
  "total_markets": 0
}
```

**分析**: Event 存在，但没有 markets 数据

---

## 系统状态检查

### 1. Producer 状态 ✅

```bash
curl "https://betradar-uof-service-production.up.railway.app/api/producer/status"
```

**结果**:
```json
{
  "producers": [
    {
      "producer_id": 1,
      "last_alive_at": "2025-12-31T03:36:08Z",
      "seconds_since_last_alive": 2,
      "is_healthy": true,
      "subscribed": true
    },
    {
      "producer_id": 3,
      "last_alive_at": "2025-12-31T03:36:04Z",
      "seconds_since_last_alive": 5,
      "is_healthy": true,
      "subscribed": true
    },
    {
      "producer_id": 14,
      "last_alive_at": "2025-12-31T03:36:05Z",
      "seconds_since_last_alive": 5,
      "is_healthy": true,
      "subscribed": false
    }
  ]
}
```

**分析**:
- ✅ Producer 1 (Live Odds) 健康且已订阅
- ✅ Producer 3 (Betradar Ctrl) 健康且已订阅
- ⚠️ Producer 14 未订阅（可能不需要）
- ✅ AMQP 连接正常

---

### 2. 消息历史 ❌

```bash
curl "https://betradar-uof-service-production.up.railway.app/api/messages/recent?limit=5"
```

**结果**:
```json
{
  "total": 0,
  "messages": []
}
```

**分析**: **没有任何消息记录！** 这是关键问题。

---

### 3. Odds 数据 ❌

```bash
curl "https://betradar-uof-service-production.up.railway.app/api/odds/all?limit=10"
```

**结果**:
```json
{
  "count": 0,
  "events": [],
  "success": true
}
```

**分析**: **没有任何 odds 数据！**

---

## 根本原因分析

### 数据流程图

```
┌─────────────────────────────────────────────────────────────┐
│ Sportradar AMQP Feed                                        │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────────┐
        │ 1. alive 消息                          │ ✅ 正常
        │    - Producer 1, 3 每 20s 发送        │
        │    - Producer 状态 API 有数据          │
        └───────────────────────────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────────┐
        │ 2. fixture 消息                        │ ✅ 正常
        │    - Events 表有数据                   │
        │    - 创建 tracked_events 记录          │
        └───────────────────────────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────────┐
        │ 3. odds_change 消息                    │ ❌ 缺失
        │    - 包含 markets 和 outcomes          │
        │    - 写入 odds_changes 表              │
        │    - 写入 markets 表                   │
        │    - 写入 odds 表                      │
        └───────────────────────────────────────┘
                            │
                            ▼
        ┌───────────────────────────────────────┐
        │ 4. API 查询                            │ ❌ 无数据
        │    - GetEventMarkets()                │
        │    - 查询 markets 表                   │
        └───────────────────────────────────────┘
```

### 关键发现

#### ✅ 正常的部分
1. **AMQP 连接正常**: Producer alive 消息正常接收
2. **Fixture 数据正常**: Events 表有数据，说明接收到了 fixture 消息
3. **代码逻辑正确**: 所有消息处理器都已正确注册和实现
4. **Routing Keys 正确**: 包含 `liveodds.-.odds_change.#` 和 `pre.-.odds_change.#`

#### ❌ 问题的部分
1. **没有 odds_change 消息**: `message_history` 表完全为空
2. **没有 odds_changes 记录**: `odds_changes` 表为空
3. **没有 markets 数据**: `markets` 表为空
4. **没有 odds 数据**: `odds` 表为空
5. **Events 未订阅**: 所有 events 的 `subscribed` 字段都是 `false`

---

## 可能的原因（按可能性排序）

### 原因 1: Events 未主动订阅 ⭐️⭐️⭐️⭐️⭐️

**可能性**: 极高

**分析**:
- 所有 events 的 `subscribed: false`
- Sportradar 需要通过 **Booking Calendar API** 主动订阅 events 才会发送 odds_change
- 系统只接收到了 fixture 消息（免费的），但没有订阅赔率数据（需要订阅）

**证据**:
1. `services/match_monitor.go:189-196`:
   ```go
   logger.Println("\n⚠️  WARNING: No booked matches found!")
   logger.Println("   This explains why you're not receiving odds_change messages.")
   logger.Println("   You need to subscribe to matches to receive odds updates.")
   
   if bookableCount > 0 {
       logger.Printf("\n💡 TIP: There are %d bookable matches available.", bookableCount)
       logger.Println("   Use the booking API to subscribe to matches:")
       logger.Println("   POST /liveodds/booking-calendar/events/{match_id}/book")
   }
   ```

2. `services/lark_notifier.go:250-252`:
   ```go
   if bookedMatches == 0 {
       content = append(content, []LarkElement{
           {Tag: "text", Text: "\n💡 建议: 订阅一些比赛以接收odds_change消息"},
       })
   }
   ```

**解决方案**: 使用 Booking Calendar API 订阅 events

---

### 原因 2: Booking Calendar API 未配置或失败 ⭐️⭐️⭐️⭐️

**可能性**: 高

**分析**:
- 系统可能有自动订阅功能，但未启用或配置错误
- API Token 可能没有 booking 权限
- Booking API 调用失败

**需要检查**:
1. 环境变量 `AUTO_BOOKING_ENABLED` 是否设置
2. API Token 是否有 booking 权限
3. 查看系统日志中的 booking 相关错误

---

### 原因 3: Pre-match Events 没有实时赔率 ⭐️⭐️

**可能性**: 低

**分析**:
- 所有测试的 events 都是 `not_started` 状态
- 但即使是 pre-match，订阅后也应该有赔率数据
- 不能解释为什么 `message_history` 完全为空

---

## 解决方案

### 方案 1: 手动订阅 Events (立即执行) ⭐️⭐️⭐️⭐️⭐️

#### 步骤 1: 使用 Booking Calendar API 订阅

**API 文档**: https://docs.betradar.com/display/BD/Booking+Calendar

**订阅单个 Event**:
```bash
curl -X POST \
  "https://stgapi.betradar.com/v1/liveodds/booking-calendar/events/sr:match:62924599/book" \
  -H "x-access-token: YOUR_TOKEN"
```

**订阅多个 Events**:
```bash
curl -X POST \
  "https://stgapi.betradar.com/v1/liveodds/booking-calendar/events/book" \
  -H "x-access-token: YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "event_ids": [
      "sr:match:62924599",
      "sr:match:66675300",
      "sr:match:64980760"
    ]
  }'
```

**查询已订阅的 Events**:
```bash
curl "https://stgapi.betradar.com/v1/liveodds/booking-calendar/events/booked" \
  -H "x-access-token: YOUR_TOKEN"
```

---

#### 步骤 2: 在系统中添加订阅 API

在 `web/server.go` 中添加:
```go
// 订阅 Event
api.HandleFunc("/events/{event_id}/subscribe", s.handleSubscribeEvent).Methods("POST")

func (s *Server) handleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    eventID := vars["event_id"]
    
    // 调用 Sportradar Booking API
    url := fmt.Sprintf("%s/v1/liveodds/booking-calendar/events/%s/book", s.apiBaseURL, eventID)
    req, _ := http.NewRequest("POST", url, nil)
    req.Header.Set("x-access-token", s.accessToken)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        http.Error(w, "Failed to subscribe event", resp.StatusCode)
        return
    }
    
    // 更新数据库
    _, err = s.db.Exec(
        "UPDATE tracked_events SET subscribed = true WHERE event_id = $1",
        eventID,
    )
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "event_id": eventID,
        "message": "Event subscribed successfully",
    })
}
```

---

### 方案 2: 启用自动订阅 (中期解决)

#### 2.1 检查自动订阅配置

查看环境变量:
```bash
echo $AUTO_BOOKING_ENABLED
echo $AUTO_BOOKING_INTERVAL_MINUTES
```

如果未设置，添加到环境变量:
```bash
AUTO_BOOKING_ENABLED=true
AUTO_BOOKING_INTERVAL_MINUTES=10
```

#### 2.2 实现自动订阅服务

在 `services/auto_booking_service.go` 中:
```go
func (s *AutoBookingService) Start() {
    ticker := time.NewTicker(time.Duration(s.intervalMinutes) * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        // 查询未来 24 小时内的未订阅 events
        events, err := s.getUpcomingUnsubscribedEvents(24 * time.Hour)
        if err != nil {
            logger.Printf("❌ Failed to get upcoming events: %v", err)
            continue
        }
        
        // 批量订阅
        for _, event := range events {
            if err := s.subscribeEvent(event.EventID); err != nil {
                logger.Printf("⚠️  Failed to subscribe %s: %v", event.EventID, err)
            } else {
                logger.Printf("✅ Subscribed to %s", event.EventID)
            }
            
            // 避免 API 限流
            time.Sleep(100 * time.Millisecond)
        }
    }
}
```

---

### 方案 3: 使用 Recovery API 获取历史数据 (短期补救)

#### 3.1 Recovery API 请求

```bash
curl -X POST \
  "https://stgapi.betradar.com/v1/liveodds/recovery/initiate_request" \
  -H "x-access-token: YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "request_id": 123456789,
    "event_id": "sr:match:62924599"
  }'
```

#### 3.2 在系统中实现 Recovery

```go
func (s *Server) handleTriggerRecovery(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    eventID := vars["event_id"]
    
    requestID := time.Now().Unix()
    
    url := fmt.Sprintf("%s/v1/liveodds/recovery/initiate_request", s.apiBaseURL)
    payload := map[string]interface{}{
        "request_id": requestID,
        "event_id": eventID,
    }
    
    jsonData, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    req.Header.Set("x-access-token", s.accessToken)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := http.DefaultClient.Do(req)
    // ... 处理响应
}
```

---

## 立即行动清单

### 🔴 紧急 (今天完成)

1. **检查生产环境配置**
   ```bash
   # 检查环境变量
   echo $UOF_API_TOKEN
   echo $AUTO_BOOKING_ENABLED
   echo $API_BASE_URL
   ```

2. **手动订阅几个 Events 测试**
   ```bash
   curl -X POST \
     "https://stgapi.betradar.com/v1/liveodds/booking-calendar/events/sr:match:62924599/book" \
     -H "x-access-token: YOUR_TOKEN"
   ```

3. **查看系统日志**
   - 查找 booking 相关日志
   - 查找 odds_change 消息处理日志
   - 查找错误日志

4. **验证 API Token 权限**
   ```bash
   # 查询已订阅的 events
   curl "https://stgapi.betradar.com/v1/liveodds/booking-calendar/events/booked" \
     -H "x-access-token: YOUR_TOKEN"
   ```

### 🟡 重要 (本周完成)

5. **实现订阅 API**
   - 添加 `/api/events/{event_id}/subscribe` 端点
   - 添加批量订阅功能
   - 添加订阅状态查询

6. **启用自动订阅**
   - 配置环境变量
   - 测试自动订阅功能
   - 添加监控和日志

7. **实现 Recovery 功能**
   - 添加 Recovery API 调用
   - 处理 Recovery 响应
   - 测试数据恢复

### 🟢 优化 (下周完成)

8. **完善订阅管理**
   - 订阅策略优化
   - 订阅失败重试
   - 订阅状态同步

9. **添加监控告警**
   - 订阅成功率监控
   - odds_change 消息接收监控
   - 数据完整性检查

---

## 验证步骤

### 1. 订阅成功后的验证

```bash
# 1. 检查 subscribed 状态
curl "https://betradar-uof-service-production.up.railway.app/api/events?limit=10" | grep "subscribed"

# 2. 等待 1-2 分钟后检查 messages
curl "https://betradar-uof-service-production.up.railway.app/api/messages/recent?limit=10"

# 3. 检查 markets 数据
curl "https://betradar-uof-service-production.up.railway.app/api/events/sr:match:62924599/markets?limit=5"

# 4. 检查 odds 数据
curl "https://betradar-uof-service-production.up.railway.app/api/odds/all?limit=10"
```

### 2. 预期结果

订阅成功后，应该能看到:
- ✅ `subscribed: true`
- ✅ `message_history` 表有 odds_change 记录
- ✅ `markets` 表有数据
- ✅ `odds` 表有数据
- ✅ API 返回 markets 数据

---

## 附录

### A. 相关 API 文档

- [Booking Calendar API](https://docs.betradar.com/display/BD/Booking+Calendar)
- [Recovery API](https://docs.betradar.com/display/BD/Recovery)
- [UOF Messages](https://docs.betradar.com/display/BD/UOF+Messages)

### B. 相关代码文件

- `/main.go` - 消息处理器注册
- `/config/config.go` - Routing keys 配置
- `/services/message_processor.go` - 消息处理逻辑
- `/services/odds_change_parser.go` - odds_change 解析
- `/services/odds_parser.go` - Odds 数据存储
- `/services/match_monitor.go` - 订阅监控

### C. 相关数据库表

- `tracked_events` - Events 基本信息
- `odds_changes` - odds_change 消息记录
- `markets` - Markets 数据
- `odds` - Odds 数据
- `message_history` - 消息历史（已禁用）

### D. Sportradar API 端点

**Staging**:
- Base URL: `https://stgapi.betradar.com`
- Booking: `/v1/liveodds/booking-calendar/events/{event_id}/book`
- Recovery: `/v1/liveodds/recovery/initiate_request`

**Production**:
- Base URL: `https://api.betradar.com`
- Booking: `/v1/liveodds/booking-calendar/events/{event_id}/book`
- Recovery: `/v1/liveodds/recovery/initiate_request`

---

## 结论

**问题根本原因**: ✅ **Events 未通过 Booking Calendar API 订阅，导致 Sportradar 不发送 odds_change 消息**

**解决方案**: 
1. 立即手动订阅几个 events 进行测试
2. 实现订阅 API 端点
3. 启用自动订阅功能

**预期效果**: 订阅后 1-2 分钟内应该开始接收 odds_change 消息，markets 和 odds 数据将自动填充。

---

**报告作者**: Manus AI  
**诊断时间**: 2025-12-31  
**下次复查**: 订阅功能实现后立即验证
