# Betradar UOF Service API 文档

**版本**: 3.0.0  
**作者**: Manus AI  
**最后更新**: 2025-10-31

---

## 概述

本文档提供了 Betradar UOF Service 的 API 接口说明。所有 API 均以 `/api` 为前缀。

---

## 1. 核心服务

### 1.1 健康检查

- **GET** `/api/health`
  - **描述**: 检查服务是否正常运行。
  - **响应**: `{"status": "ok"}`

### 1.2 消息查询

- **GET** `/api/messages`
  - **描述**: 获取最近的 UOF 消息。
  - **参数**:
    - `limit` (int, 可选): 消息数量, 默认 100
    - `offset` (int, 可选): 偏移量, 默认 0
    - `type` (string, 可选): 消息类型 (e.g., `odds_change`, `bet_stop`)
  - **响应**: 消息列表

### 1.3 比赛事件

- **GET** `/api/events`
  - **描述**: 获取增强版的比赛事件列表, 包含盘口信息。
  - **参数**:
    - `status` (string, 可选): 比赛状态 (`live`, `not_started`, `ended`)
    - `limit` (int, 可选): 数量, 默认 50
    - `offset` (int, 可选): 偏移量, 默认 0
  - **响应**: 比赛事件列表

- **GET** `/api/events/simple`
  - **描述**: 获取简版的比赛事件列表。
  - **响应**: 比赛事件列表

### 1.4 服务统计

- **GET** `/api/stats`
  - **描述**: 获取服务运行统计信息。
  - **响应**: 统计数据

---

## 2. 恢复机制

### 2.1 触发全量恢复

- **POST** `/api/recovery/trigger`
  - **描述**: 手动触发全量恢复流程。
  - **响应**: `{"status": "ok", "message": "Recovery triggered"}`

### 2.2 触发单个事件恢复

- **POST** `/api/recovery/event/{event_id}`
  - **描述**: 触发指定比赛的恢复流程。
  - **响应**: `{"status": "ok", "message": "Event recovery triggered"}`

### 2.3 获取恢复状态

- **GET** `/api/recovery/status`
  - **描述**: 获取恢复服务的状态。
  - **响应**: 恢复状态信息

---

## 3. Replay 功能

### 3.1 开始 Replay

- **POST** `/api/replay/start`
  - **描述**: 开始 Replay 流程。
  - **响应**: `{"status": "ok", "message": "Replay started"}`

### 3.2 停止 Replay

- **POST** `/api/replay/stop`
  - **描述**: 停止 Replay 流程。
  - **响应**: `{"status": "ok", "message": "Replay stopped"}`

### 3.3 获取 Replay 状态

- **GET** `/api/replay/status`
  - **描述**: 获取 Replay 服务的状态。
  - **响应**: Replay 状态信息

### 3.4 获取 Replay 列表

- **GET** `/api/replay/list`
  - **描述**: 获取 Replay 事件列表。
  - **响应**: 事件列表

---

## 4. 订阅与预订

### 4.1 自动订阅

- **POST** `/api/booking/auto`
  - **描述**: 自动订阅所有可预订的比赛。
  - **响应**: `{"status": "ok", "booked_count": 10}`

- **POST** `/api/booking/trigger`
  - **描述**: 手动触发自动订阅流程。
  - **响应**: `{"status": "ok", "message": "Auto booking triggered"}`

### 4.2 手动订阅

- **POST** `/api/booking/match/{match_id}`
  - **描述**: 订阅指定的比赛。
  - **响应**: `{"status": "ok", "message": "Match booked"}`

### 4.3 订阅查询

- **GET** `/api/booking/booked`
  - **描述**: 获取所有已订阅的比赛。
  - **响应**: 比赛列表

- **GET** `/api/booking/bookable`
  - **描述**: 获取所有可预订的比赛。
  - **响应**: 比赛列表

### 4.4 订阅同步

- **POST** `/api/booking/sync`
  - **描述**: 与 Betradar API 同步订阅列表。
  - **响应**: `{"status": "ok", "message": "Subscriptions synced"}`

### 4.5 自动订阅控制器

