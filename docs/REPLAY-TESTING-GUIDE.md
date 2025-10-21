# 🎬 Replay Server 测试指南

## 概述

使用Betradar Replay Server可以重放已结束赛事的所有消息,这对于测试和开发非常有用。通过重放,您可以:

- ✅ 测试 `odds_change`, `bet_stop`, `bet_settlement` 等消息的处理
- ✅ 验证数据存储和解析逻辑
- ✅ 调试特定场景(VAR、加时赛、点球等)
- ✅ 快速验证管道功能而无需等待真实比赛

---

## 前提条件

### 1. 环境变量设置

在运行重放测试前,需要设置以下环境变量:

```bash
export UOF_USERNAME="your_betradar_username"
export UOF_PASSWORD="your_betradar_password"
export DATABASE_URL="postgresql://user:pass@host:port/database"  # 可选,用于监控
```

**在Railway上设置**:
1. 进入Railway Dashboard → 您的服务
2. 点击 Variables 标签
3. 添加 `UOF_USERNAME` 和 `UOF_PASSWORD`

### 2. 连接到Replay服务器

Replay服务器使用**不同的AMQP地址**:

```
生产环境: stgmq.betradar.com:5671
Replay环境: global.replaymq.betradar.com:5671
```

您的服务需要能够连接到Replay服务器。可以:
- **选项A**: 临时修改配置连接到Replay服务器
- **选项B**: 运行第二个实例连接到Replay服务器
- **选项C**: 使用环境变量切换(推荐)

---

## 快速开始

### 方法1: 使用Shell脚本(最简单)

```bash
# 1. 设置凭证
export UOF_USERNAME="your_username"
export UOF_PASSWORD="your_password"

# 2. 运行重放测试
cd /home/ubuntu/uof-go-service/tools
./replay_event.sh test:match:21797788 20 60 1

# 参数说明:
# - test:match:21797788: 赛事ID
# - 20: 重放速度(20倍速)
# - 60: 运行时长(秒)
# - 1: Node ID
```

### 方法2: 使用Go程序(更灵活)

```bash
# 1. 设置凭证
export UOF_USERNAME="your_username"
export UOF_PASSWORD="your_password"
export DATABASE_URL="your_database_url"

# 2. 运行Go测试程序
cd /home/ubuntu/uof-go-service
go run tools/test_replay.go \
  -event=test:match:21797788 \
  -speed=20 \
  -duration=60 \
  -node=1
```

### 方法3: 手动API调用

```bash
# 设置变量
API_BASE="https://api.betradar.com/v1"
AUTH="$UOF_USERNAME:$UOF_PASSWORD"
EVENT_ID="test:match:21797788"

# 1. 重置重放列表
curl -u "$AUTH" -X POST "$API_BASE/replay/reset"

# 2. 添加赛事
curl -u "$AUTH" -X PUT "$API_BASE/replay/events/$EVENT_ID"

# 3. 开始重放(20倍速,node_id=1)
curl -u "$AUTH" -X POST \
  "$API_BASE/replay/play?speed=20&max_delay=10000&node_id=1&use_replay_timestamp=true"

# 4. 查看状态
curl -u "$AUTH" "$API_BASE/replay/status"

# 5. 停止重放
curl -u "$AUTH" -X POST "$API_BASE/replay/stop"
```

---

## 推荐测试赛事

以下是一些适合测试不同场景的赛事:

### 足球(Soccer)

| 赛事ID | 描述 | 适用场景 |
|--------|------|----------|
| `test:match:21797788` | 足球比赛(VAR场景) | 测试赔率变化、VAR决策 |
| `test:match:21797805` | 加时赛 | 测试加时赛消息 |
| `test:match:21797815` | 点球大战 | 测试点球大战消息 |
| `sr:match:22340005` | 平局赛事 | 测试平局结算 |

### 网球(Tennis)

| 赛事ID | 描述 | 适用场景 |
|--------|------|----------|
| `test:match:21797802` | 5盘制抢十 | 测试网球特殊规则 |
| `test:match:21796642` | 温网决赛 | 测试长盘制 |

### 篮球(Basketball)

| 赛事ID | 描述 | 适用场景 |
|--------|------|----------|
| `sr:match:11234567` | NBA总决赛2017 | 测试篮球赔率 |

### 其他运动

- **棒球**: `test:match:23517711`
- **冰球**: `sr:match:11234567`
- **电竞(CS:GO)**: 查看文档获取最新ID

**完整列表**: https://docs.sportradar.com/uof/replay-server/uof-example-replays

---

