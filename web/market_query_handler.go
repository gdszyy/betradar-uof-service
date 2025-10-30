package web

import (
	"encoding/json"
	"net/http"
	"uof-service/services"

	"github.com/gorilla/mux"
)

type MarketQueryHandler struct {
	service *services.MarketQueryService
}

func NewMarketQueryHandler(service *services.MarketQueryService) *MarketQueryHandler {
	return &MarketQueryHandler{service: service}
}

func (h *MarketQueryHandler) GetEventMarkets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}

	markets, err := h.service.GetEventMarkets(eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"event_id":      eventID,
		"total_markets": len(markets),
		"markets":       markets,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

