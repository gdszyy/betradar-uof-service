package web

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// MatchRecordsSummary 比赛记录概览
type MatchRecordsSummary struct {
	EventID        string              `json:"event_id"`
	EventInfo      *EventInfo          `json:"event_info"`
	MessagesSummary []MessageSummary   `json:"messages_summary"`
	OddsChangesSummary []OddsChangeSummary `json:"odds_changes_summary"`
	BetStopsSummary []BetStopSummary   `json:"bet_stops_summary"`
	BetSettlementsSummary []BetSettlementSummary `json:"bet_settlements_summary"`
	MarketsSummary []MarketSummary     `json:"markets_summary"`
	Statistics     *RecordStatistics   `json:"statistics"`
}

// EventInfo 赛事基本信息
type EventInfo struct {
	EventID        string    `json:"event_id"`
	SportID        string    `json:"sport_id"`
	SportName      string    `json:"sport_name"`
	Status         string    `json:"status"`
	ScheduleTime   time.Time `json:"schedule_time"`
	HomeTeamName   string    `json:"home_team_name"`
	AwayTeamName   string    `json:"away_team_name"`
	HomeScore      int       `json:"home_score"`
	AwayScore      int       `json:"away_score"`
	Subscribed     bool      `json:"subscribed"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MessageSummary 消息概要
type MessageSummary struct {
	ID          int64     `json:"id"`
	MessageType string    `json:"message_type"`
	ReceivedAt  time.Time `json:"received_at"`
	Producer    string    `json:"producer"`
	Summary     string    `json:"summary"`
}

// OddsChangeSummary 赔率变化概要
type OddsChangeSummary struct {
	ID              int64     `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	ChangeReason    string    `json:"change_reason"`
	BettingStatus   string    `json:"betting_status"`
	MarketsCount    int       `json:"markets_count"`
	OddsCount       int       `json:"odds_count"`
	SportEventStatus string   `json:"sport_event_status"`
	MatchStatus     string    `json:"match_status"`
}

// BetStopSummary 投注停止概要
type BetStopSummary struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Groups    string    `json:"groups"`
	Reason    string    `json:"reason"`
	MarketsCount int    `json:"markets_count"`
}

// BetSettlementSummary 结算概要
type BetSettlementSummary struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Certainty int       `json:"certainty"`
	MarketsCount int    `json:"markets_count"`
	Summary   string    `json:"summary"`
}

