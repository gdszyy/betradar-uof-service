# Betradar UOF Go Service

完整的Betradar Unified Odds Feed (UOF) 解决方案,使用Go语言实现,支持部署到Railway。

## 架构

```
Betradar AMQP → Go服务(Railway) → PostgreSQL → WebSocket → 前端浏览器
                                  ↓
                              REST API
```

### 核心功能

- ✅ **AMQP消费者** - 连接到Betradar AMQP服务器,接收实时消息
- ✅ **数据库存储** - 将所有消息存储到PostgreSQL数据库
- ✅ **WebSocket服务** - 实时推送消息到前端客户端
- ✅ **REST API** - 提供查询接口
- ✅ **消息过滤** - 支持按消息类型和赛事ID过滤
- ✅ **自动重连** - AMQP和WebSocket自动重连
- ✅ **生产者监控** - 监控alive消息,跟踪生产者状态

## 快速开始

### 本地开发

#### 1. 安装依赖

```bash
go mod download
```

#### 2. 配置环境变量

复制 `.env.example` 到 `.env` 并填写配置:

```bash
cp .env.example .env
```

编辑 `.env`:

```env
BETRADAR_ACCESS_TOKEN=your_access_token
BETRADAR_MESSAGING_HOST=stgmq.betradar.com:5671
BETRADAR_API_BASE_URL=https://stgapi.betradar.com/v1
ROUTING_KEYS=#
DATABASE_URL=postgres://localhost:5432/uof?sslmode=disable
PORT=8080
```

#### 3. 启动PostgreSQL

```bash
# 使用Docker
docker run -d \
  --name uof-postgres \
  -e POSTGRES_DB=uof \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:15
```

#### 4. 运行服务

```bash
go run main.go
```

服务将启动在 `http://localhost:8080`

#### 5. 访问Web界面

打开浏览器访问: `http://localhost:8080`

## Railway部署

### 方法1: 使用Railway CLI

#### 1. 安装Railway CLI

```bash
npm install -g @railway/cli
```

#### 2. 登录Railway

```bash
railway login
```

#### 3. 创建新项目

```bash
railway init
```

#### 4. 添加PostgreSQL

```bash
railway add postgresql
```

#### 5. 设置环境变量

```bash
railway variables set BETRADAR_ACCESS_TOKEN=your_token
railway variables set BETRADAR_MESSAGING_HOST=stgmq.betradar.com:5671
railway variables set BETRADAR_API_BASE_URL=https://stgapi.betradar.com/v1
railway variables set ROUTING_KEYS=#
```

#### 6. 部署

```bash
railway up
```

### 方法2: 使用Railway Dashboard

#### 1. 创建新项目

