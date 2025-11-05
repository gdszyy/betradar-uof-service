package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
	
	"github.com/gorilla/mux"
)

// MatchDetail 比赛详情结构
type MatchDetail struct {
	EventID        string    `json:"event_id"`
	SRNID          *string   `json:"srn_id"`
	SportID        *string   `json:"sport_id"`
	Status         string    `json:"status"`
	ScheduleTime   *string   `json:"schedule_time"`
	HomeTeamID     *string   `json:"home_team_id"`
	HomeTeamName   *string   `json:"home_team_name"`
	AwayTeamID     *string   `json:"away_team_id"`
	AwayTeamName   *string   `json:"away_team_name"`
	HomeScore      *int      `json:"home_score"`
	AwayScore      *int      `json:"away_score"`
	MatchStatus    *string   `json:"match_status"`
	MatchTime      *string   `json:"match_time"`
	MessageCount   int       `json:"message_count"`
	LastMessageAt  *string   `json:"last_message_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	
	// 热门度相关字段
	Attendance          *int     `json:"attendance,omitempty"`           // 到场人数
	Sellout             *bool    `json:"sellout,omitempty"`              // 是否售罄
	FeatureMatch        *bool    `json:"feature_match,omitempty"`        // 是否焦点赛
	LiveVideoAvailable  *bool    `json:"live_video_available,omitempty"` // 是否提供直播
	LiveDataAvailable   *bool    `json:"live_data_available,omitempty"`  // 是否提供实时数据
	BroadcastsCount     *int     `json:"broadcasts_count,omitempty"`     // 转播平台数量
	PopularityScore     *float64 `json:"popularity_score,omitempty"`     // 热门度评分 (0-100)
}

// handleGetMatchDetail 获取单个比赛的详细信息
func (s *Server) handleGetMatchDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	
	if eventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	
	log.Printf("[API] Getting match detail for: %s", eventID)
	
	query := `
		SELECT 
			event_id, srn_id, sport_id, status, schedule_time,
			home_team_id, home_team_name, away_team_id, away_team_name,
			home_score, away_score, match_status, match_time,
			message_count, last_message_at, created_at, updated_at
		FROM tracked_events
		WHERE event_id = $1
	`
	
	var match MatchDetail
	err := s.db.QueryRow(query, eventID).Scan(
		&match.EventID,
		&match.SRNID,
		&match.SportID,
		&match.Status,
		&match.ScheduleTime,
		&match.HomeTeamID,
		&match.HomeTeamName,
		&match.AwayTeamID,
		&match.AwayTeamName,
		&match.HomeScore,
		&match.AwayScore,
		&match.MatchStatus,
		&match.MatchTime,
		&match.MessageCount,
		&match.LastMessageAt,
		&match.CreatedAt,
		&match.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Match not found", http.StatusNotFound)
		return
	}
	
	if err != nil {
		log.Printf("[API] Error querying match detail: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query match: %v", err), http.StatusInternalServerError)
		return
	}
	
	// 使用 SR 映射器转换数据
	enhancedMatch := MapMatchDetail(match, s.srMapper)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"match":   enhancedMatch,
	})
}

// handleGetLiveMatches 获取所有进行中的比赛 (分页)
func (s *Server) handleGetLiveMatches(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting live matches...")
	
	// 获取分页参数
	pageParam := r.URL.Query().Get("page")
	pageSizeParam := r.URL.Query().Get("page_size")
	
	page := 1 // 默认第 1 页
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	
	pageSize := 100 // 默认每页 100 条
	if pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 500 {
			pageSize = ps
		}
	}
	
	offset := (page - 1) * pageSize
	
	// 查询总数
	countQuery := `
		SELECT COUNT(*)
		FROM tracked_events
		WHERE status = 'active'
		AND match_status IS NOT NULL
	`
	
	var totalCount int
	if err := s.db.QueryRow(countQuery).Scan(&totalCount); err != nil {
		log.Printf("[API] Error counting live matches: %v", err)
		totalCount = 0
	}
	
	totalPages := (totalCount + pageSize - 1) / pageSize
	
	// 查询当前页数据
	query := `
		SELECT 
			event_id, srn_id, sport_id, status, schedule_time,
			home_team_id, home_team_name, away_team_id, away_team_name,
			home_score, away_score, match_status, match_time,
			message_count, last_message_at, created_at, updated_at
		FROM tracked_events
		WHERE status = 'active'
		AND match_status IS NOT NULL
		ORDER BY schedule_time DESC NULLS LAST, created_at DESC
		LIMIT $1 OFFSET $2
	`
	
	rows, err := s.db.Query(query, pageSize, offset)
	if err != nil {
		log.Printf("[API] Error querying live matches: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query matches: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var matches []MatchDetail
	for rows.Next() {
		var match MatchDetail
		err := rows.Scan(
			&match.EventID,
			&match.SRNID,
			&match.SportID,
			&match.Status,
			&match.ScheduleTime,
			&match.HomeTeamID,
			&match.HomeTeamName,
			&match.AwayTeamID,
			&match.AwayTeamName,
			&match.HomeScore,
			&match.AwayScore,
			&match.MatchStatus,
			&match.MatchTime,
			&match.MessageCount,
			&match.LastMessageAt,
			&match.CreatedAt,
			&match.UpdatedAt,
		)
		if err != nil {
			log.Printf("[API] Error scanning match: %v", err)
			continue
		}
		matches = append(matches, match)
	}
	
	if matches == nil {
		matches = []MatchDetail{}
	}
	
	// 使用 SR 映射器转换数据
	enhancedMatches := MapMatchList(matches, s.srMapper)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"count":       len(enhancedMatches),
		"total":       totalCount,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"matches":     enhancedMatches,
	})
}

// handleGetUpcomingMatches 获取即将开始的比赛 (分页)
func (s *Server) handleGetUpcomingMatches(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting upcoming matches...")
	
	// 获取查询参数
	hoursParam := r.URL.Query().Get("hours")
	hours := 24 // 默认 24 小时
	if hoursParam != "" {
		if h, err := strconv.Atoi(hoursParam); err == nil && h > 0 {
			hours = h
		}
	}
	
	// 获取分页参数
	pageParam := r.URL.Query().Get("page")
	pageSizeParam := r.URL.Query().Get("page_size")
	
	page := 1 // 默认第 1 页
	if pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	
	pageSize := 100 // 默认每页 100 条
	if pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 500 {
			pageSize = ps
		}
	}
	
	offset := (page - 1) * pageSize
	
	// 查询总数
	countQuery := `
		SELECT COUNT(*)
		FROM tracked_events
		WHERE schedule_time IS NOT NULL
		AND schedule_time > NOW()
		AND schedule_time < NOW() + INTERVAL '1 hour' * $1
	`
	
	var totalCount int
	if err := s.db.QueryRow(countQuery, hours).Scan(&totalCount); err != nil {
		log.Printf("[API] Error counting upcoming matches: %v", err)
		totalCount = 0
	}
	
	totalPages := (totalCount + pageSize - 1) / pageSize
	
	// 查询当前页数据
	query := `
		SELECT 
			event_id, srn_id, sport_id, status, schedule_time,
			home_team_id, home_team_name, away_team_id, away_team_name,
			home_score, away_score, match_status, match_time,
			message_count, last_message_at, created_at, updated_at
		FROM tracked_events
		WHERE schedule_time IS NOT NULL
		AND schedule_time > NOW()
		AND schedule_time < NOW() + INTERVAL '1 hour' * $1
		ORDER BY schedule_time ASC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := s.db.Query(query, hours, pageSize, offset)
	if err != nil {
		log.Printf("[API] Error querying upcoming matches: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query matches: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var matches []MatchDetail
	for rows.Next() {
		var match MatchDetail
		err := rows.Scan(
			&match.EventID,
			&match.SRNID,
			&match.SportID,
			&match.Status,
			&match.ScheduleTime,
			&match.HomeTeamID,
			&match.HomeTeamName,
			&match.AwayTeamID,
			&match.AwayTeamName,
			&match.HomeScore,
			&match.AwayScore,
			&match.MatchStatus,
			&match.MatchTime,
			&match.MessageCount,
			&match.LastMessageAt,
			&match.CreatedAt,
			&match.UpdatedAt,
		)
		if err != nil {
			log.Printf("[API] Error scanning match: %v", err)
			continue
		}
		matches = append(matches, match)
	}
	
	if matches == nil {
		matches = []MatchDetail{}
	}
	
	// 使用 SR 映射器转换数据
	enhancedMatches := MapMatchList(matches, s.srMapper)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"count":       len(enhancedMatches),
		"total":       totalCount,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"hours":       hours,
		"matches":     enhancedMatches,
	})
}