## 配置服务连接到Replay服务器

### 选项A: 环境变量切换(推荐)

修改 `config/config.go`:

```go
func LoadConfig() *Config {
    // 默认使用生产环境
    amqpHost := "stgmq.betradar.com"
    
    // 如果设置了REPLAY_MODE,使用Replay服务器
    if os.Getenv("REPLAY_MODE") == "true" {
        amqpHost = "global.replaymq.betradar.com"
        log.Println("🎬 REPLAY MODE: Connecting to Replay Server")
    }
    
    return &Config{
        AMQPHost: amqpHost,
        // ... 其他配置
    }
}
```

使用时:

```bash
# Railway上设置环境变量
REPLAY_MODE=true

# 或本地运行时
export REPLAY_MODE=true
go run main.go
```

### 选项B: 运行专用Replay实例

在Railway上创建第二个服务实例:

1. 复制当前服务
2. 命名为 "uof-service-replay"
3. 设置环境变量:
   ```
   AMQP_HOST=global.replaymq.betradar.com
   REPLAY_MODE=true
   ```
4. 使用不同的数据库或schema(可选)

---

## 验证重放效果

### 1. 检查服务日志

在Railway Dashboard → Deployments → Logs中查看:

```
✅ 应该看到:
- "Connected to AMQP server"
- "Odds change for event test:match:21797788: X markets"
- "Bet stop for event test:match:21797788"
- "Bet settlement for event test:match:21797788"

❌ 不应该看到:
- "Failed to parse message"
- "Unknown message type"
```

### 2. 查询数据库

```sql
-- 检查消息总数
SELECT COUNT(*) FROM uof_messages WHERE created_at > NOW() - INTERVAL '5 minutes';

-- 检查消息类型分布
SELECT message_type, COUNT(*) as count
FROM uof_messages
WHERE created_at > NOW() - INTERVAL '5 minutes'
GROUP BY message_type
ORDER BY count DESC;

-- 检查赔率变化
SELECT COUNT(*) FROM odds_changes WHERE created_at > NOW() - INTERVAL '5 minutes';

-- 检查投注停止
SELECT COUNT(*) FROM bet_stops WHERE created_at > NOW() - INTERVAL '5 minutes';

-- 检查投注结算
SELECT COUNT(*) FROM bet_settlements WHERE created_at > NOW() - INTERVAL '5 minutes';

-- 查看具体的赔率变化
SELECT event_id, market_count, market_status, created_at
FROM odds_changes
WHERE created_at > NOW() - INTERVAL '5 minutes'
ORDER BY created_at DESC
LIMIT 10;
```

### 3. 使用API端点

```bash
SERVICE_URL="https://your-service.railway.app"

# 查看最新消息
curl "$SERVICE_URL/api/messages?limit=20"

# 查看统计
curl "$SERVICE_URL/api/stats"

# 查看跟踪的赛事
curl "$SERVICE_URL/api/events"
```

### 4. 使用WebSocket UI

打开浏览器访问: `https://your-service.railway.app/`

应该能看到实时消息流,包括:
- `[odds_change]` 消息
- `[bet_stop]` 消息
- `[bet_settlement]` 消息

---

## 重放参数说明

### speed (重放速度)

- **默认**: 10 (10倍速)
- **范围**: 1-100
- **说明**: 控制消息发送速度
- **示例**:
  - `speed=1`: 实时速度(90分钟比赛需要90分钟)
  - `speed=10`: 10倍速(90分钟比赛需要9分钟)
  - `speed=100`: 100倍速(90分钟比赛需要54秒)

### max_delay (最大延迟)

- **默认**: 10000 (10秒)
- **单位**: 毫秒
- **说明**: 两条消息间的最大延迟时间
- **用途**: 避免赛前赔率更新间隔过长

### node_id (节点ID)

- **默认**: 无
- **范围**: 1-999
- **说明**: 用于多开发者环境,隔离不同会话
- **路由键**: 消息会包含node_id在路由键中

### product_id (产品ID)

- **可选值**:
  - `1`: Live Odds (实时赔率)
  - `3`: Prematch (赛前赔率)
- **说明**: 只接收特定产品的消息

### use_replay_timestamp

- **可选值**: `true` / `false`
- **默认**: `false`
- **说明**:
  - `true`: 使用当前时间作为时间戳
  - `false`: 使用原始时间戳

---

## 常见问题

### Q1: 为什么没有收到消息?

