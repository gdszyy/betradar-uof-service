# SportRadar 产品接入指南

**版本**: 1.0.x  
**目标**: 理解 SportRadar 产品的关键业务规则和接入顺序  
**适用对象**: 开发人员

---

## 📋 目录

1. [产品概述](#产品概述)
2. [接入准备](#接入准备)
3. [UOF 接入关键规则](#uof-接入关键规则)
4. [Live Data 接入关键规则](#live-data-接入关键规则)
5. [Producer 交接机制](#producer-交接机制)
6. [数据恢复规则](#数据恢复规则)
7. [常见陷阱](#常见陷阱)

---

## 产品概述

### UOF (Unified Odds Feed)
**用途**: 实时赔率数据  
**协议**: AMQP (RabbitMQ)  
**核心概念**:
- **Producer**: 数据生产者,不同 Producer 负责不同阶段
  - `Pre-match Odds (ID: 3)`: 赛前赔率
  - `Live Odds (ID: 1)`: 比赛进行中的赔率
  - `Ctrl (ID: 3)`: 控制和管理
- **Market**: 投注市场 (如 1X2, Over/Under)
- **Outcome**: 市场中的投注选项

### Live Data (LD)
**用途**: 实时比赛事件数据  
**协议**: Socket (SSL, Port 2017)  
**核心概念**:
- **Match**: 比赛
- **Event**: 比赛事件 (进球、红牌等)
- **Sequence Number**: 消息序列号,**必须连续**

### 产品组合

```
UOF (赔率) + Live Data (事件) = 完整数据
```

---

## 接入准备

### 1. 获取凭证

**统一账号密码**:
- UOF、Live Data 和 Ctrl 后台使用**同一组账号密码**
- 联系 SportRadar 获取 Username 和 Password

**Bookmaker ID**:
- 通过 UOF API 的 `whoami` 接口获取
- 接口: `GET https://stgapi.betradar.com/v1/users/whoami.xml`
- 认证方式: HTTP Basic Auth (使用 Username/Password)

**Access Token**:
- 在 **Ctrl 后台**手动生成
- 用于 REST API 调用 (如 Recovery、Fixture 等)
- 登录 Ctrl → Settings → Generate Token

### 2. IP 白名单

**Live Data 必须配置 IP 白名单**:
1. 获取服务器出口 IP
2. 提供给 SportRadar 技术支持
3. 等待确认 (1-2 工作日)

**UOF 不需要 IP 白名单**

### 3. 了解环境

- **Integration (集成)**: 测试环境
  - UOF: `stgmq.betradar.com:5671`
  - Live Data: 使用生产服务器 (无单独集成环境)
  
- **Production (生产)**: 正式环境
  - UOF: `mq.betradar.com:5671`
  - Live Data: `livedata.betradar.com:2017`

---

## UOF 接入关键规则

### 规则 1: 首次连接必须做数据恢复

**为什么?**
- 首次连接时,队列是空的
- 不会自动接收历史数据
- 必须主动请求恢复

**恢复流程**:
```
1. 连接 AMQP
2. 立即调用 Recovery API
3. 等待接收 snapshot_complete 消息
4. 开始正常消费
```

**关键消息**: `snapshot_complete`
```xml
<snapshot_complete product="3" request_id="123" timestamp="1234567890" />
```

**恢复窗口**:
- Live Odds: **10 小时**
- Gaming: 3 小时
- Premium Cricket: 最少 7 天
- 其他: 72 小时

> ⚠️ **重要**: Recovery API 只恢复当前最新赔率,不提供历史数据。历史数据需要从 **Ctrl 后台**或 **Live Booking Calendar** 下载。

### 规则 2: Producer 交接机制

当比赛从赛前进入直播状态时,数据源会从 `Pre-match Odds` 切换到 `Live Odds`。

**交接标志**: `market.status = -2` (handed_over)

**关键规则**:
- 收到 `handed_over` 消息后,**暂停更新该市场**
- 等待 `Live Odds` producer 的新消息
- **可能出现 Live Odds 消息早于 handed_over 消息**

**处理流程**:

```
收到 odds_change 消息
    ↓
检查 product 字段
    ↓
是 LIVE producer?
    ├─ 是 → 立即使用,更新市场
    └─ 否 → 检查 market.status
            ├─ status = -2 (handed_over)
            │   ├─ 已收到 LIVE 消息? → 忽略此 handed_over
            │   └─ 未收到 LIVE 消息? → 暂停更新,等待 LIVE
            ├─ status = 0 (deactivated) → 关闭市场
            └─ 其他 status → 正常更新
```

**时序问题处理**:
- 如果先收到 LIVE 消息,后收到 handed_over → **忽略 handed_over**
- 如果先收到 handed_over,后收到 LIVE → **等待 LIVE 后更新**

### 规则 3: 消息类型

**核心消息**:
- `odds_change`: 赔率变化 (最重要)
- `bet_stop`: 停止投注
- `bet_cancel`: 取消投注
- `fixture_change`: 比赛信息变化

**控制消息**:
- `alive`: 心跳 (每 20 秒)
- `snapshot_complete`: 恢复完成

---

## Live Data 接入关键规则

### 规则 1: IP 白名单必须配置

**Live Data 服务器会在 TCP 连接建立时检查源 IP**:
- 如果 IP 不在白名单中,连接会被立即拒绝
- 错误: `connection reset by peer`

**配置流程**:
1. 获取服务器出口 IP
2. 联系 SportRadar 技术支持
3. 提供 IP 地址和账号信息
4. 等待白名单配置完成 (1-2 工作日)

### 规则 2: 建立连接

**连接参数**:
- **服务器**: `livedata.betradar.com`
- **端口**: `2017`
- **协议**: TLS/SSL (必须)

**连接流程**:
```
1. 建立 TLS 连接到 livedata.betradar.com:2017
2. 发送登录消息
3. 等待登录响应
4. 开始订阅比赛
```

### 规则 3: 登录消息格式

**正确的登录消息格式**:
```xml
<login>
<credential>
<loginname value="your_username"/>
<password value="your_password"/>
</credential>
</login>
```

**注意事项**:
- 消息必须以 `0x00` (NULL 字符) 结尾
- 使用与 UOF 相同的 Username 和 Password
- 登录成功后会收到 `<login>` 响应消息

---

## Producer 交接机制

### 交接场景

当比赛从 **Pre-match** 进入 **Live** 状态时:

1. **Pre-match Odds Producer** 发送 `market.status = -2` (handed_over)
2. **Live Odds Producer** 开始发送该比赛的赔率

### 时序问题

**可能的情况**:
- ✅ 正常: handed_over → LIVE 消息
- ⚠️ 异常: LIVE 消息 → handed_over (LIVE 消息更早)

### 处理策略

```
if (message.product == LIVE_PRODUCER) {
    // 立即使用 LIVE 数据
    updateMarket(message);
    markAsLiveReceived(market_id);
} else {
    if (market.status == -2) {  // handed_over
        if (hasReceivedLive(market_id)) {
            // 已经收到 LIVE,忽略此 handed_over
            ignore();
        } else {
            // 等待 LIVE 消息
            pauseMarketUpdate(market_id);
        }
    } else if (market.status == 0) {  // deactivated
        closeMarket(market_id);
    } else {
        // 正常更新
        updateMarket(message);
    }
}
```

---

## 数据恢复规则

### Recovery API

**用途**: 获取当前最新的赔率快照

**端点**: `POST /{product}/recovery/initiate_request`

**参数**:
- `after`: 恢复起始时间戳 (毫秒)
- `request_id`: 请求 ID (用于追踪)

**恢复窗口**:
- Live Odds: 10 小时
- 其他: 72 小时

### 恢复完成标志

```xml
<snapshot_complete product="3" request_id="123" timestamp="1234567890" />
```

收到此消息后,表示恢复完成,可以开始正常消费实时消息。

### 历史数据

**Recovery API 不提供历史数据**,只恢复当前最新赔率。

**获取历史数据**:
- 登录 **Ctrl 后台**
- 或使用 **Live Booking Calendar**
- 下载历史数据文件

---

## 常见陷阱

### 1. 忘记首次恢复

**错误**: 连接 AMQP 后直接消费,没有调用 Recovery API

**后果**: 
- 队列为空,收不到任何消息
- 或只能收到新产生的消息,错过已有数据

**解决**: 首次连接后立即调用 Recovery API

### 2. 错误处理 handed_over

**错误**: 收到 `handed_over` 后立即删除市场

**后果**: 
- LIVE 消息到达时,市场已被删除
- 无法显示直播赔率

**解决**: 收到 `handed_over` 后暂停更新,等待 LIVE 消息

### 3. 忽略时序问题

**错误**: 假设 `handed_over` 一定早于 LIVE 消息

**后果**: 
- LIVE 消息先到,被当作 Pre-match 处理
- handed_over 后到,错误地暂停了已经在用 LIVE 数据的市场

**解决**: 记录已收到 LIVE 消息的市场,忽略后续的 handed_over

### 4. 未配置 IP 白名单

**错误**: 直接尝试连接 Live Data,未配置白名单

**后果**: 
- 连接立即被拒绝
- 错误: `connection reset by peer`

**解决**: 提前联系 SportRadar 配置 IP 白名单

### 5. 混淆 Token 和 Password

**错误**: 
- 在 AMQP 连接中使用 Access Token
- 在 REST API 中使用 Password

**后果**: 认证失败

**解决**: 
- AMQP: 使用 Username + Password
- REST API: 使用 Access Token (在 Ctrl 后台生成)

### 6. 忽略 Bookmaker ID

**错误**: 不知道自己的 Bookmaker ID

**后果**: 
- 无法正确配置 AMQP vhost
- 无法订阅比赛

**解决**: 调用 `whoami` API 获取 Bookmaker ID

---

## 接入检查清单

### UOF 接入

- [ ] 获取 Username 和 Password
- [ ] 调用 `whoami` API 获取 Bookmaker ID
- [ ] 在 Ctrl 后台生成 Access Token
- [ ] 配置 AMQP 连接 (vhost: `/unifiedfeed/{bookmaker_id}`)
- [ ] 首次连接后立即调用 Recovery API
- [ ] 等待 `snapshot_complete` 消息
- [ ] 实现 Producer 交接逻辑
- [ ] 处理 LIVE 消息早于 handed_over 的情况
- [ ] 实现定期 Recovery (断线重连后)

### Live Data 接入

- [ ] 获取服务器出口 IP
- [ ] 联系 SportRadar 配置 IP 白名单
- [ ] 等待白名单配置完成
- [ ] 建立 TLS 连接到 `livedata.betradar.com:2017`
- [ ] 发送正确格式的登录消息
- [ ] 等待登录响应

---

## 参考资源

- **UOF 官方文档**: https://docs.sportradar.com/uof/
- **Live Data 官方文档**: https://docs.sportradar.com/live-data/
- **Ctrl 后台**: 联系 SportRadar 获取访问地址
- **技术支持**: 联系 SportRadar 客服

---

**文档版本**: 1.0.4  
**最后更新**: 2025-10-23

