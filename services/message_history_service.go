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
	EventID     string    `json:"event_id,omitempty"`
	MessageType string    `json:"message_type"`
	Description string    `json:"description"`
	XML         string    `json:"xml"`
}

// MessageHistoryResponse 消息历史响应
type MessageHistoryResponse struct {
	EventID  string               `json:"event_id,omitempty"`
	Total    int                  `json:"total"`
	Messages []MessageHistoryItem `json:"messages"`
}

// NewMessageHistoryService 创建消息历史服务
func NewMessageHistoryService(db *sql.DB) *MessageHistoryService {
	return &MessageHistoryService{db: db}
}

// GetRecentMessages 获取最近的 UOF 消息(不限定比赛)
func (s *MessageHistoryService) GetRecentMessages(limit int, messageType string) (*MessageHistoryResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT message_type, event_id, xml_content, received_at
		FROM uof_messages
		WHERE message_type != 'alive'
	`
	
	// 如果指定了消息类型,添加过滤条件
	if messageType != "" {
		query += ` AND message_type = $1`
	}
	
	query += ` ORDER BY received_at DESC LIMIT `
	if messageType != "" {
		query += `$2`
	} else {
		query += `$1`
	}

	var rows *sql.Rows
	var err error
	
	if messageType != "" {
		rows, err = s.db.Query(query, messageType, limit)
	} else {
		rows, err = s.db.Query(query, limit)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to query uof_messages: %w", err)
	}
	defer rows.Close()

	messages := []MessageHistoryItem{}
	for rows.Next() {
		var msgType, eventID, xmlContent string
		var receivedAt time.Time
		
		if err := rows.Scan(&msgType, &eventID, &xmlContent, &receivedAt); err != nil {
			continue
		}

		description := s.generateDescription(msgType, eventID, xmlContent)
		messages = append(messages, MessageHistoryItem{
			Timestamp:   receivedAt,
			EventID:     eventID,
			MessageType: msgType,
			Description: description,
			XML:         xmlContent,
		})
	}

	return &MessageHistoryResponse{
		EventID:  "", // 不限定比赛
		Total:    len(messages),
		Messages: messages,
	}, nil
}

// GetEventMessages 获取比赛的最近消息
func (s *MessageHistoryService) GetEventMessages(eventID string, limit int) (*MessageHistoryResponse, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT message_type, xml_content, received_at
		FROM uof_messages
		WHERE event_id = $1 AND message_type != 'alive'
		ORDER BY received_at DESC
		LIMIT $2
	`
	
	rows, err := s.db.Query(query, eventID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query uof_messages: %w", err)
	}
	defer rows.Close()

	messages := []MessageHistoryItem{}
	for rows.Next() {
		var msgType, xmlContent string
		var receivedAt time.Time
		
		if err := rows.Scan(&msgType, &xmlContent, &receivedAt); err != nil {
			continue
		}

		description := s.generateDescription(msgType, eventID, xmlContent)
		messages = append(messages, MessageHistoryItem{
			Timestamp:   receivedAt,
			EventID:     eventID,
			MessageType: msgType,
			Description: description,
			XML:         xmlContent,
		})
	}

	return &MessageHistoryResponse{
		EventID:  eventID,
		Total:    len(messages),
		Messages: messages,
	}, nil
}

// generateDescription 生成消息的自然语言描述
func (s *MessageHistoryService) generateDescription(msgType, eventID, xmlContent string) string {
	switch msgType {
	case "odds_change":
		return s.generateOddsChangeDescription(eventID, xmlContent)
	case "bet_stop":
		return s.generateBetStopDescription(eventID, xmlContent)
	case "bet_settlement":
		return s.generateBetSettlementDescription(eventID, xmlContent)
	case "fixture_change":
		return s.generateFixtureChangeDescription(eventID, xmlContent)
	case "bet_cancel":
		return s.generateBetCancelDescription(eventID, xmlContent)
	case "rollback_bet_settlement":
		return fmt.Sprintf("比赛 %s 的结算已撤销", eventID)
	case "rollback_bet_cancel":
		return fmt.Sprintf("比赛 %s 的取消已撤销", eventID)
	default:
		return fmt.Sprintf("比赛 %s 的 %s 消息", eventID, msgType)
	}
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

// generateBetCancelDescription 生成 bet_cancel 的自然语言描述
func (s *MessageHistoryService) generateBetCancelDescription(eventID, xmlContent string) string {
	type BetCancel struct {
		Markets []struct {
			ID string `xml:"id,attr"`
		} `xml:"market"`
	}

	var betCancel BetCancel
	if err := xml.Unmarshal([]byte(xmlContent), &betCancel); err != nil {
		return fmt.Sprintf("比赛 %s 的投注已取消", eventID)
	}

	return fmt.Sprintf("比赛 %s: %d个市场取消投注", eventID, len(betCancel.Markets))
}