- **GET** `/api/auto-booking/status`
  - **描述**: 获取自动订阅服务的状态。
  - **响应**: `{"enabled": true, "interval_minutes": 60}`

- **POST** `/api/auto-booking/enable`
  - **描述**: 启用自动订阅。
  - **响应**: `{"status": "ok"}`

- **POST** `/api/auto-booking/disable`
  - **描述**: 禁用自动订阅。
  - **响应**: `{"status": "ok"}`

- **POST** `/api/auto-booking/interval`
  - **描述**: 设置自动订阅的间隔时间。
  - **请求体**: `{"interval_minutes": 30}`
  - **响应**: `{"status": "ok"}`

---

## 5. 消息历史

- **GET** `/api/messages/recent`
  - **描述**: 获取最近的 UOF 消息。
  - **响应**: 消息列表

- **GET** `/api/events/{event_id}/messages`
  - **描述**: 获取指定比赛的所有消息。
  - **响应**: 消息列表

---

## 6. 盘口与赔率

### 6.1 获取比赛盘口

- **GET** `/api/events/{event_id}/markets`
  - **描述**: 获取指定比赛的所有盘口。
  - **响应**: 盘口列表

### 6.2 获取所有已订阅盘口的赔率

- **GET** `/api/odds/all`
  - **描述**: 获取所有已订阅比赛的所有盘口的赔率。
  - **响应**: 赔率列表

### 6.3 获取比赛盘口赔率

- **GET** `/api/odds/{event_id}/markets`
  - **描述**: 获取指定比赛的所有盘口的赔率。
  - **响应**: 盘口赔率列表

### 6.4 获取单个盘口赔率

- **GET** `/api/odds/{event_id}/{market_id}`
  - **描述**: 获取指定比赛的单个盘口的赔率。
  - **响应**: 赔率列表

### 6.5 获取赔率历史

- **GET** `/api/odds/{event_id}/{market_id}/{outcome_id}/history`
  - **描述**: 获取指定结果的赔率变化历史。
  - **响应**: 赔率历史列表

---

## 7. 比赛查询 (前端 API)

- **GET** `/api/matches/live`
  - **描述**: 获取正在进行的比赛。

- **GET** `/api/matches/upcoming`
  - **描述**: 获取即将开始的比赛。

- **GET** `/api/matches/status`
  - **描述**: 根据状态查询比赛。
  - **参数**: `status` (string)

- **GET** `/api/matches/search`
  - **描述**: 搜索比赛。
  - **参数**: `q` (string)

- **GET** `/api/matches/{event_id}`
  - **描述**: 获取比赛详情。

---

## 8. 市场描述

- **GET** `/api/market-descriptions/status`
  - **描述**: 获取市场描述服务的状态。

- **POST** `/api/market-descriptions/refresh`
  - **描述**: 强制刷新市场描述。

- **POST** `/api/market-descriptions/bulk-update`
  - **描述**: 批量更新存量数据的市场和结果名称。

---

## 9. 监控与管理

- **GET** `/api/producer/status`
  - **描述**: 获取 Producer 状态。

- **GET** `/api/producer/bet-acceptance`
  - **描述**: 获取投注接受状态。

- **GET** `/api/cleanup/stats`
  - **描述**: 获取数据表统计信息。

- **POST** `/api/cleanup/manual`
  - **描述**: 手动触发数据清理。

- **POST** `/api/database/reset`
  - **描述**: 重置数据库 (危险操作)。

- **GET** `/api/match/records`
  - **描述**: 获取比赛记录。

- **GET** `/api/record/detail`
  - **描述**: 获取记录详情。

---

## 10. WebSocket

- **GET** `/ws`
  - **描述**: 建立 WebSocket 连接, 实时接收赔率变化等消息。

---

## 附录: 响应字段说明

### sr_market_id vs market_id

- `sr_market_id`: Sportradar API 返回的盘口类型 ID (e.g., `1`, `186`)
- `market_id`: 数据库中 `markets` 表的主键 ID (自增整数)

在 API 响应中, 盘口相关的 ID 统一使用 `sr_market_id`。

