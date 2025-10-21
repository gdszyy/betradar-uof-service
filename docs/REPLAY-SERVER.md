# Replay Server 使用指南

## 概述

Replay Server允许您重放已结束赛事的所有消息,用于测试和开发。

**服务器地址**: `global.replaymq.betradar.com`  
**API基础URL**: `https://api.betradar.com/v1`

## 限制

- 赛事必须至少48小时前结束
- 播放列表最多600个赛事
- 每个bookmaker最多同时运行5个重放请求

## API端点

### 1. 列出重放列表
```
GET /replay/
```

### 2. 添加赛事到重放列表
```
PUT /replay/events/{event_id}
```

参数:
- `event_id`: 赛事ID (例如: sr:match:12345)
- `start_time`: (可选) 从比赛开始后多少分钟开始重放

### 3. 删除赛事
```
DELETE /replay/events/{event_id}
```

### 4. 开始重放
```
POST /replay/play
```

参数:
- `speed`: 重放速度 (默认10 = 10倍速)
- `max_delay`: 消息间最大延迟(毫秒,默认10000)
- `node_id`: (可选) 节点ID,用于多开发者环境
- `product_id`: (可选) 只接收特定生产者的消息
- `use_replay_timestamp`: true=使用当前时间, false=使用原始时间戳

### 5. 停止重放
```
POST /replay/stop
```

### 6. 重置(停止并清空列表)
```
POST /replay/reset
```

### 7. 查看状态
```
GET /replay/status
```

### 8. 获取赛事信息
```
GET /sports/{language}/sport_events/{event_id}/summary.xml
GET /sports/{language}/sport_events/{event_id}/fixture.xml
GET /sports/{language}/sport_events/{event_id}/timeline.xml
```

## 使用流程

1. **添加赛事**: `PUT /replay/events/sr:match:12345`
2. **开始重放**: `POST /replay/play?speed=10&max_delay=10000`
3. **连接AMQP**: 连接到 `global.replaymq.betradar.com:5671`
4. **接收消息**: 开始接收重放的消息
5. **停止重放**: `POST /replay/stop`

## 注意事项

- 重放的消息是发送给特殊重放用户的,可能不是100%反映您的赔率值
- 重放开始时会发送alive消息(product 1和3,每10秒)
- 如果在设置期间调用start,会收到"SETTING_UP"状态
- 使用`node_id`可以让多个开发者互不干扰

