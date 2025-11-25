# 队伍表管理和 Logo 获取功能实现文档

**版本**: 1.0.0  
**作者**: Manus AI  
**日期**: 2025-11-25

---

## 概述

本文档描述了在 `betradar-uof-service` 项目中新增的队伍表管理和 Logo 自动获取功能。该功能实现了以下核心需求：

1. **队伍表管理**：建立 `teams` 表，存储队伍的基本信息和 Logo URL
2. **自动新增队伍**：在处理比赛信息时，自动检查并创建新队伍记录
3. **解耦的 Logo 获取**：通过独立的服务异步获取队伍 Logo，并落库存储
4. **重试机制**：对获取失败的 Logo 进行自动重试

---

## 架构设计

### 1. 数据库表结构

新增 `teams` 表，包含以下字段：

| 字段名 | 类型 | 描述 |
|--------|------|------|
| `id` | SERIAL | 主键，自增 ID |
| `team_id` | VARCHAR(100) | 队伍唯一标识符（例如：`sr:competitor:12345`），唯一索引 |
| `team_name` | VARCHAR(255) | 队伍名称 |
| `sport_id` | VARCHAR(50) | 体育项目 ID |
| `sport_name` | VARCHAR(100) | 体育项目名称 |
| `category_id` | VARCHAR(50) | 类别 ID（例如：国家） |
| `category_name` | VARCHAR(200) | 类别名称 |
| `logo_url` | VARCHAR(500) | 队伍 Logo URL |
| `logo_fetched` | BOOLEAN | Logo 是否已成功获取 |
| `logo_fetch_attempted_at` | TIMESTAMP | 最后一次尝试获取 Logo 的时间 |
| `logo_fetch_retry_count` | INTEGER | Logo 获取失败后的重试次数 |
| `created_at` | TIMESTAMP | 创建时间 |
| `updated_at` | TIMESTAMP | 更新时间 |

**索引**：
- `teams_team_id_idx`：基于 `team_id` 的索引
- `teams_team_name_idx`：基于 `team_name` 的索引
- `teams_sport_id_idx`：基于 `sport_id` 的索引
- `teams_logo_fetched_idx`：基于 `logo_fetched` 的索引

---

### 2. 服务架构

#### 2.1 TeamsService（队伍管理服务）

**职责**：
- 提供队伍的 CRUD 操作
- 实现 `GetOrCreateTeam` 方法，用于检查队伍是否存在，不存在则创建
- 提供 `UpdateTeamLogo` 方法，用于更新队伍的 Logo URL
- 提供 `GetTeamsNeedingLogoFetch` 方法，用于查询需要获取 Logo 的队伍列表

**核心方法**：
```go
func (s *TeamsService) GetOrCreateTeam(teamInfo TeamInfo) (team *database.Team, isNew bool, err error)
func (s *TeamsService) GetTeamByID(teamID string) (*database.Team, error)
func (s *TeamsService) CreateTeam(teamInfo TeamInfo) (*database.Team, error)
func (s *TeamsService) UpdateTeamLogo(teamID string, logoURL string, success bool) error
func (s *TeamsService) GetTeamsNeedingLogoFetch(maxRetries int, limit int) ([]*database.Team, error)
```

---

#### 2.2 LogoFetcherService（Logo 获取服务）

**职责**：
- 异步获取队伍的 Logo URL
- 使用 **TheSportsDB API** 作为 Logo 数据源
- 定期扫描数据库，查找需要获取 Logo 的队伍
- 实现重试机制，对获取失败的队伍进行重试
- 提供同步和异步两种 Logo 获取方式

**核心方法**：
```go
func (s *LogoFetcherService) Start()
func (s *LogoFetcherService) Stop()
func (s *LogoFetcherService) FetchLogoForTeam(teamID string, teamName string) error
func (s *LogoFetcherService) ScheduleLogoFetch(teamID string, teamName string)
func (s *LogoFetcherService) fetchLogoFromTheSportsDB(teamName string) (string, error)
```

