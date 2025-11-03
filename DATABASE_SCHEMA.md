# Betradar UOF Service 数据库架构

**版本**: 2.0.0  
**作者**: Manus AI  
**最后更新**: 2025-10-31

---

## 概述

本文档详细描述了 Betradar UOF Service 使用的 PostgreSQL 数据库架构。该架构旨在存储来自 Sportradar API 的原始数据、处理后的赔率信息、比赛事件以及相关的元数据。

核心设计原则:
- **数据分离**: 将原始消息、处理后的数据和静态描述信息分离到不同的表中。
- **性能优化**: 通过索引和合理的数据类型优化查询性能。
- **一致性**: 统一使用 `sr_market_id` 表示 Sportradar 的盘口类型 ID, `market_id` 作为外键指向 `markets` 表的主键。

---

## 核心表结构

### 1. `markets`

存储每个比赛事件中的具体盘口信息。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键**, 自增 ID |
| `event_id` | `varchar` | 赛事 ID (e.g., `sr:match:12345`) |
| `sr_market_id` | `integer` | Sportradar 盘口类型 ID (e.g., `1` 代表 1X2) |
| `specifiers` | `varchar` | 盘口说明符 (e.g., `hcp=-1.5`) |
| `market_name` | `varchar` | 盘口名称 (e.g., "Match Result") |
| `status` | `varchar` | 盘口状态 (`active`, `suspended`, `settled`, `cancelled`) |
| `home_team_name` | `varchar` | 主队名称 (用于名称替换) |
| `away_team_name` | `varchar` | 客队名称 (用于名称替换) |
| `created_at` | `timestamp` | 创建时间 |
| `updated_at` | `timestamp` | 更新时间 |

**索引**:
- `markets_pkey` (id)
- `markets_event_id_idx` (event_id)
- `markets_sr_market_id_idx` (sr_market_id)
- `markets_status_idx` (status)

---

### 2. `odds`

存储每个盘口下的具体赔率信息。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键**, 自增 ID |
| `market_id` | `integer` | **外键**, 指向 `markets.id` |
| `outcome_id` | `varchar` | 结果 ID (e.g., `1`, `X`, `2`) |
| `odds` | `float8` | 赔率值 |
| `active` | `boolean` | 是否可用 |
| `outcome_name` | `varchar` | 结果名称 (e.g., "Home Win") |
| `created_at` | `timestamp` | 创建时间 |
| `updated_at` | `timestamp` | 更新时间 |

**索引**:
- `odds_pkey` (id)
- `odds_market_id_idx` (market_id)
- `odds_outcome_id_idx` (outcome_id)

---

### 3. `uof_messages`

存储从 RabbitMQ 接收到的所有原始 UOF 消息。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键**, 自增 ID |
| `message_type` | `varchar` | 消息类型 (e.g., `odds_change`, `bet_stop`) |
| `event_id` | `varchar` | 赛事 ID (如果适用) |
| `product_id` | `integer` | 产品 ID |
| `timestamp` | `bigint` | 消息时间戳 (毫秒) |
| `request_id` | `bigint` | 请求 ID (如果适用) |
| `raw_message` | `bytea` | 原始 XML 消息体 |
| `received_at` | `timestamp` | 接收时间 |

**索引**:
- `uof_messages_pkey` (id)
- `uof_messages_event_id_idx` (event_id)
- `uof_messages_message_type_idx` (message_type)
- `uof_messages_received_at_idx` (received_at)

---

## 描述与映射表

### 4. `market_descriptions`

存储所有盘口类型的静态描述信息。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键** |
| `sr_market_id` | `integer` | Sportradar 盘口类型 ID |
| `name` | `varchar` | 盘口名称模板 |
| `groups` | `varchar` | 盘口分组 |
| `created_at` | `timestamp` | 创建时间 |
| `updated_at` | `timestamp` | 更新时间 |

**索引**:
- `market_descriptions_sr_market_id_key` (sr_market_id, 唯一)

---

### 5. `outcome_descriptions`

存储所有结果类型的静态描述信息。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键** |
| `sr_market_id` | `integer` | Sportradar 盘口类型 ID |
| `outcome_id` | `varchar` | 结果 ID |
| `name` | `varchar` | 结果名称模板 |
| `created_at` | `timestamp` | 创建时间 |
| `updated_at` | `timestamp` | 更新时间 |

**索引**:
- `outcome_descriptions_sr_market_id_outcome_id_key` (sr_market_id, outcome_id, 唯一)

---

### 6. `mapping_outcomes`

存储 Sportradar 盘口与第三方产品盘口的映射关系。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键** |
| `sr_market_id` | `integer` | Sportradar 盘口类型 ID |
| `outcome_id` | `varchar` | Sportradar 结果 ID |
| `product_outcome_name` | `varchar` | 第三方产品结果名称 |
| `created_at` | `timestamp` | 创建时间 |
| `updated_at` | `timestamp` | 更新时间 |

**索引**:
- `mapping_outcomes_sr_market_id_outcome_id_key` (sr_market_id, outcome_id, 唯一)

---

## 结算与取消表

### 7. `bet_settlements`

存储投注结算信息。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键** |
| `event_id` | `varchar` | 赛事 ID |
| `producer_id` | `integer` | Producer ID |
| `sr_market_id` | `integer` | Sportradar 盘口类型 ID |
| `specifiers` | `varchar` | 盘口说明符 |
| `outcome_id` | `varchar` | 结果 ID |
| `void_factor` | `float8` | 作废因子 (0 或 1) |
| `dead_heat_factor` | `float8` | 并列因子 |
| `result` | `varchar` | 结算结果 (`win`, `loss`) |
| `created_at` | `timestamp` | 创建时间 |
| `xml_content` | `text` | 原始 XML 内容 |

---

### 8. `bet_cancels`

存储投注取消信息。

| 列名 | 数据类型 | 描述 |
| --- | --- | --- |
| `id` | `serial` | **主键** |
| `event_id` | `varchar` | 赛事 ID |
| `producer_id` | `integer` | Producer ID |
| `sr_market_id` | `integer` | Sportradar 盘口类型 ID |
| `specifiers` | `varchar` | 盘口说明符 |
| `start_time` | `timestamp` | 取消开始时间 |
| `end_time` | `timestamp` | 取消结束时间 |
| `superceded_by` | `varchar` | 被取代的盘口 |
| `created_at` | `timestamp` | 创建时间 |
| `xml_content` | `text` | 原始 XML 内容 |

---

### 9. `rollback_bet_settlements` & `rollback_bet_cancels`

这两张表的结构分别与 `bet_settlements` 和 `bet_cancels` 类似, 用于记录回滚操作。

---

## 辅助表

- **`tracked_events`**: 存储所有追踪的比赛事件的详细信息, 包括赛程、比分、状态等。
- **`sports`, `categories`, `tournaments`**: 存储体育项目、类别和锦标赛的静态数据。
- **`void_reasons`, `betstop_reasons`**: 存储作废和停止投注原因的静态数据。
- **`players`**: 存储球员信息。
- **`producer_status`**: 存储 Producer 的状态信息。

