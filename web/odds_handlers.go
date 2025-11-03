package web

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	
	"github.com/gorilla/mux"
	"uof-service/services"
)

// handleGetEventMarkets 获取比赛的所有盘口
func (s *Server) handleGetEventMarkets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	
	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	
	log.Printf("[API] Getting markets for event: %s", eventID)
	
	oddsParser := services.NewOddsParser(s.db, s.marketDescService)
	markets, err := oddsParser.GetEventMarkets(eventID)
	if err != nil {
		log.Printf("[API] Error querying markets: %v", err)
		http.Error(w, "Failed to query markets", http.StatusInternalServerError)
		return
	}
	
	if markets == nil {
		markets = []services.OddsMarketInfo{}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"event_id":  eventID,
		"count":     len(markets),
		"markets":   markets,
	})
}

// handleGetMarketOdds 获取盘口的当前赔率
func (s *Server) handleGetMarketOdds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	marketID := vars["market_id"]
	
	if eventID == "" || marketID == "" {
		http.Error(w, "event_id and market_id are required", http.StatusBadRequest)
		return
	}
	
	log.Printf("[API] Getting odds for event: %s, market: %s", eventID, marketID)
	
	oddsParser := services.NewOddsParser(s.db, s.marketDescService)
	odds, err := oddsParser.GetMarketOdds(eventID, marketID)
	if err != nil {
		log.Printf("[API] Error querying odds: %v", err)
		http.Error(w, "Failed to query odds", http.StatusInternalServerError)
		return
	}
	
	if odds == nil {
		odds = []services.OddsDetail{}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"event_id":  eventID,
		"market_id": marketID,
		"count":     len(odds),
		"odds":      odds,
	})
}

// handleGetOddsHistory 获取赔率变化历史
func (s *Server) handleGetOddsHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	marketID := vars["market_id"]
	outcomeID := vars["outcome_id"]
	
	if eventID == "" || marketID == "" || outcomeID == "" {
		http.Error(w, "event_id, market_id and outcome_id are required", http.StatusBadRequest)
		return
	}
	
	// 获取查询参数
	limitParam := r.URL.Query().Get("limit")
	limit := 50 // 默认 50 条
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
			if limit > 200 {
				limit = 200 // 最多 200 条
			}
		}
	}
	
	log.Printf("[API] Getting odds history for event: %s, market: %s, outcome: %s, limit: %d", 
		eventID, marketID, outcomeID, limit)
	
	oddsParser := services.NewOddsParser(s.db, s.marketDescService)
	history, err := oddsParser.GetOddsHistory(eventID, marketID, outcomeID, limit)
	if err != nil {
		log.Printf("[API] Error querying odds history: %v", err)
		http.Error(w, "Failed to query odds history", http.StatusInternalServerError)
		return
	}
	
	if history == nil {
		history = []services.OddsHistoryInfo{}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"event_id":   eventID,
		"market_id":  marketID,
		"outcome_id": outcomeID,
		"count":      len(history),
		"history":    history,
	})
}

// handleGetAllBookedMarketsOdds 获取所有已订阅比赛的盘口和赔率
func (s *Server) handleGetAllBookedMarketsOdds(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting all booked matches markets and odds...")
	
	// 1. 查询所有 active 状态的比赛
	query := `
		SELECT DISTINCT event_id 
		FROM tracked_events 
		WHERE status = 'active'
		ORDER BY event_id
		LIMIT 100
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("[API] Error querying active events: %v", err)
		http.Error(w, "Failed to query active events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var eventIDs []string
	for rows.Next() {
		var eventID string
		if err := rows.Scan(&eventID); err != nil {
			continue
		}
		eventIDs = append(eventIDs, eventID)
	}
	
	if len(eventIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"count":   0,
			"events":  []interface{}{},
		})
		return
	}
	
	log.Printf("[API] Found %d active events", len(eventIDs))
	
	// 2. 获取每个比赛的盘口和赔率
	oddsParser := services.NewOddsParser(s.db, s.marketDescService)
	var eventsData []map[string]interface{}
	
	for _, eventID := range eventIDs {
		// 获取比赛的盘口
		markets, err := oddsParser.GetEventMarkets(eventID)
		if err != nil {
			log.Printf("[API] Error getting markets for %s: %v", eventID, err)
			continue
		}
		
		// 获取每个盘口的赔率
		var marketsWithOdds []map[string]interface{}
		for _, market := range markets {
			odds, err := oddsParser.GetMarketOdds(eventID, market.SrMarketID)
			if err != nil {
				log.Printf("[API] Error getting odds for market %s: %v", market.SrMarketID, err)
				continue
			}
			
			marketsWithOdds = append(marketsWithOdds, map[string]interface{}{
				"sr_market_id":   market.SrMarketID,
				"market_type": market.MarketType,
				"market_name": market.MarketName,
				"specifiers":  market.Specifiers,
				"status":      market.Status,
				"odds":        odds,
				"updated_at":  market.UpdatedAt,
			})
		}
		
		if len(marketsWithOdds) > 0 {
			eventsData = append(eventsData, map[string]interface{}{
				"event_id":     eventID,
				"markets_count": len(marketsWithOdds),
				"markets":      marketsWithOdds,
			})
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(eventsData),
		"events":  eventsData,
	})
}

