# Railway 部署修复指南

## 🔥 紧急修复步骤

### 第一步: 修复数据库 Schema

1. **登录 Railway 控制台**
   - 访问 https://railway.app
   - 进入你的项目

2. **打开 PostgreSQL 数据库**
   - 点击 PostgreSQL 服务
   - 点击 "Data" 或 "Query" 标签

3. **执行修复 SQL**

复制并执行以下 SQL 语句:

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

4. **验证修改**

执行以下查询确认字段长度已更新:

```sql
SELECT 
    table_name, 
    column_name, 
    character_maximum_length 
FROM information_schema.columns 
WHERE table_name IN ('mapping_outcomes', 'outcome_descriptions', 'market_descriptions')
  AND column_name IN ('market_id', 'outcome_id')
ORDER BY table_name, column_name;
```

期望输出:
```
table_name            | column_name | character_maximum_length
----------------------|-------------|-------------------------
mapping_outcomes      | market_id   | 200
mapping_outcomes      | outcome_id  | 200
market_descriptions   | market_id   | 200
outcome_descriptions  | market_id   | 200
outcome_descriptions  | outcome_id  | 200
```

### 第二步: 重新部署服务

#### 选项 A: 通过 GitHub 自动部署 (推荐)

修复代码已推送到 GitHub,Railway 会自动检测并重新部署。

1. 在 Railway 控制台查看 "Deployments" 标签
2. 等待新的部署完成
3. 查看日志确认修复成功

#### 选项 B: 手动触发重新部署

1. 在 Railway 项目页面
2. 点击服务
3. 点击 "Settings" → "Service"
4. 点击 "Redeploy" 按钮

#### 选项 C: 重启现有服务

如果不想重新部署,只需重启:

1. 点击服务的 "Settings"
2. 点击 "Restart"

### 第三步: 验证修复

1. **查看服务日志**

在 Railway 控制台的 "Deployments" → "View Logs" 中,应该看到:

```
[MarketDescService] ✅ Loaded 1341 market descriptions from API
[MarketDescService] ✅ Parsed 4184 total mapping outcomes
[MarketDescService] Preparing to save 1341 markets with mappings
[MarketDescService] ✅ Saved 1341 markets, XXXX outcomes, and 4184 mappings to database
```

2. **确认没有错误**

不应再看到以下错误:
```
⚠️  Failed to insert mapping ... pq: value too long for type character varying(50)
```

## 📋 问题总结

### 问题原因
- 数据库字段 `outcome_id` 和 `market_id` 定义为 `VARCHAR(50)`
- Sportradar API 返回的某些 URN 标识符超过 50 字符
- 例如: `sr:goalscorer:fieldplayers_nogoal_owngoal_other:1333` (52字符)

### 解决方案
- 将字段长度扩展到 `VARCHAR(200)`
- 更新了初始化脚本和迁移文件,防止未来重建数据库时出现同样问题

### 影响范围
- 3个表: `mapping_outcomes`, `outcome_descriptions`, `market_descriptions`
- 5个字段: 每个表的 `market_id` 和/或 `outcome_id`

## 🔍 故障排查

### 如果修复后仍有问题

1. **检查表是否存在**
```sql
SELECT tablename FROM pg_tables 
WHERE tablename IN ('mapping_outcomes', 'outcome_descriptions', 'market_descriptions');
```

2. **检查字段类型**
```sql
\d mapping_outcomes
\d outcome_descriptions
\d market_descriptions
```

3. **清空并重新加载数据**
```sql
TRUNCATE mapping_outcomes, outcome_descriptions, market_descriptions CASCADE;
```

然后重启服务,让它重新从 API 加载数据。

4. **查看详细错误日志**

在 Railway 控制台启用详细日志:
- Settings → Environment → 添加 `LOG_LEVEL=debug`

### 如果需要回滚

```sql
BEGIN;

ALTER TABLE mapping_outcomes ALTER COLUMN outcome_id TYPE VARCHAR(50);
ALTER TABLE mapping_outcomes ALTER COLUMN market_id TYPE VARCHAR(50);
ALTER TABLE outcome_descriptions ALTER COLUMN outcome_id TYPE VARCHAR(50);
ALTER TABLE outcome_descriptions ALTER COLUMN market_id TYPE VARCHAR(50);
ALTER TABLE market_descriptions ALTER COLUMN market_id TYPE VARCHAR(50);

COMMIT;
```

**注意**: 回滚后原问题会重现,只在紧急情况下使用。

## 📞 支持

如果遇到问题:
1. 检查 Railway 服务日志
2. 验证数据库连接正常
3. 确认环境变量配置正确
4. 查看 GitHub Issues: https://github.com/gdszyy/betradar-uof-service/issues

## ✅ 完成清单

- [ ] 在 Railway 数据库中执行 ALTER TABLE 语句
- [ ] 验证字段长度已更新为 200
- [ ] 重新部署或重启服务
- [ ] 查看日志确认数据加载成功
- [ ] 确认没有 "value too long" 错误
- [ ] (可选) 测试市场描述 API 端点

---

**修复时间**: 预计 5-10 分钟  
**停机时间**: 约 1-2 分钟 (ALTER TABLE 期间)  
**风险等级**: 低 (非破坏性操作)
