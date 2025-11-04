# Recovery 机制集成文档

## 概述

本文档描述了 Betradar UOF 服务中三种 Recovery 机制的实现和集成情况。

## Recovery 类型

根据 Betradar UOF 文档，系统支持三种 Recovery 类型：

### 1. Present (Odds Initialization) ✅

**用途**: 请求当前的赔率数据，不带时间戳参数

**实现状态**: ✅ 完全实现

**实现位置**: `services/recovery_manager.go`

**调用方式**:
```bash
POST /api/recovery/trigger
```

**说明**: 
- 通过 `TriggerFullRecovery()` 方法实现
- 不使用 `after` 参数时，Betradar 返回默认范围内的赔率数据
- 支持多个 product (liveodds, pre)

---

### 2. Past (Odds + Stateful Messages Recovery) ✅

**用途**: 请求指定时间后的赔率数据和状态消息

**实现状态**: ✅ 完全实现

#### 2.1 Odds Recovery with Timestamp

**实现位置**: `services/recovery_manager.go` - `triggerProductRecovery()`

**配置参数**:
- `RECOVERY_AFTER_HOURS`: 恢复多少小时内的数据（默认 10 小时，最大 10 小时）

**调用方式**:
```bash
POST /api/recovery/trigger
```

**说明**:
- 使用 `after` 参数（Unix timestamp 毫秒）
- Betradar 限制：最多恢复 10 小时内的数据
- `liveodds` 产品不使用 `after` 参数（敏感）

#### 2.2 Stateful Messages Recovery

**实现位置**: `services/recovery_manager.go` - `TriggerStatefulMessagesRecovery()`

**调用方式**:
```bash
# 单个事件的 Stateful Messages 恢复
POST /api/recovery/stateful/{event_id}?product=liveodds

# 单个事件的完整恢复（包含 Odds + Stateful Messages）
POST /api/recovery/event/{event_id}?product=liveodds
```

**说明**:
- Stateful Messages 包括: bet_settlement, bet_cancel, rollback_bet_settlement, rollback_bet_cancel
- 目前仅支持单个事件的恢复
- 全量恢复不包含 Stateful Messages（需单独调用）

---

### 3. Fixture Recovery ✅

**用途**: 请求指定时间后的赛事变更信息

**实现状态**: ✅ 完全实现

**实现位置**: 
- `services/fixture_changes_service.go` - 核心服务
- `services/recovery_manager.go` - 集成到 RecoveryManager

**调用方式**:
```bash
# 使用自定义时间戳
POST /api/recovery/fixtures?after=1234567890

# 使用默认时间（最近 10 小时）
POST /api/recovery/fixtures
```

**响应示例**:
```json
{
  "status": "success",
  "message": "Fixture recovery completed",
  "after": 1234567890,
  "count": 15,
  "changes": [
    {
      "event_id": "sr:match:12345",
      "update_time": "2025-01-01T12:00:00Z",
      "change_type": "time_change",
      "next_live_time": 1735732800
    }
  ],
  "time": 1735732900
}
```

**说明**:
- 集成到全量恢复流程中（如果配置了 `RECOVERY_AFTER_HOURS`）
- 支持单独调用
- 返回完整的变更列表

---

## API 端点总览

| 端点 | 方法 | 说明 | Recovery 类型 |
|------|------|------|--------------|
| `/api/recovery/trigger` | POST | 触发全量恢复（所有产品） | Present / Past (Odds) + Fixture |
| `/api/recovery/event/{event_id}` | POST | 单个事件恢复（Odds + Stateful） | Past (单事件) |
| `/api/recovery/fixtures` | POST | Fixture 变更恢复 | Fixture |
| `/api/recovery/stateful/{event_id}` | POST | 单个事件的 Stateful Messages 恢复 | Past (Stateful only) |
| `/api/recovery/status` | GET | 查询恢复状态 | - |

---

## 配置参数

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `AUTO_RECOVERY` | `true` | 启动时自动触发恢复 |
| `RECOVERY_AFTER_HOURS` | `10` | 恢复多少小时内的数据（0=默认，最大 10） |
| `RECOVERY_PRODUCTS` | `liveodds,pre` | 需要恢复的产品列表 |
| `UOF_API_TOKEN` | - | Betradar API Token（用于 Fixture Recovery） |

