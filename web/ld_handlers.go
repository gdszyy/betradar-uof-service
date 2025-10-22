package web

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// handleLDSubscribeMatch 订阅比赛
func (s *Server) handleLDSubscribeMatch(w http.ResponseWriter, r *http.Request) {
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
	
	if s.ldClient == nil {
		http.Error(w, "LD client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if err := s.ldClient.SubscribeMatch(req.MatchID); err != nil {
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

// handleLDUnsubscribeMatch 取消订阅比赛
func (s *Server) handleLDUnsubscribeMatch(w http.ResponseWriter, r *http.Request) {
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
	
	if s.ldClient == nil {
		http.Error(w, "LD client not initialized", http.StatusServiceUnavailable)
		return
	}
	
	if err := s.ldClient.UnsubscribeMatch(req.MatchID); err != nil {
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

// handleLDGetMatches 获取比赛列表
func (s *Server) handleLDGetMatches(w http.ResponseWriter, r *http.Request) {
	// 查询参数
	status := r.URL.Query().Get("status")
	limit := r.URL.Query().Get("limit")
	
	if limit == "" {
		limit = "50"
	}
	
	query := `
		SELECT match_id, sport_id, t1_name, t2_name, match_status,
		       match_time, t1_score, t2_score, match_date, start_time,
		       subscribed, last_event_at, created_at, updated_at
		FROM ld_matches
	`
	
	args := []interface{}{}
	
	if status != "" {
		query += " WHERE match_status = $1"
		args = append(args, status)
		query += " ORDER BY last_event_at DESC LIMIT $2"
		args = append(args, limit)
	} else {
		query += " ORDER BY last_event_at DESC LIMIT $1"
		args = append(args, limit)
	}
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[API] Failed to query matches: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	type Match struct {
		MatchID      string     `json:"match_id"`
		SportID      int        `json:"sport_id"`
		T1Name       string     `json:"t1_name"`
		T2Name       string     `json:"t2_name"`
		MatchStatus  string     `json:"match_status"`
		MatchTime    string     `json:"match_time"`
		T1Score      int        `json:"t1_score"`
		T2Score      int        `json:"t2_score"`
		MatchDate    string     `json:"match_date"`
		StartTime    string     `json:"start_time"`
		Subscribed   bool       `json:"subscribed"`
		LastEventAt  *time.Time `json:"last_event_at"`
		CreatedAt    time.Time  `json:"created_at"`
		UpdatedAt    time.Time  `json:"updated_at"`
	}
	
	matches := []Match{}
	
	for rows.Next() {
		var m Match
		err := rows.Scan(
			&m.MatchID, &m.SportID, &m.T1Name, &m.T2Name, &m.MatchStatus,
			&m.MatchTime, &m.T1Score, &m.T2Score, &m.MatchDate, &m.StartTime,
			&m.Subscribed, &m.LastEventAt, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			log.Printf("[API] Failed to scan match: %v", err)
			continue
		}
		matches = append(matches, m)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"count":   len(matches),
		"matches": matches,
	})
}

// handleLDGetEvents 获取事件列表
func (s *Server) handleLDGetEvents(w http.ResponseWriter, r *http.Request) {
	matchID := r.URL.Query().Get("match_id")
	limit := r.URL.Query().Get("limit")
	importantOnly := r.URL.Query().Get("important") == "true"
	
	if limit == "" {
		limit = "100"
	}
	
	query := `
		SELECT uuid, event_id, match_id, sport_id, type, type_name,
		       info, side, mtime, stime, match_status,
		       t1_score, t2_score, player1, player2, extra_info,
		       is_important, created_at
		FROM ld_events
		WHERE 1=1
	`
	
	args := []interface{}{}
	argCount := 0
	
	if matchID != "" {
		argCount++
		query += " AND match_id = $" + string(rune('0'+argCount))
		args = append(args, matchID)
	}
	
	if importantOnly {
		query += " AND is_important = true"
	}
	
	query += " ORDER BY stime DESC LIMIT $" + string(rune('0'+argCount+1))
	args = append(args, limit)
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[API] Failed to query events: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	type Event struct {
		UUID        string    `json:"uuid"`
		EventID     string    `json:"event_id"`
		MatchID     string    `json:"match_id"`
		SportID     int       `json:"sport_id"`
		Type        int       `json:"type"`
		TypeName    string    `json:"type_name"`
		Info        string    `json:"info"`
		Side        string    `json:"side"`
		MTime       string    `json:"mtime"`
		STime       int64     `json:"stime"`
		MatchStatus string    `json:"match_status"`
		T1Score     int       `json:"t1_score"`
		T2Score     int       `json:"t2_score"`
		Player1     string    `json:"player1"`
		Player2     string    `json:"player2"`
		ExtraInfo   string    `json:"extra_info"`
		IsImportant bool      `json:"is_important"`
		CreatedAt   time.Time `json:"created_at"`
	}
	
	events := []Event{}
	
	for rows.Next() {
		var e Event
		err := rows.Scan(
			&e.UUID, &e.EventID, &e.MatchID, &e.SportID, &e.Type, &e.TypeName,
			&e.Info, &e.Side, &e.MTime, &e.STime, &e.MatchStatus,
			&e.T1Score, &e.T2Score, &e.Player1, &e.Player2, &e.ExtraInfo,
			&e.IsImportant, &e.CreatedAt,
		)
		if err != nil {
			log.Printf("[API] Failed to scan event: %v", err)
			continue
		}
		events = append(events, e)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"count":  len(events),
		"events": events,
	})
}

