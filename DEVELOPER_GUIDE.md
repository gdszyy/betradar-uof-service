# Betradar UOF Service 开发者模块映射指南

为了方便技术人员进行问题定位和迭代开发，本项目的所有 Go 源文件已按功能职责划分为以下五大核心模块。

---

## 1. 核心引擎模块 (Core Engine)
**职责**：负责与外部数据源建立连接、消息的初步接收、内部路由分发以及系统的启动引导。
**定位建议**：如果出现连接中断、消息丢失或系统无法启动的问题，请优先检查此模块。

| 文件路径 | 功能描述 |
| :--- | :--- |
| `main.go` | 服务入口，负责所有组件的初始化和生命周期管理。 |
| `services/amqp_connector.go` | 管理与 Betradar RabbitMQ 的连接及重连逻辑。 |
| `services/amqp_consumer.go` | 原始消息 Ingestor，负责接收并持久化原始 XML。 |
| `services/broker.go` | 消息代理接口定义。 |
| `services/in_memory_broker.go` | 内存消息队列实现，负责 Topic 订阅与分发。 |
| `services/message_processor.go` | 业务分发中枢，根据消息类型调用对应的 Parser。 |
| `services/message_store.go` | 负责原始消息在数据库中的存取。 |

---

## 2. 业务解析模块 (Business Parsers)
**职责**：负责将原始 XML 消息解析为业务模型，并执行核心业务逻辑（如赔率更新、结算、赛事变更）。
**定位建议**：如果出现赔率不更新、比分错误、结算异常等业务逻辑问题，请检查此模块。

| 文件路径 | 功能描述 |
| :--- | :--- |
| `services/odds_change_parser.go` | **核心**：处理赔率变动、比分及比赛状态更新。 |
| `services/fixture_parser.go` | 处理赛事元数据（时间、对阵、联赛）的解析。 |
| `services/bet_settlement_parser.go` | 处理投注结算逻辑。 |
| `services/bet_stop_processor.go` | 处理盘口暂停（Bet Stop）逻辑。 |
| `services/bet_cancel_processor.go` | 处理投注取消逻辑。 |
| `services/rollback_*.go` | 处理结算或取消的回滚逻辑。 |
| `services/odds_parser.go` | 通用的赔率结构解析工具。 |

---

## 3. 元数据与增强服务 (Metadata & Enrichment)
**职责**：提供赛事、市场、队伍、球员等基础数据的查询、翻译及分类增强功能。
**定位建议**：如果出现市场名称显示错误、队伍 Logo 缺失、分类（Tab/Chip）逻辑异常，请检查此模块。

| 文件路径 | 功能描述 |
| :--- | :--- |
| `services/market_descriptions_service.go` | 管理市场定义，处理动态名称替换（如 `{{total}}`）。 |
| `services/market_groups_service.go` | **关键**：负责市场的 Tab 和 Chip 分类逻辑。 |
| `services/teams_service.go` | 维护队伍信息。 |
| `services/logo_fetcher_service.go` | 自动抓取并缓存队伍 Logo。 |
| `services/players_service.go` | 球员信息管理。 |
| `services/srn_mapping_service.go` | 处理 Sportradar 内部 ID 的映射关系。 |
| `services/static_data_service.go` | 定期同步体育、联赛、类别等静态元数据。 |

---

## 4. 运维监控与自动化 (Ops & Automation)
**职责**：负责系统健康检查、数据清理、自动订阅管理以及第三方通知（飞书）。
**定位建议**：如果出现磁盘空间不足、数据源中断未告警、自动订阅失效，请检查此模块。

| 文件路径 | 功能描述 |
| :--- | :--- |
| `services/producer_monitor.go` | 监控 Betradar 数据源（Producer）的活跃度。 |
| `services/lark_notifier.go` | 飞书机器人通知集成。 |
| `services/stale_live_cleanup.go` | 清理过期的滚球赛事数据。 |
| `services/subscription_cleanup.go` | 自动取消已结束赛事的订阅。 |
| `services/auto_booking.go` | 赛事的自动预订（Booking）逻辑。 |
| `services/message_stats.go` | 统计消息流量及处理延迟。 |

---

## 5. 接口与分发层 (API & Distribution)
**职责**：通过 REST API 和 WebSocket 为前端提供数据访问接口。
**定位建议**：如果前端请求报错、WebSocket 断连或数据推送不及时，请检查此模块。

| 文件路径 | 功能描述 |
| :--- | :--- |
| `web/server.go` | HTTP 服务器配置及路由定义。 |
| `web/websocket.go` | WebSocket Hub 实现，负责实时数据广播。 |
| `web/*_handler.go` | 各类业务接口的具体处理函数（如 `odds_handlers.go`）。 |
| `handlers/market_tab_chip_handler.go` | 专门处理市场分类配置的接口。 |

---

## 快速导航建议
*   **新功能开发**：通常涉及 `services/message_processor.go` (注册新消息) 和 `web/server.go` (暴露新接口)。
*   **数据准确性排查**：查看 `services/odds_change_parser.go` 中的数据库写入逻辑。
*   **性能优化**：关注 `services/in_memory_broker.go` 的并发处理能力。
