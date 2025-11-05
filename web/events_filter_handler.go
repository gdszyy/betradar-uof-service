package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	
	"uof-service/services"
)

// handleGetEventsWithFilters 获取比赛列表(支持多种筛选)
// GET /api/events
func (s *Server) handleGetEventsWithFilters(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Getting events with filters...")
	
	// 解析查询参数
	filters := parseEventFilters(r)
	
	// 生成缓存键
	cacheKey := services.GenerateCacheKey("events_filter", filters)
	
	// 尝试从缓存获取
	if s.queryCache != nil {
		if cached, found := s.queryCache.Get(cacheKey); found {
			log.Printf("[API] Cache hit for events filter query")
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			json.NewEncoder(w).Encode(cached)
			return
		}
	}
	
	// 构建 SQL 查询
	query, args := buildEventFilterQuery(filters)
	
	// 查询总数
	countQuery, countArgs := buildEventCountQuery(filters)
	var totalCount int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		log.Printf("[API] Error counting events: %v", err)
		totalCount = 0
	}
	
	totalPages := (totalCount + filters.PageSize - 1) / filters.PageSize
	
	// 查询数据
	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[API] Error querying events: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query events: %v", err), http.StatusInternalServerError)
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
			&match.Attendance,
			&match.Sellout,
			&match.FeatureMatch,
			&match.LiveVideoAvailable,
			&match.LiveDataAvailable,
			&match.BroadcastsCount,
			&match.PopularityScore,
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

	// 构建响应
	response := map[string]interface{}{
		"success":     true,
		"count":       len(enhancedMatches),
		"total":       totalCount,
		"page":        filters.Page,
		"page_size":   filters.PageSize,
		"total_pages": totalPages,
		"filters":     filters.toMap(),
		"matches":     enhancedMatches,
	}
	
	// 缓存结果
	if s.queryCache != nil {
		s.queryCache.Set(cacheKey, response)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
			json.NewEncoder(w).Encode(response)
}

// EventFilters 事件筛选参数
type EventFilters struct {
	// 分页参数
	Page     int
	PageSize int
	
	// 状态筛选
	IsLive *bool // true=live, false=非live, nil=全部
	Status string // active, ended, etc.
	IncludeEnded bool // true=包含已结束的比赛, false=排除已结束的比赛(默认)
	
	// 体育类型筛选 (支持多选,逗号分隔)
	SportIDs []string
	
	// 开赛时间筛选 (左闭右闭)
	StartTimeFrom *time.Time
	StartTimeTo   *time.Time
	
	// 盘口组筛选
	MarketGroup string
	
	// 盘口类型筛选 (支持多选,逗号分隔)
	MarketIDs []string
	
	// 队伍筛选 (支持多选,逗号分隔)
	TeamIDs  []string
	TeamName string
	
	// 联赛筛选 (支持多选,逗号分隔)
	LeagueIDs  []string
	LeagueName string
	
	// 搜索
	Search string
	
	// 热门度筛选
	Popular      *bool  // true=只返回热门比赛, false=排除热门比赛, nil=全部
	SortBy       string // 排序字段: popularity, time, default
	MinPopularity float64 // 最小热门度评分 (0-100)
}

