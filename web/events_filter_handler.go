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
	
	// 体育类型筛选
	SportID string
	
	// 开赛时间筛选 (左闭右闭)
	StartTimeFrom *time.Time
	StartTimeTo   *time.Time
	
	// 盘口组筛选
	MarketGroup string
	
	// 盘口类型筛选
	MarketID string
	
	// 队伍筛选
	TeamID   string
	TeamName string
	
	// 联赛筛选
	LeagueID   string
	LeagueName string
	
	// 搜索
	Search string
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
	
	// 体育类型筛选
	filters.SportID = r.URL.Query().Get("sport_id")
	
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
	
	// 盘口类型筛选
	filters.MarketID = r.URL.Query().Get("market_id")
	
	// 队伍筛选
	filters.TeamID = r.URL.Query().Get("team_id")
	filters.TeamName = r.URL.Query().Get("team_name")
	
	// 联赛筛选
	filters.LeagueID = r.URL.Query().Get("league_id")
	filters.LeagueName = r.URL.Query().Get("league_name")
	
	// 搜索
	filters.Search = r.URL.Query().Get("search")
	
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
			e.message_count, e.last_message_at, e.created_at, e.updated_at
		FROM tracked_events e
	`
	
	// 是否需要 JOIN markets 表
	needMarketsJoin := filters.MarketID != "" // MarketGroup 暂时不支持
	
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
	
	// 体育类型筛选
	if filters.SportID != "" {
		conditions = append(conditions, fmt.Sprintf("e.sport_id = $%d", argIndex))
		args = append(args, filters.SportID)
		argIndex++
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
	
	// 盘口类型筛选
	if filters.MarketID != "" {
		conditions = append(conditions, fmt.Sprintf("m.sr_market_id = $%d", argIndex))
		args = append(args, filters.MarketID)
		argIndex++
	}
	
	// 队伍筛选
	if filters.TeamID != "" {
		conditions = append(conditions, fmt.Sprintf("(e.home_team_id = $%d OR e.away_team_id = $%d)", argIndex, argIndex))
		args = append(args, filters.TeamID)
		argIndex++
	}
	
	if filters.TeamName != "" {
		conditions = append(conditions, fmt.Sprintf("(e.home_team_name ILIKE $%d OR e.away_team_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filters.TeamName+"%")
		argIndex++
	}
	
	// 联赛筛选 (需要从 srn_id 中提取)
	if filters.LeagueID != "" {
		conditions = append(conditions, fmt.Sprintf("e.srn_id LIKE $%d", argIndex))
		args = append(args, "%:"+filters.LeagueID+":%")
		argIndex++
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
	
	// 添加 WHERE 子句
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// 排序
	query += " ORDER BY e.schedule_time DESC NULLS LAST, e.created_at DESC"
	
	// 分页
	offset := (filters.Page - 1) * filters.PageSize
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filters.PageSize, offset)
	
	return query, args
}

// buildEventCountQuery 构建计数查询
func buildEventCountQuery(filters *EventFilters) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	query := "SELECT COUNT(DISTINCT e.event_id) FROM tracked_events e"
	
	// 是否需要 JOIN markets 表
	needMarketsJoin := filters.MarketID != "" // MarketGroup 暂时不支持
	
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
	
	if filters.SportID != "" {
		conditions = append(conditions, fmt.Sprintf("e.sport_id = $%d", argIndex))
		args = append(args, filters.SportID)
		argIndex++
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
	
	if filters.MarketID != "" {
		conditions = append(conditions, fmt.Sprintf("m.sr_market_id = $%d", argIndex))
		args = append(args, filters.MarketID)
		argIndex++
	}
	
	if filters.TeamID != "" {
		conditions = append(conditions, fmt.Sprintf("(e.home_team_id = $%d OR e.away_team_id = $%d)", argIndex, argIndex))
		args = append(args, filters.TeamID)
		argIndex++
	}
	
	if filters.TeamName != "" {
		conditions = append(conditions, fmt.Sprintf("(e.home_team_name ILIKE $%d OR e.away_team_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filters.TeamName+"%")
		argIndex++
	}
	
	if filters.LeagueID != "" {
		conditions = append(conditions, fmt.Sprintf("e.srn_id LIKE $%d", argIndex))
		args = append(args, "%:"+filters.LeagueID+":%")
		argIndex++
	}
	
	if filters.Search != "" {
		searchCondition := fmt.Sprintf("(e.event_id ILIKE $%d OR e.home_team_name ILIKE $%d OR e.away_team_name ILIKE $%d)", argIndex, argIndex, argIndex)
		conditions = append(conditions, searchCondition)
		args = append(args, "%"+filters.Search+"%")
		argIndex++
	}
	
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	
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
	if f.SportID != "" {
		m["sport_id"] = f.SportID
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
	if f.MarketID != "" {
		m["market_id"] = f.MarketID
	}
	if f.TeamID != "" {
		m["team_id"] = f.TeamID
	}
	if f.TeamName != "" {
		m["team_name"] = f.TeamName
	}
	if f.LeagueID != "" {
		m["league_id"] = f.LeagueID
	}
	if f.LeagueName != "" {
		m["league_name"] = f.LeagueName
	}
	if f.Search != "" {
		m["search"] = f.Search
	}
	
	return m
}