**检查清单**:
1. ✅ 服务是否连接到 `global.replaymq.betradar.com`?
2. ✅ 重放是否已启动? (调用 `/replay/play`)
3. ✅ 重放状态是否为 "PLAYING"? (调用 `/replay/status`)
4. ✅ node_id 是否匹配?
5. ✅ AMQP订阅的routing key是否正确?

### Q2: 收到的消息类型为空?

这个问题已经修复!如果仍然出现:
1. 确认使用最新代码(包含XML解析修复)
2. 检查 `services/amqp_consumer.go` 中的 `parseMessage` 函数

### Q3: odds_changes表为空?

可能原因:
1. 重放的赛事没有赔率变化(尝试足球比赛)
2. `handleOddsChange` 函数有问题
3. eventID或productID解析失败

**调试方法**:
```sql
-- 检查是否收到odds_change消息
SELECT COUNT(*) FROM uof_messages 
WHERE message_type = 'odds_change' 
AND created_at > NOW() - INTERVAL '5 minutes';

-- 查看原始XML
SELECT xml_content FROM uof_messages 
WHERE message_type = 'odds_change' 
LIMIT 1;
```

### Q4: 重放一直在 "SETTING_UP" 状态?

这是正常的!重放服务器需要时间准备数据:
- 通常需要5-30秒
- 使用 `WaitUntilReady()` 函数自动等待
- 或手动轮询 `/replay/status`

### Q5: 如何停止重放?

```bash
# 方法1: API调用
curl -u "$AUTH" -X POST "https://api.betradar.com/v1/replay/stop"

# 方法2: 重置(停止并清空列表)
curl -u "$AUTH" -X POST "https://api.betradar.com/v1/replay/reset"

# 方法3: 使用脚本
./replay_event.sh test:match:12345 10 30 1  # 30秒后自动停止
```

---

## 最佳实践

### 1. 开发时使用高速重放

```bash
# 100倍速,快速验证功能
./replay_event.sh test:match:21797788 100 30 1
```

### 2. 调试时使用慢速重放

```bash
# 1倍速,便于观察每条消息
./replay_event.sh test:match:21797788 1 300 1
```

### 3. 使用不同的node_id隔离测试

```bash
# 开发者A使用node_id=1
./replay_event.sh test:match:12345 10 60 1

# 开发者B使用node_id=2
./replay_event.sh test:match:67890 10 60 2
```

### 4. 测试特定场景

```bash
# 测试赔率变化
./replay_event.sh test:match:21797788 20 60 1

# 测试加时赛
./replay_event.sh test:match:21797805 20 60 1

# 测试点球大战
./replay_event.sh test:match:21797815 20 60 1
```

### 5. 监控数据库变化

```bash
# 在一个终端运行重放
./replay_event.sh test:match:21797788 10 60 1

# 在另一个终端监控数据库
watch -n 5 'psql $DATABASE_URL -c "SELECT message_type, COUNT(*) FROM uof_messages WHERE created_at > NOW() - INTERVAL \"1 minute\" GROUP BY message_type;"'
```

---

## 集成到CI/CD

### GitHub Actions示例

```yaml
name: Test with Replay Server

on: [push, pull_request]

jobs:
  replay-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Go
        uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      
      - name: Run Replay Test
        env:
          UOF_USERNAME: ${{ secrets.UOF_USERNAME }}
          UOF_PASSWORD: ${{ secrets.UOF_PASSWORD }}
          DATABASE_URL: ${{ secrets.TEST_DATABASE_URL }}
        run: |
          go run tools/test_replay.go \
            -event=test:match:21797788 \
            -speed=100 \
            -duration=30 \
            -node=1
      
      - name: Verify Results
        run: |
          # 检查数据库中是否有新消息
          psql $DATABASE_URL -c "SELECT COUNT(*) FROM odds_changes;"
```

---

## 相关资源

- **官方文档**: https://docs.sportradar.com/uof/replay-server
- **API文档**: https://docs.sportradar.com/uof/replay-server/uof-replay-server-api
- **示例赛事**: https://docs.sportradar.com/uof/replay-server/uof-example-replays
- **API交互文档**: https://iodocs.betradar.com/replay

---

## 下一步

1. ✅ 设置环境变量
2. ✅ 选择测试赛事
3. ✅ 运行重放脚本
4. ✅ 验证数据存储
5. ✅ 检查WebSocket推送
6. ✅ 查看API响应

**准备好了吗?** 运行您的第一个重放测试:

```bash
export UOF_USERNAME="your_username"
export UOF_PASSWORD="your_password"
cd /home/ubuntu/uof-go-service/tools
./replay_event.sh test:match:21797788 20 60 1
```

祝测试顺利! 🎉