// parseEventFilters 解析查询参数
func parseEventFilters(r *http.Request) *EventFilters {
	filters := &EventFilters{
		Page:     1,
		PageSize: 100,
	}
	
	// 分页参数
	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			filters.Page = p
		}
	}
	
	if pageSizeParam := r.URL.Query().Get("page_size"); pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 500 {
			filters.PageSize = ps
		}
	}
	
	// 状态筛选
	if isLiveParam := r.URL.Query().Get("is_live"); isLiveParam != "" {
		isLive := isLiveParam == "true" || isLiveParam == "1"
		filters.IsLive = &isLive
	}
	
	filters.Status = r.URL.Query().Get("status")
	
	// include_ended 参数 (默认 false, 排除已结束的比赛)
	if includeEndedParam := r.URL.Query().Get("include_ended"); includeEndedParam != "" {
		filters.IncludeEnded = includeEndedParam == "true" || includeEndedParam == "1"
	}
	
	// 体育类型筛选 (支持多选,逗号分隔)
	if sportID := r.URL.Query().Get("sport_id"); sportID != "" {
		filters.SportIDs = strings.Split(sportID, ",")
		// 去除空格
		for i := range filters.SportIDs {
			filters.SportIDs[i] = strings.TrimSpace(filters.SportIDs[i])
		}
	}
	
	// 开赛时间筛选
	if startFrom := r.URL.Query().Get("start_time_from"); startFrom != "" {
		if t, err := parseDateTime(startFrom); err == nil {
			filters.StartTimeFrom = &t
		}
	}
	
	if startTo := r.URL.Query().Get("start_time_to"); startTo != "" {
		if t, err := parseDateTime(startTo); err == nil {
			filters.StartTimeTo = &t
		}
	}
	
	// 盘口组筛选
	filters.MarketGroup = r.URL.Query().Get("market_group")
	
	// 盘口类型筛选 (支持多选,逗号分隔)
	if marketID := r.URL.Query().Get("market_id"); marketID != "" {
		filters.MarketIDs = strings.Split(marketID, ",")
		for i := range filters.MarketIDs {
			filters.MarketIDs[i] = strings.TrimSpace(filters.MarketIDs[i])
		}
	}
	
	// 队伍筛选 (支持多选,逗号分隔)
	if teamID := r.URL.Query().Get("team_id"); teamID != "" {
		filters.TeamIDs = strings.Split(teamID, ",")
		for i := range filters.TeamIDs {
			filters.TeamIDs[i] = strings.TrimSpace(filters.TeamIDs[i])
		}
	}
	filters.TeamName = r.URL.Query().Get("team_name")
	
	// 联赛筛选 (支持多选,逗号分隔)
	if leagueID := r.URL.Query().Get("league_id"); leagueID != "" {
		filters.LeagueIDs = strings.Split(leagueID, ",")
		for i := range filters.LeagueIDs {
			filters.LeagueIDs[i] = strings.TrimSpace(filters.LeagueIDs[i])
		}
	}
	filters.LeagueName = r.URL.Query().Get("league_name")
	
	// 搜索
	filters.Search = r.URL.Query().Get("search")
	
	// 热门度筛选
	if popularParam := r.URL.Query().Get("popular"); popularParam != "" {
		popular := popularParam == "true" || popularParam == "1"
		filters.Popular = &popular
	}
	
	filters.SortBy = r.URL.Query().Get("sort_by")
	
	if minPopParam := r.URL.Query().Get("min_popularity"); minPopParam != "" {
		if minPop, err := strconv.ParseFloat(minPopParam, 64); err == nil && minPop >= 0 && minPop <= 100 {
			filters.MinPopularity = minPop
		}
	}
	
	return filters
}

// parseDateTime 解析日期时间 (支持日期或日期+时间)
func parseDateTime(s string) (time.Time, error) {
	// 尝试解析完整的日期时间: 2025-11-03T15:00:00Z
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	
	// 尝试解析日期时间: 2025-11-03 15:00:00
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t, nil
	}
	
	// 尝试解析日期: 2025-11-03 (默认为当天 00:00:00)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	
	return time.Time{}, fmt.Errorf("invalid datetime format: %s", s)
}

