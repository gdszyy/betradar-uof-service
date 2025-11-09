package services

import (
	"database/sql"
	"fmt"
	"time"
)

// DataCleanupService 数据清理服务
type DataCleanupService struct {
	db     *sql.DB
	config CleanupConfig
}

// CleanupConfig 清理配置
type CleanupConfig struct {
	RetainDaysMessages  int // uof_messages 保留天数
	RetainDaysOdds      int // odds_changes, markets, odds 保留天数
	RetainDaysBets      int // bet_stops, bet_settlements, bet_cancels 保留天数
	RetainDaysLiveData  int // ld_events, ld_lineups 保留天数
	RetainDaysEvents    int // tracked_events, ld_matches 保留天数
}

// CleanupResult 清理结果
type CleanupResult struct {
	TableName    string
	DeletedRows  int64
	RetainedDays int
	Error        error
}

// NewDataCleanupService 创建数据清理服务
func NewDataCleanupService(db *sql.DB, config CleanupConfig) *DataCleanupService {
	return &DataCleanupService{
		db:     db,
		config: config,
	}
}

// ExecuteCleanup 执行数据清理
// 保留策略：
// - uof_messages: 保留 7 天（原始消息，占用空间最大）
// - odds_changes: 保留 7 天（赔率变化记录）
// - bet_stops: 保留 7 天
// - bet_settlements: 保留 7 天
// - bet_cancels: 保留 7 天
// - odds_history: 保留 7 天
// - tracked_events: 保留 30 天（赛事信息，需要更长时间）
// - markets: 保留 7 天（盘口数据）
// - odds: 保留 7 天（赔率详情）
// - ld_events: 保留 3 天（Live Data 事件）
// - ld_matches: 保留 30 天（比赛信息）
// - ld_lineups: 保留 7 天（阵容信息）
func (s *DataCleanupService) ExecuteCleanup() ([]CleanupResult, error) {
	results := []CleanupResult{}

	// 清理配置：表名 -> 保留天数（从配置读取）
	cleanupConfig := map[string]int{
		"uof_messages":    s.config.RetainDaysMessages,  // 原始消息（占用最大）
		"odds_changes":    s.config.RetainDaysOdds,      // 赔率变化
		"bet_stops":       s.config.RetainDaysBets,      // 投注停止
		"bet_settlements": s.config.RetainDaysBets,      // 投注结算
		"bet_cancels":     s.config.RetainDaysBets,      // 投注取消
		"odds_history":    s.config.RetainDaysOdds,      // 赔率历史
		"markets":         s.config.RetainDaysOdds,      // 盘口数据
		"odds":            s.config.RetainDaysOdds,      // 赔率详情
		"ld_events":       s.config.RetainDaysLiveData,  // Live Data 事件（更新频繁）
		"ld_lineups":      s.config.RetainDaysLiveData,  // 阵容信息
		"tracked_events":  s.config.RetainDaysEvents,    // 赛事信息（保留更长时间）
		"ld_matches":      s.config.RetainDaysEvents,    // 比赛信息（保留更长时间）
	}

	// 按表清理数据
	for tableName, retainDays := range cleanupConfig {
		result := s.cleanupTable(tableName, retainDays)
		results = append(results, result)
	}

	return results, nil
}

// cleanupTable 清理单个表的数据
func (s *DataCleanupService) cleanupTable(tableName string, retainDays int) CleanupResult {
	result := CleanupResult{
		TableName:    tableName,
		RetainedDays: retainDays,
	}

	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -retainDays)

	// 根据表名选择时间字段
	timeField := s.getTimeField(tableName)
	if timeField == "" {
		result.Error = fmt.Errorf("no time field found for table %s", tableName)
		return result
	}

	// 构建删除 SQL
	query := fmt.Sprintf("DELETE FROM %s WHERE %s < $1", tableName, timeField)

	// 执行删除
	res, err := s.db.Exec(query, cutoffTime)
	if err != nil {
		result.Error = fmt.Errorf("failed to delete from %s: %w", tableName, err)
		return result
	}

	// 获取删除的行数
	deletedRows, err := res.RowsAffected()
	if err != nil {
		result.Error = fmt.Errorf("failed to get rows affected: %w", err)
		return result
	}

	result.DeletedRows = deletedRows
	return result
}

// getTimeField 获取表的时间字段
func (s *DataCleanupService) getTimeField(tableName string) string {
	// 时间字段映射
	timeFields := map[string]string{
		"uof_messages":    "received_at",
		"odds_changes":    "created_at",
		"bet_stops":       "created_at",
		"bet_settlements": "created_at",
		"bet_cancels":     "created_at",
		"odds_history":    "created_at",
		"markets":         "updated_at",
		"odds":            "updated_at",
		"ld_events":       "created_at",
		"ld_lineups":      "created_at",
		"tracked_events":  "schedule_time", // 修正：使用 schedule_time 而不是 created_at
		"ld_matches":      "created_at",
	}

	return timeFields[tableName]
}