**配置参数**：
- `apiKey`: TheSportsDB API Key（免费版：`123`）
- `apiBaseURL`: TheSportsDB API 基础 URL
- `maxRetries`: 最大重试次数（默认：3）
- `batchSize`: 每次批量处理的队伍数量（默认：10）
- `interval`: 定期扫描间隔（默认：5 分钟）

---

#### 2.3 OddsChangeParser（赔率变化解析器）

**修改内容**：
- 新增 `teamsService` 和 `logoFetcher` 依赖
- 在解析 `odds_change` 消息时，提取主客队信息
- 调用 `processTeam` 方法，检查队伍是否存在，不存在则创建并安排 Logo 获取

**核心逻辑**：
```go
func (p *OddsChangeParser) processTeam(teamID, teamName, eventID string) {
    // 1. 尝试获取或创建队伍记录
    team, isNew, err := p.teamsService.GetOrCreateTeam(teamInfo)
    
    // 2. 如果是新队伍，异步安排 Logo 获取
    if isNew {
        p.logoFetcher.ScheduleLogoFetch(teamID, teamName)
    }
    
    // 3. 如果队伍已存在但 Logo 未获取，也可以尝试重新获取
    if !isNew && !team.LogoFetched && team.LogoFetchRetryCount < 3 {
        p.logoFetcher.ScheduleLogoFetch(teamID, teamName)
    }
}
```

---

### 3. 数据流程

```
1. 接收 odds_change 消息
   ↓
2. OddsChangeParser 解析消息
   ↓
3. 提取主客队信息（team_id, team_name）
   ↓
4. 调用 TeamsService.GetOrCreateTeam()
   ├─ 队伍已存在 → 返回现有记录
   └─ 队伍不存在 → 创建新记录，标记为新队伍
   ↓
5. 如果是新队伍，调用 LogoFetcherService.ScheduleLogoFetch()
   ↓
6. LogoFetcherService 异步调用 TheSportsDB API
   ↓
7. 获取 Logo URL 后，调用 TeamsService.UpdateTeamLogo()
   ↓
8. 更新 teams 表中的 logo_url 和 logo_fetched 字段
```

---

### 4. Logo 获取策略

#### 4.1 数据源选择

**首选方案：TheSportsDB API**

- **API 端点**：`https://www.thesportsdb.com/api/v1/json/123/searchteams.php?t={team_name}`
- **优点**：
  - 免费开放
  - 数据覆盖广泛
  - 返回结构化 JSON 数据
  - 包含 `strBadge`（队徽）和 `strLogo` 字段
- **缺点**：
  - 依赖队伍名称匹配，可能存在误匹配
  - 部分小众队伍可能无数据

#### 4.2 获取流程

1. **实时获取**：当检测到新队伍时，立即异步调度 Logo 获取任务
2. **定期扫描**：每 5 分钟扫描一次数据库，查找 `logo_fetched = false` 且重试次数未超限的队伍
3. **批量处理**：每次最多处理 10 个队伍，避免 API 请求过于频繁
4. **重试机制**：获取失败的队伍会自动重试，最多重试 3 次

#### 4.3 错误处理

- **API 请求失败**：记录错误日志，增加重试计数
- **未找到队伍**：标记为获取失败，增加重试计数
- **超过重试次数**：不再尝试获取，等待手动干预

---

## 文件清单

### 新增文件

1. **`database/migrations/013_create_teams_table.sql`**
   - 数据库迁移脚本，创建 `teams` 表

2. **`database/models.go`**
   - 新增 `Team` 模型结构

3. **`services/teams_service.go`**
   - 队伍管理服务实现

4. **`services/logo_fetcher_service.go`**
   - Logo 获取服务实现

### 修改文件

1. **`services/odds_change_parser.go`**
   - 新增 `teamsService` 和 `logoFetcher` 依赖
   - 新增 `processTeam` 方法
   - 修改 `NewOddsChangeParser` 构造函数

2. **`services/message_processor.go`**
   - 在 `NewMessageProcessor` 中初始化 `TeamsService` 和 `LogoFetcherService`
   - 启动 `LogoFetcherService`

---

## 使用说明

### 1. 数据库迁移

在部署前，需要运行数据库迁移脚本以创建 `teams` 表：

