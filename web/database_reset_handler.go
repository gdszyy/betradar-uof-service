package web

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleResetDatabase 清空数据库所有数据（保留表结构）
func (s *Server) handleResetDatabase(w http.ResponseWriter, r *http.Request) {
	log.Println("[DatabaseReset] Starting database reset...")
	
	// 获取确认参数
	confirm := r.URL.Query().Get("confirm")
	if confirm != "yes" {
		http.Error(w, "Missing confirmation. Add ?confirm=yes to proceed", http.StatusBadRequest)
		return
	}
	
	// 按照依赖关系顺序删除数据
	tables := []string{
		"odds",                  // 赔率数据（依赖 markets）
		"markets",               // 盘口数据（依赖 odds_changes）
		"bet_settlements",       // 结算数据
		"bet_stops",             // 停止投注数据
		"odds_changes",          // 赔率变化数据
		"ld_lineups",            // 阵容数据
		"ld_events",             // Live Data 事件
		"ld_matches",            // Live Data 比赛
		"uof_messages",          // 原始消息
		"tracked_events",        // 跟踪的赛事
		"srn_mapping",           // SRN 映射
	}
	
	deletedCounts := make(map[string]int64)
	totalDeleted := int64(0)
	
	for _, table := range tables {
		result, err := s.db.Exec("DELETE FROM " + table)
		if err != nil {
			log.Printf("[DatabaseReset] ❌ Failed to delete from %s: %v", table, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		count, _ := result.RowsAffected()
		deletedCounts[table] = count
		totalDeleted += count
		
		log.Printf("[DatabaseReset] ✅ Deleted %d rows from %s", count, table)
	}
	
	// 重置序列（如果有的话）
	sequences := []string{
		"uof_messages_id_seq",
		"tracked_events_id_seq",
		"odds_changes_id_seq",
		"bet_stops_id_seq",
		"bet_settlements_id_seq",
		"markets_id_seq",
		"odds_id_seq",
		"ld_events_id_seq",
		"ld_matches_id_seq",
		"ld_lineups_id_seq",
		"srn_mapping_id_seq",
	}
	
	for _, seq := range sequences {
		_, err := s.db.Exec("ALTER SEQUENCE IF EXISTS " + seq + " RESTART WITH 1")
		if err != nil {
			log.Printf("[DatabaseReset] ⚠️  Failed to reset sequence %s: %v", seq, err)
			// 不中断，继续执行
		}
	}
	
	log.Printf("[DatabaseReset] ✅ Database reset completed. Total deleted: %d rows", totalDeleted)
	
	// 发送 Lark 通知
	if s.larkNotifier != nil {
		s.larkNotifier.NotifyDatabaseReset(deletedCounts, totalDeleted)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"total_deleted":  totalDeleted,
		"deleted_counts": deletedCounts,
		"message":        "Database reset completed successfully",
	})
}

