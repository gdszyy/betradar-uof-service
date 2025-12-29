# 🎬 Replay Server 功能总结

## 概述

已成功添加完整的Replay Server支持,用于测试和验证Betradar UOF服务的数据管道。

---

## 📦 新增文件

### 核心代码

1. **`services/replay_client.go`** (430行)
   - Replay API客户端
   - 支持所有Replay API端点
   - 便捷方法: `QuickReplay()`
   - 自动等待准备就绪: `WaitUntilReady()`

### 测试工具

2. **`tools/test_replay.go`** (200行)
   - Go语言重放测试程序
   - 实时监控数据库变化
   - 显示消息类型分布
   - 统计专门表数据

3. **`tools/replay_event.sh`** (150行)
   - Shell脚本重放工具
   - 完整的API调用流程
   - 自动等待和状态检查
   - 彩色输出和进度显示

4. **`tools/quick_replay_test.sh`** (100行)
   - 快速测试脚本
   - 交互式界面
   - 推荐测试赛事列表
   - 自动数据库统计

### 文档

5. **`docs/REPLAY-SERVER.md`**
   - Replay Server基础知识
   - API端点参考
   - 使用流程说明

6. **`docs/REPLAY-TESTING-GUIDE.md`** (500行)
   - 完整的使用指南
   - 推荐测试赛事列表
   - 配置说明
   - 故障排查
   - 最佳实践
   - CI/CD集成示例

---

## 🎯 功能特性

### 1. API客户端功能

```go
client := services.NewReplayClient(username, password)

// 列出重放列表
events, _ := client.ListEvents()

// 添加赛事
client.AddEvent("test:match:12345", 0)

// 开始重放
client.Play(PlayOptions{
    Speed:              20,
    MaxDelay:           10000,
    NodeID:             1,
    UseReplayTimestamp: true,
})

// 查看状态
status, _ := client.GetStatus()

// 停止重放
client.Stop()

// 一键重放(最简单)
client.QuickReplay("test:match:12345", 20, 1)
```

### 2. 命令行工具

```bash
# 方法1: Shell脚本
./replay_event.sh test:match:21797788 20 60 1

# 方法2: Go程序
go run tools/test_replay.go \
  -event=test:match:21797788 \
  -speed=20 \
  -duration=60 \
  -node=1

# 方法3: 快速测试
./quick_replay_test.sh
```

### 3. 支持的参数

| 参数 | 说明 | 默认值 | 范围 |
|------|------|--------|------|
| `speed` | 重放速度倍数 | 10 | 1-100 |
| `max_delay` | 最大延迟(毫秒) | 10000 | 1-60000 |
| `node_id` | 节点ID(多会话隔离) | - | 1-999 |
| `product_id` | 产品ID(1=live, 3=pre) | - | 1,3 |
| `use_replay_timestamp` | 使用当前时间戳 | false | true/false |
| `start_time` | 从比赛开始后X分钟开始 | 0 | 0-200 |

---

## 📋 推荐测试赛事

### 足球 (Soccer)

| 赛事ID | 描述 | 测试场景 |
|--------|------|----------|
| `test:match:21797788` | VAR场景 | ⭐ 推荐 - 丰富的赔率变化 |
| `test:match:21797805` | 加时赛 | 测试加时赛消息 |
| `test:match:21797815` | 点球大战 | 测试点球消息 |
| `sr:match:22340005` | 平局 | 测试平局结算 |

### 网球 (Tennis)

| 赛事ID | 描述 | 测试场景 |
|--------|------|----------|
| `test:match:21797802` | 5盘制抢十 | 测试网球特殊规则 |
| `test:match:21796642` | 温网决赛 | 测试长盘制 |

### 其他运动

- **棒球**: `test:match:23517711`
- **篮球**: NBA总决赛
- **冰球**: NHL决赛
- **电竞**: CS:GO, Dota2, LoL

**完整列表**: https://docs.sportradar.com/uof/replay-server/uof-example-replays

---

## 🚀 快速开始

### 步骤1: 设置环境变量

```bash
export UOF_USERNAME="your_betradar_username"
export UOF_PASSWORD="your_betradar_password"
export DATABASE_URL="postgresql://..."  # 可选
```

### 步骤2: 运行测试

```bash
cd /home/ubuntu/uof-go-service/tools
./quick_replay_test.sh
```

### 步骤3: 验证结果

```sql
-- 检查消息类型
SELECT message_type, COUNT(*) 
FROM uof_messages 
WHERE created_at > NOW() - INTERVAL '5 minutes'
GROUP BY message_type;

-- 检查赔率变化
SELECT * FROM odds_changes LIMIT 10;

-- 检查投注停止
SELECT * FROM bet_stops LIMIT 10;

-- 检查投注结算
SELECT * FROM bet_settlements LIMIT 10;
```

---

## 🔧 配置服务连接Replay服务器

### 选项A: 环境变量切换(推荐)

在 `config/config.go` 中添加:

```go
func LoadConfig() *Config {
    amqpHost := "stgmq.betradar.com"
    
    if os.Getenv("REPLAY_MODE") == "true" {
        amqpHost = "global.replaymq.betradar.com"
        log.Println("🎬 REPLAY MODE: Using Replay Server")
    }
    
    return &Config{
        AMQPHost: amqpHost,
        // ...
    }
}
```

使用:

