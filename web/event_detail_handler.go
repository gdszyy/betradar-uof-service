package web

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// EventDetailResponse 赛事详情响应结构
type EventDetailResponse struct {
	EventID       string                      `json:"event_id"`
	SRNID         *string                     `json:"srn_id"`
	SportID       string                      `json:"sport_id"`
	Status        string                      `json:"status"`
	ScheduleTime  *string                     `json:"schedule_time"`
	HomeTeamID    *string                     `json:"home_team_id"`
	HomeTeamName  *string                     `json:"home_team_name"`
	AwayTeamID    *string                     `json:"away_team_id"`
	AwayTeamName  *string                     `json:"away_team_name"`
	HomeScore     *int                        `json:"home_score"`
	AwayScore     *int                        `json:"away_score"`
	MatchStatus   *string                     `json:"match_status"`
	MatchTime     *string                     `json:"match_time"`
	CreatedAt     string                      `json:"created_at"`
	UpdatedAt     string                      `json:"updated_at"`
	Markets       []MarketWithSpecifiersGroup `json:"markets"`
}

// MarketWithSpecifiersGroup 按market和specifier分组的盘口信息
type MarketWithSpecifiersGroup struct {
	MarketID      string                  `json:"market_id"`
	MarketName    string                  `json:"market_name"`
	MarketType    string                  `json:"market_type"`
	Status        string                  `json:"status"`
	ProducerID    int                     `json:"producer_id"`
	SpecifierGroups []SpecifierMarketGroup `json:"specifier_groups"`
}

// SpecifierMarketGroup 按specifier分组的市场信息
type SpecifierMarketGroup struct {
	Specifier           string                 `json:"specifier"`
	SpecifierDict       map[string]string      `json:"specifier_dict"`
	TabID               *string                `json:"tab_id"`
	ChipID              *string                `json:"chip_id"`
	RemainingSpecifiers map[string]string      `json:"remaining_specifiers"`
	Outcomes            []OutcomeWithTabChip   `json:"outcomes"`
	UpdatedAt           string                 `json:"updated_at"`
}

// OutcomeWithTabChip 包含TabID和ChipID的outcome信息
type OutcomeWithTabChip struct {
	OutcomeID   string  `json:"outcome_id"`
	OutcomeName string  `json:"outcome_name"`
	Odds        float64 `json:"odds"`
	Active      bool    `json:"active"`
}