// MarketSummary 盘口概要
type MarketSummary struct {
	ID         int64     `json:"id"`
	MarketID   int       `json:"market_id"`
	MarketName string    `json:"market_name"`
	Specifiers string    `json:"specifiers"`
	Status     string    `json:"status"`
	OddsCount  int       `json:"odds_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// RecordStatistics 记录统计
type RecordStatistics struct {
	TotalMessages      int `json:"total_messages"`
	TotalOddsChanges   int `json:"total_odds_changes"`
	TotalBetStops      int `json:"total_bet_stops"`
	TotalBetSettlements int `json:"total_bet_settlements"`
	TotalMarkets       int `json:"total_markets"`
	TotalOdds          int `json:"total_odds"`
}

// RecordDetail 记录详情
type RecordDetail struct {
	RecordType string      `json:"record_type"` // "message", "odds_change", "bet_stop", "bet_settlement", "market"
	RecordID   int64       `json:"record_id"`
	RawData    interface{} `json:"raw_data"`
}

// handleGetMatchRecords 获取比赛记录概览
func (s *Server) handleGetMatchRecords(w http.ResponseWriter, r *http.Request) {
	eventID := r.URL.Query().Get("event_id")
	if eventID == "" {
		http.Error(w, "Missing event_id parameter", http.StatusBadRequest)
		return
	}

	log.Printf("[MatchRecords] Fetching records for event: %s", eventID)

	// 1. 获取赛事基本信息
	eventInfo, err := s.getEventInfo(eventID)
	if err != nil {
		log.Printf("[MatchRecords] ❌ Failed to get event info: %v", err)
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	// 2. 获取消息概要
	messagesSummary, err := s.getMessagesSummary(eventID)
	if err != nil {
		log.Printf("[MatchRecords] ⚠️  Failed to get messages summary: %v", err)
	}

	// 3. 获取赔率变化概要
	oddsChangesSummary, err := s.getOddsChangesSummary(eventID)
	if err != nil {
		log.Printf("[MatchRecords] ⚠️  Failed to get odds changes summary: %v", err)
	}

	// 4. 获取投注停止概要
	betStopsSummary, err := s.getBetStopsSummary(eventID)
	if err != nil {
		log.Printf("[MatchRecords] ⚠️  Failed to get bet stops summary: %v", err)
	}

	// 5. 获取结算概要
	betSettlementsSummary, err := s.getBetSettlementsSummary(eventID)
	if err != nil {
		log.Printf("[MatchRecords] ⚠️  Failed to get bet settlements summary: %v", err)
	}

	// 6. 获取盘口概要
	marketsSummary, err := s.getMarketsSummary(eventID)
	if err != nil {
		log.Printf("[MatchRecords] ⚠️  Failed to get markets summary: %v", err)
	}

	// 7. 统计信息
	statistics := &RecordStatistics{
		TotalMessages:      len(messagesSummary),
		TotalOddsChanges:   len(oddsChangesSummary),
		TotalBetStops:      len(betStopsSummary),
		TotalBetSettlements: len(betSettlementsSummary),
		TotalMarkets:       len(marketsSummary),
	}

	// 统计总赔率数
	for _, market := range marketsSummary {
		statistics.TotalOdds += market.OddsCount
	}

	// 构建响应
	summary := MatchRecordsSummary{
		EventID:               eventID,
		EventInfo:             eventInfo,
		MessagesSummary:       messagesSummary,
		OddsChangesSummary:    oddsChangesSummary,
		BetStopsSummary:       betStopsSummary,
		BetSettlementsSummary: betSettlementsSummary,
		MarketsSummary:        marketsSummary,
		Statistics:            statistics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)

	log.Printf("[MatchRecords] ✅ Returned %d messages, %d odds changes, %d bet stops, %d settlements, %d markets",
		statistics.TotalMessages, statistics.TotalOddsChanges, statistics.TotalBetStops,
		statistics.TotalBetSettlements, statistics.TotalMarkets)
}

// handleGetRecordDetail 获取记录详情
func (s *Server) handleGetRecordDetail(w http.ResponseWriter, r *http.Request) {
	recordType := r.URL.Query().Get("type")
	recordIDStr := r.URL.Query().Get("id")

	if recordType == "" || recordIDStr == "" {
		http.Error(w, "Missing type or id parameter", http.StatusBadRequest)
		return
	}

	recordID, err := strconv.ParseInt(recordIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid id parameter", http.StatusBadRequest)
		return
	}

	log.Printf("[RecordDetail] Fetching %s record with ID: %d", recordType, recordID)

	var detail *RecordDetail

	switch recordType {
	case "message":
		detail, err = s.getMessageDetail(recordID)
	case "odds_change":
		detail, err = s.getOddsChangeDetail(recordID)
	case "bet_stop":
		detail, err = s.getBetStopDetail(recordID)
	case "bet_settlement":
		detail, err = s.getBetSettlementDetail(recordID)
	case "market":
		detail, err = s.getMarketDetail(recordID)
	default:
		http.Error(w, "Invalid record type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("[RecordDetail] ❌ Failed to get record detail: %v", err)
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)

	log.Printf("[RecordDetail] ✅ Returned %s record %d", recordType, recordID)
}

// getEventInfo 获取赛事基本信息
func (s *Server) getEventInfo(eventID string) (*EventInfo, error) {
	query := `
		SELECT event_id, sport_id, sport_name, status, schedule_time,
		       home_team_name, away_team_name, home_score, away_score,
		       subscribed, created_at, updated_at
		FROM tracked_events
		WHERE event_id = $1
	`

	var info EventInfo
	err := s.db.QueryRow(query, eventID).Scan(
		&info.EventID, &info.SportID, &info.SportName, &info.Status, &info.ScheduleTime,
		&info.HomeTeamName, &info.AwayTeamName, &info.HomeScore, &info.AwayScore,
		&info.Subscribed, &info.CreatedAt, &info.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &info, nil
}

// getMessagesSummary 获取消息概要
func (s *Server) getMessagesSummary(eventID string) ([]MessageSummary, error) {
	query := `
		SELECT id, message_type, received_at, producer
		FROM uof_messages
		WHERE event_id = $1
		ORDER BY received_at DESC
		LIMIT 100
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []MessageSummary
	for rows.Next() {
		var msg MessageSummary
		var producer sql.NullString

		err := rows.Scan(&msg.ID, &msg.MessageType, &msg.ReceivedAt, &producer)
		if err != nil {
			continue
		}

		if producer.Valid {
			msg.Producer = producer.String
		}

		// 生成概要
		msg.Summary = generateMessageSummary(msg.MessageType, msg.ReceivedAt)

		messages = append(messages, msg)
	}

	return messages, nil
}

