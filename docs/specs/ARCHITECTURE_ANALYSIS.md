# Betradar UOF Service 架构分析报告

## 1. 项目概述
`betradar-uof-service` 是一个基于 Go 语言开发的高性能服务，旨在接入、处理并分发来自 Betradar Unified Odds Feed (UOF) 的实时体育赛事数据。该服务不仅负责数据的持久化存储，还通过 REST API 和 WebSocket 为前端应用提供实时数据支持。

## 2. 核心架构设计
项目采用了**分层架构**与**事件驱动架构**相结合的设计模式，通过内部消息代理（Broker）实现了数据接入与业务处理的解耦。

### 2.1 架构分层
| 层次 | 组件 | 功能描述 |
| :--- | :--- | :--- |
| **接入层 (Ingestor)** | `AMQPConnector`, `AMQPConsumer` | 负责与 Betradar 的 RabbitMQ 服务器建立连接，接收原始 XML 消息。 |
| **消息层 (Broker)** | `InMemoryBroker` | 内部消息代理，作为 Kafka 的轻量级替代方案，实现消息的异步分发和削峰填谷。 |
| **处理层 (Processor)** | `MessageProcessor`, 各类 `Parser` | 核心业务逻辑层，负责解析 XML 消息并根据业务规则更新数据库。 |
| **服务层 (Service)** | `PlayersService`, `ScheduleService`, `MarketDescService` 等 | 提供基础数据支持、元数据管理、监控及清理等辅助功能。 |
| **分发层 (Web/API)** | `web.Server`, `web.Hub` (WebSocket) | 提供 RESTful API 接口和 WebSocket 实时推送服务。 |

---

## 3. 数据处理流程
数据在系统中的流动遵循以下路径：

1.  **数据采集**：`AMQPConsumer` 接收到 XML 消息后，首先将其原始数据持久化到 `uof_messages` 表中，确保数据可追溯。
2.  **消息分发**：消息被推送到 `InMemoryBroker`，并根据消息类型（如 `odds_change`, `fixture`）分发到不同的 Topic。
3.  **业务解析**：`MessageProcessor` 订阅相关 Topic，调用对应的解析器（如 `OddsChangeParser`）进行深度解析：
    *   更新赛事状态（`tracked_events`）。
    *   记录赔率变动（`odds_changes`）。
    *   处理结算信息（`bet_settlements`）。
    *   同步队伍及球员信息。
4.  **数据增强**：在处理过程中，系统会调用 `MarketDescriptionsService` 获取市场名称，调用 `MarketGroupsService` 为市场分配前端展示所需的 `tab_id` 和 `chip_id`。
5.  **实时推送**：处理后的结构化数据通过 `web.Hub` 实时广播给所有连接的 WebSocket 客户端。

---

## 4. 模块功能详解

### 4.1 核心业务模块
*   **Message Processor**：业务调度的中枢，协调各个解析器完成数据处理。
*   **Odds Change Parser**：处理最频繁的消息类型，负责实时赔率更新及比赛比分同步。
*   **Fixture Parser**：处理赛事元数据，包括开赛时间、对阵双方、联赛信息等。
*   **Market Descriptions Service**：管理 Betradar 复杂的市场定义，支持动态替换占位符（如 `{{home_team}}`）以生成可读的市场名称。

### 4.2 辅助支撑模块
*   **Market Groups Service**：根据预设规则（JSON/CSV 配置）对市场进行分类，解决前端“滚球”与“早盘”界面的市场展示逻辑。
*   **Producer Monitor**：实时监控 Betradar 各个数据源（Producer）的状态，一旦发现中断立即通过 Feishu 发送告警。
*   **Lark Notifier**：集成飞书机器人，负责服务启动、错误告警及每日数据汇总报告。
*   **Cleanup Services**：包括 `StaleLiveCleanup` 和 `SubscriptionCleanup`，定期清理过期数据，保证数据库性能。

---

## 5. 数据模型设计
系统核心表结构如下：
*   `uof_messages`：全量原始消息审计日志。
*   `tracked_events`：当前正在跟踪的活跃赛事状态。
*   `odds_changes` / `bet_settlements`：业务明细数据。
*   `teams`：队伍基础信息及 Logo 缓存状态。
*   `producer_status`：数据源健康状况记录。

---

## 6. 总结
`betradar-uof-service` 的设计充分考虑了体育数据高并发、高实时性的特点。通过**内存消息队列**实现了极低的处理延迟，通过**模块化设计**保证了业务逻辑的可维护性，是一套成熟的体育博彩数据接入解决方案。