// handleGetMatchesByStatus 按状态获取比赛
func (s *Server) handleGetMatchesByStatus(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}
	
	log.Printf("[API] Getting matches by status: %s", status)
	
	query := `
		SELECT 
			event_id, srn_id, sport_id, status, schedule_time,
			home_team_id, home_team_name, away_team_id, away_team_name,
			home_score, away_score, match_status, match_time,
			message_count, last_message_at, created_at, updated_at
		FROM tracked_events
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 100
	`
	
	rows, err := s.db.Query(query, status)
	if err != nil {
		log.Printf("[API] Error querying matches by status: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query matches: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var matches []MatchDetail
	for rows.Next() {
		var match MatchDetail
		err := rows.Scan(
			&match.EventID,
			&match.SRNID,
			&match.SportID,
			&match.Status,
			&match.ScheduleTime,
			&match.HomeTeamID,
			&match.HomeTeamName,
			&match.AwayTeamID,
			&match.AwayTeamName,
			&match.HomeScore,
			&match.AwayScore,
			&match.MatchStatus,
			&match.MatchTime,
			&match.MessageCount,
			&match.LastMessageAt,
			&match.CreatedAt,
			&match.UpdatedAt,
		)
		if err != nil {
			log.Printf("[API] Error scanning match: %v", err)
			continue
		}
		matches = append(matches, match)
	}
	
	if matches == nil {
		matches = []MatchDetail{}
	}
	
	// 使用 SR 映射器转换数据
	enhancedMatches := MapMatchList(matches, s.srMapper)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  status,
		"count":   len(enhancedMatches),
		"matches": enhancedMatches,
	})
}

