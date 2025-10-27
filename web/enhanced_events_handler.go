package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// EnhancedEvent 增强的赛事信息
type EnhancedEvent struct {
	// 基本信息
	EventID        string  `json:"event_id"`
	SRNID          *string `json:"srn_id"`
	SportID        string  `json:"sport_id"`
	Status         string  `json:"status"`
	ScheduleTime   *string `json:"schedule_time"`
	
	// 球队信息
	HomeTeamID     *string `json:"home_team_id"`
	HomeTeamName   *string `json:"home_team_name"`
	AwayTeamID     *string `json:"away_team_id"`
	AwayTeamName   *string `json:"away_team_name"`
	
	// 比分和状态
	HomeScore      *int    `json:"home_score"`
	AwayScore      *int    `json:"away_score"`
	MatchStatus    *string `json:"match_status"`
	MatchTime      *string `json:"match_time"`
	
	// 统计信息
	MessageCount   int     `json:"message_count"`
	LastMessageAt  *string `json:"last_message_at"`
	Subscribed     bool    `json:"subscribed"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	
	// 映射后的字段
	Sport              string `json:"sport"`
	SportName          string `json:"sport_name"`
	MatchStatusMapped  string `json:"match_status_mapped"`
	MatchStatusName    string `json:"match_status_name"`
	MatchTimeMapped    string `json:"match_time_mapped"`
	HomeTeamIDMapped   string `json:"home_team_id_mapped"`
	AwayTeamIDMapped   string `json:"away_team_id_mapped"`
	IsLive             bool   `json:"is_live"`
	IsEnded            bool   `json:"is_ended"`
	
	// 盘口信息
	Markets []MarketInfo `json:"markets"`
}

// MarketInfo 盘口信息
type MarketInfo struct {
	MarketID    string        `json:"market_id"`
	MarketName  string        `json:"market_name"`
	Specifiers  string        `json:"specifiers"`
	Status      string        `json:"status"`
	Outcomes    []OutcomeInfo `json:"outcomes"`
	UpdatedAt   string        `json:"updated_at"`
}

// OutcomeInfo 结果信息
type OutcomeInfo struct {
	OutcomeID string  `json:"outcome_id"`
	Name      string  `json:"name"`
	Odds      float64 `json:"odds"`
	Active    bool    `json:"active"`
}

// handleGetEnhancedEvents 获取增强的赛事信息
func (s *Server) handleGetEnhancedEvents(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting enhanced events with markets...")
	
	// 查询参数
	status := r.URL.Query().Get("status")
	subscribed := r.URL.Query().Get("subscribed")
	sportID := r.URL.Query().Get("sport_id")
	search := r.URL.Query().Get("search")
	limit := r.URL.Query().Get("limit")
	
	if limit == "" {
		limit = "100"
	}
	
	// 构建查询
	query := `
		SELECT 
			event_id, srn_id, sport_id, status, schedule_time,
			home_team_id, home_team_name, away_team_id, away_team_name,
			home_score, away_score, match_status, match_time,
			message_count, last_message_at, subscribed,
			created_at, updated_at
		FROM tracked_events
	`
	
	args := []interface{}{}
	whereClauses := []string{}
	
	// 添加 status 过滤
	if status != "" {
		whereClauses = append(whereClauses, "status = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, status)
	}
	
	// 添加 subscribed 过滤
	if subscribed != "" {
		subscribedBool := subscribed == "true"
		whereClauses = append(whereClauses, "subscribed = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, subscribedBool)
	}
	
	// 添加 sport_id 过滤
	if sportID != "" {
		whereClauses = append(whereClauses, "sport_id = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, sportID)
	}
	
	// 添加 search 过滤 (队伍名称)
	if search != "" {
		searchPattern := "%" + search + "%"
		whereClauses = append(whereClauses, "(home_team_name ILIKE $"+fmt.Sprintf("%d", len(args)+1)+" OR away_team_name ILIKE $"+fmt.Sprintf("%d", len(args)+2)+")")
		args = append(args, searchPattern, searchPattern)
	}
	
	// 组合 WHERE 子句
	if len(whereClauses) > 0 {
		query += " WHERE " + whereClauses[0]
		for i := 1; i < len(whereClauses); i++ {
			query += " AND " + whereClauses[i]
		}
	}
	
	// 添加排序和限制
	query += " ORDER BY last_message_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, limit)
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[API] Failed to query events: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var events []EnhancedEvent
	
	for rows.Next() {
		var event EnhancedEvent
		var scheduleTime, lastMessageAt sql.NullTime
		var srnID, homeTeamID, homeTeamName, awayTeamID, awayTeamName sql.NullString
		var homeScore, awayScore sql.NullInt64
		var matchStatus, matchTime sql.NullString
		
		err := rows.Scan(
			&event.EventID, &srnID, &event.SportID, &event.Status, &scheduleTime,
			&homeTeamID, &homeTeamName, &awayTeamID, &awayTeamName,
			&homeScore, &awayScore, &matchStatus, &matchTime,
			&event.MessageCount, &lastMessageAt, &event.Subscribed,
			&event.CreatedAt, &event.UpdatedAt,
		)
		
		if err != nil {
			log.Printf("[API] Failed to scan row: %v", err)
			continue
		}
		
		// 处理 NULL 值
		if srnID.Valid {
			event.SRNID = &srnID.String
		}
		if scheduleTime.Valid {
			t := scheduleTime.Time.Format("2006-01-02T15:04:05Z")
			event.ScheduleTime = &t
		}
		if homeTeamID.Valid {
			event.HomeTeamID = &homeTeamID.String
		}
		if homeTeamName.Valid {
			event.HomeTeamName = &homeTeamName.String
		}
		if awayTeamID.Valid {
			event.AwayTeamID = &awayTeamID.String
		}
		if awayTeamName.Valid {
			event.AwayTeamName = &awayTeamName.String
		}
		if homeScore.Valid {
			score := int(homeScore.Int64)
			event.HomeScore = &score
		}
		if awayScore.Valid {
			score := int(awayScore.Int64)
			event.AwayScore = &score
		}
		if matchStatus.Valid {
			event.MatchStatus = &matchStatus.String
		}
		if matchTime.Valid {
			event.MatchTime = &matchTime.String
		}
		if lastMessageAt.Valid {
			t := lastMessageAt.Time.Format("2006-01-02T15:04:05Z")
			event.LastMessageAt = &t
		}
		
		// 使用映射器转换数据
		event.Sport = s.srMapper.MapSport(event.SportID)
		event.SportName = s.srMapper.MapSportChinese(event.SportID)
		
		if event.MatchStatus != nil {
			event.MatchStatusMapped = s.srMapper.MapMatchStatus(*event.MatchStatus)
			event.MatchStatusName = s.srMapper.MapMatchStatusChinese(*event.MatchStatus)
			event.IsLive = s.srMapper.IsMatchLive(*event.MatchStatus)
			event.IsEnded = s.srMapper.IsMatchEnded(*event.MatchStatus)
		}
		
		if event.MatchTime != nil {
			event.MatchTimeMapped = s.srMapper.FormatMatchTime(*event.MatchTime)
		}
		
		if event.HomeTeamID != nil {
			event.HomeTeamIDMapped = s.srMapper.ExtractCompetitorIDFromURN(*event.HomeTeamID)
		}
		
		if event.AwayTeamID != nil {
			event.AwayTeamIDMapped = s.srMapper.ExtractCompetitorIDFromURN(*event.AwayTeamID)
		}
		
		// 获取盘口信息
		markets, err := s.getEventMarkets(event.EventID)
		if err != nil {
			log.Printf("[API] Failed to get markets for %s: %v", event.EventID, err)
			event.Markets = []MarketInfo{} // 空数组而不是 null
		} else {
			event.Markets = markets
		}
		
		events = append(events, event)
	}
	
	if err := rows.Err(); err != nil {
		log.Printf("[API] Error iterating rows: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Printf("[API] Returning %d enhanced events", len(events))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(events),
		"events":  events,
	})
}

// getEventMarkets 获取赛事的盘口信息
func (s *Server) getEventMarkets(eventID string) ([]MarketInfo, error) {
	query := `
		SELECT DISTINCT ON (market_id, specifiers)
			id, market_id, specifiers, status, updated_at
		FROM markets
		WHERE event_id = $1
		ORDER BY market_id, specifiers, updated_at DESC
	`
	
	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var markets []MarketInfo
	
	for rows.Next() {
		var market MarketInfo
		var marketPK int
		var specifiers sql.NullString
		
		err := rows.Scan(&marketPK, &market.MarketID, &specifiers, &market.Status, &market.UpdatedAt)
		if err != nil {
			log.Printf("[API] Failed to scan market: %v", err)
			continue
		}
		
		if specifiers.Valid {
			market.Specifiers = specifiers.String
		}
		
		// 获取市场名称 (简化版,可以后续从 market descriptions 获取)
		market.MarketName = s.getMarketName(market.MarketID)
		
		// 获取该盘口的赔率 (使用 marketPK)
		outcomes, err := s.getMarketOutcomes(marketPK)
		if err != nil {
			log.Printf("[API] Failed to get outcomes for market %s: %v", market.MarketID, err)
			market.Outcomes = []OutcomeInfo{}
		} else {
			market.Outcomes = outcomes
		}
		
		markets = append(markets, market)
	}
	
	return markets, nil
}

// getMarketOutcomes 获取盘口的赔率
func (s *Server) getMarketOutcomes(marketPK int) ([]OutcomeInfo, error) {
	query := `
		SELECT outcome_id, odds_value, active, updated_at
		FROM odds
		WHERE market_id = $1
		ORDER BY outcome_id
	`
	
	rows, err := s.db.Query(query, marketPK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var outcomes []OutcomeInfo
	
	for rows.Next() {
		var outcome OutcomeInfo
		var updatedAt string
		
		err := rows.Scan(&outcome.OutcomeID, &outcome.Odds, &outcome.Active, &updatedAt)
		if err != nil {
			log.Printf("[API] Failed to scan outcome: %v", err)
			continue
		}
		
		// 获取结果名称 (简化版)
		outcome.Name = s.getOutcomeName(outcome.OutcomeID)
		
		outcomes = append(outcomes, outcome)
	}
	
	return outcomes, nil
}

// getMarketName 获取市场名称 (简化版)
func (s *Server) getMarketName(marketID string) string {
	// 常见市场的映射
	marketNames := map[string]string{
		"1":   "1X2",
		"18":  "Total Goals",
		"52":  "Asian Handicap",
		"60":  "Correct Score",
		"10":  "Double Chance",
		"29":  "Both Teams to Score",
		"186": "Next Goal",
	}
	
	if name, ok := marketNames[marketID]; ok {
		return name
	}
	
	return "Market " + marketID
}

// getOutcomeName 获取结果名称 (简化版)
func (s *Server) getOutcomeName(outcomeID string) string {
	// 常见结果的映射
	outcomeNames := map[string]string{
		"1":  "Home",
		"2":  "Away",
		"3":  "Draw",
		"4":  "Over",
		"5":  "Under",
		"6":  "Yes",
		"7":  "No",
		"12": "Home/Draw",
		"13": "Home/Away",
		"23": "Draw/Away",
	}
	
	if name, ok := outcomeNames[outcomeID]; ok {
		return name
	}
	
	return "Outcome " + outcomeID
}