// getOddsChangesSummary 获取赔率变化概要
func (s *Server) getOddsChangesSummary(eventID string) ([]OddsChangeSummary, error) {
	query := `
		SELECT oc.id, oc.timestamp, oc.change_reason, oc.betting_status,
		       oc.sport_event_status, oc.match_status,
		       COUNT(DISTINCT m.id) as markets_count,
		       COUNT(o.id) as odds_count
		FROM odds_changes oc
		LEFT JOIN markets m ON m.odds_change_id = oc.id
		LEFT JOIN odds o ON o.market_id = m.id
		WHERE oc.event_id = $1
		GROUP BY oc.id, oc.timestamp, oc.change_reason, oc.betting_status,
		         oc.sport_event_status, oc.match_status
		ORDER BY oc.timestamp DESC
		LIMIT 50
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var oddsChanges []OddsChangeSummary
	for rows.Next() {
		var oc OddsChangeSummary
		var changeReason, bettingStatus, sportEventStatus, matchStatus sql.NullString

		err := rows.Scan(
			&oc.ID, &oc.Timestamp, &changeReason, &bettingStatus,
			&sportEventStatus, &matchStatus,
			&oc.MarketsCount, &oc.OddsCount,
		)
		if err != nil {
			continue
		}

		if changeReason.Valid {
			oc.ChangeReason = changeReason.String
		}
		if bettingStatus.Valid {
			oc.BettingStatus = bettingStatus.String
		}
		if sportEventStatus.Valid {
			oc.SportEventStatus = sportEventStatus.String
		}
		if matchStatus.Valid {
			oc.MatchStatus = matchStatus.String
		}

		oddsChanges = append(oddsChanges, oc)
	}

	return oddsChanges, nil
}

// getBetStopsSummary 获取投注停止概要
func (s *Server) getBetStopsSummary(eventID string) ([]BetStopSummary, error) {
	query := `
		SELECT id, timestamp, groups, reason, market_count
		FROM bet_stops
		WHERE event_id = $1
		ORDER BY timestamp DESC
		LIMIT 50
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var betStops []BetStopSummary
	for rows.Next() {
		var bs BetStopSummary
		var groups, reason sql.NullString

		err := rows.Scan(&bs.ID, &bs.Timestamp, &groups, &reason, &bs.MarketsCount)
		if err != nil {
			continue
		}

		if groups.Valid {
			bs.Groups = groups.String
		}
		if reason.Valid {
			bs.Reason = reason.String
		}

		betStops = append(betStops, bs)
	}

	return betStops, nil
}

