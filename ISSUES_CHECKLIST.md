# Betradar UOF Service - 问题清单

**更新日期**: 2025-10-23  
**当前版本**: v1.0.6  
**项目**: https://github.com/gdszyy/betradar-uof-service

---

## 📊 问题概览

| 类别 | 数量 | 优先级 |
|------|------|--------|
| 🔴 阻塞性问题 | 2 | 高 |
| 🟡 功能缺失 | 5 | 中 |
| 🟢 优化改进 | 8 | 低 |
| 📝 文档缺失 | 3 | 中 |
| ⚙️ 配置问题 | 2 | 中 |
| **总计** | **20** | - |

---

## 🔴 阻塞性问题 (高优先级)

### 1. Live Data 不可用 - IP 白名单未完成

**状态**: 🔴 阻塞  
**影响**: 无法使用 Betradar Live Data 服务  
**临时方案**: ✅ 已用 The Sports API 替代 (v1.0.6)

**问题描述**:
- Betradar Live Data 需要 IP 白名单
- 提交白名单申请后等待时间长
- 目前无法连接到 Live Data WebSocket

**临时解决方案**:
- v1.0.6 已集成 The Sports API
- 使用 The Sports MQTT 替代 Live Data
- 数据存储到相同的数据库表

**长期解决方案**:
- [ ] 等待 Betradar 完成 IP 白名单配置
- [ ] 测试 Live Data 连接
- [ ] 实现双数据源支持 (Live Data + The Sports)
- [ ] 数据源自动切换机制

**相关文档**:
- RELEASE_NOTES_v1.0.6.md - The Sports 集成说明

---

### 2. Replay API - Playlist Empty 错误

**状态**: 🔴 未解决  
**影响**: 无法使用 Replay API 进行历史数据回放测试  
**优先级**: 高

**问题描述**:
```
POST /replay/reset                              → 200 OK
PUT  /replay/events/test:match:21797788         → 200 OK
GET  /replay/                                   → 200 OK (event in list ✓)
POST /replay/play?speed=30&node_id=1            → 400 "Playlist is empty"
```

**现象**:
- 事件成功添加到 Replay 列表
- GET /replay/ 确认事件存在
- 但 POST /replay/play 报错 "Playlist is empty"
- 事件似乎在 verify 和 play 之间消失

**已尝试的方案**:
- ✅ 增加等待时间 (3秒 → 10秒)
- ✅ 使用 `x-access-token` 认证
- ✅ 添加 `node_id` 参数
- ✅ 多次重试验证
- ❌ 问题仍然存在

**待确认**:
- [ ] `test:match:21797788` 是否有效?
- [ ] 是否满足 48 小时规则?
- [ ] 是否需要更长的等待时间?
- [ ] 是否有其他可用的测试事件?

**下一步**:
- [ ] 联系 Sportradar 技术支持
- [ ] 获取可用的测试事件列表
- [ ] 确认正确的 API 调用流程

**相关文档**:
- docs/SR-TECHNICAL-MEETING-ISSUES.md
- docs/REPLAY-TESTING-GUIDE.md

---

## 🟡 功能缺失 (中优先级)

### 3. The Sports 凭证未配置

**状态**: 🟡 需要配置  
**影响**: The Sports API 功能无法使用  
**优先级**: 中

**问题描述**:
- v1.0.6 已集成 The Sports SDK
- 但缺少 API 凭证配置
- 需要在 Railway 中添加环境变量

**所需环境变量**:
```bash
THESPORTS_API_TOKEN=your_api_token
THESPORTS_USERNAME=your_mqtt_username
THESPORTS_SECRET=your_mqtt_secret
```

**解决步骤**:
1. [ ] 注册 The Sports API 账户
2. [ ] 获取 API Token 和 MQTT 凭证
3. [ ] 在 Railway 中配置环境变量
4. [ ] 测试连接: `POST /api/thesports/connect`
5. [ ] 验证数据接收

**相关文档**:
- RELEASE_NOTES_v1.0.6.md - 配置说明

---

### 4. 缺少自动订阅配置文档

