# UOF 项目监控与验收机制说明

## 1. 数据留存策略调整
为了优化数据库存储空间，系统已将 `uof_messages` 表的留存策略从默认的 7 天调整为 **3 天**。
- **执行逻辑**：每天凌晨 2 点，`DataCleanupService` 会自动删除 `received_at` 早于 3 天前的原始消息记录。
- **修改位置**：`main.go` 中的 `cleanupConfig` 初始化部分。

---

## 2. 飞书通知过滤
根据需求，系统已关闭所有常规业务报送，仅保留**异常告警**。
- **已关闭的通知**：
    - 服务启动通知 (`NotifyServiceStart`)
    - 每日数据清理汇总 (`NotifyDataCleanup`)
    - 自动订阅结果汇总 (`NotifyPrematchBooking`)
    - 每小时赛事订阅监控 (`NotifyMatchMonitor`)
- **保留的通知**：
    - 系统错误告警 (`NotifyError`)
    - 业务异常告警（由新监控模块触发）

---

## 3. 业务验收监控机制 (Business Monitor)
系统新增了 `BusinessMonitor` 服务，专门用于识别并记录业务层面的异常情况。

### 3.1 异常记录库
所有识别到的异常都会记录在 `exceptions` 表中，包含以下字段：
- `type`: 异常类型（如 `LATE_START`）
- `event_id`: 关联的赛事 ID
- `message`: 异常详细描述
- `severity`: 严重程度（high, medium, low）
- `created_at`: 记录时间

### 3.2 核心监控项
| 监控项 | 异常类型 | 判定规则 | 严重程度 |
| :--- | :--- | :--- | :--- |
| **开赛监控** | `LATE_START` | 超过预计开赛时间 15 分钟，但状态仍为 `not_started`。 | High |
| **赔率停滞** | `ODDS_STAGNATION` | 滚球中 (`live`) 的赛事超过 30 分钟未收到任何赔率更新消息。 | Medium |
| **结算缺失** | `MISSING_SETTLEMENT` | 赛事已结束 (`ended`) 超过 2 小时，但未收到结算消息。 | High |

### 3.3 告警流程
1. 监控任务每 10 分钟执行一次扫描。
2. 发现异常后，首先检查 `exceptions` 表，避免对同一赛事的同一异常重复告警。
3. 将异常写入数据库。
4. 调用 `LarkNotifier.NotifyError` 将异常详情实时推送到飞书。

---

## 4. 开发者参考
- **监控代码位置**：`services/business_monitor.go`
- **数据库迁移脚本**：`database/migrations/019_create_exceptions_table.sql`
- **启动逻辑**：在 `main.go` 中通过 `businessMonitor.Start()` 启动。