// getBetSettlementsSummary 获取结算概要
func (s *Server) getBetSettlementsSummary(eventID string) ([]BetSettlementSummary, error) {
	query := `
		SELECT id, timestamp, certainty, market_count
		FROM bet_settlements
		WHERE event_id = $1
		ORDER BY timestamp DESC
		LIMIT 50
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settlements []BetSettlementSummary
	for rows.Next() {
		var bs BetSettlementSummary

		err := rows.Scan(&bs.ID, &bs.Timestamp, &bs.Certainty, &bs.MarketsCount)
		if err != nil {
			continue
		}

		// 生成概要
		certaintyText := "Unknown"
		if bs.Certainty == 1 {
			certaintyText = "Confirmed"
		} else if bs.Certainty == 2 {
			certaintyText = "Final"
		}
		bs.Summary = certaintyText + " settlement with " + strconv.Itoa(bs.MarketsCount) + " markets"

		settlements = append(settlements, bs)
	}

	return settlements, nil
}

// getMarketsSummary 获取盘口概要
func (s *Server) getMarketsSummary(eventID string) ([]MarketSummary, error) {
	query := `
		SELECT m.id, m.market_id, m.market_name, m.specifiers, m.status,
		       COUNT(o.id) as odds_count, m.created_at, m.updated_at
		FROM markets m
		LEFT JOIN odds o ON o.market_id = m.id
		WHERE m.event_id = $1
		GROUP BY m.id, m.market_id, m.market_name, m.specifiers, m.status,
		         m.created_at, m.updated_at
		ORDER BY m.created_at DESC
		LIMIT 100
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []MarketSummary
	for rows.Next() {
		var m MarketSummary
		var specifiers, status sql.NullString

		err := rows.Scan(
			&m.ID, &m.MarketID, &m.MarketName, &specifiers, &status,
			&m.OddsCount, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if specifiers.Valid {
			m.Specifiers = specifiers.String
		}
		if status.Valid {
			m.Status = status.String
		}

		markets = append(markets, m)
	}

	return markets, nil
}

// getMessageDetail 获取消息详情
func (s *Server) getMessageDetail(id int64) (*RecordDetail, error) {
	query := `
		SELECT id, event_id, message_type, producer, timestamp, received_at, raw_message
		FROM uof_messages
		WHERE id = $1
	`

	var detail struct {
		ID          int64     `json:"id"`
		EventID     string    `json:"event_id"`
		MessageType string    `json:"message_type"`
		Producer    string    `json:"producer"`
		Timestamp   int64     `json:"timestamp"`
		ReceivedAt  time.Time `json:"received_at"`
		RawMessage  string    `json:"raw_message"`
	}

	var producer sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&detail.ID, &detail.EventID, &detail.MessageType, &producer,
		&detail.Timestamp, &detail.ReceivedAt, &detail.RawMessage,
	)

	if err != nil {
		return nil, err
	}

	if producer.Valid {
		detail.Producer = producer.String
	}

	return &RecordDetail{
		RecordType: "message",
		RecordID:   id,
		RawData:    detail,
	}, nil
}