**状态**: 🟡 文档缺失  
**影响**: 用户不知道如何配置自动订阅间隔  
**优先级**: 中

**问题描述**:
- v1.0.6 实现了自动订阅功能
- 默认 30 分钟间隔硬编码在代码中
- 缺少配置文档说明如何调整

**需要的文档**:
- [ ] 创建 `docs/AUTO_BOOKING.md`
- [ ] 说明自动订阅工作原理
- [ ] 如何调整订阅间隔
- [ ] 如何手动触发订阅
- [ ] 故障排查指南

**临时方案**:
- 修改 `main.go` 中的硬编码值:
  ```go
  autoBookingScheduler := services.NewAutoBookingScheduler(autoBooking, 30*time.Minute)
  ```

---

### 5. 缺少 The Sports 集成文档

**状态**: 🟡 文档缺失  
**影响**: 用户不了解 The Sports API 的使用方法  
**优先级**: 中

**问题描述**:
- The Sports SDK 已集成
- 但缺少详细的使用文档
- 用户不知道如何订阅比赛、查询数据

**需要的文档**:
- [ ] 创建 `docs/THESPORTS_INTEGRATION.md`
- [ ] The Sports API 架构说明
- [ ] MQTT Topic 订阅指南
- [ ] REST API 使用示例
- [ ] 数据映射说明 (The Sports → LD 表)
- [ ] 故障排查

**临时方案**:
- 参考 RELEASE_NOTES_v1.0.6.md 中的 API 端点说明
- 查看 `thesports/` 目录中的代码注释

---

### 6. 缺少 API 文档

**状态**: 🟡 文档缺失  
**影响**: 用户不了解完整的 API 端点  
**优先级**: 中

**问题描述**:
- README.md 中只有部分 API 文档
- 缺少新增的 API 端点说明:
  - `/api/booking/*` - 自动订阅 API
  - `/api/thesports/*` - The Sports API
  - `/api/recovery/*` - 恢复 API
  - `/api/replay/*` - Replay API
  - `/api/ld/*` - Live Data API

**需要的文档**:
- [ ] 创建 `docs/API.md`
- [ ] 完整的 API 端点列表
- [ ] 请求/响应示例
- [ ] 错误代码说明
- [ ] 认证方式 (如果需要)

---

### 7. Live Data 状态检查未实现

**状态**: 🟡 TODO 标记  
**影响**: 无法准确获取 LD 连接状态  
**优先级**: 低

**问题描述**:
- `web/ld_control.go:73` 有 TODO 注释:
  ```go
  // TODO: 添加实际的连接状态检查
  ```
- 当前 `/api/ld/status` 返回的是假数据
- 无法真实反映 LD 客户端连接状态

**解决方案**:
- [ ] 在 `LDClient` 中添加 `IsConnected()` 方法
- [ ] 在 `handleLDStatus` 中调用实际状态
- [ ] 返回真实的连接信息

**代码位置**:
- `web/ld_control.go:73`
- `services/ld_client.go`

---

### 8. 不支持动态调整自动订阅间隔

**状态**: 🟡 功能缺失  
**影响**: 需要修改代码才能调整订阅间隔  
**优先级**: 低

**问题描述**:
- 自动订阅间隔硬编码为 30 分钟
- 无法通过环境变量配置
- 无法通过 API 动态调整

**建议的解决方案**:
- [ ] 添加环境变量 `AUTO_BOOKING_INTERVAL` (分钟)
- [ ] 在 `config.go` 中读取配置
- [ ] 添加 API 端点调整间隔:
  ```
  PUT /api/booking/config
  {
    "interval_minutes": 15
  }
  ```

**临时方案**:
- 修改 `main.go` 中的硬编码值

---

## 🟢 优化改进 (低优先级)

### 9. The Sports 仅支持足球

**状态**: 🟢 功能限制  
**影响**: 无法获取篮球、网球等其他体育数据  
**优先级**: 低

**问题描述**:
- The Sports SDK 包含篮球和电竞支持
- 但当前只集成了足球数据
- `thesports/basketball.go` 和 `thesports/esports.go` 未使用

