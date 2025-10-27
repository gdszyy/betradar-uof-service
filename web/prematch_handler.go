package web

import (
	"encoding/json"
	"log"
	"net/http"
	"uof-service/services"
)

// handleTriggerPrematchBooking 手动触发 pre-match 订阅
func (s *Server) handleTriggerPrematchBooking(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] 🔄 Received pre-match booking trigger request")
	
	// 创建 pre-match 服务
	prematchService := services.NewPrematchService(s.config, s.db)
	
	// 执行订阅
	result, err := prematchService.ExecutePrematchBooking()
	if err != nil {
		log.Printf("[API] ❌ Pre-match booking failed: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	
	log.Printf("[API] ✅ Pre-match booking completed: %d/%d successful", result.Success, result.Bookable)
	
	// 发送通知
	if result.Success > 0 {
		s.larkNotifier.NotifyPrematchBooking(result.TotalEvents, result.Bookable, result.Success, result.Failed)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"total_events":   result.TotalEvents,
		"bookable":       result.Bookable,
		"already_booked": result.AlreadyBooked,
		"success_count":  result.Success,
		"failed_count":   result.Failed,
	})
}

// handleGetPrematchEvents 获取 pre-match 赛事列表
func (s *Server) handleGetPrematchEvents(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting pre-match events...")
	
	// 查询数据库中的 pre-match 赛事 (schedule_time > now)
	query := `
		SELECT 
			event_id, sport_id, status, schedule_time,
			home_team_name, away_team_name,
			subscribed, message_count, last_message_at,
			created_at, updated_at
		FROM tracked_events
		WHERE schedule_time > NOW()
		ORDER BY schedule_time ASC
		LIMIT 1000
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		log.Printf("[API] Failed to query pre-match events: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to query database",
		})
		return
	}
	defer rows.Close()
	
	type PrematchEvent struct {
		EventID        string  `json:"event_id"`
		SportID        string  `json:"sport_id"`
		Status         string  `json:"status"`
		ScheduleTime   *string `json:"schedule_time"`
		HomeTeamName   *string `json:"home_team_name"`
		AwayTeamName   *string `json:"away_team_name"`
		Subscribed     bool    `json:"subscribed"`
		MessageCount   int     `json:"message_count"`
		LastMessageAt  *string `json:"last_message_at"`
		CreatedAt      string  `json:"created_at"`
		UpdatedAt      string  `json:"updated_at"`
	}
	
	var events []PrematchEvent
	
	for rows.Next() {
		var event PrematchEvent
		err := rows.Scan(
			&event.EventID,
			&event.SportID,
			&event.Status,
			&event.ScheduleTime,
			&event.HomeTeamName,
			&event.AwayTeamName,
			&event.Subscribed,
			&event.MessageCount,
			&event.LastMessageAt,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			log.Printf("[API] Failed to scan row: %v", err)
			continue
		}
		
		events = append(events, event)
	}
	
	log.Printf("[API] Found %d pre-match events", len(events))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(events),
		"events":  events,
	})
}

// handleGetPrematchStats 获取 pre-match 统计信息
func (s *Server) handleGetPrematchStats(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting pre-match stats...")
	
	type Stats struct {
		TotalEvents      int `json:"total_events"`
		SubscribedEvents int `json:"subscribed_events"`
		EventsWithOdds   int `json:"events_with_odds"`
	}
	
	var stats Stats
	
	// 查询总数
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM tracked_events WHERE schedule_time > NOW()
	`).Scan(&stats.TotalEvents)
	if err != nil {
		log.Printf("[API] Failed to query total events: %v", err)
	}
	
	// 查询已订阅数量
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM tracked_events 
		WHERE schedule_time > NOW() AND subscribed = true
	`).Scan(&stats.SubscribedEvents)
	if err != nil {
		log.Printf("[API] Failed to query subscribed events: %v", err)
	}
	
	// 查询有赔率的数量
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT event_id) FROM markets 
		WHERE event_id IN (
			SELECT event_id FROM tracked_events WHERE schedule_time > NOW()
		)
	`).Scan(&stats.EventsWithOdds)
	if err != nil {
		log.Printf("[API] Failed to query events with odds: %v", err)
	}
	
	log.Printf("[API] Pre-match stats: %d total, %d subscribed, %d with odds",
		stats.TotalEvents, stats.SubscribedEvents, stats.EventsWithOdds)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

