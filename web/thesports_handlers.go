package web

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleTheSportsConnect 连接到 The Sports
func (s *Server) handleTheSportsConnect(w http.ResponseWriter, r *http.Request) {
	if s.theSportsClient == nil {
		http.Error(w, "The Sports client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if s.theSportsClient.IsConnected() {
		http.Error(w, "Already connected", http.StatusBadRequest)
		return
	}
	
	go func() {
		if err := s.theSportsClient.Connect(); err != nil {
			log.Printf("[API] Failed to connect to The Sports: %v", err)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "connecting",
		"message": "The Sports connection initiated",
	})
}

// handleTheSportsDisconnect 断开 The Sports 连接
func (s *Server) handleTheSportsDisconnect(w http.ResponseWriter, r *http.Request) {
	if s.theSportsClient == nil {
		http.Error(w, "The Sports client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if !s.theSportsClient.IsConnected() {
		http.Error(w, "Not connected", http.StatusBadRequest)
		return
	}
	
	if err := s.theSportsClient.Disconnect(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "disconnected",
		"message": "The Sports disconnected successfully",
	})
}

// handleTheSportsStatus 获取 The Sports 连接状态
func (s *Server) handleTheSportsStatus(w http.ResponseWriter, r *http.Request) {
	if s.theSportsClient == nil {
		http.Error(w, "The Sports client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	connected := s.theSportsClient.IsConnected()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"connected": connected,
		"status":    map[string]interface{}{
			"connected": connected,
		},
	})
}

// handleTheSportsSubscribeMatch 订阅比赛
func (s *Server) handleTheSportsSubscribeMatch(w http.ResponseWriter, r *http.Request) {
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
	
	if s.theSportsClient == nil {
		http.Error(w, "The Sports client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if err := s.theSportsClient.SubscribeMatch(req.MatchID); err != nil {
		log.Printf("[API] Failed to subscribe match %s: %v", req.MatchID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"match_id": req.MatchID,
		"message":  "Match subscribed successfully",
	})
}

// handleTheSportsUnsubscribeMatch 取消订阅比赛
func (s *Server) handleTheSportsUnsubscribeMatch(w http.ResponseWriter, r *http.Request) {
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
	
	if s.theSportsClient == nil {
		http.Error(w, "The Sports client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if err := s.theSportsClient.UnsubscribeMatch(req.MatchID); err != nil {
		log.Printf("[API] Failed to unsubscribe match %s: %v", req.MatchID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"match_id": req.MatchID,
		"message":  "Match unsubscribed successfully",
	})
}

// handleTheSportsGetTodayMatches 获取今日比赛
func (s *Server) handleTheSportsGetTodayMatches(w http.ResponseWriter, r *http.Request) {
	if s.theSportsClient == nil {
		http.Error(w, "The Sports client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	matches, err := s.theSportsClient.GetTodayMatches()
	if err != nil {
		log.Printf("[API] Failed to get today matches: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"count":   len(matches),
		"matches": matches,
	})
}

// handleTheSportsGetLiveMatches 获取直播比赛
func (s *Server) handleTheSportsGetLiveMatches(w http.ResponseWriter, r *http.Request) {
	if s.theSportsClient == nil {
		http.Error(w, "The Sports client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	matches, err := s.theSportsClient.GetLiveMatches()
	if err != nil {
		log.Printf("[API] Failed to get live matches: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"count":   len(matches),
		"matches": matches,
	})
}