**扩展计划**:
- [ ] 添加篮球数据集成
- [ ] 添加电竞数据集成
- [ ] 添加网球数据集成
- [ ] 支持多体育项目切换

**相关文件**:
- `thesports/basketball.go` - 篮球 API (已有)
- `thesports/esports.go` - 电竞 API (已有)
- `services/thesports_client.go` - 需要扩展

---

### 10. 缺少数据源切换机制

**状态**: 🟢 未实现  
**影响**: 无法在 Live Data 和 The Sports 之间切换  
**优先级**: 低

**问题描述**:
- 当前只能使用一个数据源
- 无法根据可用性自动切换
- 无法对比两个数据源的数据

**建议的功能**:
- [ ] 数据源抽象接口
- [ ] 支持多数据源并存
- [ ] 自动故障切换
- [ ] 数据对比和验证
- [ ] 数据质量评分

**架构设计**:
```go
type DataSource interface {
    Connect() error
    Disconnect() error
    IsConnected() bool
    GetMatches() ([]Match, error)
    SubscribeMatch(matchID string) error
}

type DataSourceManager struct {
    sources []DataSource
    primary DataSource
    fallback DataSource
}
```

---

### 11. 缺少数据质量监控

**状态**: 🟢 未实现  
**影响**: 无法评估数据源质量  
**优先级**: 低

**问题描述**:
- 无法监控数据完整性
- 无法检测数据延迟
- 无法对比不同数据源

**建议的功能**:
- [ ] 数据延迟监控
- [ ] 数据完整性检查
- [ ] 数据准确性对比
- [ ] 质量报告和告警

---

### 12. 性能优化 - MQTT 消息处理

**状态**: 🟢 可优化  
**影响**: 高频消息可能导致性能问题  
**优先级**: 低

**问题描述**:
- The Sports MQTT 消息处理是同步的
- 高频消息可能阻塞
- 缺少消息队列缓冲

**优化方案**:
- [ ] 使用消息队列缓冲
- [ ] 批量处理消息
- [ ] 异步数据库写入
- [ ] 添加性能监控

**代码位置**:
- `services/thesports_client.go:handleMessage()`

---

### 13. 缺少单元测试

**状态**: 🟢 测试不足  
**影响**: 代码质量和稳定性  
**优先级**: 低

**问题描述**:
- 大部分服务缺少单元测试
- 只有 `thesports/client_test.go` 有测试
- 缺少集成测试

**需要的测试**:
- [ ] `services/auto_booking_test.go`
- [ ] `services/thesports_client_test.go`
- [ ] `web/handlers_test.go`
- [ ] 集成测试

---

### 14. 错误处理可以改进

**状态**: 🟢 可优化  
**影响**: 错误信息不够详细  
**优先级**: 低

**问题描述**:
- 部分错误只记录日志,未返回给调用者
- 错误信息不够详细
- 缺少错误码定义

**改进方案**:
- [ ] 定义统一的错误码
- [ ] 改进错误消息格式
- [ ] 添加错误上下文信息
- [ ] 错误追踪和监控

---

### 15. 日志级别不可配置

**状态**: 🟢 功能缺失  
**影响**: 无法动态调整日志详细程度  
**优先级**: 低

**问题描述**:
- 日志级别硬编码
- 无法通过环境变量配置
- 生产环境可能有过多日志

**建议的功能**:
- [ ] 添加环境变量 `LOG_LEVEL` (debug/info/warn/error)
- [ ] 使用结构化日志库 (如 zap, logrus)
- [ ] 支持日志输出格式配置 (JSON/文本)

---

### 16. 缺少健康检查详细信息

**状态**: 🟢 可优化  
**影响**: 健康检查信息不够详细  
**优先级**: 低

**问题描述**:
- `/api/health` 只返回简单的 OK
- 缺少各组件的健康状态
- 无法快速定位问题