// buildEventFilterQuery 构建查询 SQL
func buildEventFilterQuery(filters *EventFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	// 基础查询
	query := `
		SELECT DISTINCT
			e.event_id, e.srn_id, e.sport_id, e.status, e.schedule_time,
			e.home_team_id, e.home_team_name, e.away_team_id, e.away_team_name,
			e.home_score, e.away_score, e.match_status, e.match_time,
			e.message_count, e.last_message_at, e.created_at, e.updated_at,
			e.attendance, e.sellout, e.feature_match, e.live_video_available,
			e.live_data_available, e.broadcasts_count, e.popularity_score
		FROM tracked_events e
	`
	
	// 是否需要 JOIN markets 表
	needMarketsJoin := len(filters.MarketIDs) > 0 // MarketGroup 暂时不支持
	
	if needMarketsJoin {
		query += " LEFT JOIN markets m ON e.event_id = m.event_id"
	}
	
	// 状态筛选
	if filters.IsLive != nil {
		if *filters.IsLive {
			// Live: status = 'active' AND match_status IS NOT NULL
			conditions = append(conditions, "e.status = 'active' AND e.match_status IS NOT NULL")
		} else {
			// 非 Live: status != 'active' OR match_status IS NULL
			conditions = append(conditions, "(e.status != 'active' OR e.match_status IS NULL)")
		}
	}
	
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}
	
	// 默认排除已结束的比赛 (除非明确请求包含)
	if !filters.IncludeEnded && filters.Status == "" {
		// 如果没有明确指定 status, 则排除 ended 状态
		conditions = append(conditions, "e.status != 'ended'")
	}
	
	// 体育类型筛选 (支持多选)
	if len(filters.SportIDs) > 0 {
		placeholders := []string{}
		for _, sportID := range filters.SportIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, sportID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("e.sport_id IN (%s)", strings.Join(placeholders, ", ")))
	}
	
	// 开赛时间筛选 (左闭右闭)
	if filters.StartTimeFrom != nil {
		conditions = append(conditions, fmt.Sprintf("e.schedule_time >= $%d", argIndex))
		args = append(args, filters.StartTimeFrom)
		argIndex++
	}
	
	if filters.StartTimeTo != nil {
		// 如果只有日期,设置为当天 23:59:59
		endTime := *filters.StartTimeTo
		if endTime.Hour() == 0 && endTime.Minute() == 0 && endTime.Second() == 0 {
			endTime = endTime.Add(24*time.Hour - time.Second)
		}
		conditions = append(conditions, fmt.Sprintf("e.schedule_time <= $%d", argIndex))
		args = append(args, endTime)
		argIndex++
	}
	
	// 盘口组筛选 (暂时禁用,因为 markets 表没有 groups 字段)
	// if filters.MarketGroup != "" {
	// 	conditions = append(conditions, fmt.Sprintf("m.groups LIKE $%d", argIndex))
	// 	args = append(args, "%"+filters.MarketGroup+"%")
	// 	argIndex++
	// }
	
	// 盘口类型筛选 (支持多选)
	if len(filters.MarketIDs) > 0 {
		placeholders := []string{}
		for _, marketID := range filters.MarketIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, marketID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("m.sr_market_id IN (%s)", strings.Join(placeholders, ", ")))
	}
	// 队伍 ID 筛选 (支持多选)
	if len(filters.TeamIDs) > 0 {
		placeholders := []string{}
		for _, teamID := range filters.TeamIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, teamID)
			argIndex++
		}
		inClause := strings.Join(placeholders, ", ")
		conditions = append(conditions, fmt.Sprintf("(e.home_team_id IN (%s) OR e.away_team_id IN (%s))", inClause, inClause))
	}
	if filters.TeamName != "" {
		conditions = append(conditions, fmt.Sprintf("(e.home_team_name ILIKE $%d OR e.away_team_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filters.TeamName+"%")
		argIndex++
	}
		// 联赛 ID 筛选 (从 srn_id 提取,支持多选)
	if len(filters.LeagueIDs) > 0 {
		leagueConditions := []string{}
		for _, leagueID := range filters.LeagueIDs {
			leagueConditions = append(leagueConditions, fmt.Sprintf("e.srn_id LIKE $%d", argIndex))
			args = append(args, "%:"+leagueID+":%")
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(leagueConditions, " OR ")))
	}
	
	if filters.LeagueName != "" {
		// 联赛名称搜索 (需要 JOIN 联赛表,暂时不支持)
		log.Printf("[API] Warning: league_name filter not yet supported")
	}
	
	// 搜索 (队伍名称或赛事 ID)
	if filters.Search != "" {
		searchCondition := fmt.Sprintf("(e.event_id ILIKE $%d OR e.home_team_name ILIKE $%d OR e.away_team_name ILIKE $%d)", argIndex, argIndex, argIndex)
		conditions = append(conditions, searchCondition)
		args = append(args, "%"+filters.Search+"%")
		argIndex++
	}
	
	// 热门度筛选
	if filters.Popular != nil {
		if *filters.Popular {
			// 只返回热门比赛: 焦点赛 OR 售罄 OR 转播数 > 0 OR 热门度评分 > 50
			conditions = append(conditions, "(e.feature_match = TRUE OR e.sellout = TRUE OR e.broadcasts_count > 0 OR e.popularity_score > 50)")
		} else {
			// 排除热门比赛
			conditions = append(conditions, "(e.feature_match = FALSE OR e.feature_match IS NULL) AND (e.sellout = FALSE OR e.sellout IS NULL) AND (e.broadcasts_count = 0 OR e.broadcasts_count IS NULL) AND (e.popularity_score <= 50 OR e.popularity_score IS NULL)")
		}
	}
	
	// 最小热门度评分
	if filters.MinPopularity > 0 {
		conditions = append(conditions, fmt.Sprintf("e.popularity_score >= $%d", argIndex))
		args = append(args, filters.MinPopularity)
		argIndex++
	}
	
	// 添加 WHERE 子句
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// 排序
	if filters.SortBy == "popularity" {
		// 按热门度排序 (热门度评分高的在前)
		query += " ORDER BY e.popularity_score DESC NULLS LAST, e.schedule_time DESC"
	} else if filters.SortBy == "time" {
		// 按时间排序
		query += " ORDER BY e.schedule_time DESC NULLS LAST, e.created_at DESC"
	} else {
		// 默认排序: 先按时间
		query += " ORDER BY e.schedule_time DESC NULLS LAST, e.created_at DESC"
	}
	
	// 分页
	offset := (filters.Page - 1) * filters.PageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filters.PageSize, offset)
	argIndex += 2 // 修正：增加 argIndex
	return query, args
}