```bash
# Railway上设置
REPLAY_MODE=true

# 本地运行
export REPLAY_MODE=true
go run main.go
```

### 选项B: 专用Replay实例

在Railway上创建第二个服务:

1. 复制现有服务
2. 命名: "uof-service-replay"
3. 设置环境变量:
   ```
   AMQP_HOST=global.replaymq.betradar.com
   REPLAY_MODE=true
   ```

---

## 📊 预期结果

### 成功指标

运行重放测试后,应该看到:

#### 1. 服务日志
```
✅ Connected to AMQP server: global.replaymq.betradar.com
✅ Odds change for event test:match:21797788: 15 markets, status=1
✅ Bet stop for event test:match:21797788: market_status=1
✅ Bet settlement for event test:match:21797788: market_count=8
```

#### 2. 数据库数据
```
消息类型分布:
  odds_change: 150+
  bet_stop: 20+
  bet_settlement: 15+
  alive: 10+
  fixture_change: 5+

专门表:
  odds_changes: 150+ rows
  bet_stops: 20+ rows
  bet_settlements: 15+ rows
  tracked_events: 1+ rows
```

#### 3. API响应
```json
{
  "total_messages": 200+,
  "odds_changes": 150+,
  "bet_stops": 20+,
  "bet_settlements": 15+
}
```

#### 4. WebSocket UI
- 实时显示 `[odds_change]` 消息
- 显示完整的XML内容
- 统计数据实时更新

---

## 🐛 故障排查

### 问题1: "Access forbidden" 错误

**原因**: UOF凭证未设置或不正确

**解决**:
```bash
export UOF_USERNAME="correct_username"
export UOF_PASSWORD="correct_password"
```

### 问题2: 没有收到消息

**检查**:
1. 服务是否连接到 `global.replaymq.betradar.com`?
2. 重放是否已启动? (调用 `/replay/play`)
3. node_id 是否匹配?

**调试**:
```bash
# 检查重放状态
curl -u "$UOF_USERNAME:$UOF_PASSWORD" \
  "https://api.betradar.com/v1/replay/status"
```

### 问题3: odds_changes表为空

**原因**: 可能选择的赛事没有赔率变化

**解决**: 使用推荐的足球比赛
```bash
./replay_event.sh test:match:21797788 20 60 1
```

### 问题4: 重放一直在 "SETTING_UP"

**原因**: 正常现象,需要等待

**解决**: 使用 `WaitUntilReady()` 或等待30秒

---

## 📚 文档链接

- **Replay Server文档**: `docs/REPLAY-SERVER.md`
- **测试指南**: `docs/REPLAY-TESTING-GUIDE.md`
- **官方文档**: https://docs.sportradar.com/uof/replay-server
- **API参考**: https://docs.sportradar.com/uof/replay-server/uof-replay-server-api
- **示例赛事**: https://docs.sportradar.com/uof/replay-server/uof-example-replays

---

## 🎯 使用场景

### 1. 开发测试
```bash
# 快速验证功能(100倍速)
./replay_event.sh test:match:21797788 100 30 1
```

### 2. 调试问题
```bash
# 慢速重放,便于观察(1倍速)
./replay_event.sh test:match:21797788 1 300 1
```

### 3. 性能测试
```bash
# 多个赛事同时重放
./replay_event.sh test:match:21797788 50 60 1 &
./replay_event.sh test:match:21797805 50 60 2 &
./replay_event.sh test:match:21797815 50 60 3 &
```

### 4. CI/CD集成
```yaml
- name: Test Pipeline
  run: |
    go run tools/test_replay.go \
      -event=test:match:21797788 \
      -speed=100 \
      -duration=30
```

### 5. 演示和培训
```bash
# 实时速度,展示完整比赛流程
./replay_event.sh test:match:21797788 1 5400 1
```

---

## ✅ 验证清单

使用以下清单验证Replay功能:

- [ ] 设置了 `UOF_USERNAME` 和 `UOF_PASSWORD`
- [ ] 运行 `./quick_replay_test.sh` 成功
- [ ] 服务日志显示 "Odds change" 消息
- [ ] `odds_changes` 表有数据
- [ ] `bet_stops` 表有数据
- [ ] `bet_settlements` 表有数据
- [ ] WebSocket UI显示实时消息
- [ ] API `/api/stats` 显示正确统计
- [ ] 消息类型正确解析(不为空)

---

## 🎉 总结

### 已实现

✅ 完整的Replay API客户端  
✅ 多种测试工具(Go, Shell)  
✅ 详细的文档和指南  
✅ 推荐测试赛事列表  
✅ 故障排查指南  
✅ CI/CD集成示例  
✅ 最佳实践建议  

### 优势

- 🚀 **快速验证**: 无需等待真实比赛
- 🐛 **便于调试**: 可重复测试特定场景
- 📊 **完整覆盖**: 测试所有消息类型
- 🔧 **易于使用**: 一键运行测试
- 📚 **文档完善**: 详细的使用指南

### 下一步

1. 设置环境变量
2. 运行快速测试: `./quick_replay_test.sh`
3. 验证数据存储
4. 集成到开发流程

---

**准备好开始测试了吗?**

```bash
export UOF_USERNAME="your_username"
export UOF_PASSWORD="your_password"
cd /home/ubuntu/uof-go-service/tools
./quick_replay_test.sh
```

祝测试顺利! 🎬🎉