// handleSearchMatches 搜索比赛
func (s *Server) handleSearchMatches(w http.ResponseWriter, r *http.Request) {
	keyword := r.URL.Query().Get("q")
	if keyword == "" {
		http.Error(w, "Search keyword is required", http.StatusBadRequest)
		return
	}
	
	log.Printf("[API] Searching matches with keyword: %s", keyword)
	
	query := `
		SELECT 
			event_id, srn_id, sport_id, status, schedule_time,
			home_team_id, home_team_name, away_team_id, away_team_name,
			home_score, away_score, match_status, match_time,
			message_count, last_message_at, created_at, updated_at
		FROM tracked_events
		WHERE 
			home_team_name ILIKE '%' || $1 || '%'
			OR away_team_name ILIKE '%' || $1 || '%'
			OR event_id ILIKE '%' || $1 || '%'
		ORDER BY created_at DESC
		LIMIT 50
	`
	
	rows, err := s.db.Query(query, keyword)
	if err != nil {
		log.Printf("[API] Error searching matches: %v", err)
		http.Error(w, fmt.Sprintf("Failed to search matches: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var matches []MatchDetail
	for rows.Next() {
		var match MatchDetail
		err := rows.Scan(
			&match.EventID,
			&match.SRNID,
			&match.SportID,
			&match.Status,
			&match.ScheduleTime,
			&match.HomeTeamID,
			&match.HomeTeamName,
			&match.AwayTeamID,
			&match.AwayTeamName,
			&match.HomeScore,
			&match.AwayScore,
			&match.MatchStatus,
			&match.MatchTime,
			&match.MessageCount,
			&match.LastMessageAt,
			&match.CreatedAt,
			&match.UpdatedAt,
		)
		if err != nil {
			log.Printf("[API] Error scanning match: %v", err)
			continue
		}
		matches = append(matches, match)
	}
	
	if matches == nil {
		matches = []MatchDetail{}
	}
	
	// 使用 SR 映射器转换数据
	enhancedMatches := MapMatchList(matches, s.srMapper)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"keyword": keyword,
		"count":   len(enhancedMatches),
		"matches": enhancedMatches,
	})
}

