package web

import (
	"encoding/json"
	"net/http"
	
	"uof-service/services"
)

// ProducerHandler 处理 Producer 相关的 API 请求
type ProducerHandler struct {
	monitor *services.ProducerMonitor
}

// NewProducerHandler 创建 Producer 处理器
func NewProducerHandler(monitor *services.ProducerMonitor) *ProducerHandler {
	return &ProducerHandler{
		monitor: monitor,
	}
}

// HandleGetStatus 获取所有 Producer 的健康状态
func (h *ProducerHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.monitor.GetProducerStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"producers": statuses,
	})
}

// HandleGetBetAcceptance 检查是否可以接受投注
func (h *ProducerHandler) HandleGetBetAcceptance(w http.ResponseWriter, r *http.Request) {
	canAccept, reason := h.monitor.CanAcceptBets()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"can_accept_bets": canAccept,
		"reason":          reason,
	})
}

