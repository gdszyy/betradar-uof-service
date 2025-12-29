# 修复 tracked_events 表字段为空问题

## 问题诊断

### 数据库现状
通过数据库查询发现 `tracked_events` 表存在大量字段为空的情况：

| 字段 | 填充率 | 说明 |
|------|--------|------|
| 总记录数 | 17,795 条 | - |
| home_team_name | 60.9% (10,835) | **大量为空** |
| away_team_name | 60.9% (10,835) | **大量为空** |
| sport_id | 97.4% (17,340) | 基本正常 |
| sport | 3.5% (626) | **几乎全空** |
| match_status | 0.2% (40) | **几乎全空** |
| srn_id | 0% (0) | **完全为空** |

### 根本原因

**PrematchService 的数据插入逻辑存在严重缺陷**：

1. **缺失字段解析**：`PrematchEvent` 结构体没有定义 `Competitors` 和 `Sport` 字段
2. **硬编码错误值**：`sport_id` 被硬编码为 `"unknown"`
3. **未插入队伍信息**：完全没有插入 `home_team_id`, `home_team_name`, `away_team_id`, `away_team_name`

#### 原代码问题（services/prematch_service.go）

```go
// ❌ 原结构体 - 缺少 Competitors 和 Sport 字段
type PrematchEvent struct {
    ID        string `xml:"id,attr"`
    Scheduled string `xml:"scheduled,attr"`
    Status    string `xml:"status,attr"`
    LiveOdds  string `xml:"liveodds,attr"`
}

// ❌ 原插入逻辑 - 缺少队伍信息
query := `
    INSERT INTO tracked_events (
        event_id, sport_id, status, schedule_time, 
        subscribed, created_at, updated_at
    ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    ...
`
_, err := s.db.Exec(
    query,
    event.ID,
    "unknown", // ❌ 硬编码为 "unknown"
    event.Status,
    scheduleTime,
    false,
    time.Now(),
    time.Now(),
)
```

## 修复方案

### 1. 扩展 PrematchEvent 结构体

添加 `Competitors` 和 `Sport` 字段以正确解析 XML：

```go
// ✅ 新增结构体
type PrematchCompetitor struct {
    ID        string `xml:"id,attr"`
    Name      string `xml:"name,attr"`
    Qualifier string `xml:"qualifier,attr"`
}

type PrematchSport struct {
    ID   string `xml:"id,attr"`
    Name string `xml:"name,attr"`
}

// ✅ 修复后的 PrematchEvent
type PrematchEvent struct {
    ID          string               `xml:"id,attr"`
    Scheduled   string               `xml:"scheduled,attr"`
    Status      string               `xml:"status,attr"`
    LiveOdds    string               `xml:"liveodds,attr"`
    Sport       PrematchSport        `xml:"sport"`
    Competitors []PrematchCompetitor `xml:"competitors>competitor"`
}
```

### 2. 修复 StorePrematchEvents 方法

完整解析并插入队伍信息：

```go
func (s *PrematchService) StorePrematchEvents(events []PrematchEvent) (int, error) {
    stored := 0
    
    for _, event := range events {
        // 解析时间
        var scheduleTime *time.Time
        if event.Scheduled != "" {
            t, err := time.Parse(time.RFC3339, event.Scheduled)
            if err == nil {
                scheduleTime = &t
            }
        }
        
        // ✅ 提取队伍信息
        var homeTeamID, homeTeamName, awayTeamID, awayTeamName string
        for _, comp := range event.Competitors {
            if comp.Qualifier == "home" {
                homeTeamID = comp.ID
                homeTeamName = comp.Name
            } else if comp.Qualifier == "away" {
                awayTeamID = comp.ID
                awayTeamName = comp.Name
            }
        }
        
        // ✅ 获取真实的 sport_id
        sportID := event.Sport.ID
        if sportID == "" {
            sportID = "unknown"
        }
        
        // ✅ 完整的插入语句
        query := `
            INSERT INTO tracked_events (
                event_id, sport_id, status, schedule_time, 
                home_team_id, home_team_name, away_team_id, away_team_name,
                subscribed, created_at, updated_at
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            ON CONFLICT (event_id) DO UPDATE SET
                sport_id = CASE WHEN EXCLUDED.sport_id = 'unknown' THEN tracked_events.sport_id ELSE EXCLUDED.sport_id END,
                status = EXCLUDED.status,
                schedule_time = EXCLUDED.schedule_time,
                home_team_id = CASE WHEN EXCLUDED.home_team_id = '' THEN tracked_events.home_team_id ELSE EXCLUDED.home_team_id END,
                home_team_name = CASE WHEN EXCLUDED.home_team_name = '' THEN tracked_events.home_team_name ELSE EXCLUDED.home_team_name END,
                away_team_id = CASE WHEN EXCLUDED.away_team_id = '' THEN tracked_events.away_team_id ELSE EXCLUDED.away_team_id END,
                away_team_name = CASE WHEN EXCLUDED.away_team_name = '' THEN tracked_events.away_team_name ELSE EXCLUDED.away_team_name END,
                updated_at = EXCLUDED.updated_at
        `
        
        _, err := s.db.Exec(
            query,
            event.ID,
            sportID,
            event.Status,
            scheduleTime,
            homeTeamID,
            homeTeamName,
            awayTeamID,
            awayTeamName,
            false,
            time.Now(),
            time.Now(),
        )
        
        if err != nil {
            logger.Printf("[PrematchService] ⚠️  Failed to store %s: %v", event.ID, err)
            continue
        }
        
        stored++
    }
    
    return stored, nil
}
```

## 修复效果

修复后，`PrematchService` 将能够：

1. ✅ 正确解析 XML 中的 `<sport>` 元素，获取真实的 `sport_id`
2. ✅ 正确解析 `<competitors>` 元素，提取 `home` 和 `away` 队伍信息
3. ✅ 完整插入所有必要字段到 `tracked_events` 表
4. ✅ 使用 `CASE WHEN` 逻辑避免用空值覆盖已有数据

## 验证方法

修复后，可通过以下 SQL 验证数据完整性：

```sql
-- 检查字段填充率
SELECT 
    COUNT(*) as total_records,
    COUNT(home_team_name) as has_home_team_name,
    COUNT(away_team_name) as has_away_team_name,
    COUNT(sport_id) as has_sport_id,
    ROUND(COUNT(home_team_name)::numeric / COUNT(*) * 100, 2) as home_fill_rate,
    ROUND(COUNT(away_team_name)::numeric / COUNT(*) * 100, 2) as away_fill_rate,
    ROUND(COUNT(sport_id)::numeric / COUNT(*) * 100, 2) as sport_fill_rate
FROM tracked_events;

-- 查看最新记录
SELECT event_id, sport_id, home_team_name, away_team_name, schedule_time
FROM tracked_events 
WHERE created_at > NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC 
LIMIT 10;
```

## 相关文件

- **修复文件**: `services/prematch_service.go`
- **参考实现**: `services/cold_start.go` (正确的实现示例)
- **数据库 Schema**: `database/init_all_tables.sql`

## 注意事项

1. **历史数据**：此修复只影响新插入的数据，历史空数据需要通过其他方式补全
2. **数据保护**：UPDATE 逻辑使用 `CASE WHEN` 确保不会用空值覆盖已有的有效数据
3. **兼容性**：修复后的代码与 `ColdStart` 服务保持一致的数据处理逻辑

## 部署建议

1. 合并此修复到 main 分支
2. 重新部署到 Railway
3. 运行 PrematchService 验证新数据完整性
4. 考虑为历史空数据编写补全脚本（可选）