**建议的改进**:
```json
{
  "status": "ok",
  "timestamp": 1234567890,
  "components": {
    "database": {
      "status": "ok",
      "latency_ms": 5
    },
    "amqp": {
      "status": "ok",
      "connected": true
    },
    "thesports": {
      "status": "ok",
      "connected": true
    },
    "auto_booking": {
      "status": "ok",
      "last_run": "2025-10-23T00:00:00Z",
      "next_run": "2025-10-23T00:30:00Z"
    }
  }
}
```

---

## ⚙️ 配置问题

### 17. 环境变量文档不完整

**状态**: ⚙️ 文档问题  
**影响**: 用户不清楚所有可用的配置选项  
**优先级**: 中

**问题描述**:
- `.env.example` 缺少部分环境变量说明
- README.md 中的环境变量表不完整
- 缺少配置示例

**缺少的环境变量说明**:
- `UOF_USERNAME` - UOF 用户名
- `UOF_PASSWORD` - UOF 密码
- `AUTO_RECOVERY` - 自动恢复开关
- `RECOVERY_AFTER_HOURS` - 恢复时间范围
- `RECOVERY_PRODUCTS` - 恢复产品列表

**解决方案**:
- [ ] 更新 `.env.example` 添加所有变量
- [ ] 更新 README.md 环境变量表
- [ ] 添加配置说明文档

---

### 18. Railway 部署配置不完整

**状态**: ⚙️ 部署问题  
**影响**: 新用户部署困难  
**优先级**: 中

**问题描述**:
- Railway 部署步骤不够详细
- 缺少数据库配置说明
- 缺少环境变量配置清单

**需要补充的内容**:
- [ ] Railway PostgreSQL 配置步骤
- [ ] 完整的环境变量清单
- [ ] 部署后验证步骤
- [ ] 常见部署问题排查

---

## 📝 待办事项优先级排序

### 立即处理 (本周)
1. 🔴 配置 The Sports API 凭证 (#3)
2. 📝 创建 API 文档 (#6)
3. 📝 创建自动订阅配置文档 (#4)

### 短期 (本月)
4. 🔴 联系 Sportradar 解决 Replay API 问题 (#2)
5. 📝 创建 The Sports 集成文档 (#5)
6. ⚙️ 完善环境变量文档 (#17)
7. 🟡 实现 LD 状态检查 (#7)

### 中期 (下个月)
8. 🟡 支持动态调整自动订阅间隔 (#8)
9. 🔴 等待 Live Data IP 白名单完成 (#1)
10. 🟢 添加篮球和电竞支持 (#9)
11. ⚙️ 完善 Railway 部署文档 (#18)

### 长期 (季度)
12. 🟢 实现数据源切换机制 (#10)
13. 🟢 添加数据质量监控 (#11)
14. 🟢 性能优化 (#12)
15. 🟢 添加单元测试 (#13)
16. 🟢 改进错误处理 (#14)
17. 🟢 支持日志级别配置 (#15)
18. 🟢 改进健康检查 (#16)

---

## 📊 问题统计

### 按类别
- 🔴 阻塞性问题: 2 个
- 🟡 功能缺失: 5 个
- 🟢 优化改进: 8 个
- 📝 文档缺失: 3 个
- ⚙️ 配置问题: 2 个

### 按优先级
- 高优先级: 2 个
- 中优先级: 10 个
- 低优先级: 8 个

### 按状态
- 未解决: 18 个
- 已有临时方案: 2 个 (#1, #3)

---

## 🎯 下一步行动

### 本周目标
1. ✅ 完成 v1.0.6 发布 (已完成)
2. [ ] 配置 The Sports API 凭证
3. [ ] 创建 API 文档
4. [ ] 创建自动订阅配置文档

### 本月目标
1. [ ] 解决 Replay API 问题
2. [ ] 完善所有文档
3. [ ] 测试 Live Data 连接
4. [ ] 实现 LD 状态检查

### 长期目标
1. [ ] 实现双数据源支持
2. [ ] 添加数据质量监控
3. [ ] 性能优化和测试
4. [ ] 完善错误处理和日志

---

**文档生成时间**: 2025-10-23  
**文档维护者**: 开发团队  
**下次更新**: 每周更新

