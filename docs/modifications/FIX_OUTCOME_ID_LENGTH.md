# 修复 outcome_id 字段长度限制问题

## 问题描述

在 Railway 部署环境中,`MarketDescService` 在保存市场映射数据时遇到数据库字段长度限制错误:

```
2025/11/10 06:33:16 [MarketDescService] ⚠️  Failed to insert mapping 893/sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333: pq: value too long for type character varying(50)
```

## 根本原因

数据库表 `mapping_outcomes` 和 `outcome_descriptions` 中的 `outcome_id` 和 `market_id` 字段定义为 `VARCHAR(50)`,但实际数据中存在超过50字符的标识符。

**问题数据示例:**
- `sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333` (52字符)
- `sr:goalscorer:fieldplayers_nogoal_owngoal_other:1334` (52字符)
- `sr:goalscorer:fieldplayers_nogoal_owngoal_other:1335` (52字符)

由于插入操作在事务中执行,第一个失败后会导致整个事务回滚,后续所有插入都会失败。

## 解决方案

将相关字段的长度从 `VARCHAR(50)` 扩展到 `VARCHAR(200)`,以容纳更长的 URN 格式标识符。

### 涉及的表和字段

1. **mapping_outcomes** 表
   - `market_id`: VARCHAR(50) → VARCHAR(200)
   - `outcome_id`: VARCHAR(50) → VARCHAR(200)

2. **outcome_descriptions** 表
   - `market_id`: VARCHAR(50) → VARCHAR(200)
   - `outcome_id`: VARCHAR(50) → VARCHAR(200)

3. **market_descriptions** 表
   - `market_id`: VARCHAR(50) → VARCHAR(200)

## 修复步骤

### 方案 1: 在 Railway 数据库中直接执行 SQL (推荐)

1. 登录 Railway 控制台
2. 进入项目的 PostgreSQL 数据库
3. 执行以下 SQL 脚本:

```sql
BEGIN;

-- 修改 mapping_outcomes 表
ALTER TABLE mapping_outcomes 
ALTER COLUMN outcome_id TYPE VARCHAR(200);

ALTER TABLE mapping_outcomes 
ALTER COLUMN market_id TYPE VARCHAR(200);

-- 修改 outcome_descriptions 表
ALTER TABLE outcome_descriptions 
ALTER COLUMN outcome_id TYPE VARCHAR(200);

ALTER TABLE outcome_descriptions 
ALTER COLUMN market_id TYPE VARCHAR(200);

-- 修改 market_descriptions 表
ALTER TABLE market_descriptions 
ALTER COLUMN market_id TYPE VARCHAR(200);

COMMIT;
```

4. 验证修改:

```sql
SELECT 
    table_name, 
    column_name, 
    data_type, 
    character_maximum_length 
FROM information_schema.columns 
WHERE table_name IN ('mapping_outcomes', 'outcome_descriptions', 'market_descriptions')
  AND column_name IN ('market_id', 'outcome_id')
ORDER BY table_name, column_name;
```

5. 重启 Railway 服务,让服务重新加载市场描述数据

### 方案 2: 通过迁移脚本修复

如果项目使用了数据库迁移工具,可以使用新创建的迁移文件:

```bash
# 执行迁移
psql $DATABASE_URL -f database/migrations/012_fix_outcome_id_length.sql
```

或者使用快速修复脚本:

```bash
psql $DATABASE_URL -f HOTFIX_OUTCOME_ID_LENGTH.sql
```

## 已修改的文件

为了防止未来重新创建数据库时出现同样的问题,以下文件已更新:

1. ✅ `database/init_all_tables.sql` - 初始化脚本
2. ✅ `database/migrations/008_create_market_descriptions_cache.sql` - 原始迁移文件
3. ✅ `database/migrations/012_fix_outcome_id_length.sql` - 新增修复迁移
4. ✅ `HOTFIX_OUTCOME_ID_LENGTH.sql` - 快速修复脚本

## 验证修复

修复完成后,重启服务,观察日志应该看到:

```
[MarketDescService] ✅ Loaded 1341 market descriptions from API
[MarketDescService] ✅ Parsed 4184 total mapping outcomes
[MarketDescService] ✅ Saved 1341 markets, XXXX outcomes, and 4184 mappings to database
```

不应再出现 "value too long for type character varying(50)" 错误。

## 注意事项

1. **执行前备份**: 虽然这是非破坏性的 ALTER 操作,但建议在生产环境执行前备份数据库
2. **停机时间**: ALTER TABLE 操作可能会短暂锁定表,建议在低峰期执行
3. **回滚计划**: 如需回滚,可以将字段改回 VARCHAR(50),但需确保没有超长数据

## 技术细节

### 为什么选择 VARCHAR(200)?

- Sportradar URN 格式通常为: `sr:<type>:<subtype>:<id>`
- 最长的已知格式约为 52 字符
- VARCHAR(200) 提供了足够的缓冲空间,同时不会造成显著的存储开销
- PostgreSQL 的 VARCHAR 是变长存储,实际只占用数据长度+1-4字节

### 性能影响

- **索引**: 字段长度增加可能略微影响索引性能,但影响微乎其微
- **存储**: 由于是变长字段,实际存储空间不变
- **查询**: 对查询性能无明显影响

## 相关链接

- [PostgreSQL ALTER TABLE 文档](https://www.postgresql.org/docs/current/sql-altertable.html)
- [Sportradar URN 格式说明](https://docs.betradar.com/display/BD/UOF+-+Unique+identifiers)