---

## 使用示例

### 1. 启动时自动恢复

```bash
# 设置环境变量
export AUTO_RECOVERY=true
export RECOVERY_AFTER_HOURS=10

# 启动服务（自动触发全量恢复）
./uof-service
```

### 2. 手动触发全量恢复

```bash
curl -X POST http://localhost:8080/api/recovery/trigger
```

### 3. 恢复单个事件

```bash
# 恢复事件的 Odds + Stateful Messages
curl -X POST "http://localhost:8080/api/recovery/event/sr:match:12345?product=liveodds"
```

### 4. 恢复 Fixture 变更

```bash
# 恢复最近 10 小时的变更
curl -X POST http://localhost:8080/api/recovery/fixtures

# 恢复指定时间后的变更
curl -X POST "http://localhost:8080/api/recovery/fixtures?after=1735732800"
```

### 5. 查询恢复状态

```bash
curl http://localhost:8080/api/recovery/status?limit=20
```

---

## 实现细节

### 全量恢复流程

`TriggerFullRecovery()` 执行以下步骤：

1. **Odds Recovery**: 遍历所有配置的产品，触发 Odds 恢复
   - 使用 `after` 参数（如果配置了 `RECOVERY_AFTER_HOURS`）
   - `liveodds` 产品不使用 `after` 参数
   - 处理频率限制错误，自动重试

2. **Fixture Recovery**: 如果配置了 `RECOVERY_AFTER_HOURS`
   - 调用 `TriggerFixtureRecoverySince()` 获取赛事变更
   - 最多恢复 10 小时内的变更

3. **注意**: Stateful Messages 不包含在全量恢复中
   - 需要针对特定事件单独调用
   - 使用 `/api/recovery/event/{event_id}` 端点

### 频率限制处理

- 检测到 `403 Forbidden` + "Too many requests" 时
- 自动安排 15 分钟后重试
- 如果再次失败，延长到 30 分钟
- 不阻塞启动流程

### 缓存和性能

- Fixture Changes Service 使用独立的 HTTP 客户端
- 30 秒超时
- 无缓存（每次调用都获取最新数据）

---

## 注意事项

1. **Betradar 限制**:
   - Odds Recovery 最多恢复 10 小时内的数据
   - 调用频率有限制（触发频率限制时自动重试）

2. **liveodds 产品**:
   - 对 `after` 参数敏感
   - 建议不使用 `after` 参数

3. **Stateful Messages**:
   - 仅支持单个事件的恢复
   - 不包含在全量恢复中

4. **Fixture Recovery**:
   - 需要配置 `UOF_API_TOKEN`
   - 使用 Betradar REST API（非 AMQP）

---

## 测试

### 编译测试

```bash
go build -o /tmp/test-build
```

### 功能测试

1. **测试全量恢复**:
   ```bash
   curl -X POST http://localhost:8080/api/recovery/trigger
   ```

2. **测试 Fixture Recovery**:
   ```bash
   curl -X POST http://localhost:8080/api/recovery/fixtures
   ```

3. **测试单事件恢复**:
   ```bash
   curl -X POST "http://localhost:8080/api/recovery/event/sr:match:12345?product=liveodds"
   ```

4. **查看恢复状态**:
   ```bash
   curl http://localhost:8080/api/recovery/status
   ```

---

## 更新日志

### 2025-01-XX

- ✅ 集成 Fixture Changes Service 到 RecoveryManager
- ✅ 添加 `TriggerFixtureRecovery()` 方法
- ✅ 将 Fixture Recovery 集成到全量恢复流程
- ✅ 添加 `/api/recovery/fixtures` 端点
- ✅ 添加 `/api/recovery/stateful/{event_id}` 端点
- ✅ 修复类型冲突（APISport, APITournament）
- ✅ 文档完善

---

## 参考文档

- [Betradar UOF Messages 完整文档](./BetradarUnifiedOddsFeed(UOF)-Messages完整文档.md)
- [UOF API 完整文档](./UOF_API_Complete_Documentation.md)
- [Market ID 迁移文档](./MIGRATION_SR_MARKET_ID.md)