// getOddsChangeDetail 获取赔率变化详情
func (s *Server) getOddsChangeDetail(id int64) (*RecordDetail, error) {
	// 获取 odds_change 基本信息
	query := `
		SELECT id, event_id, timestamp, change_reason, betting_status,
		       sport_event_status, match_status, betstop_reason
		FROM odds_changes
		WHERE id = $1
	`

	var detail struct {
		ID               int64     `json:"id"`
		EventID          string    `json:"event_id"`
		Timestamp        time.Time `json:"timestamp"`
		ChangeReason     string    `json:"change_reason"`
		BettingStatus    string    `json:"betting_status"`
		SportEventStatus string    `json:"sport_event_status"`
		MatchStatus      string    `json:"match_status"`
		BetstopReason    string    `json:"betstop_reason"`
		Markets          []map[string]interface{} `json:"markets"`
	}

	var changeReason, bettingStatus, sportEventStatus, matchStatus, betstopReason sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&detail.ID, &detail.EventID, &detail.Timestamp, &changeReason, &bettingStatus,
		&sportEventStatus, &matchStatus, &betstopReason,
	)

	if err != nil {
		return nil, err
	}

	if changeReason.Valid {
		detail.ChangeReason = changeReason.String
	}
	if bettingStatus.Valid {
		detail.BettingStatus = bettingStatus.String
	}
	if sportEventStatus.Valid {
		detail.SportEventStatus = sportEventStatus.String
	}
	if matchStatus.Valid {
		detail.MatchStatus = matchStatus.String
	}
	if betstopReason.Valid {
		detail.BetstopReason = betstopReason.String
	}

	// 获取关联的 markets 和 odds
	marketsQuery := `
		SELECT m.id, m.market_id, m.market_name, m.specifiers, m.status,
		       o.id as odd_id, o.outcome_id, o.outcome_name, o.odds_value, o.active, o.probabilities
		FROM markets m
		LEFT JOIN odds o ON o.market_id = m.id
		WHERE m.odds_change_id = $1
		ORDER BY m.id, o.id
	`

	rows, err := s.db.Query(marketsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	marketsMap := make(map[int64]map[string]interface{})

	for rows.Next() {
		var marketID int64
		var marketIDNum int
		var marketName string
		var specifiers, status sql.NullString
		var oddID sql.NullInt64
		var outcomeID, outcomeName sql.NullString
		var oddsValue, probabilities sql.NullFloat64
		var active sql.NullBool

		err := rows.Scan(
			&marketID, &marketIDNum, &marketName, &specifiers, &status,
			&oddID, &outcomeID, &outcomeName, &oddsValue, &active, &probabilities,
		)
		if err != nil {
			continue
		}

		// 如果 market 不存在，创建它
		if _, exists := marketsMap[marketID]; !exists {
			marketsMap[marketID] = map[string]interface{}{
				"id":          marketID,
				"market_id":   marketIDNum,
				"market_name": marketName,
				"specifiers":  "",
				"status":      "",
				"odds":        []map[string]interface{}{},
			}

			if specifiers.Valid {
				marketsMap[marketID]["specifiers"] = specifiers.String
			}
			if status.Valid {
				marketsMap[marketID]["status"] = status.String
			}
		}

		// 添加 odd
		if oddID.Valid {
			odd := map[string]interface{}{
				"id":           oddID.Int64,
				"outcome_id":   "",
				"outcome_name": "",
				"odds_value":   0.0,
				"active":       false,
				"probabilities": 0.0,
			}

			if outcomeID.Valid {
				odd["outcome_id"] = outcomeID.String
			}
			if outcomeName.Valid {
				odd["outcome_name"] = outcomeName.String
			}
			if oddsValue.Valid {
				odd["odds_value"] = oddsValue.Float64
			}
			if active.Valid {
				odd["active"] = active.Bool
			}
			if probabilities.Valid {
				odd["probabilities"] = probabilities.Float64
			}

			odds := marketsMap[marketID]["odds"].([]map[string]interface{})
			marketsMap[marketID]["odds"] = append(odds, odd)
		}
	}

	// 转换为数组
	for _, market := range marketsMap {
		detail.Markets = append(detail.Markets, market)
	}

	return &RecordDetail{
		RecordType: "odds_change",
		RecordID:   id,
		RawData:    detail,
	}, nil
}

// getBetStopDetail 获取投注停止详情
func (s *Server) getBetStopDetail(id int64) (*RecordDetail, error) {
	query := `
		SELECT id, event_id, timestamp, groups, reason, market_count, market_status
		FROM bet_stops
		WHERE id = $1
	`

	var detail struct {
		ID           int64     `json:"id"`
		EventID      string    `json:"event_id"`
		Timestamp    time.Time `json:"timestamp"`
		Groups       string    `json:"groups"`
		Reason       string    `json:"reason"`
		MarketCount  int       `json:"market_count"`
		MarketStatus string    `json:"market_status"`
	}

	var groups, reason, marketStatus sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&detail.ID, &detail.EventID, &detail.Timestamp, &groups, &reason,
		&detail.MarketCount, &marketStatus,
	)

	if err != nil {
		return nil, err
	}

	if groups.Valid {
		detail.Groups = groups.String
	}
	if reason.Valid {
		detail.Reason = reason.String
	}
	if marketStatus.Valid {
		detail.MarketStatus = marketStatus.String
	}

	return &RecordDetail{
		RecordType: "bet_stop",
		RecordID:   id,
		RawData:    detail,
	}, nil
}