```bash
# 方式 1：通过应用程序自动迁移
# 应用程序启动时会自动执行 database.Migrate(db)

# 方式 2：手动执行 SQL 脚本
psql -h <host> -U <user> -d <database> -f database/migrations/013_create_teams_table.sql
```

### 2. 服务启动

服务启动时，`LogoFetcherService` 会自动启动，并开始定期扫描需要获取 Logo 的队伍。

### 3. 查询队伍 Logo

可以通过以下 SQL 查询队伍的 Logo 信息：

```sql
-- 查询所有已获取 Logo 的队伍
SELECT team_id, team_name, logo_url, created_at
FROM teams
WHERE logo_fetched = true;

-- 查询 Logo 获取失败的队伍
SELECT team_id, team_name, logo_fetch_retry_count, logo_fetch_attempted_at
FROM teams
WHERE logo_fetched = false
ORDER BY logo_fetch_retry_count DESC;

-- 查询特定队伍的 Logo
SELECT team_id, team_name, logo_url, logo_fetched
FROM teams
WHERE team_name LIKE '%Arsenal%';
```

### 4. 手动触发 Logo 获取

如果需要手动为特定队伍获取 Logo，可以通过以下方式：

```sql
-- 重置队伍的 Logo 获取状态，使其重新进入获取队列
UPDATE teams
SET logo_fetched = false,
    logo_fetch_retry_count = 0,
    logo_fetch_attempted_at = NULL
WHERE team_id = 'sr:competitor:12345';
```

---

## 性能优化

1. **异步处理**：Logo 获取采用异步方式，不阻塞主业务流程
2. **批量处理**：每次最多处理 10 个队伍，避免数据库和 API 压力过大
3. **索引优化**：为 `team_id`、`team_name`、`logo_fetched` 等字段创建索引
4. **请求限流**：每次 API 请求后添加 500ms 延迟，避免触发 API 限流

---

## 监控和日志

### 日志输出示例

```
[TeamsService] ✅ Created new team: Arsenal (ID: sr:competitor:42)
[OddsChangeParser] 🆕 New team detected: Arsenal (ID: sr:competitor:42), scheduling logo fetch
[LogoFetcherService] 🚀 Starting Logo fetcher service...
[LogoFetcherService] 📥 Fetching logos for pending teams...
[LogoFetcherService] 📋 Found 5 teams needing logo fetch
[LogoFetcherService] ✅ Updated logo for team sr:competitor:42: https://www.thesportsdb.com/images/media/team/badge/abc123.png
[LogoFetcherService] ✅ Logo fetch completed: 5 success, 0 failure
```

### 监控指标

- **队伍总数**：`SELECT COUNT(*) FROM teams;`
- **已获取 Logo 的队伍数**：`SELECT COUNT(*) FROM teams WHERE logo_fetched = true;`
- **Logo 获取成功率**：`SELECT (COUNT(*) FILTER (WHERE logo_fetched = true) * 100.0 / COUNT(*)) AS success_rate FROM teams;`
- **待获取 Logo 的队伍数**：`SELECT COUNT(*) FROM teams WHERE logo_fetched = false AND logo_fetch_retry_count < 3;`

---

## 未来扩展

1. **多数据源支持**：除了 TheSportsDB，还可以集成 Sportradar 官方 API、Wikipedia 等数据源
2. **图片缓存**：将 Logo 图片下载到本地或 CDN，避免外链失效
3. **图片质量检测**：自动检测 Logo 图片的分辨率和质量，优先选择高质量图片
4. **手动管理接口**：提供 Web 界面或 API，允许管理员手动上传或修改队伍 Logo
5. **Logo 更新检测**：定期检查队伍 Logo 是否有更新，自动刷新

---

## 总结

本次实现为 `betradar-uof-service` 项目添加了完整的队伍表管理和 Logo 自动获取功能。该功能具有以下特点：

- **解耦设计**：Logo 获取服务独立运行，不影响主业务流程
- **自动化**：新队伍自动创建，Logo 自动获取，无需人工干预
- **容错性强**：实现了重试机制和错误处理，确保系统稳定运行
- **易于扩展**：服务架构清晰，便于后续添加新功能

该功能已完全集成到项目中，可以立即投入使用。
