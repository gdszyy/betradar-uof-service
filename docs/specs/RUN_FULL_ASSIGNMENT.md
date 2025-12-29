# 市场 Tab/Chip 完整分配指南

## 概述

本指南说明如何运行市场 Tab/Chip 的完整分配。完整分配会为所有未映射的市场分配 `tab_id`。

## 方式一：本地运行（推荐用于测试）

### 前置条件

- Python 3.7+
- psycopg2 库
- 数据库访问权限

### 步骤

1. **安装依赖**
```bash
pip install psycopg2-binary
```

2. **设置数据库连接**
```bash
export DATABASE_URL="postgresql://postgres:password@host:port/database"
```

3. **运行分配脚本**
```bash
python3 scripts/run_full_assignment.py
```

### 示例

```bash
# 连接到 Railway 数据库
export DATABASE_URL="postgresql://postgres:qcriEvdpsnxvfPLaGuCuTqtivHpKoodg@turntable.proxy.rlwy.net:48608/railway"

# 运行分配
python3 scripts/run_full_assignment.py
```

## 方式二：Railway 后台任务（推荐用于生产环境）

### 步骤

1. **登录 Railway**
```bash
railway login
```

2. **连接到项目**
```bash
cd /path/to/betradar-uof-service
railway link
```

3. **运行分配脚本**
```bash
railway run python3 scripts/run_full_assignment.py
```

或者使用环境变量：
```bash
railway run python3 scripts/run_full_assignment.py
```

### 监控进度

```bash
# 查看日志
railway logs

# 查看数据库状态
railway run psql -c "SELECT COUNT(*), COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) FROM markets"
```

## 方式三：API 触发（如果已部署）

### 通过 HTTP 请求触发

```bash
# 触发完整分配
curl -X POST https://betradar-uof-service-production.up.railway.app/api/v1/assign-tab-chip \
  -H "Content-Type: application/json" \
  -d '{"mode": "full"}'

# 触发增量分配（仅新市场）
curl -X POST https://betradar-uof-service-production.up.railway.app/api/v1/assign-tab-chip \
  -H "Content-Type: application/json" \
  -d '{"mode": "incremental"}'
```

## 分配规则

脚本按以下优先级分配 Tab：

### 1. 基于 market_type（第一优先级）
- `regular_play` → `regular_play`
- `player_props` → `player_props`
- `micro_market` → `micro_market`
- `bookings` → `bookings`
- `corners` → `corners`
- `1st_half` → `1st_half`
- `combo` → `combo`
- `2nd_half` → `2nd_half`
- `scorers` → `scorers`

### 2. 基于 specifiers（第二优先级）
- `inningnr` → `innings`
- `setnr` → `sets`
- `mapnr` → `maps`
- `quarternr` → `quarters`
- `periodnr` → `periods`
- `framenr` → `frames`
- `overnr` → `overs`
- `drivenr` → `drives`

### 3. 默认值（第三优先级）
- 其他所有市场 → `regular_play`

## 性能指标

- **处理速度**：~10,000 市场/秒（使用批量 SQL 更新）
- **总耗时**：~20 秒（处理 182,000+ 市场）
- **内存占用**：< 100 MB

## 监控和验证

### 查看分配进度

```sql
-- 查看映射统计
SELECT 
    COUNT(*) as total,
    COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) as mapped,
    COUNT(CASE WHEN tab_id IS NULL THEN 1 END) as unmapped
FROM markets;

-- 查看 Tab 分布
SELECT tab_id, COUNT(*) as count
FROM markets
WHERE tab_id IS NOT NULL
GROUP BY tab_id
ORDER BY count DESC;

-- 查看未映射市场（应该为 0）
SELECT COUNT(*) FROM markets WHERE tab_id IS NULL OR tab_id = '';
```

### 查看分配日志

```sql
-- 查看分配日志
SELECT * FROM market_tab_chip_assignment_log
ORDER BY created_at DESC
LIMIT 10;

-- 查看未映射市场记录
SELECT * FROM market_tab_chip_unmapped
ORDER BY created_at DESC
LIMIT 10;
```

## 常见问题

### Q: 分配失败了怎么办？

A: 检查以下几点：
1. 数据库连接是否正常
2. 是否有足够的磁盘空间
3. 查看错误日志了解具体原因

### Q: 如何重新运行分配？

A: 脚本是幂等的，可以安全地多次运行。它会：
1. 检查未映射的市场
2. 按规则分配
3. 如果市场已映射，则跳过

### Q: 分配需要多长时间？

A: 取决于未映射市场的数量：
- 182,000 个市场：~20 秒
- 1,000 个市场：< 1 秒

### Q: 可以中断分配吗？

A: 可以。脚本使用事务，如果中断：
- 已提交的更新会保留
- 未提交的更新会回滚

## 故障排查

### 连接错误

```
错误：无法连接到数据库
```

**解决方案**：
- 检查 DATABASE_URL 是否正确
- 检查网络连接
- 检查数据库是否在线

### 权限错误

```
错误：permission denied for table markets
```

**解决方案**：
- 确保用户有 UPDATE 权限
- 检查数据库用户权限

### 超时错误

```
错误：timeout
```

**解决方案**：
- 增加连接超时时间
- 减少批量大小
- 检查数据库性能

## 相关文件

- `scripts/run_full_assignment.py` - 完整分配脚本
- `scripts/import_incremental.sh` - 增量导入脚本
- `database/migrations/011_add_tab_chip_fields.sql` - 数据库迁移
- `OPTIMIZATION_GUIDE.md` - 优化指南

## 支持

如有问题，请查看：
- `QUICKSTART.md` - 快速开始
- `MARKET_TAB_CHIP_IMPLEMENTATION.md` - 详细实现
- `DEPLOYMENT_GUIDE.md` - 部署指南
