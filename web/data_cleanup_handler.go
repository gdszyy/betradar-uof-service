package web

import (
	"encoding/json"
	"net/http"
	
	"uof-service/services"
)

// handleGetTableStats 获取表统计信息
func (s *Server) handleGetTableStats(w http.ResponseWriter, r *http.Request) {
	dataCleanup := services.NewDataCleanupService(s.db)
	
	// 获取表行数
	rowCounts, err := dataCleanup.GetTableRowCounts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 获取表大小
	tableSizes, err := dataCleanup.GetTableSizes()
	if err != nil {
		// 如果没有权限查询表大小，只返回行数
		tableSizes = make(map[string]int64)
	}
	
	// 合并结果
	type TableStat struct {
		TableName string `json:"table_name"`
		RowCount  int64  `json:"row_count"`
		SizeBytes int64  `json:"size_bytes"`
		SizeMB    float64 `json:"size_mb"`
	}
	
	stats := []TableStat{}
	for tableName, rowCount := range rowCounts {
		sizeBytes := tableSizes[tableName]
		sizeMB := float64(sizeBytes) / 1024 / 1024
		stats = append(stats, TableStat{
			TableName: tableName,
			RowCount:  rowCount,
			SizeBytes: sizeBytes,
			SizeMB:    sizeMB,
		})
	}
	
	response := map[string]interface{}{
		"success": true,
		"stats":   stats,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleManualCleanup 手动触发数据清理
func (s *Server) handleManualCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	dataCleanup := services.NewDataCleanupService(s.db)
	
	// 执行清理
	results, err := dataCleanup.ExecuteCleanup()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 统计结果
	totalDeleted := int64(0)
	cleanupResults := []map[string]interface{}{}
	
	for _, result := range results {
		totalDeleted += result.DeletedRows
		cleanupResults = append(cleanupResults, map[string]interface{}{
			"table_name":    result.TableName,
			"deleted_rows":  result.DeletedRows,
			"retained_days": result.RetainedDays,
			"error":         result.Error,
		})
	}
	
	response := map[string]interface{}{
		"success":       true,
		"total_deleted": totalDeleted,
		"results":       cleanupResults,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