// GetTableSizes 获取所有表的大小（需要数据库权限）
func (s *DataCleanupService) GetTableSizes() (map[string]int64, error) {
	query := `
		SELECT 
			tablename,
			pg_total_relation_size(schemaname||'.'||tablename) AS size_bytes
		FROM pg_tables
		WHERE schemaname = 'public'
		ORDER BY size_bytes DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query table sizes: %w", err)
	}
	defer rows.Close()

	sizes := make(map[string]int64)
	for rows.Next() {
		var tableName string
		var sizeBytes int64
		if err := rows.Scan(&tableName, &sizeBytes); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		sizes[tableName] = sizeBytes
	}

	return sizes, nil
}

// GetTableRowCounts 获取所有表的行数
func (s *DataCleanupService) GetTableRowCounts() (map[string]int64, error) {
	tables := []string{
		"uof_messages", "odds_changes", "bet_stops", "bet_settlements", "bet_cancels",
		"odds_history", "markets", "odds", "ld_events", "ld_lineups",
		"tracked_events", "ld_matches",
	}

	counts := make(map[string]int64)
	for _, tableName := range tables {
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		var count int64
		if err := s.db.QueryRow(query).Scan(&count); err != nil {
			// 如果表不存在，跳过
			continue
		}
		counts[tableName] = count
	}

	return counts, nil
}

// ArchiveToFile 归档数据到文件（留存逻辑，暂不实现）
// 功能说明：
// 1. 导出即将删除的数据到 CSV/JSON 文件
// 2. 压缩文件（gzip）
// 3. 上传到 S3/OSS（可选）
//
// 示例实现：
// func (s *DataCleanupService) ArchiveToFile(tableName string, cutoffTime time.Time) error {
//     // 1. 查询需要归档的数据
//     query := fmt.Sprintf("SELECT * FROM %s WHERE %s < $1", tableName, timeField)
//     rows, err := s.db.Query(query, cutoffTime)
//     if err != nil {
//         return err
//     }
//     defer rows.Close()
//
//     // 2. 创建归档文件
//     filename := fmt.Sprintf("archive/%s_%s.csv.gz", tableName, cutoffTime.Format("20060102"))
//     file, err := os.Create(filename)
//     if err != nil {
//         return err
//     }
//     defer file.Close()
//
//     // 3. 写入数据（CSV 格式）
//     gzWriter := gzip.NewWriter(file)
//     defer gzWriter.Close()
//     csvWriter := csv.NewWriter(gzWriter)
//     defer csvWriter.Flush()
//
//     // 4. 遍历行并写入
//     for rows.Next() {
//         // ... 扫描行并写入 CSV
//     }
//
//     // 5. 上传到 S3（可选）
//     // s3Client.Upload(filename, ...)
//
//     return nil
// }

// CleanupOldEvents 清理已结束的赛事及其关联数据
// 功能说明：
// 1. 查找 status = 'ended' 且超过 N 天的赛事
// 2. 删除关联的 markets、odds 数据
// 3. 删除赛事本身
//
// 示例实现：
// func (s *DataCleanupService) CleanupOldEvents(retainDays int) error {
//     cutoffTime := time.Now().AddDate(0, 0, -retainDays)
//
//     // 1. 查找需要删除的赛事 ID
//     query := `
//         SELECT event_id FROM tracked_events
//         WHERE match_status = 'ended' AND updated_at < $1
//     `
//     rows, err := s.db.Query(query, cutoffTime)
//     if err != nil {
//         return err
//     }
//     defer rows.Close()
//
//     eventIDs := []string{}
//     for rows.Next() {
//         var eventID string
//         if err := rows.Scan(&eventID); err != nil {
//             return err
//         }
//         eventIDs = append(eventIDs, eventID)
//     }
//
//     // 2. 删除关联数据
//     for _, eventID := range eventIDs {
//         // 删除 markets
//         s.db.Exec("DELETE FROM markets WHERE event_id = $1", eventID)
//         // 删除 odds
//         s.db.Exec("DELETE FROM odds WHERE event_id = $1", eventID)
//         // 删除 odds_changes
//         s.db.Exec("DELETE FROM odds_changes WHERE event_id = $1", eventID)
//         // 删除赛事
//         s.db.Exec("DELETE FROM tracked_events WHERE event_id = $1", eventID)
//     }
//
//     return nil
// }
