package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"time"
)

// MessageHistoryService 消息历史服务
type MessageHistoryService struct {
	db *sql.DB
}

// MessageHistoryItem 消息历史项
type MessageHistoryItem struct {
	Timestamp   time.Time `json:"timestamp"`
	MessageType string    `json:"message_type"`
	Description string    `json:"description"`
	XML         string    `json:"xml"`
}

// MessageHistoryResponse 消息历史响应
type MessageHistoryResponse struct {
	EventID  string               `json:"event_id"`
	Total    int                  `json:"total"`
	Messages []MessageHistoryItem `json:"messages"`
}

// NewMessageHistoryService 创建消息历史服务
func NewMessageHistoryService(db *sql.DB) *MessageHistoryService {
	return &MessageHistoryService{db: db}
}

// GetEventMessages 获取比赛的最近消息
func (s *MessageHistoryService) GetEventMessages(eventID string, limit int) (*MessageHistoryResponse, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100 // 最多返回 100 条
	}

	messages := []MessageHistoryItem{}

	// 1. 从 odds_changes 表获取消息
	oddsQuery := `
		SELECT timestamp, xml_content
		FROM odds_changes
		WHERE event_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err := s.db.Query(oddsQuery, eventID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query odds_changes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var timestamp time.Time
		var xmlContent string
		if err := rows.Scan(&timestamp, &xmlContent); err != nil {
			continue
		}

		description := s.generateOddsChangeDescription(eventID, xmlContent)
		messages = append(messages, MessageHistoryItem{
			Timestamp:   timestamp,
			MessageType: "odds_change",
			Description: description,
			XML:         xmlContent,
		})
	}

	// 2. 从 bet_stops 表获取消息
	betStopQuery := `
		SELECT timestamp, xml_content
		FROM bet_stops
		WHERE event_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err = s.db.Query(betStopQuery, eventID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query bet_stops: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var timestamp time.Time
		var xmlContent string
		if err := rows.Scan(&timestamp, &xmlContent); err != nil {
			continue
		}

		description := s.generateBetStopDescription(eventID, xmlContent)
		messages = append(messages, MessageHistoryItem{
			Timestamp:   timestamp,
			MessageType: "bet_stop",
			Description: description,
			XML:         xmlContent,
		})
	}

	// 3. 从 bet_settlements 表获取消息
	settlementQuery := `
		SELECT timestamp, xml_content
		FROM bet_settlements
		WHERE event_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err = s.db.Query(settlementQuery, eventID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query bet_settlements: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var timestamp time.Time
		var xmlContent string
		if err := rows.Scan(&timestamp, &xmlContent); err != nil {
			continue
		}

		description := s.generateBetSettlementDescription(eventID, xmlContent)
		messages = append(messages, MessageHistoryItem{
			Timestamp:   timestamp,
			MessageType: "bet_settlement",
			Description: description,
			XML:         xmlContent,
		})
	}

	// 4. 从 fixture_changes 表获取消息
	fixtureQuery := `
		SELECT timestamp, xml_content
		FROM fixture_changes
		WHERE event_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`
	rows, err = s.db.Query(fixtureQuery, eventID, limit)
	if err != nil {
		// fixture_changes 表可能不存在,忽略错误
	} else {
		defer rows.Close()
		for rows.Next() {
			var timestamp time.Time
			var xmlContent string
			if err := rows.Scan(&timestamp, &xmlContent); err != nil {
				continue
			}

			description := s.generateFixtureChangeDescription(eventID, xmlContent)
			messages = append(messages, MessageHistoryItem{
				Timestamp:   timestamp,
				MessageType: "fixture_change",
				Description: description,
				XML:         xmlContent,
			})
		}
	}

	// 按时间倒序排序
	sortMessagesByTimestamp(messages)

	// 只返回 limit 条
	if len(messages) > limit {
		messages = messages[:limit]
	}

	return &MessageHistoryResponse{
		EventID:  eventID,
		Total:    len(messages),
		Messages: messages,
	}, nil
}

// generateOddsChangeDescription 生成 odds_change 的自然语言描述
func (s *MessageHistoryService) generateOddsChangeDescription(eventID, xmlContent string) string {
	type OddsChange struct {
		Odds struct {
			Markets []struct {
				Outcomes []struct{} `xml:"outcome"`
			} `xml:"market"`
		} `xml:"odds"`
		SportEventStatus *struct {
			HomeScore *int `xml:"home_score,attr"`
			AwayScore *int `xml:"away_score,attr"`
			MatchStatus string `xml:"match_status,attr"`
		} `xml:"sport_event_status"`
	}

	var oddsChange OddsChange
	if err := xml.Unmarshal([]byte(xmlContent), &oddsChange); err != nil {
		return fmt.Sprintf("比赛 %s 的赔率已更新", eventID)
	}

	marketCount := len(oddsChange.Odds.Markets)
	outcomeCount := 0
	for _, market := range oddsChange.Odds.Markets {
		outcomeCount += len(market.Outcomes)
	}

	description := fmt.Sprintf("比赛 %s: %d个市场, %d个结果", eventID, marketCount, outcomeCount)

	if oddsChange.SportEventStatus != nil {
		if oddsChange.SportEventStatus.HomeScore != nil && oddsChange.SportEventStatus.AwayScore != nil {
			description += fmt.Sprintf(", 比分 %d-%d", 
				*oddsChange.SportEventStatus.HomeScore, 
				*oddsChange.SportEventStatus.AwayScore)
		}
		if oddsChange.SportEventStatus.MatchStatus != "" {
			description += fmt.Sprintf(", 状态: %s", oddsChange.SportEventStatus.MatchStatus)
		}
	}

	return description
}

// generateBetStopDescription 生成 bet_stop 的自然语言描述
func (s *MessageHistoryService) generateBetStopDescription(eventID, xmlContent string) string {
	type BetStop struct {
		Groups string `xml:"groups,attr"`
	}

	var betStop BetStop
	if err := xml.Unmarshal([]byte(xmlContent), &betStop); err != nil {
		return fmt.Sprintf("比赛 %s 的市场已暂停", eventID)
	}

	if betStop.Groups == "all" || betStop.Groups == "" {
		return fmt.Sprintf("比赛 %s 的所有市场已暂停", eventID)
	}
	return fmt.Sprintf("比赛 %s 的市场组 %s 已暂停", eventID, betStop.Groups)
}

// generateBetSettlementDescription 生成 bet_settlement 的自然语言描述
func (s *MessageHistoryService) generateBetSettlementDescription(eventID, xmlContent string) string {
	type BetSettlement struct {
		Certainty int `xml:"certainty,attr"`
		Outcomes struct {
			Markets []struct {
				Outcomes []struct{} `xml:"outcome"`
			} `xml:"market"`
		} `xml:"outcomes"`
	}

	var settlement BetSettlement
	if err := xml.Unmarshal([]byte(xmlContent), &settlement); err != nil {
		return fmt.Sprintf("比赛 %s 的市场已结算", eventID)
	}

	marketCount := len(settlement.Outcomes.Markets)
	outcomeCount := 0
	for _, market := range settlement.Outcomes.Markets {
		outcomeCount += len(market.Outcomes)
	}

	certaintyText := ""
	switch settlement.Certainty {
	case 1:
		certaintyText = "低确定性"
	case 2:
		certaintyText = "高确定性"
	case 3:
		certaintyText = "最高确定性"
	default:
		certaintyText = fmt.Sprintf("certainty=%d", settlement.Certainty)
	}

	return fmt.Sprintf("比赛 %s 的 %d个市场已结算: %d个结果 (%s)",
		eventID, marketCount, outcomeCount, certaintyText)
}

// generateFixtureChangeDescription 生成 fixture_change 的自然语言描述
func (s *MessageHistoryService) generateFixtureChangeDescription(eventID, xmlContent string) string {
	type FixtureChange struct {
		StartTime  int64 `xml:"start_time,attr"`
		ChangeType int   `xml:"change_type,attr"`
	}

	var fixtureChange FixtureChange
	if err := xml.Unmarshal([]byte(xmlContent), &fixtureChange); err != nil {
		return fmt.Sprintf("比赛 %s 的赛事信息已更新", eventID)
	}

	if fixtureChange.ChangeType == 5 {
		return fmt.Sprintf("比赛 %s 的直播覆盖已取消", eventID)
	}

	if fixtureChange.StartTime > 0 {
		startTime := time.UnixMilli(fixtureChange.StartTime)
		return fmt.Sprintf("比赛 %s 的开赛时间变更为 %s", 
			eventID, startTime.Format("2006-01-02 15:04"))
	}

	return fmt.Sprintf("比赛 %s 的赛事信息已更新", eventID)
}

// GetRecentMessages 获取最近的 UOF 消息(不限定比赛)
func (s *MessageHistoryService) GetRecentMessages(limit int, messageType string) (*MessageHistoryResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	messages := []MessageHistoryItem{}

	// 根据 messageType 查询不同的表
	if messageType == "" || messageType == "odds_change" {
		if err := s.fetchRecentOddsChanges(&messages, limit); err != nil {
			return nil, err
		}
	}

	if messageType == "" || messageType == "bet_stop" {
		if err := s.fetchRecentBetStops(&messages, limit); err != nil {
			return nil, err
		}
	}

	if messageType == "" || messageType == "bet_settlement" {
		if err := s.fetchRecentBetSettlements(&messages, limit); err != nil {
			return nil, err
		}
	}

	if messageType == "" || messageType == "fixture_change" {
		if err := s.fetchRecentFixtureChanges(&messages, limit); err != nil {
			// fixture_changes 表可能不存在,忽略错误
		}
	}

	// 按时间倒序排序
	sortMessagesByTimestamp(messages)

	// 只返回 limit 条
	if len(messages) > limit {
		messages = messages[:limit]
	}

	return &MessageHistoryResponse{
		EventID:  "", // 不限定比赛
		Total:    len(messages),
		Messages: messages,
	}, nil
}

// fetchRecentOddsChanges 获取最近的 odds_change 消息
func (s *MessageHistoryService) fetchRecentOddsChanges(messages *[]MessageHistoryItem, limit int) error {
	query := `
		SELECT event_id, timestamp, xml_content
		FROM odds_changes
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return fmt.Errorf("failed to query odds_changes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventID string
		var timestamp time.Time
		var xmlContent string
		if err := rows.Scan(&eventID, &timestamp, &xmlContent); err != nil {
			continue
		}

		description := s.generateOddsChangeDescription(eventID, xmlContent)
		*messages = append(*messages, MessageHistoryItem{
			Timestamp:   timestamp,
			MessageType: "odds_change",
			Description: description,
			XML:         xmlContent,
		})
	}

	return nil
}