// buildEventCountQuery 构建计数查询
func buildEventCountQuery(filters *EventFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	query := "SELECT COUNT(DISTINCT e.event_id) FROM tracked_events e"
	
	// 是否需要 JOIN markets 表
	needMarketsJoin := len(filters.MarketIDs) > 0 // MarketGroup 暂时不支持
	
	if needMarketsJoin {
		query += " LEFT JOIN markets m ON e.event_id = m.event_id"
	}
	
	// 复用筛选条件逻辑
	if filters.IsLive != nil {
		if *filters.IsLive {
			conditions = append(conditions, "e.status = 'active' AND e.match_status IS NOT NULL")
		} else {
			conditions = append(conditions, "(e.status != 'active' OR e.match_status IS NULL)")
		}
	}
	
	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", argIndex))
		args = append(args, filters.Status)
		argIndex++
	}
	
	// 默认排除已结束的比赛 (除非明确请求包含)
	if !filters.IncludeEnded && filters.Status == "" {
		conditions = append(conditions, "e.status != 'ended'")
	}
	
	// 体育类型筛选 (支持多选)
	if len(filters.SportIDs) > 0 {
		placeholders := []string{}
		for _, sportID := range filters.SportIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, sportID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("e.sport_id IN (%s)", strings.Join(placeholders, ", ")))
	}
	
	if filters.StartTimeFrom != nil {
		conditions = append(conditions, fmt.Sprintf("e.schedule_time >= $%d", argIndex))
		args = append(args, filters.StartTimeFrom)
		argIndex++
	}
	
	if filters.StartTimeTo != nil {
		endTime := *filters.StartTimeTo
		if endTime.Hour() == 0 && endTime.Minute() == 0 && endTime.Second() == 0 {
			endTime = endTime.Add(24*time.Hour - time.Second)
		}
		conditions = append(conditions, fmt.Sprintf("e.schedule_time <= $%d", argIndex))
		args = append(args, endTime)
		argIndex++
	}
	
	// 盘口组筛选 (暂时禁用,因为 markets 表没有 groups 字段)
	// if filters.MarketGroup != "" {
	// 	conditions = append(conditions, fmt.Sprintf("m.groups LIKE $%d", argIndex))
	// 	args = append(args, "%"+filters.MarketGroup+"%")
	// 	argIndex++
	// }
	
	// 盘口类型筛选 (支持多选)
	if len(filters.MarketIDs) > 0 {
		placeholders := []string{}
		for _, marketID := range filters.MarketIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, marketID)
			argIndex++
		}
		conditions = append(conditions, fmt.Sprintf("m.sr_market_id IN (%s)", strings.Join(placeholders, ", ")))
	}
	
	// 队伍 ID 筛选 (支持多选)
	if len(filters.TeamIDs) > 0 {
		placeholders := []string{}
		for _, teamID := range filters.TeamIDs {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, teamID)
			argIndex++
		}
		inClause := strings.Join(placeholders, ", ")
		conditions = append(conditions, fmt.Sprintf("(e.home_team_id IN (%s) OR e.away_team_id IN (%s))", inClause, inClause))
	}
	
	if filters.TeamName != "" {
		conditions = append(conditions, fmt.Sprintf("(e.home_team_name ILIKE $%d OR e.away_team_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filters.TeamName+"%")
		argIndex++
	}
	
	// 联赛 ID 筛选 (从 srn_id 提取,支持多选)
	if len(filters.LeagueIDs) > 0 {
		leagueConditions := []string{}
		for _, leagueID := range filters.LeagueIDs {
			leagueConditions = append(leagueConditions, fmt.Sprintf("e.srn_id LIKE $%d", argIndex))
			args = append(args, "%:"+leagueID+":%")
			argIndex++
		}
			conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(leagueConditions, " OR ")))
		}
		
		if filters.LeagueName != "" {
			// 联赛名称搜索 (需要 JOIN 联赛表,暂时不支持)
			log.Printf("[API] Warning: league_name filter not yet supported")
		}
		
		// 搜索 (队伍名称或赛事 ID)
		if filters.Search != "" {
		searchCondition := fmt.Sprintf("(e.event_id ILIKE $%d OR e.home_team_name ILIKE $%d OR e.away_team_name ILIKE $%d)", argIndex, argIndex, argIndex)
		conditions = append(conditions, searchCondition)
		args = append(args, "%"+filters.Search+"%")
		argIndex++
	}
	
	// 热门度筛选 (与 buildEventFilterQuery 保持一致)
	if filters.Popular != nil {
		if *filters.Popular {
			conditions = append(conditions, "(e.feature_match = TRUE OR e.sellout = TRUE OR e.broadcasts_count > 0 OR e.popularity_score > 50)")
		} else {
			conditions = append(conditions, "(e.feature_match = FALSE OR e.feature_match IS NULL) AND (e.sellout = FALSE OR e.sellout IS NULL) AND (e.broadcasts_count = 0 OR e.broadcasts_count IS NULL) AND (e.popularity_score <= 50 OR e.popularity_score IS NULL)")
		}
	}
	
	if filters.MinPopularity > 0 {
		conditions = append(conditions, fmt.Sprintf("e.popularity_score >= $%d", argIndex))
		args = append(args, filters.MinPopularity)
		argIndex++
	}
	
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	log.Printf("[DEBUG] SQL Query: %s", query)
	log.Printf("[DEBUG] SQL Args: %v", args)
	return query, args
}

