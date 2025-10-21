# Feishu (Lark) Bot Integration

## 概述

本服务已集成飞书(Lark)机器人,用于实时监控和通知。所有重要事件和统计信息都会自动发送到指定的飞书群组。

## 功能特性

### 1. 自动通知

服务会自动发送以下类型的通知到飞书:

#### 🚀 服务启动通知
- 服务启动时自动发送
- 包含 Bookmaker ID 和订阅的产品列表
- 帮助确认服务正常启动

#### 📊 消息统计报告 (每5分钟)
- 自动统计接收到的消息类型和数量
- 按消息类型分组显示
- 第一次报告显示"启动至今"的统计
- 后续报告显示"过去5分钟"的增量统计

#### 🎯 比赛监控报告 (每小时)
- 自动查询已订阅的比赛
- 显示总比赛数、已订阅数、Pre-match 和 Live 比赛数
- 如果没有订阅比赛会发出警告
- 帮助诊断为什么没有收到 odds_change 消息

#### ✅ 恢复完成通知
- Recovery 请求完成时自动通知
- 包含 Product ID 和 Request ID

#### ❌ 错误通知
- 关键错误发生时自动通知
- 包含组件名称和错误详情

### 2. 手动触发

#### 手动触发监控检查

可以通过 API 手动触发监控检查:

```bash
curl -X POST https://betradar-uof-service-copy-production.up.railway.app/api/monitor/trigger
```

响应:
```json
{
  "status": "triggered",
  "message": "Monitor check triggered. Results will be sent to Feishu webhook.",
  "time": 1729485600
}
```

## 配置

### 环境变量

在 Railway 或本地 `.env` 文件中配置:

```bash
# 飞书 Webhook URL (必需)
LARK_WEBHOOK_URL=https://open.larksuite.com/open-apis/bot/v2/hook/your-webhook-id

# Bookmaker 配置 (可选,用于通知显示)
BOOKMAKER_ID=your_bookmaker_id
PRODUCTS=liveodds,pre
```

### 获取 Webhook URL

1. 在飞书中创建一个群组
2. 群组设置 → 群机器人 → 添加机器人 → 自定义机器人
3. 设置机器人名称和描述
4. 复制 Webhook 地址
5. 将地址配置到环境变量中

## 通知示例

### 服务启动通知

```
🚀 服务启动

Bookmaker ID: test-bookmaker
Products: [liveodds pre]
时间: 2025-10-21 13:40:00
```

### 消息统计报告

```
📊 消息统计 (过去 5 分钟)

总消息数: 242
  odds_change: 150
  bet_stop: 45
  bet_settlement: 30
  fixture_change: 12
  alive: 5
时间: 2025-10-21 13:45:00
```

### 比赛监控报告

```
✅ Match订阅监控

状态: 正常
总比赛数: 100
已订阅: 25
  - Pre-match: 15
  - Live: 10

时间: 2025-10-21 14:00:00
```

### 警告示例 (无订阅比赛)

```
⚠️ Match订阅监控

状态: 警告: 没有订阅的比赛
总比赛数: 100
已订阅: 0
  - Pre-match: 0
  - Live: 0

💡 建议: 订阅一些比赛以接收odds_change消息

时间: 2025-10-21 14:00:00
```

## 实现细节

### 核心组件

#### 1. LarkNotifier (`services/lark_notifier.go`)
- 负责发送飞书消息
- 支持文本消息和富文本消息
- 提供各种预定义的通知方法

#### 2. MessageStatsTracker (`services/message_stats.go`)
- 追踪消息统计
- 每5分钟自动报告一次
- 第一次报告后重置计数器

#### 3. MatchMonitor (`services/match_monitor.go`)
- 查询已订阅的比赛
- 分析订阅状态
- 每小时自动检查一次

### 集成点

1. **main.go**: 初始化通知器,启动定期任务
2. **amqp_consumer.go**: 在消息处理中记录统计,在恢复完成时发送通知
3. **web/server.go**: 提供手动触发监控的 API 接口

## 测试

### 本地测试

运行测试脚本验证 Feishu 集成:

```bash
cd /home/ubuntu/uof-go-service
go run tools/test_feishu.go
```

测试脚本会发送以下通知:
1. 文本消息
2. 服务启动通知
3. 消息统计通知
4. 比赛监控通知
5. 恢复完成通知
6. 错误通知

### 生产环境验证

部署后,检查飞书群组应该能看到:
1. 服务启动通知 (立即)
2. 比赛监控报告 (启动后立即,然后每小时)
3. 消息统计报告 (启动后5分钟,然后每5分钟)

## 故障排除

### 没有收到通知

1. **检查环境变量**: 确认 `LARK_WEBHOOK_URL` 已正确配置
2. **检查日志**: 查看服务日志中是否有错误信息
3. **验证 Webhook**: 使用 curl 测试 webhook 是否可用:

```bash
curl -X POST "YOUR_WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -d '{"msg_type":"text","content":{"text":"测试消息"}}'
```

### 通知频率过高

如果觉得通知太频繁,可以调整:

1. **消息统计间隔**: 修改 `main.go` 中的 `5*time.Minute`
2. **监控间隔**: 修改 `main.go` 中的 `1*time.Hour`

### 禁用通知

如果需要临时禁用通知,只需移除或清空 `LARK_WEBHOOK_URL` 环境变量,服务会自动禁用通知功能。

## API 文档

### POST /api/monitor/trigger

手动触发监控检查。

**请求**:
```bash
curl -X POST https://your-service.railway.app/api/monitor/trigger
```

**响应**:
```json
{
  "status": "triggered",
  "message": "Monitor check triggered. Results will be sent to Feishu webhook.",
  "time": 1729485600
}
```

**说明**:
- 触发后会立即发送通知到飞书
- 用于调试或按需检查订阅状态

## 最佳实践

1. **创建专用群组**: 为监控通知创建单独的飞书群组,避免干扰正常沟通
2. **设置群组名称**: 使用清晰的名称,如 "UOF Service Monitoring"
3. **添加相关人员**: 将运维和开发人员加入监控群组
4. **保存 Webhook**: 妥善保管 Webhook URL,避免泄露
5. **定期检查**: 定期查看通知,确保服务正常运行

## 未来改进

可能的改进方向:

1. **自定义通知规则**: 允许配置哪些事件需要通知
2. **通知级别**: 区分信息、警告、错误等级别
3. **@提醒**: 在关键错误时 @ 相关人员
4. **交互式卡片**: 使用飞书的交互式卡片提供更丰富的信息
5. **命令支持**: 通过飞书消息控制服务(如触发恢复)

## 相关文档

- [飞书机器人开发文档](https://open.feishu.cn/document/ukTMukTMukTM/ucTM5YjL3ETO24yNxkjN)
- [Match Monitoring 文档](./MATCH-MONITORING.md)
- [Recovery API 文档](../RECOVERY-API.md)

## 支持

如有问题或建议,请联系开发团队或提交 GitHub Issue。

