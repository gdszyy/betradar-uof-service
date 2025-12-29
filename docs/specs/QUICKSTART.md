# 快速开始指南 - 市场卡片展示方案

## 5 分钟快速部署

### 前置条件

- Go 1.16+ 已安装
- PostgreSQL 数据库可访问
- 环境变量 `DATABASE_URL` 已配置

### 快速部署步骤

#### 1. 克隆项目并进入目录

```bash
cd betradar-uof-service
```

#### 2. 运行数据库迁移

```bash
psql $DATABASE_URL -f database/migrations/011_add_tab_chip_fields.sql
```

#### 3. 导入配置数据

```bash
bash scripts/import_all.sh
```

这个脚本会自动：
- 编译导入工具
- 导入 17 个 Tab 配置
- 导入 158 个 Chip 配置
- 为所有市场分配 tab_id 和 chip_id

#### 4. 在主应用中集成 API

编辑 `main.go`，在路由初始化中添加：

```go
import (
    "betradar-uof-service/handlers"
    "betradar-uof-service/services"
)

// 在 main() 函数中
tabChipService := services.NewMarketTabChipService(db)
tabChipHandler := handlers.NewMarketTabChipHandler(tabChipService)

router.HandleFunc("/api/v1/events/{eventId}/tabs", tabChipHandler.GetTabsForEvent).Methods("GET")
router.HandleFunc("/api/v1/events/{eventId}/markets", tabChipHandler.GetMarketsByTabChip).Methods("GET")
router.HandleFunc("/api/v1/tabs/{tabId}/chips", tabChipHandler.GetChipsForTab).Methods("GET")
router.HandleFunc("/api/v1/events/{eventId}/market-cards", tabChipHandler.GetMarketCardData).Methods("GET")
router.HandleFunc("/api/v1/health", tabChipHandler.HealthCheck).Methods("GET")
```

#### 5. 构建并运行

```bash
go build -o main .
./main
```

#### 6. 测试 API

```bash
# 健康检查
curl http://localhost:8080/api/v1/health

# 获取事件的 Tab（替换 sr:match:12345 为实际的 event_id）
curl "http://localhost:8080/api/v1/events/sr:match:12345/tabs"

# 按 Tab 查询市场
curl "http://localhost:8080/api/v1/events/sr:match:12345/markets?tab=regular_play"

# 获取 Tab 的 Chip
curl "http://localhost:8080/api/v1/tabs/quarters/chips"
```

## 常用命令

### 查看导入进度

```bash
psql $DATABASE_URL -c "
  SELECT 
    COUNT(*) as total_markets,
    COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) as with_tab,
    ROUND(100.0 * COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) / COUNT(*), 2) as percentage
  FROM markets
"
```

### 查看 Tab 分布

```bash
psql $DATABASE_URL -c "
  SELECT tab_id, COUNT(*) as count 
  FROM markets 
  WHERE tab_id IS NOT NULL 
  GROUP BY tab_id 
  ORDER BY count DESC
"
```

### 查看特定市场的 Tab/Chip

```bash
psql $DATABASE_URL -c "
  SELECT id, market_type, specifiers, tab_id, chip_id 
  FROM markets 
  WHERE event_id = 'sr:match:12345' 
  LIMIT 10
"
```

### 运行单元测试

```bash
go test ./services -v
```

## 文件说明

| 文件 | 说明 |
|------|------|
| `database/migrations/011_add_tab_chip_fields.sql` | 数据库迁移脚本 |
| `database/models_tab_chip.go` | 数据模型定义 |
| `services/market_tab_chip_service.go` | 业务逻辑实现 |
| `handlers/market_tab_chip_handler.go` | REST API 处理器 |
| `cmd/import_tab_chip/main.go` | 导入工具 |
| `cmd/assign_tab_chip/main.go` | 分配工具 |
| `scripts/import_all.sh` | 完整导入脚本 |
| `final_tab_chip_config.csv` | Tab 配置数据 |
| `final_chip_enumeration.csv` | Chip 配置数据 |
| `MARKET_TAB_CHIP_IMPLEMENTATION.md` | 详细文档 |
| `DEPLOYMENT_GUIDE.md` | 部署指南 |

## API 端点

### 获取事件的所有 Tab

```
GET /api/v1/events/{eventId}/tabs
```

**响应示例**:
```json
{
  "event_id": "sr:match:12345",
  "tabs": [
    {
      "id": "regular_play",
      "label": "常规玩法",
      "type": "group",
      "market_count": 198
    }
  ],
  "count": 1
}
```

### 按 Tab 和 Chip 查询市场

```
GET /api/v1/events/{eventId}/markets?tab={tabId}&chip={chipId}
```

**参数**:
- `tab` (必需) - Tab ID
- `chip` (可选) - Chip ID，省略则返回该 Tab 的所有市场

**响应示例**:
```json
{
  "event_id": "sr:match:12345",
  "tab_id": "regular_play",
  "markets": [
    {
      "market_id": 1,
      "market_type": "1X2",
      "market_name": "Match Odds",
      "tab_id": "regular_play"
    }
  ],
  "count": 1
}
```

### 获取 Tab 的所有 Chip

```
GET /api/v1/tabs/{tabId}/chips
```

**响应示例**:
```json
{
  "tab_id": "quarters",
  "chips": [
    {
      "id": "quarters_quarternr_1",
      "label": "第1节",
      "specifier": "quarternr",
      "value": "1"
    }
  ],
  "count": 4
}
```

### 获取完整市场卡片数据

```
GET /api/v1/events/{eventId}/market-cards
```

**响应示例**:
```json
{
  "event_id": "sr:match:12345",
  "tabs": [...],
  "markets": {
    "regular_play": [...],
    "quarters": [...]
  },
  "chips": {
    "quarters": [...]
  },
  "tab_count": 5
}
```

## 故障排查

### 问题：导入脚本失败

```bash
# 检查 CSV 文件
ls -la final_tab_chip_config.csv final_chip_enumeration.csv

# 检查数据库连接
psql $DATABASE_URL -c "SELECT 1"

# 手动运行导入工具
go run cmd/import_tab_chip/main.go \
  -db "$DATABASE_URL" \
  -tabs final_tab_chip_config.csv \
  -chips final_chip_enumeration.csv
```

### 问题：API 返回 404

```bash
# 检查路由是否注册
curl -v http://localhost:8080/api/v1/health

# 检查应用日志
tail -f app.log
```

### 问题：市场没有 tab_id

```bash
# 检查分配进度
psql $DATABASE_URL -c "
  SELECT COUNT(CASE WHEN tab_id IS NOT NULL THEN 1 END) 
  FROM markets
"

# 手动运行分配工具
go run cmd/assign_tab_chip/main.go -db "$DATABASE_URL"
```

## 下一步

- 查看 [详细文档](MARKET_TAB_CHIP_IMPLEMENTATION.md) 了解完整实现
- 查看 [部署指南](DEPLOYMENT_GUIDE.md) 部署到 Railway
- 查看 [API 文档](MARKET_TAB_CHIP_IMPLEMENTATION.md#api-使用示例) 了解 API 详情

## 支持

如有问题，请：
1. 查看日志：`railway logs`
2. 检查数据库：`psql $DATABASE_URL`
3. 运行测试：`bash scripts/test_tab_chip.sh`
4. 查看文档：`MARKET_TAB_CHIP_IMPLEMENTATION.md`