// fetchRecentBetStops 获取最近的 bet_stop 消息
func (s *MessageHistoryService) fetchRecentBetStops(messages *[]MessageHistoryItem, limit int) error {
	query := `
		SELECT event_id, timestamp, xml_content
		FROM bet_stops
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return fmt.Errorf("failed to query bet_stops: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventID string
		var timestamp time.Time
		var xmlContent string
		if err := rows.Scan(&eventID, &timestamp, &xmlContent); err != nil {
			continue
		}

		description := s.generateBetStopDescription(eventID, xmlContent)
		*messages = append(*messages, MessageHistoryItem{
			Timestamp:   timestamp,
			MessageType: "bet_stop",
			Description: description,
			XML:         xmlContent,
		})
	}

	return nil
}

// fetchRecentBetSettlements 获取最近的 bet_settlement 消息
func (s *MessageHistoryService) fetchRecentBetSettlements(messages *[]MessageHistoryItem, limit int) error {
	query := `
		SELECT event_id, timestamp, xml_content
		FROM bet_settlements
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return fmt.Errorf("failed to query bet_settlements: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventID string
		var timestamp time.Time
		var xmlContent string
		if err := rows.Scan(&eventID, &timestamp, &xmlContent); err != nil {
			continue
		}

		description := s.generateBetSettlementDescription(eventID, xmlContent)
		*messages = append(*messages, MessageHistoryItem{
			Timestamp:   timestamp,
			MessageType: "bet_settlement",
			Description: description,
			XML:         xmlContent,
		})
	}

	return nil
}

// fetchRecentFixtureChanges 获取最近的 fixture_change 消息
func (s *MessageHistoryService) fetchRecentFixtureChanges(messages *[]MessageHistoryItem, limit int) error {
	query := `
		SELECT event_id, timestamp, xml_content
		FROM fixture_changes
		ORDER BY timestamp DESC
		LIMIT $1
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return fmt.Errorf("failed to query fixture_changes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventID string
		var timestamp time.Time
		var xmlContent string
		if err := rows.Scan(&eventID, &timestamp, &xmlContent); err != nil {
			continue
		}

		description := s.generateFixtureChangeDescription(eventID, xmlContent)
		*messages = append(*messages, MessageHistoryItem{
			Timestamp:   timestamp,
			MessageType: "fixture_change",
			Description: description,
			XML:         xmlContent,
		})
	}

	return nil
}

// sortMessagesByTimestamp 按时间戳倒序排序消息
func sortMessagesByTimestamp(messages []MessageHistoryItem) {
	// 简单的冒泡排序
	for i := 0; i < len(messages); i++ {
		for j := i + 1; j < len(messages); j++ {
			if messages[i].Timestamp.Before(messages[j].Timestamp) {
				messages[i], messages[j] = messages[j], messages[i]
			}
		}
	}
}

