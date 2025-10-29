package web

import (
	"encoding/json"
	"log"
	"net/http"
	"uof-service/services"
)

// SyncSubscriptionsHandler 同步订阅状态
func (s *Server) SyncSubscriptionsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	log.Println("[API] 🔄 Received subscription sync request")
	
	// 创建同步服务
	syncService := services.NewSubscriptionSyncService(
		s.db,
		s.config.APIBaseURL,
		s.config.BetradarAccessToken,
		s.config.SubscriptionSyncIntervalMinutes,
	)
	
	// 执行一次同步
	result, err := syncService.SyncOnce()
	if err != nil {
		log.Printf("[API] ❌ Sync failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Printf("[API] ✅ Sync completed: %d updated", result.UpdatedToTrue)
	
	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"total_booked":    result.TotalBooked,
		"updated_to_true": result.UpdatedToTrue,
		"not_found":       result.NotFound,
		"failed":          result.Failed,
		"message":         "Subscription sync completed successfully",
	})
}

