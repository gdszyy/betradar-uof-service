# 比赛订阅管理功能

**版本**: v1.0.8  
**更新日期**: 2025-10-23  
**重要性**: 🟢 中 - 优化订阅管理,避免达到订阅上限

---

## 📋 功能概述

自动管理 Live Data 比赛订阅的生命周期,在比赛结束后主动取消订阅,避免占用订阅名额导致无法订阅新比赛。

---

## 🎯 解决的问题

### 问题背景

根据 Betradar Live Data 文档:
> 不取消订阅可能导致占用订阅名额,达到上限后会返回 "maximum number of subscriptions" 错误

### 解决方案

1. **自动检测比赛结束**: 监听比赛状态变化
2. **延迟取消订阅**: 比赛结束后等待一段时间(默认10分钟)
3. **批量清理**: 定期检查并批量取消已结束比赛的订阅
4. **手动管理**: 提供 API 端点手动管理订阅

---

## 🔧 核心功能

### 1. 自动订阅管理

#### 订阅记录
```go
type MatchSubscription struct {
    MatchID         string    // 比赛ID
    SubscribedAt    time.Time // 订阅时间
    LastEventAt     time.Time // 最后事件时间
    Status          string    // 比赛状态: live, ended, closed
    EventCount      int       // 事件数量
    AutoUnsubscribe bool      // 是否自动取消订阅
}
```

#### 状态跟踪
- 记录每个订阅的比赛
- 跟踪比赛状态变化
- 统计接收到的事件数量
- 记录最后事件时间

### 2. 自动清理机制

#### 清理条件
1. **比赛已结束**: 状态为 `ended` 或 `closed`
2. **超过清理时间**: 比赛结束后超过 10 分钟(可配置)
3. **长时间无活动**: 超过 24 小时没有接收到事件

#### 清理流程
```
1. 每 5 分钟检查一次(可配置)
2. 找出符合清理条件的比赛
3. 批量发送取消订阅请求
4. 从订阅列表中移除
5. 发送飞书通知
```

### 3. 取消订阅方式

#### 单个比赛取消
```xml
<matchstop matchid="1101335"/>
```

**确认回复**:
```xml
<matchstop matchid="1101335" reason="User unsubscribed to match"/>
```

#### 批量取消
```xml
<matchunsubscription>
  <match matchid="1101335"/>
  <match matchid="1062714"/>
</matchunsubscription>
```

---

## 📊 API 端点

### 1. 获取所有订阅

**端点**: `GET /api/subscriptions`

**响应**:
```json
{
  "status": "success",
  "count": 5,
  "subscriptions": [
    {
      "MatchID": "1101335",
      "SubscribedAt": "2025-10-23T10:00:00Z",
      "LastEventAt": "2025-10-23T12:30:00Z",
      "Status": "live",
      "EventCount": 145,
      "AutoUnsubscribe": true
    }
  ]
}
```

---

### 2. 获取订阅统计

**端点**: `GET /api/subscriptions/stats`

**响应**:
```json
{
  "status": "success",
  "stats": {
    "total": 5,
    "live": 3,
    "ended": 2,
    "closed": 0
  }
}
```

---

### 3. 取消订阅单个比赛

**端点**: `POST /api/subscriptions/unsubscribe`

**请求**:
```json
{
  "match_id": "1101335"
}
```

**响应**:
```json
{
  "status": "success",
  "message": "Match unsubscribed successfully",
  "match_id": "1101335"
}
```

---

### 4. 批量取消订阅

**端点**: `POST /api/subscriptions/unsubscribe/batch`

**请求**:
```json
{
  "match_ids": ["1101335", "1062714", "1098765"]
}
```

**响应**:
```json
{
  "status": "success",
  "message": "Matches unsubscribed successfully",
  "count": 3
}
```

---

### 5. 手动清理已结束比赛

**端点**: `POST /api/subscriptions/cleanup`

**响应**:
```json
{
  "status": "success",
  "message": "Ended matches cleaned up successfully",
  "count": 2
}
```

---

## 🚀 使用示例

### 查看当前订阅
```bash
curl https://your-app.railway.app/api/subscriptions
```

### 查看订阅统计
```bash
curl https://your-app.railway.app/api/subscriptions/stats
```

### 取消单个订阅
```bash
curl -X POST https://your-app.railway.app/api/subscriptions/unsubscribe \
  -H "Content-Type: application/json" \
  -d '{"match_id": "1101335"}'
```

### 批量取消订阅
```bash
curl -X POST https://your-app.railway.app/api/subscriptions/unsubscribe/batch \
  -H "Content-Type: application/json" \
  -d '{"match_ids": ["1101335", "1062714"]}'
```

### 手动清理
```bash
curl -X POST https://your-app.railway.app/api/subscriptions/cleanup
```

---

## ⚙️ 配置选项

### 自动清理配置

```go
manager := NewMatchSubscriptionManager(ldClient, larkNotifier)

// 设置是否启用自动清理
manager.SetAutoCleanup(true)

// 设置清理检查间隔(默认 5 分钟)
manager.SetCleanupInterval(5 * time.Minute)

// 设置比赛结束后清理时间(默认 10 分钟)
manager.SetCleanupAfterEnded(10 * time.Minute)
```

---

## 📝 日志示例

