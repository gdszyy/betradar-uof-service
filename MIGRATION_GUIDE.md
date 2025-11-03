# Betradar UOF Service 数据库迁移指南

**版本**: 2.0.0  
**作者**: Manus AI  
**最后更新**: 2025-10-31

---

## 概述

本文档提供了完整的数据库迁移指南, 用于将 Betradar UOF Service 的数据库从旧版本升级到最新版本。迁移过程分为两个主要阶段:

1. **重命名 `market_id` 字段**: 将所有相关的 `market_id` 字段重命名为 `sr_market_id`, 以解决语义歧义。
2. **创建和修改表结构**: 创建缺失的表 (`bet_cancels`, `rollback_bet_settlements`, `rollback_bet_cancels`) 并修改 `bet_settlements` 表的结构。

---

## 迁移前准备

1. **备份数据库**: 在执行任何迁移操作之前, 强烈建议备份您的数据库。
2. **停止应用程序**: 停止正在运行的 Betradar UOF Service 实例, 以避免在迁移过程中写入数据。

---

## 阶段 1: 重命名 market_id 字段

此阶段将 `market_id` 字段重命名为 `sr_market_id`, 以准确反映其作为 Sportradar Market ID 的含义。

### 1.1 执行脚本

使用以下命令执行迁移脚本:

```bash
PGPASSWORD='your_password' psql -h your_host -p 5432 -U your_user -d your_db -f migrate_market_id_to_sr_market_id.sql
```

### 1.2 脚本内容 (`migrate_market_id_to_sr_market_id.sql`)

```sql
-- 数据库迁移脚本: 将所有表的 market_id 字段重命名为 sr_market_id
-- 目的: 区分 Sportradar Market ID 和 Database Primary Key

-- 1. bet_settlements 表
ALTER TABLE IF EXISTS bet_settlements RENAME COLUMN market_id TO sr_market_id;

-- 2. bet_cancels 表
ALTER TABLE IF EXISTS bet_cancels RENAME COLUMN market_id TO sr_market_id;

-- 3. rollback_bet_settlements 表
ALTER TABLE IF EXISTS rollback_bet_settlements RENAME COLUMN market_id TO sr_market_id;

-- 4. rollback_bet_cancels 表
ALTER TABLE IF EXISTS rollback_bet_cancels RENAME COLUMN market_id TO sr_market_id;

-- 5. outcomes 表 (如果存在 market_id 字段)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'outcomes' AND column_name = 'market_id'
    ) THEN
        ALTER TABLE outcomes RENAME COLUMN market_id TO sr_market_id;
    END IF;
END $$;
```

---

## 阶段 2: 创建和修改表结构

此阶段创建在旧版本中缺失的表, 并为 `bet_settlements` 表添加详细字段。

### 2.1 执行脚本

使用以下命令执行迁移脚本:

```bash
PGPASSWORD='your_password' psql -h your_host -p 5432 -U your_user -d your_db -f create_missing_tables.sql
```

### 2.2 脚本内容 (`create_missing_tables.sql`)

```sql
-- 创建缺失的表和修改现有表结构

-- 1. 修改 bet_settlements 表,添加详细字段
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS sr_market_id VARCHAR(50);
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS specifiers TEXT;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS void_factor FLOAT;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS outcome_id VARCHAR(50);
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS result INTEGER;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS dead_heat_factor FLOAT;
ALTER TABLE bet_settlements ADD COLUMN IF NOT EXISTS producer_id INTEGER;

-- 添加唯一约束
CREATE UNIQUE INDEX IF NOT EXISTS idx_bet_settlements_unique 
ON bet_settlements (event_id, sr_market_id, specifiers, outcome_id, producer_id);

-- 2. 创建 bet_cancels 表
CREATE TABLE IF NOT EXISTS bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    sr_market_id VARCHAR(50) NOT NULL,
    specifiers TEXT,
    void_reason INTEGER,
    start_time BIGINT,
    end_time BIGINT,
    superceded_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, sr_market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_bet_cancels_event_id ON bet_cancels(event_id);

-- 3. 创建 rollback_bet_settlements 表
CREATE TABLE IF NOT EXISTS rollback_bet_settlements (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    sr_market_id VARCHAR(50) NOT NULL,
    specifiers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, sr_market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_rollback_bet_settlements_event_id ON rollback_bet_settlements(event_id);

-- 4. 创建 rollback_bet_cancels 表
CREATE TABLE IF NOT EXISTS rollback_bet_cancels (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    producer_id INTEGER NOT NULL,
    timestamp BIGINT NOT NULL,
    sr_market_id VARCHAR(50) NOT NULL,
    specifiers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (event_id, sr_market_id, specifiers, producer_id)
);

CREATE INDEX IF NOT EXISTS idx_rollback_bet_cancels_event_id ON rollback_bet_cancels(event_id);
```

---

## 迁移后验证

1. **检查表结构**: 连接到数据库并使用 `\d table_name` 命令检查以下表的结构是否正确:
   - `bet_settlements`
   - `bet_cancels`
   - `rollback_bet_settlements`
   - `rollback_bet_cancels`

2. **启动应用程序**: 启动 Betradar UOF Service。
3. **监控日志**: 检查应用程序日志, 确认没有与数据库相关的错误, 特别是 "failed to commit transaction" 或 "column does not exist" 等错误。

---

## 回滚方案

如果迁移失败, 您可以:
1. **恢复数据库备份**: 这是最安全的回滚方法。
2. **手动反向迁移**: 如果没有备份, 您可以手动执行 `ALTER TABLE ... RENAME COLUMN ...` 和 `DROP TABLE ...` 等命令来恢复到原始状态。**此方法风险较高, 不推荐。**

