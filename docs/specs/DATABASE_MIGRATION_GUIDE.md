# 数据库迁移执行指南

## 概述

本文档说明如何在 Railway 平台上执行数据库迁移,为 `tracked_events` 表添加必需的字段。

## 迁移内容

迁移脚本位于: `database/migrations/001_add_srn_mapping.sql`

该脚本将为 `tracked_events` 表添加以下字段:

- `srn_id` - SportRadar SRN ID 映射
- `schedule_time` - 比赛计划开始时间
- `home_team_id` / `home_team_name` - 主队信息
- `away_team_id` / `away_team_name` - 客队信息
- `home_score` / `away_score` - 比分
- `match_status` - 比赛状态码
- `match_time` - 当前比赛时间

## 执行方法

### 方法 1: 使用 Railway CLI (推荐)

```bash
# 1. 安装 Railway CLI
npm install -g @railway/cli

# 2. 登录 Railway
railway login

# 3. 链接到项目
railway link

# 4. 执行迁移
railway run go run cmd/migrate/main.go
```

### 方法 2: 直接连接数据库执行

```bash
# 1. 从 Railway 控制台获取数据库连接信息
# 进入 Railway 项目 -> PostgreSQL 服务 -> Connect

# 2. 使用 psql 连接
psql "postgresql://username:password@host:port/database"

# 3. 执行迁移脚本
\i database/migrations/001_add_srn_mapping.sql

# 4. 验证字段已添加
\d tracked_events
```

### 方法 3: 通过 Railway Web Console

1. 进入 Railway 项目控制台
2. 选择 PostgreSQL 服务
3. 点击 "Data" 标签
4. 点击 "Query" 按钮
5. 复制粘贴 `001_add_srn_mapping.sql` 的内容
6. 点击 "Run Query" 执行

## 验证迁移

执行以下 SQL 查询验证字段已正确添加:

```sql
SELECT column_name, data_type, is_nullable
FROM information_schema.columns
WHERE table_name = 'tracked_events'
AND column_name IN (
    'srn_id', 'schedule_time', 
    'home_team_id', 'home_team_name',
    'away_team_id', 'away_team_name',
    'home_score', 'away_score',
    'match_status', 'match_time'
);
```

预期结果应显示所有 10 个字段。

## 注意事项

1. **使用 IF NOT EXISTS**: 迁移脚本使用 `ADD COLUMN IF NOT EXISTS`,可以安全地重复执行
2. **零停机时间**: 添加字段操作不会影响现有数据和服务运行
3. **索引创建**: 脚本会自动创建必要的索引以优化查询性能
4. **默认值**: `home_score` 和 `away_score` 默认为 0

## 回滚 (如需要)

如果需要回滚迁移,执行以下 SQL:

```sql
-- 删除添加的字段
ALTER TABLE tracked_events DROP COLUMN IF EXISTS srn_id;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS schedule_time;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS home_team_id;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS home_team_name;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS away_team_id;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS away_team_name;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS home_score;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS away_score;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS match_status;
ALTER TABLE tracked_events DROP COLUMN IF EXISTS match_time;

-- 删除索引
DROP INDEX IF EXISTS idx_tracked_events_srn_id;
DROP INDEX IF EXISTS idx_tracked_events_schedule_time;
DROP INDEX IF EXISTS idx_tracked_events_home_team_id;
DROP INDEX IF EXISTS idx_tracked_events_away_team_id;
DROP INDEX IF EXISTS idx_tracked_events_match_status;
```

## 故障排除

### 错误: "column already exists"

这是正常的,说明字段已经存在。迁移脚本使用 `IF NOT EXISTS` 可以安全忽略此错误。

### 错误: "permission denied"

确保使用的数据库用户具有 ALTER TABLE 权限。Railway 默认提供的连接凭据应该有足够权限。

### 错误: "relation does not exist"

确保 `tracked_events` 表已存在。如果不存在,需要先运行初始化脚本创建表结构。

## 相关文件

- 迁移脚本: `database/migrations/001_add_srn_mapping.sql`
- 迁移工具: `cmd/migrate/main.go`
- 数据库模型: `database/models.go`