### 订阅添加
```
[SubscriptionManager] ✅ Added subscription for match 1101335
```

### 比赛结束
```
[SubscriptionManager] 🏁 Match 1101335 ended (status: ended)
```

### 自动清理
```
[SubscriptionManager] 🧹 Cleaning up 2 ended matches
[SubscriptionManager] 🛑 Batch unsubscribing 2 matches
[SubscriptionManager] ✅ Batch unsubscribed 2 matches
```

### 取消订阅
```
[SubscriptionManager] 🛑 Unsubscribing from match: 1101335
[SubscriptionManager] ✅ Unsubscribed from match 1101335 (events: 145, duration: 2h30m15s)
```

---

## 🔔 飞书通知

### 比赛结束通知
```
🏁 **比赛结束**

比赛ID: 1101335
状态: ended
事件数: 145
订阅时长: 2h30m15s
将在 10m0s 后自动取消订阅
```

### 自动清理通知
```
🧹 **自动清理订阅**

已取消 2 个已结束比赛的订阅
释放订阅名额,避免达到上限
```

---

## 🎯 最佳实践

### 1. 及时清理
- 启用自动清理功能
- 设置合理的清理延迟(10-15分钟)
- 定期检查订阅统计

### 2. 监控订阅数量
```bash
# 定期检查订阅统计
watch -n 60 'curl -s https://your-app.railway.app/api/subscriptions/stats | jq'
```

### 3. 手动干预
```bash
# 如果订阅数量过多,手动清理
curl -X POST https://your-app.railway.app/api/subscriptions/cleanup
```

### 4. 设置告警
- 监控订阅总数
- 当接近上限时发送告警
- 自动触发清理

---

## ⚠️ 注意事项

### 1. 清理延迟

**为什么需要延迟?**
- 比赛结束后可能还有延迟事件
- 给系统时间处理最后的数据
- 避免过早取消导致数据丢失

**推荐配置**:
- 足球: 10-15 分钟
- 篮球: 5-10 分钟
- 其他: 根据实际情况调整

### 2. 订阅上限

不同的 Betradar 账户可能有不同的订阅上限:
- 基础账户: ~50 个并发订阅
- 高级账户: ~200 个并发订阅
- 企业账户: 更高或无限制

**建议**: 联系 Betradar 确认您的账户上限

### 3. 批量操作

批量取消订阅时:
- 建议每批不超过 50 个
- 避免一次性取消过多导致网络问题
- 可以分批次执行

---

## 🔍 故障排查

### 问题 1: 取消订阅失败

**错误信息**:
```
[SubscriptionManager] ❌ Failed to unsubscribe match 1101335: not connected
```

**解决方案**:
1. 检查 LD 客户端连接状态
2. 确认比赛ID正确
3. 查看 LD 服务器日志

---

### 问题 2: 自动清理不工作

**症状**: 已结束的比赛没有被自动清理

**排查步骤**:
1. 检查自动清理是否启用
```bash
# 查看日志
railway logs | grep "SubscriptionManager.*Started"
```

2. 检查清理间隔配置
3. 查看是否有错误日志

---

### 问题 3: 订阅数量持续增长

**症状**: 订阅数量不断增加,不减少

**解决方案**:
1. 手动触发清理
```bash
curl -X POST https://your-app.railway.app/api/subscriptions/cleanup
```

2. 检查清理条件
3. 调整清理延迟时间

---

## 📈 性能影响

### 资源消耗
- **内存**: 每个订阅约 200 字节
- **CPU**: 清理检查每 5 分钟一次,影响极小
- **网络**: 批量取消订阅时有短暂流量峰值

### 优化建议
1. 合理设置清理间隔(不要太频繁)
2. 使用批量取消而非单个取消
3. 监控订阅数量,避免过多积压

---

## 🔄 集成示例

### 在 main.go 中集成

```go
// 创建订阅管理器
subscriptionManager := services.NewMatchSubscriptionManager(ldClient, larkNotifier)

// 设置到事件处理器
eventHandler.SetSubscriptionManager(subscriptionManager)

// 设置到 Web 服务器
server.SetSubscriptionManager(subscriptionManager)

// 启动订阅管理器
go subscriptionManager.Start()

// 在 LD 客户端订阅比赛时添加记录
func (c *LDClient) SubscribeMatch(matchID string) error {
    // ... 发送订阅消息 ...
    
    // 添加到订阅管理器
    if c.subscriptionManager != nil {
        c.subscriptionManager.AddSubscription(matchID)
    }
    
    return nil
}
```

---

## 📚 相关文档

- [Live Data 集成指南](LD_INTEGRATION_GUIDE.md)
- [Betradar 官方文档](https://docs.betradar.com)
- [Release Notes v1.0.8](../RELEASE_NOTES_v1.0.8.md)

---

## 🎉 总结

### 核心优势
1. ✅ 自动管理订阅生命周期
2. ✅ 避免达到订阅上限
3. ✅ 释放不需要的订阅名额
4. ✅ 提供手动管理接口
5. ✅ 完善的监控和通知

### 使用建议
- 启用自动清理功能
- 定期检查订阅统计
- 监控订阅数量变化
- 根据实际情况调整配置

---

**文档版本**: v1.0.8  
**最后更新**: 2025-10-23

