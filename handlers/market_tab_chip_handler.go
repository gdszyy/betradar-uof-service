package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"betradar-uof-service/services"
)

// MarketTabChipHandler handles HTTP requests for market tab and chip operations
type MarketTabChipHandler struct {
	service *services.MarketTabChipService
}

// NewMarketTabChipHandler creates a new instance of MarketTabChipHandler
func NewMarketTabChipHandler(service *services.MarketTabChipService) *MarketTabChipHandler {
	return &MarketTabChipHandler{
		service: service,
	}
}

// GetMarketsByTabChip retrieves markets for a specific tab and chip
// GET /api/v1/events/{eventId}/markets?tab={tabId}&chip={chipId}
func (h *MarketTabChipHandler) GetMarketsByTabChip(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimPrefix(r.URL.Path, "/api/v1/events/")
	eventID = strings.Split(eventID, "/")[0]

	tabID := r.URL.Query().Get("tab")
	chipID := r.URL.Query().Get("chip")

	if eventID == "" || tabID == "" {
		http.Error(w, "Missing required parameters: eventId and tab", http.StatusBadRequest)
		return
	}

	markets, err := h.service.GetMarketsByTabChip(eventID, tabID, chipID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id": eventID,
		"tab_id":   tabID,
		"chip_id":  chipID,
		"markets":  markets,
		"count":    len(markets),
	})
}

// GetTabsForEvent retrieves all tabs available for an event
// GET /api/v1/events/{eventId}/tabs
func (h *MarketTabChipHandler) GetTabsForEvent(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimPrefix(r.URL.Path, "/api/v1/events/")
	eventID = strings.Split(eventID, "/")[0]

	if eventID == "" {
		http.Error(w, "Missing required parameter: eventId", http.StatusBadRequest)
		return
	}

	tabs, err := h.service.GetTabsForEvent(eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id": eventID,
		"tabs":     tabs,
		"count":    len(tabs),
	})
}

// GetChipsForTab retrieves all chips for a specific tab
// GET /api/v1/tabs/{tabId}/chips
func (h *MarketTabChipHandler) GetChipsForTab(w http.ResponseWriter, r *http.Request) {
	tabID := strings.TrimPrefix(r.URL.Path, "/api/v1/tabs/")
	tabID = strings.Split(tabID, "/")[0]

	if tabID == "" {
		http.Error(w, "Missing required parameter: tabId", http.StatusBadRequest)
		return
	}

	chips, err := h.service.GetChipsForTab(tabID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tab_id": tabID,
		"chips":  chips,
		"count":  len(chips),
	})
}

// GetMarketCardData retrieves complete market card data for an event
// GET /api/v1/events/{eventId}/market-cards
func (h *MarketTabChipHandler) GetMarketCardData(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimPrefix(r.URL.Path, "/api/v1/events/")
	eventID = strings.Split(eventID, "/")[0]

	if eventID == "" {
		http.Error(w, "Missing required parameter: eventId", http.StatusBadRequest)
		return
	}

	// Get tabs for event
	tabs, err := h.service.GetTabsForEvent(eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Build response with tabs and markets grouped by tab
	marketsByTab := make(map[string]interface{})
	chipsByTab := make(map[string]interface{})

	for _, tab := range tabs {
		// Get markets for this tab
		markets, err := h.service.GetMarketsByTabChip(eventID, tab.ID, "")
		if err != nil {
			continue
		}

		// Get chips for this tab
		chips, err := h.service.GetChipsForTab(tab.ID)
		if err != nil {
			continue
		}

		marketsByTab[tab.ID] = markets
		chipsByTab[tab.ID] = chips
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id":    eventID,
		"tabs":        tabs,
		"markets":     marketsByTab,
		"chips":       chipsByTab,
		"tab_count":   len(tabs),
	})
}

// HealthCheck returns the health status of the service
// GET /api/v1/health
func (h *MarketTabChipHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"service": "market-tab-chip",
	})
}
