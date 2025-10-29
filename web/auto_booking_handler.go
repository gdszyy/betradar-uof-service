package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"uof-service/services"
)

// AutoBookingHandler 自动订阅配置处理器
type AutoBookingHandler struct {
	controller *services.AutoBookingController
}

// NewAutoBookingHandler 创建自动订阅配置处理器
func NewAutoBookingHandler(controller *services.AutoBookingController) *AutoBookingHandler {
	return &AutoBookingHandler{
		controller: controller,
	}
}

// GetStatus 获取自动订阅状态
// GET /api/auto-booking/status
func (h *AutoBookingHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := h.controller.GetStatus()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// Enable 启用自动订阅
// POST /api/auto-booking/enable
func (h *AutoBookingHandler) Enable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.controller.Enable()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Auto-booking enabled",
		"data":    h.controller.GetStatus(),
	})
}

// Disable 禁用自动订阅
// POST /api/auto-booking/disable
func (h *AutoBookingHandler) Disable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.controller.Disable()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Auto-booking disabled",
		"data":    h.controller.GetStatus(),
	})
}

// SetInterval 设置自动订阅间隔
// POST /api/auto-booking/interval?minutes=30
func (h *AutoBookingHandler) SetInterval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	minutesStr := r.URL.Query().Get("minutes")
	if minutesStr == "" {
		http.Error(w, "Missing 'minutes' parameter", http.StatusBadRequest)
		return
	}

	minutes, err := strconv.Atoi(minutesStr)
	if err != nil || minutes < 1 {
		http.Error(w, "Invalid 'minutes' parameter (must be >= 1)", http.StatusBadRequest)
		return
	}

	h.controller.SetInterval(minutes)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Interval updated",
		"data":    h.controller.GetStatus(),
	})
}