// getBetSettlementDetail 获取结算详情
func (s *Server) getBetSettlementDetail(id int64) (*RecordDetail, error) {
	query := `
		SELECT id, event_id, timestamp, producer, product, certainty, market_count
		FROM bet_settlements
		WHERE id = $1
	`

	var detail struct {
		ID          int64     `json:"id"`
		EventID     string    `json:"event_id"`
		Timestamp   time.Time `json:"timestamp"`
		Producer    string    `json:"producer"`
		Product     int       `json:"product"`
		Certainty   int       `json:"certainty"`
		MarketCount int       `json:"market_count"`
	}

	var producer sql.NullString

	err := s.db.QueryRow(query, id).Scan(
		&detail.ID, &detail.EventID, &detail.Timestamp, &producer,
		&detail.Product, &detail.Certainty, &detail.MarketCount,
	)

	if err != nil {
		return nil, err
	}

	if producer.Valid {
		detail.Producer = producer.String
	}

	return &RecordDetail{
		RecordType: "bet_settlement",
		RecordID:   id,
		RawData:    detail,
	}, nil
}

// getMarketDetail 获取盘口详情
func (s *Server) getMarketDetail(id int64) (*RecordDetail, error) {
	query := `
		SELECT m.id, m.event_id, m.market_id, m.market_name, m.specifiers,
		       m.status, m.favourite, m.created_at, m.updated_at,
		       o.id as odd_id, o.outcome_id, o.outcome_name, o.odds_value,
		       o.active, o.probabilities
		FROM markets m
		LEFT JOIN odds o ON o.market_id = m.id
		WHERE m.id = $1
		ORDER BY o.id
	`

	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var detail struct {
		ID         int64                    `json:"id"`
		EventID    string                   `json:"event_id"`
		MarketID   int                      `json:"market_id"`
		MarketName string                   `json:"market_name"`
		Specifiers string                   `json:"specifiers"`
		Status     string                   `json:"status"`
		Favourite  bool                     `json:"favourite"`
		CreatedAt  time.Time                `json:"created_at"`
		UpdatedAt  time.Time                `json:"updated_at"`
		Odds       []map[string]interface{} `json:"odds"`
	}

	first := true
	for rows.Next() {
		var specifiers, status sql.NullString
		var favourite sql.NullBool
		var oddID sql.NullInt64
		var outcomeID, outcomeName sql.NullString
		var oddsValue, probabilities sql.NullFloat64
		var active sql.NullBool

		err := rows.Scan(
			&detail.ID, &detail.EventID, &detail.MarketID, &detail.MarketName, &specifiers,
			&status, &favourite, &detail.CreatedAt, &detail.UpdatedAt,
			&oddID, &outcomeID, &outcomeName, &oddsValue, &active, &probabilities,
		)
		if err != nil {
			continue
		}

		if first {
			if specifiers.Valid {
				detail.Specifiers = specifiers.String
			}
			if status.Valid {
				detail.Status = status.String
			}
			if favourite.Valid {
				detail.Favourite = favourite.Bool
			}
			first = false
		}

		// 添加 odd
		if oddID.Valid {
			odd := map[string]interface{}{
				"id":            oddID.Int64,
				"outcome_id":    "",
				"outcome_name":  "",
				"odds_value":    0.0,
				"active":        false,
				"probabilities": 0.0,
			}

			if outcomeID.Valid {
				odd["outcome_id"] = outcomeID.String
			}
			if outcomeName.Valid {
				odd["outcome_name"] = outcomeName.String
			}
			if oddsValue.Valid {
				odd["odds_value"] = oddsValue.Float64
			}
			if active.Valid {
				odd["active"] = active.Bool
			}
			if probabilities.Valid {
				odd["probabilities"] = probabilities.Float64
			}

			detail.Odds = append(detail.Odds, odd)
		}
	}

	if first {
		// 没有找到记录
		return nil, sql.ErrNoRows
	}

	return &RecordDetail{
		RecordType: "market",
		RecordID:   id,
		RawData:    detail,
	}, nil
}

// generateMessageSummary 生成消息概要
func generateMessageSummary(messageType string, receivedAt time.Time) string {
	return messageType + " received at " + receivedAt.Format("2006-01-02 15:04:05")
}

