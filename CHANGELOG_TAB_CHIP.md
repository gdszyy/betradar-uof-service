# 市场卡片展示方案 - 变更日志

## 版本 1.0.0 (2025-12-02)

### 新增功能

#### 数据库
- 添加 `markets.tab_id` 字段 - 市场所属的 Tab ID
- 添加 `markets.chip_id` 字段 - 市场所属的 Chip ID
- 创建 `market_tabs` 表 - 存储 17 个 Tab 配置
- 创建 `market_chips` 表 - 存储 158 个 Chip 配置
- 创建 `market_tab_chip_mapping` 表 - 记录 Market 与 Tab/Chip 的关系
- 创建 `market_groups_cache` 表 - 缓存市场的 Groups
- 创建 `market_specifiers_cache` 表 - 缓存解析后的 Specifiers
- 创建 `market_tab_chip_view` 视图 - 便于查询市场的 Tab/Chip 信息
- 添加 7 个性能索引

#### 业务逻辑
- 实现 `MarketTabChipService` - 核心业务逻辑服务
  - `AssignTabChipToMarket()` - 为单个市场分配 Tab/Chip
  - `AssignTabChipToAllMarkets()` - 批量分配所有市场
  - `GetMarketsByTabChip()` - 按 Tab/Chip 查询市场
  - `GetTabsForEvent()` - 获取事件的所有 Tab
  - `GetChipsForTab()` - 获取 Tab 的所有 Chip
  - `determineTabID()` - Tab 分配规则引擎
  - `determineChipID()` - Chip 分配规则引擎
  - `parseSpecifiers()` - Specifier 解析器

#### REST API
- 实现 `MarketTabChipHandler` - REST API 处理器
  - `GET /api/v1/events/{eventId}/tabs` - 获取事件的所有 Tab
  - `GET /api/v1/events/{eventId}/markets?tab={tabId}&chip={chipId}` - 按 Tab/Chip 查询市场
  - `GET /api/v1/tabs/{tabId}/chips` - 获取 Tab 的所有 Chip
  - `GET /api/v1/events/{eventId}/market-cards` - 获取完整市场卡片数据
  - `GET /api/v1/health` - 健康检查

#### 数据导入工具
- 实现 `import_tab_chip` 工具 - 从 CSV 导入 Tab/Chip 配置
- 实现 `assign_tab_chip` 工具 - 为市场分配 Tab/Chip
- 创建 `import_all.sh` 脚本 - 自动化完整导入流程

#### 配置数据
- 提供 17 个 Tab 配置（final_tab_chip_config.csv）
- 提供 158 个 Chip 配置（final_chip_enumeration.csv）

#### 测试
- 实现单元测试
  - `TestParseSpecifiers` - 测试 Specifier 解析
  - `TestDetermineTabID` - 测试 Tab 确定逻辑
  - `TestDetermineChipID` - 测试 Chip 确定逻辑
  - `TestSpecifierPairMarshaling` - 测试 JSON 序列化
- 创建 `test_tab_chip.sh` 脚本 - 完整的测试套件

#### 文档
- `MARKET_TAB_CHIP_IMPLEMENTATION.md` - 详细实现文档
- `DEPLOYMENT_GUIDE.md` - 部署指南
- `QUICKSTART.md` - 快速开始指南
- `IMPLEMENTATION_SUMMARY.md` - 实现总结
- `CHANGELOG_TAB_CHIP.md` - 本变更日志

#### 部署
- 创建 `railway.toml` - Railway 部署配置
- 创建 `railway-post-deploy.sh` - 部署后自动化脚本

### Tab 配置详情

