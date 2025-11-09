package web

import (
	"encoding/json"
	"net/http"

	"uof-service/logger"
	"uof-service/services"
)

// MarketDescriptionsHandler Market Descriptions API 处理器
type MarketDescriptionsHandler struct {
	service *services.MarketDescriptionsService
}

// NewMarketDescriptionsHandler 创建处理器
func NewMarketDescriptionsHandler(service *services.MarketDescriptionsService) *MarketDescriptionsHandler {
	return &MarketDescriptionsHandler{
		service: service,
	}
}

// HandleGetStatus 获取服务状态
func (h *MarketDescriptionsHandler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.service.GetStatus()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"data":   status,
	})
}

// HandleForceRefresh 强制刷新 (从 API 重新加载)
func (h *MarketDescriptionsHandler) HandleForceRefresh(w http.ResponseWriter, r *http.Request) {
logger.Println("[API] Force refresh of market descriptions requested")
	
	h.service.ForceRefresh()
	
	logger.Println("[API] ✅ Market descriptions refresh initiated")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
	"status":  "ok",
	"message": "Market descriptions refresh initiated",
	})
}

// HandleBulkUpdate 批量更新存量数据
func (h *MarketDescriptionsHandler) HandleBulkUpdate(w http.ResponseWriter, r *http.Request) {
logger.Println("[API] Bulk update of existing markets/outcomes requested")
	
	updatedMarkets, updatedOutcomes, err := h.service.UpdateExistingMarkets()
	if err != nil {
		logger.Printf("[API] ⚠️  Failed to bulk update: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	
	logger.Printf("[API] ✅ Bulk update completed: %d markets, %d outcomes", updatedMarkets, updatedOutcomes)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
	"status": "ok",
	"data": map[string]interface{}{
	// "updated_markets":  updatedMarkets,
	// "updated_outcomes": updatedOutcomes,
	},
	})

}
