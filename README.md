# Betradar UOF Service - 开发者指南

本项目是一个基于 Betradar Unified Odds Feed (UOF) 的高性能实时赔率处理引擎。本文档旨在帮助开发者快速理解项目架构、代码组织及核心逻辑，以便进行高效的协作开发。

## 1. 项目全景架构

系统采用分层解耦设计，确保了高并发消息处理的稳定性和可扩展性：

- **接入层 (Ingestor)**: 负责与 Betradar AMQP 服务器建立长连接，接收原始 XML 消息并进行初步校验。
- **消息层 (Broker)**: 内部采用内存 Broker 机制，实现生产者与消费者的解耦。
- **处理层 (Processor)**: 核心业务逻辑所在地，负责 XML 解析、状态机维护及业务规则校验。
- **存储层 (Storage)**: 基于 PostgreSQL 的持久化存储，记录赛事、赔率及异常流水。
- **分发层 (Distribution)**: 通过 WebSocket 实时推送处理结果，并提供 RESTful API 供外部查询。

## 2. 核心模块职责

| 模块 | 目录/文件 | 职责描述 |
| :--- | :--- | :--- |
| **启动入口** | `main.go` | 负责全局配置加载、数据库连接初始化及各后台服务的生命周期管理。 |
| **消息接入** | `services/amqp_*` | 处理 AMQP 连接管理、自动重连及原始消息流转。 |
| **业务处理** | `services/message_processor.go` | 消息路由中心，根据消息类型分发至不同的 Parser。 |
| **解析引擎** | `services/*_parser.go` | 负责 Odds、Settlement、Fixture 等不同业务消息的 XML 解析与模型转换。 |
| **数据模型** | `database/models.go` | 定义系统核心实体（Event, Market, Outcome）的数据库映射。 |
| **接口服务** | `web/server.go` | 统一的 HTTP 路由管理，包含 API 逻辑与 WebSocket Hub 维护。 |
| **业务监控** | `services/business_monitor.go` | 独立的监控协程，负责开赛延迟、赔率停滞等业务异常的巡检。 |

## 3. 协作开发规范

- **代码风格**: 遵循标准 Go 代码规范，提交前请运行 `go fmt`。
- **异常处理**: 业务异常必须记录至 `exceptions` 表，并通过 `LarkNotifier` 发送告警。
- **数据留存**: `uof_messages` 等高频表的数据留存策略统一在 `main.go` 中配置（当前为 3 天）。
- **文档更新**: 新增业务逻辑时，请同步更新 `docs/ARCHITECTURE_ANALYSIS.md`。

## 4. 技术栈
- **语言**: Go 1.21+
- **数据库**: PostgreSQL
- **消息协议**: AMQP (RabbitMQ 兼容)
- **实时通信**: WebSocket

---
**注意**: 关于如何配置环境及运行服务，请参阅 [使用说明文档](docs/USAGE_GUIDE.md)。
