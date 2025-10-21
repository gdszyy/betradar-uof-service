# AMQP Exchange 配置说明

## 问题描述

在使用 Betradar UOF 的 AMQP 服务时,必须使用正确的 exchange 配置,否则会遇到以下错误:

```
access to amq.default is denied
```

## 正确配置

### Exchange 名称

所有 AMQP 操作(包括消息消费和发布)都必须使用 **`unifiedfeed`** exchange,而不是默认的空 exchange (`""` 或 `amq.default`)。

### VHost

VHost 格式为: `/unifiedfeed/<bookmaker_id>`

例如: `/unifiedfeed/12345`

### 认证

- **Username**: 您的 access token
- **Password**: 留空

### 队列配置

- **队列名称**: 留空,让服务器自动生成
- **Durable**: false
- **Exclusive**: true
- **Auto-delete**: true (delete when unused)
- **Passive**: 不需要预先声明,服务器会自动创建

## 代码示例

### 消费消息 (订阅)

```go
// 1. 声明队列 (服务器自动生成名称)
queue, err := channel.QueueDeclare(
    "",    // name (empty for auto-generated)
    false, // durable
    true,  // delete when unused
    true,  // exclusive
    false, // no-wait
    nil,   // arguments
)

// 2. 绑定到 unifiedfeed exchange
err = channel.QueueBind(
    queue.Name,
    "#",           // routing key (订阅所有消息)
    "unifiedfeed", // exchange - 必须使用 unifiedfeed
    false,
    nil,
)

// 3. 开始消费
msgs, err := channel.Consume(
    queue.Name,
    "",    // consumer
    true,  // auto-ack
    false, // exclusive
    false, // no-local
    false, // no-wait
    nil,   // args
)
```

### 发布消息 (查询 Match List)

```go
// 创建临时响应队列
responseQueue, err := channel.QueueDeclare(
    "",    // 让服务器生成队列名
    false, // durable
    true,  // delete when unused
    true,  // exclusive
    false, // no-wait
    nil,   // arguments
)

// 发布到 unifiedfeed exchange
err = channel.Publish(
    "unifiedfeed",  // exchange - 必须使用 unifiedfeed
    "matchlist",    // routing key
    false,          // mandatory
    false,          // immediate
    amqp.Publishing{
        ContentType:   "text/xml",
        CorrelationId: fmt.Sprintf("%d", time.Now().Unix()),
        ReplyTo:       responseQueue.Name,
        Body:          xmlData,
    },
)
```

## 常见错误

### ❌ 错误示例 1: 使用空 exchange

```go
err = channel.Publish(
    "",          // ❌ 错误: 不能使用空 exchange
    "matchlist",
    false,
    false,
    amqp.Publishing{...},
)
```

**错误信息**: `access to amq.default is denied`

### ❌ 错误示例 2: 使用 amq.default

```go
err = channel.Publish(
    "amq.default",  // ❌ 错误: 不能使用 amq.default
    "matchlist",
    false,
    false,
    amqp.Publishing{...},
)
```

**错误信息**: `access to amq.default is denied`

### ✅ 正确示例

```go
err = channel.Publish(
    "unifiedfeed",  // ✅ 正确: 使用 unifiedfeed exchange
    "matchlist",
    false,
    false,
    amqp.Publishing{...},
)
```

## 周末访问限制

如果您使用的是 **Integration Token** (而不是 Production Token),请注意:

- **周末**: RabbitMQ 访问可能受限
- **工作日**: 完全访问

详见 Betradar 官方文档: [Messages and Weekend access](https://docs.sportradar.com/uof/)

## 相关文档

- [Betradar UOF AMQP 文档](https://docs.sportradar.com/uof/)
- [Match Monitoring 文档](./MATCH-MONITORING.md)
- [Feishu Integration 文档](./FEISHU-INTEGRATION.md)

## 故障排除

### 问题: 无法发送 match list 请求

**症状**: 
```
failed to publish request: access to amq.default is denied
```

**解决方案**: 
确认代码中使用的是 `"unifiedfeed"` exchange,而不是 `""` 或 `"amq.default"`。

### 问题: 队列声明失败

**症状**: 
```
failed to declare queue: access denied
```

**解决方案**: 
- 确认队列名称为空字符串 `""`
- 确认 `exclusive` 设置为 `true`
- 不要尝试创建持久化队列 (`durable: false`)

### 问题: VHost 错误

**症状**: 
```
vhost not found
```

**解决方案**: 
- 确认 VHost 格式为 `/unifiedfeed/<bookmaker_id>`
- 通过 `/users/whoami.xml` API 获取正确的 VHost

## 测试

我们提供了测试脚本来验证 exchange 配置:

```bash
cd /home/ubuntu/uof-go-service
go run tools/test_match_list.go
```

该脚本会:
1. 连接到 AMQP 服务器
2. 使用 `unifiedfeed` exchange 发送 match list 请求
3. 接收并解析响应
4. 显示已订阅的比赛列表

## 总结

记住这个关键规则:

> **所有 Betradar UOF AMQP 操作都必须使用 `unifiedfeed` exchange**

这包括:
- 消息订阅 (QueueBind)
- Match list 查询 (Publish)
- 其他任何 AMQP 发布操作

不要使用空 exchange 或 `amq.default`。

