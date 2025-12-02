# Sportradar UOF 前后端实时数据交互技术方案 v2.0 实施说明

**实施日期**: 2024-12-02  
**实施人员**: Manus AI  
**版本**: v2.0

## 1. 概述

本文档记录了基于《Sportradar UOF 前后端实时数据交互技术方案 v2.0》的后端代码实施情况。

## 2. 核心修改内容

### 2.1 bet_stop 状态修正

**文件**: `services/bet_stop_processor.go`

**修改内容**:
- 将 `bet_stop` 消息的默认状态从 `-1` (Suspended) 修正为 `0` (Deactivated)
- 根据技术方案 v2.0 的要求,`bet_stop` 消息现在正确映射到市场状态 `Deactivated` (停用)
- 保留 `market_status` 属性的支持,如果消息中包含该属性,则使用该值
- 增强日志输出,明确显示状态值和状态名称

**代码变更**:
```go
// 修改前
targetStatus := -1 // 默认: -1 = Suspended

// 修改后
targetStatus := 0 // 默认: 0 = Deactivated (已修正从 v1.0 的 -1)
```

### 2.2 Producer 心跳监控阈值调整

**文件**: `config/config.go`

**修改内容**:
- Producer 心跳断联判断阈值已从 20 秒收紧至 **10 秒**
- 配置默认值: `ProducerDownThresholdSeconds: 10`
- 可通过环境变量 `PRODUCER_DOWN_THRESHOLD_SECONDS` 覆盖

**说明**:
此修改实现了更灵敏的 Producer 健康状态监控,能够更快地检测到连接问题。

### 2.3 WebSocket 精细化订阅模型

**文件**: `web/websocket.go`

**修改内容**:
1. **扩展 Client 结构**: 新增 `marketIDs map[int]bool` 字段,支持按盘口ID订阅
2. **扩展 WSMessage 结构**: 新增 `MarketID *int` 字段,用于消息中携带盘口ID
3. **增强订阅逻辑**: 支持前端通过 `market_ids` 参数订阅特定盘口
4. **优化过滤逻辑**: `shouldReceive` 方法现在支持三级过滤:
   - 消息类型过滤 (message_types)
   - 赛事ID过滤 (event_ids)
   - 盘口ID过滤 (market_ids) **[新增]**

**前端订阅消息格式**:
```json
{
  "action": "subscribe",
  "params": {
    "eventIds": ["sr:match:12345", "sr:match:67890"],
    "marketIds": [1, 18, 235],
    "messageTypes": ["odds_update", "market_status"]
  }
}
```

### 2.4 盘口分组展示支持

**新增文件**:
1. `config/market_tabs_config.json` - 盘口分组配置文件
2. `web/market_tabs_handler.go` - 盘口分组配置 API Handler

**修改文件**:
- `web/server.go` - 添加新的 API 端点 `/api/config/market-tabs`

**新增 API 端点**:
```
GET /api/config/market-tabs
```

**功能说明**:
- 提供前端所需的 Tab 和 Chip 配置信息
- 支持 17 种盘口分组类型 (常规玩法、半场、节、局、地图、盘、回合等)
- 配置文件采用 JSON 格式,易于维护和扩展

**配置结构**:
```json
{
  "tabs": [
    {
      "id": "regular_play",
      "label": "常规玩法",
      "type": "group",
      "groups": ["regular_play"],
      "chipSpecifiers": []
    },
    ...
  ],
  "chips": {
    "quarter_1": {
      "label": "第1节",
      "specifier": "quarternr",
      "value": "1"
    },
    ...
  }
}
```

## 3. 后续工作建议

### 3.1 Market 数据增强

**待实施**: 在处理 `fixture` 或 `odds_change` 消息时,需要为每个 market 计算并附加 `tabId` 字段。

**建议实施位置**:
- `services/odds_change_parser.go`
- `services/fixture_parser.go`

**实施逻辑**:
```go
// 伪代码示例
func calculateTabID(market Market) string {
    // 根据 market.Groups 和 market.Specifiers 计算 tabId
    if contains(market.Groups, "regular_play") {
        return "regular_play"
    }
    if contains(market.Groups, "quarters") {
        return "quarters"
    }
    // ... 其他分组逻辑
    return "other"
}
```

### 3.2 API 响应增强

**待实施**: 在 `/api/events/{eventId}` 端点的响应中,为每个 market 对象添加 `tabId` 字段。

**建议实施位置**:
- `web/enhanced_events_handler.go` 或相关的事件详情处理器

### 3.3 比赛交接与订阅逻辑

**说明**: 根据技术方案 v2.0,比赛交接 (Handover) 和赛事订阅 (Booking) 应完全由后端处理。

**当前状态**: 需要检查现有的 `services/auto_booking.go` 和 `services/recovery_manager.go` 是否已正确实现此逻辑。

## 4. 测试建议

### 4.1 bet_stop 状态测试
- 验证 `bet_stop` 消息是否正确将市场状态更新为 `0` (Deactivated)
- 验证带有 `market_status` 属性的 `bet_stop` 消息是否使用该属性值

### 4.2 Producer 监控测试
- 模拟 Producer 断线场景,验证 10 秒后是否触发告警
- 验证 Producer 恢复后告警是否正确清除

### 4.3 WebSocket 订阅测试
- 测试按 `eventIds` 订阅
- 测试按 `marketIds` 订阅
- 测试同时使用多种过滤器的组合订阅

### 4.4 盘口分组配置测试
- 访问 `/api/config/market-tabs` 端点,验证返回的配置格式
- 验证配置文件的完整性和正确性

## 5. 环境变量配置

以下环境变量与本次实施相关:

```bash
# Producer 监控配置
PRODUCER_CHECK_INTERVAL_SECONDS=60  # 检查间隔(秒)
PRODUCER_DOWN_THRESHOLD_SECONDS=10  # 下线阈值(秒) - 已从 20 修改为 10
```

## 6. 兼容性说明

所有修改均向后兼容,不会影响现有功能:
- `bet_stop` 状态修正仅影响新接收的消息
- WebSocket 订阅模型扩展了功能,但保留了原有订阅方式
- 新增的 API 端点不影响现有端点

## 7. 文档更新

建议更新以下文档:
- `API_DOCUMENTATION.md` - 添加 `/api/config/market-tabs` 端点说明
- `FRONTEND_API_REFERENCE.md` - 更新 WebSocket 订阅消息格式
- `README.md` - 添加 v2.0 技术方案实施说明的链接

## 8. 版本信息

- **技术方案版本**: v2.0
- **实施版本**: v1.0.8 (建议)
- **Git 分支**: feature/tech-spec-v2.0 (建议)

---

**实施完成日期**: 2024-12-02  
**审核状态**: 待审核  
**部署状态**: 待部署