| 优先级 | Tab ID | Tab 名称 | 类型 | Chip 数量 |
|--------|--------|---------|------|----------|
| 1 | `regular_play` | 常规玩法 | group | 0 |
| 2 | `innings` | 分局 | specifier_aggregate | 11 |
| 3 | `player_props` | 球员道具 | group | 1 |
| 4 | `micro_market` | 微盘口 | group | 3 |
| 5 | `sets` | 分盘 | specifier_aggregate | 14 |
| 6 | `maps` | 分地图 | specifier_aggregate | 10 |
| 7 | `bookings` | 罚牌 | group | 0 |
| 8 | `corners` | 角球 | group | 5 |
| 9 | `1st_half` | 上半场 | group | 6 |
| 10 | `quarters` | 分节 | specifier_aggregate | 4 |
| 11 | `combo` | 组合玩法 | group | 0 |
| 12 | `2nd_half` | 下半场 | group | 5 |
| 13 | `periods` | 分时段 | specifier_aggregate | 3 |
| 14 | `frames` | 分Frame | specifier_aggregate | 19 |
| 15 | `scorers` | 射手 | group | 5 |
| 16 | `overs` | 分Over | specifier_aggregate | 50 |
| 17 | `drives` | 分Drive | specifier_aggregate | 20 |

### 文件清单

#### 新增文件
```
database/
  ├── migrations/
  │   └── 011_add_tab_chip_fields.sql
  └── models_tab_chip.go

services/
  ├── market_tab_chip_service.go
  └── market_tab_chip_service_test.go

handlers/
  └── market_tab_chip_handler.go

cmd/
  ├── import_tab_chip/
  │   └── main.go
  └── assign_tab_chip/
      └── main.go

scripts/
  ├── import_all.sh
  └── test_tab_chip.sh

文档:
  ├── MARKET_TAB_CHIP_IMPLEMENTATION.md
  ├── DEPLOYMENT_GUIDE.md
  ├── QUICKSTART.md
  ├── IMPLEMENTATION_SUMMARY.md
  └── CHANGELOG_TAB_CHIP.md

配置:
  ├── final_tab_chip_config.csv
  ├── final_chip_enumeration.csv
  ├── railway.toml
  └── railway-post-deploy.sh
```

#### 修改文件
- `main.go` - 需要添加 API 处理器注册（见 QUICKSTART.md）

### 使用指南

#### 快速开始
```bash
# 1. 运行数据库迁移
psql $DATABASE_URL -f database/migrations/011_add_tab_chip_fields.sql

# 2. 导入配置
bash scripts/import_all.sh

# 3. 在 main.go 中注册 API 处理器（见 QUICKSTART.md）

# 4. 启动应用
go run main.go
```

#### 部署到 Railway
```bash
# 1. 提交代码
git add .
git commit -m "Add market tab/chip display implementation"
git push origin main

# 2. Railway 自动部署并运行 railway-post-deploy.sh
```

### 破坏性变更

无。本版本是完全向后兼容的，不会影响现有功能。

### 已知问题

1. **动态 Chip** - 某些 Chip 的值是动态的，需要在查询时计算
2. **性能** - 大量市场的分配可能需要较长时间

### 改进建议

1. 添加缓存层提高查询性能
2. 实现分页查询支持大量数据
3. 添加 WebSocket 支持实时更新
4. 实现市场搜索和过滤功能

### 贡献者

- 实现者：AI Assistant
- 审核者：待定
- 部署者：待定

### 许可证

与项目保持一致

---

## 升级指南

### 从之前的版本升级

由于这是首次发布，无升级指南。

### 回滚

如需回滚，请执行：
```bash
# 1. 删除新增的表和列
psql $DATABASE_URL -c "
  DROP TABLE IF EXISTS market_tab_chip_mapping CASCADE;
  DROP TABLE IF EXISTS market_chips CASCADE;
  DROP TABLE IF EXISTS market_tabs CASCADE;
  DROP TABLE IF EXISTS market_groups_cache CASCADE;
  DROP TABLE IF EXISTS market_specifiers_cache CASCADE;
  ALTER TABLE markets DROP COLUMN IF EXISTS tab_id;
  ALTER TABLE markets DROP COLUMN IF EXISTS chip_id;
"

# 2. 回滚代码
git revert <commit_hash>
git push origin main
```

---

## 支持

如有问题，请查看：
- 详细文档：`MARKET_TAB_CHIP_IMPLEMENTATION.md`
- 部署指南：`DEPLOYMENT_GUIDE.md`
- 快速开始：`QUICKSTART.md`
- 测试脚本：`bash scripts/test_tab_chip.sh`
