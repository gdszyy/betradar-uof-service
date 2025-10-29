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
		s.config.AccessToken,
		s.config.APIBaseURL,
		s.config.SubscriptionSyncIntervalMinutes,
	)
	
	// 启动服务并执行一次同步
	err := syncService.Start()
	if err != nil {
		log.Printf("[API] ❌ Failed to start sync service: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// 等待一小段时间让同步完成
	// 注意: Start() 是异步的,这里简化处理
	result := map[string]interface{}{
		"success": true,
		"message": "Subscription sync started",
	}
	log.Println("[API] ✅ Subscription sync service started")
	
	// 返回结果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

