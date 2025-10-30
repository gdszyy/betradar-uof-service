package web

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"uof-service/services"
)

// MessageHistoryHandler 消息历史 API handler
type MessageHistoryHandler struct {
	service *services.MessageHistoryService
}

// NewMessageHistoryHandler 创建消息历史 handler
func NewMessageHistoryHandler(service *services.MessageHistoryService) *MessageHistoryHandler {
	return &MessageHistoryHandler{
		service: service,
	}
}

// GetEventMessages 获取比赛的最近消息
// GET /api/events/{event_id}/messages?limit=5
func (h *MessageHistoryHandler) GetEventMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	// 获取 limit 参数
	limit := 5
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// 获取消息历史
	response, err := h.service.GetEventMessages(eventID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回 JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRecentMessages 获取最近的 UOF 消息(不限定比赛)
// GET /api/messages/recent?limit=10&type=odds_change
func (h *MessageHistoryHandler) GetRecentMessages(w http.ResponseWriter, r *http.Request) {
	// 获取 limit 参数
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// 获取 type 参数
	messageType := r.URL.Query().Get("type")

	// 获取消息历史
	response, err := h.service.GetRecentMessages(limit, messageType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回 JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

