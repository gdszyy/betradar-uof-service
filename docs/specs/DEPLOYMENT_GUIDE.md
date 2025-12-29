# 部署指南 - 市场卡片展示方案

## 概述

本指南说明如何将市场卡片展示方案部署到 Railway 平台。

## 前置条件

1. **Railway 账户** - 已创建并配置 betradar-uof-service 项目
2. **数据库** - PostgreSQL 数据库已连接到 Railway
3. **环境变量** - `DATABASE_URL` 已配置
4. **Git 仓库** - 代码已推送到 GitHub

## 部署步骤

### 步骤 1: 准备配置文件

确保以下文件在项目根目录：

```bash
betradar-uof-service/
├── final_tab_chip_config.csv
├── final_chip_enumeration.csv
└── ...
```

**验证文件**:
```bash
ls -la final_tab_chip_config.csv final_chip_enumeration.csv
```

### 步骤 2: 提交代码更改

```bash
cd betradar-uof-service

# 添加新文件
git add database/migrations/011_add_tab_chip_fields.sql
git add database/models_tab_chip.go
git add services/market_tab_chip_service.go
git add handlers/market_tab_chip_handler.go
git add cmd/import_tab_chip/main.go
git add cmd/assign_tab_chip/main.go
git add scripts/import_all.sh
git add scripts/test_tab_chip.sh
git add MARKET_TAB_CHIP_IMPLEMENTATION.md
git add DEPLOYMENT_GUIDE.md
git add final_tab_chip_config.csv
git add final_chip_enumeration.csv

# 提交
git commit -m "Add market tab/chip display implementation

- Add database migration for tab_id and chip_id fields
- Implement MarketTabChipService for tab/chip assignment
- Add REST API endpoints for market card display
- Add import tools for tab/chip configuration
- Add comprehensive documentation and deployment guide"

# 推送到 GitHub
git push origin main
```

### 步骤 3: Railway 自动部署

Railway 将自动检测到推送，并开始构建和部署：

1. **构建阶段** - Railway 运行 `go build` 编译应用
2. **部署阶段** - 新版本部署到生产环境
3. **启动阶段** - 应用启动并监听端口

可以在 Railway 仪表板查看部署进度。

### 步骤 4: 运行数据库迁移

连接到 Railway 数据库并运行迁移：

```bash
# 方式 1: 使用 Railway CLI
railway run psql -f database/migrations/011_add_tab_chip_fields.sql

# 方式 2: 使用 psql 直接连接
psql $DATABASE_URL -f database/migrations/011_add_tab_chip_fields.sql
```

### 步骤 5: 导入 Tab/Chip 配置

运行导入脚本：

```bash
# 方式 1: 使用 Railway CLI
railway run bash scripts/import_all.sh

# 方式 2: 手动运行
export DATABASE_URL="postgresql://..."
bash scripts/import_all.sh
```

**预期输出**:
```
ℹ Project directory: /app
ℹ Script directory: /app/scripts
✓ Found configuration files
ℹ Tab config: /app/final_tab_chip_config.csv
ℹ Chip config: /app/final_chip_enumeration.csv

=== Step 1: Running Database Migration ===
✓ Database migration completed

=== Step 2: Building Import Binary ===
✓ Binary built successfully

=== Step 3: Importing Tab and Chip Configurations ===
✓ Tab and chip configurations imported successfully

=== Step 4: Assigning Tab/Chip to Markets ===
✓ Tab/Chip assignment completed successfully

=== Step 6: Verifying Import ===
Total tabs in database: 17
Total chips in database: 158
Markets with tab_id assigned: 1234
Markets with chip_id assigned: 567

✓ Import Completed Successfully
```

### 步骤 6: 集成 API 处理器

在 `main.go` 中添加以下代码：

```go
package main

import (
	"betradar-uof-service/handlers"
	"betradar-uof-service/services"
	"github.com/gorilla/mux"
)

func main() {
	// ... 现有代码 ...

	// 初始化 Tab/Chip 服务
	tabChipService := services.NewMarketTabChipService(db)
	tabChipHandler := handlers.NewMarketTabChipHandler(tabChipService)

	// 注册路由
	router := mux.NewRouter()
	
	// 现有路由
	// ...

	// 新增 Tab/Chip 路由
	router.HandleFunc("/api/v1/events/{eventId}/tabs", tabChipHandler.GetTabsForEvent).Methods("GET")
	router.HandleFunc("/api/v1/events/{eventId}/markets", tabChipHandler.GetMarketsByTabChip).Methods("GET")
	router.HandleFunc("/api/v1/tabs/{tabId}/chips", tabChipHandler.GetChipsForTab).Methods("GET")
	router.HandleFunc("/api/v1/events/{eventId}/market-cards", tabChipHandler.GetMarketCardData).Methods("GET")
	router.HandleFunc("/api/v1/health", tabChipHandler.HealthCheck).Methods("GET")

	// 启动服务器
	log.Fatal(http.ListenAndServe(":8080", router))
}
```

### 步骤 7: 重新部署

```bash
# 提交 main.go 的更改
git add main.go
git commit -m "Integrate market tab/chip API endpoints"
git push origin main
```