访问 [railway.app](https://railway.app) 并创建新项目

#### 2. 添加PostgreSQL数据库

点击 "New" → "Database" → "PostgreSQL"

#### 3. 部署Go服务

点击 "New" → "GitHub Repo" → 选择您的仓库

#### 4. 配置环境变量

在项目设置中添加以下环境变量:

- `BETRADAR_ACCESS_TOKEN`: 您的Betradar访问令牌
- `BETRADAR_MESSAGING_HOST`: stgmq.betradar.com:5671
- `BETRADAR_API_BASE_URL`: https://stgapi.betradar.com/v1
- `ROUTING_KEYS`: # (或自定义routing keys)
- `DATABASE_URL`: (自动设置,连接到PostgreSQL)
- `PORT`: (自动设置)

#### 5. 部署

Railway会自动检测Dockerfile并部署

## API文档

### REST API

#### 健康检查

```
GET /api/health
```

响应:
```json
{
  "status": "ok",
  "time": 1234567890
}
```

#### 获取消息列表

```
GET /api/messages?limit=50&offset=0&event_id=sr:match:12345&message_type=odds_change
```

参数:
- `limit`: 每页数量(默认50,最大100)
- `offset`: 偏移量(默认0)
- `event_id`: 过滤赛事ID(可选)
- `message_type`: 过滤消息类型(可选)

响应:
```json
{
  "messages": [
    {
      "id": 1,
      "message_type": "odds_change",
      "event_id": "sr:match:12345",
      "product_id": 1,
      "sport_id": "sr:sport:1",
      "routing_key": "hi.-.live.odds_change.1.sr:match.12345.-",
      "xml_content": "<odds_change>...</odds_change>",
      "timestamp": 1234567890,
      "received_at": "2024-01-01T00:00:00Z",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ],
  "limit": 50,
  "offset": 0
}
```

#### 获取跟踪的赛事

```
GET /api/events
```

响应:
```json
{
  "events": [
    {
      "id": 1,
      "event_id": "sr:match:12345",
      "sport_id": "sr:sport:1",
      "status": "active",
      "message_count": 150,
      "last_message_at": "2024-01-01T00:00:00Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### 获取特定赛事的消息

```
GET /api/events/{event_id}/messages
```

响应:
```json
{
  "event_id": "sr:match:12345",
  "messages": [...]
}
```

#### 获取统计信息

```
GET /api/stats
```

响应:
```json
{
  "total_messages": 10000,
  "total_events": 50,
  "odds_changes": 5000,
  "bet_stops": 200,
  "bet_settlements": 150
}
```

### WebSocket API

#### 连接

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

#### 订阅消息

发送:
```json
{
  "type": "subscribe",
  "message_types": ["odds_change", "bet_stop"],
  "event_ids": ["sr:match:12345", "sr:match:67890"]
}
```

#### 取消订阅

发送:
```json
{
  "type": "unsubscribe"
}
```

#### 接收消息

```json
{
  "type": "message",
  "message_type": "odds_change",
  "event_id": "sr:match:12345",
  "product_id": 1,
  "routing_key": "hi.-.live.odds_change.1.sr:match.12345.-",
  "xml": "<odds_change>...</odds_change>",
  "timestamp": 1234567890
}
```

## 前端客户端使用

### 引入客户端

```html
<script src="/uof-client.js"></script>
```

### 创建客户端

```javascript
const client = window.createUOFClient({
  wsUrl: 'ws://localhost:8080/ws',  // 可选,默认自动检测
  apiUrl: 'http://localhost:8080/api', // 可选,默认自动检测
  autoReconnect: true,
  reconnectInterval: 3000
});
```

### 连接

```javascript
client.connect();
```

### 监听事件

```javascript
// 连接成功
client.on('connected', () => {
  console.log('Connected');
});

// 接收消息
client.on('message', (msg) => {
  console.log('Message:', msg);
});

// 赔率变化
client.on('odds_change', (msg) => {
  console.log('Odds change:', msg.event_id);
  console.log('XML:', msg.xml);
});

// 投注停止
client.on('bet_stop', (msg) => {
  console.log('Bet stop:', msg.event_id);
});

// 投注结算
client.on('bet_settlement', (msg) => {
  console.log('Bet settlement:', msg.event_id);
});
```

### 订阅特定消息

```javascript
// 只订阅特定消息类型
client.subscribe(['odds_change', 'bet_stop'], []);

// 只订阅特定赛事
client.subscribe([], ['sr:match:12345', 'sr:match:67890']);

// 同时过滤消息类型和赛事
client.subscribe(['odds_change'], ['sr:match:12345']);
```

### 调用API

```javascript
// 获取消息列表
const messages = await client.getMessages({
  limit: 50,
  offset: 0,
  event_id: 'sr:match:12345'
});

// 获取跟踪的赛事
const events = await client.getTrackedEvents();

// 获取特定赛事的消息
const eventMessages = await client.getEventMessages('sr:match:12345');

// 获取统计信息
const stats = await client.getStats();
```

## 数据库表结构

### uof_messages
存储所有UOF消息

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| message_type | VARCHAR(50) | 消息类型 |
| event_id | VARCHAR(100) | 赛事ID |
| product_id | INTEGER | 产品ID |
| sport_id | VARCHAR(50) | 运动ID |
| routing_key | VARCHAR(255) | 路由键 |
| xml_content | TEXT | XML内容 |
| timestamp | BIGINT | 消息时间戳 |
| received_at | TIMESTAMP | 接收时间 |
| created_at | TIMESTAMP | 创建时间 |

### tracked_events
跟踪的赛事

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| event_id | VARCHAR(100) | 赛事ID(唯一) |
| sport_id | VARCHAR(50) | 运动ID |
| status | VARCHAR(20) | 状态 |
| message_count | INTEGER | 消息数量 |
| last_message_at | TIMESTAMP | 最后消息时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### odds_changes
赔率变化记录

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| event_id | VARCHAR(100) | 赛事ID |
| product_id | INTEGER | 产品ID |
| timestamp | BIGINT | 时间戳 |
| odds_change_reason | VARCHAR(50) | 变化原因 |
| markets_count | INTEGER | 市场数量 |
| xml_content | TEXT | XML内容 |
| created_at | TIMESTAMP | 创建时间 |

### bet_stops
投注停止记录

### bet_settlements
投注结算记录

### producer_status
生产者状态

## 配置说明

### Routing Keys

Routing key格式:
```
{priority}.{pre}.{live}.{message_type}.{sport_id}.{urn_type}.{event_id}.{node_id}
```

示例:
- `#` - 订阅所有消息
- `*.*.live.odds_change.#` - 所有实时赔率变化
- `*.*.live.odds_change.1.#` - 足球实时赔率变化
- `*.pre.-.bet_settlement.#` - 所有赛前投注结算

### 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| BETRADAR_ACCESS_TOKEN | Betradar访问令牌 | (必填) |
| BETRADAR_MESSAGING_HOST | AMQP服务器地址 | stgmq.betradar.com:5671 |
| BETRADAR_API_BASE_URL | API服务器地址 | https://stgapi.betradar.com/v1 |
| ROUTING_KEYS | 路由键(逗号分隔) | # |
| DATABASE_URL | PostgreSQL连接URL | (必填) |
| PORT | HTTP服务器端口 | 8080 |
| ENVIRONMENT | 环境(development/production) | development |

## 监控和日志

### 查看日志

```bash
# Railway
railway logs

# Docker
docker logs uof-service

# 本地
# 日志输出到stdout
```

### 监控指标

- 总消息数
- 跟踪的赛事数
- 赔率变化数
- 投注停止数
- 投注结算数
- 生产者状态

## 故障排除

### AMQP连接失败

**检查项:**
- Access Token是否有效
- 网络是否能访问stgmq.betradar.com:5671
- 防火墙是否开放5671端口

### 数据库连接失败

**检查项:**
- DATABASE_URL是否正确
- PostgreSQL是否运行
- 数据库权限是否正确

### WebSocket连接失败

**检查项:**
- 服务是否运行
- 端口是否开放
- CORS配置是否正确

## 生产环境建议

1. **使用环境变量管理配置**
2. **启用数据库连接池**
3. **配置日志级别**
4. **设置合理的routing keys过滤**
5. **定期清理旧消息**
6. **监控服务健康状态**
7. **配置告警**

## 技术栈

- **语言**: Go 1.21
- **Web框架**: Gorilla Mux
- **WebSocket**: Gorilla WebSocket
- **AMQP**: streadway/amqp
- **数据库**: PostgreSQL
- **部署**: Railway / Docker

## 许可证

MIT License

## 支持

如有问题,请参考:
- [Betradar官方文档](https://docs.betradar.com)
- [Railway文档](https://docs.railway.app)