// handleGetEventDetail 获取赛事详情（包括所有市场、specifier和outcomes）
// GET /api/events/{eventId}
func (s *Server) handleGetEventDetail(w http.ResponseWriter, r *http.Request) {
	// 从URL路径中提取eventId
	eventID := extractEventIDFromPath(r.URL.Path, "/api/events/")
	
	if eventID == "" {
		http.Error(w, "Missing required parameter: eventId", http.StatusBadRequest)
		return
	}

	log.Printf("[API] Getting event detail for event_id: %s", eventID)

	// 获取赛事基本信息
	eventDetail, err := s.getEventDetailFromDB(eventID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Event not found", http.StatusNotFound)
		} else {
			log.Printf("Error getting event detail: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// 获取市场信息（包括specifier和outcomes）
	markets, err := s.getEventMarketsWithSpecifiers(eventID)
	if err != nil {
		log.Printf("Error getting markets: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	eventDetail.Markets = markets

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eventDetail)
}

// getEventDetailFromDB 从数据库获取赛事基本信息
func (s *Server) getEventDetailFromDB(eventID string) (*EventDetailResponse, error) {
	query := `
		SELECT 
			event_id,
			srn_id,
			sport_id,
			status,
			schedule_time,
			home_team_id,
			home_team_name,
			away_team_id,
			away_team_name,
			home_score,
			away_score,
			match_status,
			match_time,
			created_at,
			updated_at
		FROM tracked_events
		WHERE event_id = $1
	`

	var event EventDetailResponse
	err := s.db.QueryRow(query, eventID).Scan(
		&event.EventID,
		&event.SRNID,
		&event.SportID,
		&event.Status,
		&event.ScheduleTime,
		&event.HomeTeamID,
		&event.HomeTeamName,
		&event.AwayTeamID,
		&event.AwayTeamName,
		&event.HomeScore,
		&event.AwayScore,
		&event.MatchStatus,
		&event.MatchTime,
		&event.CreatedAt,
		&event.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &event, nil
}

// getEventMarketsWithSpecifiers 获取赛事的所有市场及其specifier分组
func (s *Server) getEventMarketsWithSpecifiers(eventID string) ([]MarketWithSpecifiersGroup, error) {
	query := `
		SELECT 
			m.id,
			m.sr_market_id,
			m.market_name,
			m.market_type,
			m.status,
			m.producer_id,
			m.specifiers,
			m.updated_at
		FROM markets m
		WHERE m.event_id = $1
		ORDER BY m.sr_market_id, m.specifiers
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()

	// 使用map来按market_id分组
	marketMap := make(map[string]*MarketWithSpecifiersGroup)

	for rows.Next() {
		var marketID int
		var srMarketID, marketType, status string
		var marketName sql.NullString
		var producerID int
		var specifiers sql.NullString
		var updatedAt string

		if err := rows.Scan(&marketID, &srMarketID, &marketName, &marketType, &status, &producerID, &specifiers, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan market: %w", err)
		}

		// 获取该市场的outcomes
		outcomes, err := s.getEventDetailOutcomes(marketID)
		if err != nil {
			log.Printf("Warning: failed to get outcomes for market %d: %v", marketID, err)
			outcomes = []OutcomeWithTabChip{}
		}

		// 获取TabID和ChipID
		tabID, chipID, err := s.getMarketTabChip(marketID, eventID)
		if err != nil {
			log.Printf("Warning: failed to get tab/chip for market %d: %v", marketID, err)
		}

		// 解析specifier
		specifierDict := parseSpecifiers(specifiers.String)

		// 获取剩余specifier（去除TabID或ChipID关联的specifier）
		remainingSpecifiers := getRemainingSpecifiers(specifierDict, tabID, chipID)

		// 创建specifier分组
		specifierGroup := SpecifierMarketGroup{
			Specifier:           specifiers.String,
			SpecifierDict:       specifierDict,
			TabID:               tabID,
			ChipID:              chipID,
			RemainingSpecifiers: remainingSpecifiers,
			Outcomes:            outcomes,
			UpdatedAt:           updatedAt,
		}

		// 处理 marketName 可能为 NULL 的情况
		marketNameStr := ""
		if marketName.Valid {
			marketNameStr = marketName.String
		}

		// 如果market已存在，则添加specifier分组；否则创建新的market
		if market, exists := marketMap[srMarketID]; exists {
			market.SpecifierGroups = append(market.SpecifierGroups, specifierGroup)
		} else {
			marketMap[srMarketID] = &MarketWithSpecifiersGroup{
				MarketID:        srMarketID,
				MarketName:      marketNameStr,
				MarketType:      marketType,
				Status:          status,
				ProducerID:      producerID,
				SpecifierGroups: []SpecifierMarketGroup{specifierGroup},
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating markets: %w", err)
	}

	// 将map转换为slice
	var markets []MarketWithSpecifiersGroup
	for _, market := range marketMap {
		markets = append(markets, *market)
	}

	return markets, nil
}

// getEventDetailOutcomes 获取市场的所有outcomes
func (s *Server) getEventDetailOutcomes(marketID int) ([]OutcomeWithTabChip, error) {
	query := `
		SELECT 
			outcome_id,
			outcome_name,
			odds_value,
			active
		FROM odds
		WHERE market_id = $1
		ORDER BY outcome_id
	`

	rows, err := s.db.Query(query, marketID)
	if err != nil {
		return nil, fmt.Errorf("failed to query outcomes: %w", err)
	}
	defer rows.Close()

	var outcomes []OutcomeWithTabChip
	for rows.Next() {
		var outcome OutcomeWithTabChip
		var oddsValue sql.NullFloat64

		if err := rows.Scan(&outcome.OutcomeID, &outcome.OutcomeName, &oddsValue, &outcome.Active); err != nil {
			return nil, fmt.Errorf("failed to scan outcome: %w", err)
		}

		if oddsValue.Valid {
			outcome.Odds = oddsValue.Float64
		}

		outcomes = append(outcomes, outcome)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outcomes: %w", err)
	}

	return outcomes, nil
}

// getMarketTabChip 获取市场的TabID和ChipID
func (s *Server) getMarketTabChip(marketID int, eventID string) (*string, *string, error) {
	query := `
		SELECT 
			tab_id,
			chip_id
		FROM markets
		WHERE id = $1 AND event_id = $2
	`

	var tabID, chipID sql.NullString
	err := s.db.QueryRow(query, marketID, eventID).Scan(&tabID, &chipID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var tabIDPtr, chipIDPtr *string
	if tabID.Valid {
		tabIDPtr = &tabID.String
	}
	if chipID.Valid {
		chipIDPtr = &chipID.String
	}

	return tabIDPtr, chipIDPtr, nil
}

// parseSpecifiers 解析specifier字符串为字典
func parseSpecifiers(specifiersStr string) map[string]string {
	result := make(map[string]string)
	if specifiersStr == "" {
		return result
	}

	// 按|分割specifier对
	pairs := strings.Split(specifiersStr, "|")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	return result
}

// getRemainingSpecifiers 获取去除TabID或ChipID关联specifier后的剩余specifier
func getRemainingSpecifiers(specifierDict map[string]string, tabID *string, chipID *string) map[string]string {
	remaining := make(map[string]string)
	
	// 复制原specifier字典
	for k, v := range specifierDict {
		remaining[k] = v
	}

	// 定义TabID和ChipID关联的specifier
	tabSpecifierMap := map[string][]string{
		"innings":  {"inningnr"},
		"sets":     {"setnr"},
		"maps":     {"mapnr"},
		"quarters": {"quarternr"},
		"periods":  {"periodnr"},
		"frames":   {"framenr"},
		"overs":    {"overnr"},
		"drives":   {"drivenr"},
		"1st_half": {"goalnr"},
		"2nd_half": {"goalnr"},
		"corners":  {"cornernr"},
	}

	chipSpecifierMap := map[string][]string{
		"innings":  {"inningnr"},
		"sets":     {"setnr"},
		"maps":     {"mapnr"},
		"quarters": {"quarternr"},
		"periods":  {"periodnr"},
		"frames":   {"framenr"},
		"overs":    {"overnr"},
		"drives":   {"drivenr"},
	}

	// 去除TabID关联的specifier
	if tabID != nil {
		if specs, ok := tabSpecifierMap[*tabID]; ok {
			for _, spec := range specs {
				delete(remaining, spec)
			}
		}
	}

	// 去除ChipID关联的specifier
	if chipID != nil {
		if specs, ok := chipSpecifierMap[*chipID]; ok {
			for _, spec := range specs {
				delete(remaining, spec)
			}
		}
	}

	return remaining
}

// extractEventIDFromPath 从URL路径中提取eventId
func extractEventIDFromPath(path string, prefix string) string {
	// 移除前缀
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	remaining := strings.TrimPrefix(path, prefix)
	
	// 提取eventId（直到下一个/或字符串结束）
	parts := strings.Split(remaining, "/")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}

	return ""
}

// handleGetEventDetailByID 获取赛事详情（兼容不同的URL格式）
// 支持 /api/events/{eventId} 和 /api/event/{eventId}
func (s *Server) handleGetEventDetailByID(w http.ResponseWriter, r *http.Request) {
	// 尝试从URL参数中获取eventId
	eventID := r.URL.Query().Get("event_id")
	
	if eventID == "" {
		// 尝试从路径中提取
		eventID = extractEventIDFromPath(r.URL.Path, "/api/event/")
		if eventID == "" {
			eventID = extractEventIDFromPath(r.URL.Path, "/api/events/")
		}
	}

	if eventID == "" {
		http.Error(w, "Missing required parameter: eventId", http.StatusBadRequest)
		return
	}

	// 验证eventId格式（应该是sr:match:数字）
	if !isValidEventID(eventID) {
		http.Error(w, "Invalid eventId format", http.StatusBadRequest)
		return
	}

	s.handleGetEventDetail(w, r)
}

// isValidEventID 验证eventId格式
func isValidEventID(eventID string) bool {
	// eventId应该匹配 sr:match:数字 或 sr:stage:数字 等格式
	pattern := `^sr:[a-z_]+:\d+$`
	matched, _ := regexp.MatchString(pattern, eventID)
	return matched
}
