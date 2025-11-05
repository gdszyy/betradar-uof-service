package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	
	"uof-service/services"
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
	MarketID       string        `json:"sr_market_id"`
	MarketName     string        `json:"market_name"`
	Specifiers     string        `json:"specifiers,omitempty"`
	Status         string        `json:"status"`
	ProducerID     int           `json:"producer_id"`
	Outcomes       []OutcomeInfo `json:"outcomes"`
	OutcomesCount  int           `json:"outcomes_count"`
	UpdatedAt      string        `json:"updated_at"`
}

// OutcomeInfo 结果信息
type OutcomeInfo struct {
	OutcomeID   string  `json:"outcome_id"`
		Name        string  `json:"name"`
	OutcomeName string  `json:"outcome_name"` // 新增字段
	Odds        float64 `json:"odds"`
	Probability float64 `json:"probability"`
	Active      bool    `json:"active"`
}

// handleGetEnhancedEvents 获取增强的赛事信息
func (s *Server) handleGetEnhancedEvents(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting enhanced events with markets...")
	
	// 查询参数
	status := r.URL.Query().Get("status")
	subscribed := r.URL.Query().Get("subscribed")
	sportID := r.URL.Query().Get("sport_id")
	search := r.URL.Query().Get("search")
	producer := r.URL.Query().Get("producer")
	isLive := r.URL.Query().Get("is_live")
	isEnded := r.URL.Query().Get("is_ended")
	hasMarkets := r.URL.Query().Get("has_markets")
	
	page := 1
	pageSize := 100
	
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}
	
	if pageSizeParam := r.URL.Query().Get("page_size"); pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 500 {
			pageSize = ps
		}
	} else if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 500 {
			pageSize = l
		}
	}
	
		// 构建 SQL 查询
		args := []interface{}{}
		whereClauses := []string{}
		
		// 添加 status 过滤
		if status != "" {
			whereClauses = append(whereClauses, "te.status = $"+fmt.Sprintf("%d", len(args)+1))
			args = append(args, status)
		}
		
		// 添加 subscribed 过滤
		if subscribed != "" {
			subscribedBool := subscribed == "true"
			whereClauses = append(whereClauses, "te.subscribed = $"+fmt.Sprintf("%d", len(args)+1))
			args = append(args, subscribedBool)
		}
		
			// 添加 sport_id 过滤
			if sportID != "" {
				whereClauses = append(whereClauses, "te.sport_id::text = $"+fmt.Sprintf("%d", len(args)+1))
				args = append(args, sportID)
			}
		
		// 添加 search 过滤 (event_id 精确匹配或队伍名称模糊匹配)
		if search != "" {
			searchPattern := "%" + search + "%"
			whereClauses = append(whereClauses, "(te.event_id = $"+fmt.Sprintf("%d", len(args)+1)+" OR te.home_team_name ILIKE $"+fmt.Sprintf("%d", len(args)+2)+" OR te.away_team_name ILIKE $"+fmt.Sprintf("%d", len(args)+3)+")")
			args = append(args, search, searchPattern, searchPattern)
		}
		
		// 添加 is_ended 过滤 (排除已结束的比赛)
		if isEnded != "" {
			if isEnded == "false" {
				// 排除已结束的比赛 (status 不是 ended, closed, cancelled, abandoned)
				whereClauses = append(whereClauses, "te.status NOT IN ('ended', 'closed', 'cancelled', 'abandoned')")
			} else if isEnded == "true" {
				// 只返回已结束的比赛
				whereClauses = append(whereClauses, "te.status IN ('ended', 'closed', 'cancelled', 'abandoned')")
			}
		}
		
	// 添加 is_live 过滤
	if isLive == "true" {
		// 只返回 live 的比赛 (status = 'live')
		whereClauses = append(whereClauses, "te.status = 'live'")
	}
	
	// 添加 producer 过滤
	if producer != "" {
		// 通过 markets 表过滤 producer_id (使用 defensive cast 避免类型不匹配)
		if producerID, err := strconv.Atoi(producer); err == nil {
			whereClauses = append(whereClauses, "EXISTS (SELECT 1 FROM markets m WHERE m.event_id::text = te.event_id AND m.producer_id = $"+fmt.Sprintf("%d", len(args)+1)+")")
			args = append(args, producerID)
		}
	}
	
	// 添加 has_markets 过滤
	if hasMarkets == "true" {
		// 只返回有 markets 数据的比赛 (使用 defensive cast 避免类型不匹配)
		whereClauses = append(whereClauses, "EXISTS (SELECT 1 FROM markets m WHERE m.event_id::text = te.event_id)")
	}
		
		// 构建 WHERE 子句
		whereClause := ""
		if len(whereClauses) > 0 {
			whereClause = " WHERE " + whereClauses[0]
			for i := 1; i < len(whereClauses); i++ {
				whereClause += " AND " + whereClauses[i]
			}
		}
		
		// 使用 LEFT JOIN 从 markets 表获取有数据的比赛
		// WHERE 子句必须在 GROUP BY 之前
		query := `
			SELECT 
				te.event_id, te.srn_id, te.sport_id, te.status, te.schedule_time,
				te.home_team_id, te.home_team_name, te.away_team_id, te.away_team_name,
				te.home_score, te.away_score, te.match_status, te.match_time,
				te.message_count, te.last_message_at, te.subscribed,
				te.created_at, te.updated_at,
				COALESCE(MAX(m.updated_at), te.last_message_at) as last_update
			FROM tracked_events te
			LEFT JOIN markets m ON m.event_id::text = te.event_id
		` + whereClause + `
			GROUP BY te.event_id, te.srn_id, te.sport_id, te.status, te.schedule_time,
				te.home_team_id, te.home_team_name, te.away_team_id, te.away_team_name,
				te.home_score, te.away_score, te.match_status, te.match_time,
				te.message_count, te.last_message_at, te.subscribed,
				te.created_at, te.updated_at
		`
	
		// 添加排序和限制 (支持 page/page_size)
		query += " ORDER BY last_update DESC NULLS LAST, te.event_id"
		
		limit := pageSize
		offset := (page - 1) * pageSize
		
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
		args = append(args, limit, offset)
	
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
		
		var lastUpdate sql.NullTime
		err := rows.Scan(
			&event.EventID, &srnID, &event.SportID, &event.Status, &scheduleTime,
			&homeTeamID, &homeTeamName, &awayTeamID, &awayTeamName,
			&homeScore, &awayScore, &matchStatus, &matchTime,
			&event.MessageCount, &lastMessageAt, &event.Subscribed,
			&event.CreatedAt, &event.UpdatedAt,
			&lastUpdate, // last_update 字段
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
		
		// 解引用 HomeTeamName 和 AwayTeamName
		localHomeTeamName := ""
		if event.HomeTeamName != nil {
			localHomeTeamName = *event.HomeTeamName
		}
		localAwayTeamName := ""
		if event.AwayTeamName != nil {
			localAwayTeamName = *event.AwayTeamName
		}
		
		// 获取盘口信息 (按 producer 过滤)
		markets, err := s.getEventMarketsWithProducer(event.EventID, producer, localHomeTeamName, localAwayTeamName)
			if err != nil {
				log.Printf("[API] Failed to get markets for %s: %v", event.EventID, err)
				event.Markets = []MarketInfo{} // 空数组而不是 null
			} else {
				// 确保不为 nil，即使没有 markets 也返回空数组
				if markets == nil {
					event.Markets = []MarketInfo{}
				} else {
					event.Markets = markets
				}
			}
			
			// 如果 has_markets=true，过滤掉没有 markets 的比赛
			if hasMarkets == "true" && len(event.Markets) == 0 {
				continue
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
	// 传入空字符串作为默认值
	return s.getEventMarketsWithProducer(eventID, "", "", "")
}

// getEventMarketsWithProducer 获取赛事的盘口信息 (按 producer 过滤)
func (s *Server) getEventMarketsWithProducer(eventID string, producer string, homeTeamName string, awayTeamName string) ([]MarketInfo, error) {
	query := `
		SELECT DISTINCT ON (sr_market_id, specifiers)
			id, sr_market_id, specifiers, status, producer_id, updated_at
		FROM markets
		WHERE event_id = $1
	`
	
	args := []interface{}{eventID}
	
	// 添加 producer 过滤
	if producer != "" {
		query += " AND producer_id = $2"
		args = append(args, producer)
	}
	
	query += " ORDER BY sr_market_id, specifiers, updated_at DESC"
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// 初始化为空数组，而不是 nil
	markets := make([]MarketInfo, 0)
	
	for rows.Next() {
		var market MarketInfo
		var marketPK int
		var specifiers sql.NullString
		
		var producerID sql.NullInt64
		
		err := rows.Scan(&marketPK, &market.MarketID, &specifiers, &market.Status, &producerID, &market.UpdatedAt)
		if err != nil {
			log.Printf("[API] Failed to scan market: %v", err)
			continue
		}
		
		if specifiers.Valid {
			market.Specifiers = specifiers.String
		}
		
		if producerID.Valid {
			market.ProducerID = int(producerID.Int64)
		}
		
	// 获取市场名称 (简化版,可以后续从 market descriptions 获取)
				market.MarketName = s.getMarketName(market.MarketID, homeTeamName, awayTeamName, market.Specifiers)
			
				// 获取该盘口的赔率 (使用 marketPK)
				outcomes, err := s.getMarketOutcomes(marketPK, market.MarketID, homeTeamName, awayTeamName, market.Specifiers)
		if err != nil {
			log.Printf("[API] Failed to get outcomes for market %s: %v", market.MarketID, err)
			market.Outcomes = []OutcomeInfo{}
		} else {
			market.Outcomes = outcomes
			market.OutcomesCount = len(outcomes)
		}
		
		markets = append(markets, market)
	}
	
	return markets, nil
}

// getMarketOutcomes 获取盘口的赔率
func (s *Server) getMarketOutcomes(marketPK int, marketID string, homeTeamName string, awayTeamName string, specifiers string) ([]OutcomeInfo, error) {
	query := `
		SELECT outcome_id, odds_value, probability, active, updated_at
		FROM odds
		WHERE market_id = $1
		ORDER BY outcome_id
	`
	
	rows, err := s.db.Query(query, marketPK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	// 初始化为空数组，而不是 nil
	outcomes := make([]OutcomeInfo, 0)
	
	for rows.Next() {
		var outcome OutcomeInfo
		var updatedAt string
		
		var probability sql.NullFloat64
		err := rows.Scan(&outcome.OutcomeID, &outcome.Odds, &probability, &outcome.Active, &updatedAt)
		if probability.Valid {
			outcome.Probability = probability.Float64
		}
		if err != nil {
			log.Printf("[API] Failed to scan outcome: %v", err)
			continue
		}
		
		// 获取结果名称 (简化版)
			outcome.OutcomeName = s.getOutcomeName(marketID, outcome.OutcomeID, homeTeamName, awayTeamName, specifiers)
		outcomes = append(outcomes, outcome)
	}
	
	return outcomes, nil
}

// getMarketName 获取市场名称
func (s *Server) getMarketName(marketID string, homeTeamName string, awayTeamName string, specifiers string) string {
	// 优先使用 Market Descriptions Service
	if s.marketDescService != nil {
		ctx := services.ReplacementContext{
			HomeTeamName: homeTeamName,
			AwayTeamName: awayTeamName,
			Specifiers:   specifiers,
		}
			name := s.marketDescService.GetMarketName(marketID, specifiers, &ctx)
		// 如果不是默认的 "Market X" 格式,说明找到了
		if name != "Market "+marketID {
			return name
		}
	}
	
	// Fallback: 常见市场的硬编码映射
	marketNames := map[string]string{
		"1":   "1X2",
		"10":  "Double Chance",
		"18":  "Total Goals",
		"29":  "Both Teams to Score",
		"52":  "Asian Handicap",
		"60":  "Correct Score",
		"186": "Next Goal",
		"219": "Total Goals (1st Half)",
		"26":  "Odd/Even",
		"14":  "1st Half 1X2",
		"16":  "Draw No Bet",
		"47":  "Total Goals (2nd Half)",
		"45":  "1st Half Asian Handicap",
		"68":  "1st Half Double Chance",
		"74":  "1st Half Draw No Bet",
		"223": "Total Corners",
		"237": "Total Cards",
	}
	
	if name, ok := marketNames[marketID]; ok {
		return name
	}
	
	return "Market " + marketID
}

// getOutcomeName 获取结果名称
func (s *Server) getOutcomeName(marketID string, outcomeID string, homeTeamName string, awayTeamName string, specifiers string) string {
	// 优先使用 Market Descriptions Service
	if s.marketDescService != nil {
		ctx := services.ReplacementContext{
			HomeTeamName: homeTeamName,
			AwayTeamName: awayTeamName,
			Specifiers:   specifiers,
		}
			name := s.marketDescService.GetOutcomeName(marketID, outcomeID, specifiers, &ctx)
		// 如果不是默认的 "Outcome X" 格式,说明找到了
		if name != "Outcome "+outcomeID {
			return name
		}
	}
	
	// Fallback: 常见结果的硬编码映射
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

