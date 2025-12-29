# 数据库迁移执行指南 - sr_market_id

## ⚠️ 重要提示

当前代码已推送到 GitHub,但**数据库迁移尚未执行**。这会导致应用程序运行时出现以下错误:

```
Failed to parse and store odds: failed to commit transaction: pq: Could not complete operation in a failed transaction
```

**原因**: 代码中使用了 `sr_market_id` 字段,但数据库表中仍然是 `market_id` 字段。

---

## 📋 需要执行的迁移

### 方法 1: 使用迁移脚本 (推荐)

```bash
cd /home/ubuntu/betradar-uof-service
PGPASSWORD='qcriEvdpsnxvfPLaGuCuTqtivHpKoodg' psql -h 103.235.47.102 -p 5432 -U postgres -d betradar_uof -f migrate_market_id_to_sr_market_id.sql
```

### 方法 2: 手动执行 SQL 语句

如果迁移脚本无法执行,可以手动执行以下 SQL:

```sql
-- 1. bet_settlements 表
ALTER TABLE IF EXISTS bet_settlements RENAME COLUMN market_id TO sr_market_id;

-- 2. bet_cancels 表
ALTER TABLE IF EXISTS bet_cancels RENAME COLUMN market_id TO sr_market_id;

-- 3. rollback_bet_settlements 表
ALTER TABLE IF EXISTS rollback_bet_settlements RENAME COLUMN market_id TO sr_market_id;

-- 4. rollback_bet_cancels 表
ALTER TABLE IF EXISTS rollback_bet_cancels RENAME COLUMN market_id TO sr_market_id;

-- 5. outcomes 表 (如果有 market_id 字段)
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

## ✅ 验证迁移是否成功

执行以下 SQL 查询,检查字段是否已重命名:

```sql
SELECT 
    table_name, 
    column_name, 
    data_type 
FROM information_schema.columns 
WHERE column_name IN ('market_id', 'sr_market_id')
    AND table_schema = 'public'
ORDER BY table_name, column_name;
```

**预期结果**:
- `bet_settlements` 应该有 `sr_market_id` 字段
- `bet_cancels` 应该有 `sr_market_id` 字段
- `rollback_bet_settlements` 应该有 `sr_market_id` 字段
- `rollback_bet_cancels` 应该有 `sr_market_id` 字段
- `markets` 应该有 `sr_market_id` 字段
- `market_descriptions` 应该有 `sr_market_id` 字段
- `outcome_descriptions` 应该有 `sr_market_id` 字段
- `mapping_outcomes` 应该有 `sr_market_id` 字段
- `odds` 应该有 `market_id` 字段 (这个不变,因为是外键指向 markets.id)

---

## 🔄 迁移后需要做的事情

### 1. 重启应用程序
```bash
# 如果使用 systemd
sudo systemctl restart betradar-uof-service

# 或者手动重启
pkill -f betradar-uof-service
cd /home/ubuntu/betradar-uof-service
./betradar-uof-service
```

### 2. 检查日志
```bash
# 查看应用日志,确认没有数据库错误
tail -f /path/to/logs/betradar-uof-service.log
```

### 3. 验证功能
- 测试赔率变化是否正常存储
- 测试市场查询 API
- 测试结算和取消功能

---

## 🚨 回滚方案 (如果出现问题)

如果迁移后出现问题,可以回滚:

```sql
-- 回滚 bet_settlements 表
ALTER TABLE IF EXISTS bet_settlements RENAME COLUMN sr_market_id TO market_id;

-- 回滚 bet_cancels 表
ALTER TABLE IF EXISTS bet_cancels RENAME COLUMN sr_market_id TO market_id;

-- 回滚 rollback_bet_settlements 表
ALTER TABLE IF EXISTS rollback_bet_settlements RENAME COLUMN sr_market_id TO market_id;

-- 回滚 rollback_bet_cancels 表
ALTER TABLE IF EXISTS rollback_bet_cancels RENAME COLUMN sr_market_id TO market_id;
```

然后回滚代码:
```bash
cd /home/ubuntu/betradar-uof-service
git revert HEAD~2..HEAD
git push origin main
```

---

## 📞 联系方式

如果迁移过程中遇到问题,请联系开发团队。

---

**创建时间**: 2025-10-31  
**迁移脚本**: `migrate_market_id_to_sr_market_id.sql`  
**相关提交**: `970bd56`, `d91b085`  
**推送仓库**: https://github.com/gdszyy/betradar-uof-service/