Railway 将自动重新部署应用。

### 步骤 8: 验证部署

```bash
# 检查应用状态
curl https://betradar-uof-service-production.up.railway.app/api/v1/health

# 预期响应
{
  "status": "ok",
  "service": "market-tab-chip"
}
```

## 验证清单

部署完成后，检查以下项目：

- [ ] 应用在 Railway 上成功部署
- [ ] 数据库迁移已执行
- [ ] Tab 配置已导入（17 个）
- [ ] Chip 配置已导入（158 个）
- [ ] 市场已分配 tab_id 和 chip_id
- [ ] API 端点可访问
- [ ] 健康检查返回成功

## 故障排查

### 问题 1: 迁移失败

**症状**: 运行迁移脚本时出错

**解决方案**:
```bash
# 检查表是否已存在
psql $DATABASE_URL -c "\dt market_tabs"

# 如果表存在，检查列
psql $DATABASE_URL -c "\d markets"

# 如果需要重新运行迁移，先删除表（谨慎操作！）
psql $DATABASE_URL -c "DROP TABLE IF EXISTS market_tabs CASCADE;"
psql $DATABASE_URL -f database/migrations/011_add_tab_chip_fields.sql
```

### 问题 2: 导入失败

**症状**: 导入脚本返回错误

**解决方案**:
```bash
# 检查 CSV 文件格式
head -5 final_tab_chip_config.csv
head -5 final_chip_enumeration.csv

# 检查数据库连接
psql $DATABASE_URL -c "SELECT 1"

# 手动导入（调试）
go run cmd/import_tab_chip/main.go \
  -db "$DATABASE_URL" \
  -tabs final_tab_chip_config.csv \
  -chips final_chip_enumeration.csv
```

### 问题 3: API 端点返回 404

**症状**: 调用 API 端点时返回 404

**解决方案**:
```bash
# 检查路由是否注册
curl -v https://betradar-uof-service-production.up.railway.app/api/v1/health

# 检查应用日志
railway logs

# 确认 main.go 中已注册路由
grep -n "tabChipHandler" main.go
```

### 问题 4: 市场没有 tab_id

**症状**: 查询市场时 tab_id 为 NULL

**解决方案**:
```bash
# 检查分配进度
psql $DATABASE_URL -c "
  SELECT 
    COUNT(*) as total,
    COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) as with_tab
  FROM markets
"

# 手动运行分配
go run cmd/assign_tab_chip/main.go -db "$DATABASE_URL"

# 检查特定市场
psql $DATABASE_URL -c "
  SELECT id, market_type, specifiers, tab_id, chip_id 
  FROM markets 
  WHERE id = 1 
  LIMIT 5
"
```

## 监控和维护

### 定期检查

```bash
# 每周检查数据完整性
psql $DATABASE_URL -c "
  SELECT 
    COUNT(*) as total_markets,
    COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) as with_tab,
    ROUND(100.0 * COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) / COUNT(*), 2) as percentage
  FROM markets
"

# 检查 Tab 分布
psql $DATABASE_URL -c "
  SELECT tab_id, COUNT(*) as count 
  FROM markets 
  WHERE tab_id IS NOT NULL 
  GROUP BY tab_id 
  ORDER BY count DESC
"
```

### 日志分析

```bash
# 查看应用日志
railway logs --follow

# 搜索错误
railway logs | grep -i error

# 搜索特定操作
railway logs | grep -i "tab_chip"
```

## 回滚计划

如果需要回滚到之前的版本：

```bash
# 查看部署历史
railway deployments

# 回滚到之前的版本
railway rollback <deployment_id>

# 或者重新部署之前的代码
git revert <commit_hash>
git push origin main
```

## 性能优化

### 查询优化

```bash
# 分析慢查询
psql $DATABASE_URL -c "
  EXPLAIN ANALYZE
  SELECT * FROM markets 
  WHERE event_id = 'sr:match:12345' AND tab_id = 'quarters'
"

# 检查索引使用情况
psql $DATABASE_URL -c "
  SELECT schemaname, tablename, indexname 
  FROM pg_indexes 
  WHERE tablename IN ('markets', 'market_tabs', 'market_chips')
"
```

### 缓存策略

考虑在应用层添加缓存：

```go
// 缓存 Tab 配置（24 小时）
var tabCache map[string]*database.MarketTab
var tabCacheExpiry time.Time

func GetTabsForEvent(eventID string) ([]*database.MarketTab, error) {
    if time.Now().Before(tabCacheExpiry) && tabCache != nil {
        return tabCache[eventID], nil
    }
    
    tabs, err := service.GetTabsForEvent(eventID)
    if err == nil {
        tabCache[eventID] = tabs
        tabCacheExpiry = time.Now().Add(24 * time.Hour)
    }
    return tabs, err
}
```

## 总结

部署步骤：
1. 准备配置文件
2. 提交代码更改
3. Railway 自动部署
4. 运行数据库迁移
5. 导入 Tab/Chip 配置
6. 集成 API 处理器
7. 重新部署应用
8. 验证部署成功

如有问题，请参考故障排查部分或查看应用日志。
