package web

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleLDConnect 连接到 LD 服务器
func (s *Server) handleLDConnect(w http.ResponseWriter, r *http.Request) {
	if s.ldClient == nil {
		http.Error(w, "LD client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	log.Println("[API] Attempting to connect to Live Data server...")
	
	go func() {
		if err := s.ldClient.Connect(); err != nil {
			log.Printf("[LD] ❌ Failed to connect: %v", err)
			if s.larkNotifier != nil {
				s.larkNotifier.NotifyError("Live Data Client", err.Error())
			}
		} else {
			log.Println("[LD] ✅ Live Data client connected")
			if s.larkNotifier != nil {
				s.larkNotifier.SendText("🟢 Live Data 客户端已连接")
			}
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "connecting",
		"message": "Connection attempt started. Check logs for results.",
	})
}

// handleLDDisconnect 断开 LD 连接
func (s *Server) handleLDDisconnect(w http.ResponseWriter, r *http.Request) {
	if s.ldClient == nil {
		http.Error(w, "LD client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if err := s.ldClient.Close(); err != nil {
		log.Printf("[LD] ❌ Failed to disconnect: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Println("[LD] ✅ Live Data client disconnected")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "disconnected",
		"message": "Successfully disconnected from Live Data server",
	})
}

// handleLDStatus 获取 LD 连接状态
func (s *Server) handleLDStatus(w http.ResponseWriter, r *http.Request) {
	if s.ldClient == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "not_initialized",
			"connected": false,
			"message":   "LD client not initialized",
		})
		return
	}
	
	// TODO: 添加实际的连接状态检查
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "initialized",
		"connected": false, // 需要从 ldClient 获取实际状态
		"message":   "LD client initialized but not connected (IP whitelist required)",
	})
}