// toMap 将筛选参数转换为 map (用于响应)
func (f *EventFilters) toMap() map[string]interface{} {
	m := make(map[string]interface{})
	
	if f.IsLive != nil {
		m["is_live"] = *f.IsLive
	}
	if f.Status != "" {
		m["status"] = f.Status
	}
	if len(f.SportIDs) > 0 {
		m["sport_id"] = strings.Join(f.SportIDs, ",")
	}
	if f.StartTimeFrom != nil {
		m["start_time_from"] = f.StartTimeFrom.Format(time.RFC3339)
	}
	if f.StartTimeTo != nil {
		m["start_time_to"] = f.StartTimeTo.Format(time.RFC3339)
	}
	if f.MarketGroup != "" {
		m["market_group"] = f.MarketGroup
	}
	if len(f.MarketIDs) > 0 {
		m["market_id"] = strings.Join(f.MarketIDs, ",")
	}
	if len(f.TeamIDs) > 0 {
		m["team_id"] = strings.Join(f.TeamIDs, ",")
	}
	if f.TeamName != "" {
		m["team_name"] = f.TeamName
	}
	if len(f.LeagueIDs) > 0 {
		m["league_id"] = strings.Join(f.LeagueIDs, ",")
	}
	if f.LeagueName != "" {
		m["league_name"] = f.LeagueName
	}
	if f.Search != "" {
		m["search"] = f.Search
	}
	
	return m
}

