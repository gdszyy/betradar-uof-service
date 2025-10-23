package web

import (
	"encoding/json"
	"net/http"
)

// handleGetSubscriptions 获取所有订阅
func (s *Server) handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	if s.subscriptionManager == nil {
		http.Error(w, "Subscription manager not available", http.StatusServiceUnavailable)
		return
	}
	
	subscriptions := s.subscriptionManager.GetSubscriptions()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "success",
		"count":         len(subscriptions),
		"subscriptions": subscriptions,
	})
}

// handleGetSubscriptionStats 获取订阅统计
func (s *Server) handleGetSubscriptionStats(w http.ResponseWriter, r *http.Request) {
	if s.subscriptionManager == nil {
		http.Error(w, "Subscription manager not available", http.StatusServiceUnavailable)
		return
	}
	
	stats := s.subscriptionManager.GetSubscriptionStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"stats":  stats,
	})
}

// handleUnsubscribeMatch 取消订阅单个比赛
func (s *Server) handleUnsubscribeMatch(w http.ResponseWriter, r *http.Request) {
	if s.subscriptionManager == nil {
		http.Error(w, "Subscription manager not available", http.StatusServiceUnavailable)
		return
	}
	
	// 解析请求
	var req struct {
		MatchID string `json:"match_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if req.MatchID == "" {
		http.Error(w, "match_id is required", http.StatusBadRequest)
		return
	}
	
	// 取消订阅
	if err := s.subscriptionManager.UnsubscribeMatch(req.MatchID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Match unsubscribed successfully",
		"match_id": req.MatchID,
	})
}

// handleUnsubscribeMatches 批量取消订阅
func (s *Server) handleUnsubscribeMatches(w http.ResponseWriter, r *http.Request) {
	if s.subscriptionManager == nil {
		http.Error(w, "Subscription manager not available", http.StatusServiceUnavailable)
		return
	}
	
	// 解析请求
	var req struct {
		MatchIDs []string `json:"match_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if len(req.MatchIDs) == 0 {
		http.Error(w, "match_ids is required", http.StatusBadRequest)
		return
	}
	
	// 批量取消订阅
	if err := s.subscriptionManager.UnsubscribeMatches(req.MatchIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Matches unsubscribed successfully",
		"count":   len(req.MatchIDs),
	})
}

// handleCleanupEndedMatches 手动触发清理已结束的比赛
func (s *Server) handleCleanupEndedMatches(w http.ResponseWriter, r *http.Request) {
	if s.subscriptionManager == nil {
		http.Error(w, "Subscription manager not available", http.StatusServiceUnavailable)
		return
	}
	
	// 获取所有订阅
	subscriptions := s.subscriptionManager.GetSubscriptions()
	
	// 找出已结束的比赛
	var endedMatchIDs []string
	for _, sub := range subscriptions {
		if sub.Status == "ended" || sub.Status == "closed" {
			endedMatchIDs = append(endedMatchIDs, sub.MatchID)
		}
	}
	
	if len(endedMatchIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "No ended matches to cleanup",
			"count":   0,
		})
		return
	}
	
	// 批量取消订阅
	if err := s.subscriptionManager.UnsubscribeMatches(endedMatchIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Ended matches cleaned up successfully",
		"count":   len(endedMatchIDs),
	})
}

