# Betradar UOF Service 使用说明

本文档介绍了如何配置、部署和运行 Betradar UOF Service。

## 1. 环境要求
- **Go**: 1.21 或更高版本
- **PostgreSQL**: 14 或更高版本
- **Betradar 账号**: 需具备 UOF 访问权限的 Bookmaker ID 及 Access Token

## 2. 配置说明
项目通过环境变量进行配置，核心配置项如下：

| 环境变量 | 说明 | 示例 |
| :--- | :--- | :--- |
| `DATABASE_URL` | 数据库连接字符串 | `postgres://user:pass@localhost:5432/uof` |
| `BETRADAR_ACCESS_TOKEN` | Betradar API 访问令牌 | `your_token_here` |
| `LARK_WEBHOOK` | 飞书机器人 Webhook 地址 | `https://open.feishu.cn/...` |
| `UOF_BOOKMAKER_ID` | 您的 Bookmaker ID | `1234` |

## 3. 快速启动

### 本地运行
1. 克隆仓库并进入目录。
2. 安装依赖：`go mod download`。
3. 启动服务：`go run main.go`。

### Docker 部署
```bash
docker build -t uof-service .
docker run --env-file .env uof-service
```

## 4. 常用接口
- **WebSocket**: `ws://localhost:8080/ws` - 实时接收赔率和赛事更新。
- **健康检查**: `GET /health` - 检查服务运行状态。
- **监控触发**: `POST /api/monitor/trigger` - 手动触发一次业务异常巡检。

## 5. 运维建议
- **日志查看**: 服务日志默认输出至标准输出，建议配合 ELK 或 Grafana Loki 使用。
- **数据清理**: 系统会自动清理 3 天前的原始消息，请确保数据库磁盘空间充足。
- **告警响应**: 收到飞书告警后，请优先检查 `exceptions` 表中的详细错误上下文。
