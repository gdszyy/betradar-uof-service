package services

import (
	"database/sql"
	"time"
)

type MessageStore struct {
	db *sql.DB
}

func NewMessageStore(db *sql.DB) *MessageStore {
	return &MessageStore{db: db}
}

// SaveMessage 保存消息到数据库
func (s *MessageStore) SaveMessage(messageType, eventID string, productID *int, sportID *string, routingKey, xmlContent string, timestamp int64) error {
	query := `
		INSERT INTO uof_messages (message_type, event_id, product_id, sport_id, routing_key, xml_content, timestamp, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var eventIDPtr *string
	if eventID != "" {
		eventIDPtr = &eventID
	}

	_, err := s.db.Exec(query, messageType, eventIDPtr, productID, sportID, routingKey, xmlContent, timestamp, time.Now())
	return err
}

// SaveOddsChange 保存赔率变化
func (s *MessageStore) SaveOddsChange(eventID string, productID int, timestamp int64, xmlContent string, marketsCount int) error {
	query := `
		INSERT INTO odds_changes (event_id, product_id, timestamp, markets_count, xml_content)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := s.db.Exec(query, eventID, productID, timestamp, marketsCount, xmlContent)
	return err
}

// SaveBetStop 保存投注停止
func (s *MessageStore) SaveBetStop(eventID string, productID int, timestamp int64, xmlContent string) error {
	query := `
		INSERT INTO bet_stops (event_id, product_id, timestamp, xml_content)
		VALUES ($1, $2, $3, $4)
	`

	_, err := s.db.Exec(query, eventID, productID, timestamp, xmlContent)
	return err
}

// SaveBetSettlement 保存投注结算
func (s *MessageStore) SaveBetSettlement(eventID string, productID int, timestamp int64, xmlContent string) error {
	query := `
		INSERT INTO bet_settlements (event_id, product_id, timestamp, xml_content)
		VALUES ($1, $2, $3, $4)
	`

	_, err := s.db.Exec(query, eventID, productID, timestamp, xmlContent)
	return err
}

// UpdateProducerStatus 更新生产者状态
func (s *MessageStore) UpdateProducerStatus(productID int, lastAlive int64, subscribed int) error {
	query := `
		INSERT INTO producer_status (product_id, status, last_alive, subscribed, updated_at)
		VALUES ($1, 'online', $2, $3, $4)
		ON CONFLICT (product_id) 
		DO UPDATE SET 
			status = 'online',
			last_alive = $2,
			subscribed = $3,
			updated_at = $4
	`

	_, err := s.db.Exec(query, productID, lastAlive, subscribed, time.Now())
	return err
}

// UpdateTrackedEvent 更新跟踪的赛事
func (s *MessageStore) UpdateTrackedEvent(eventID string) error {
	query := `
		INSERT INTO tracked_events (event_id, message_count, last_message_at, updated_at)
		VALUES ($1, 1, $2, $2)
		ON CONFLICT (event_id)
		DO UPDATE SET
			message_count = tracked_events.message_count + 1,
			last_message_at = $2,
			updated_at = $2
	`

	_, err := s.db.Exec(query, eventID, time.Now())
	return err
}

// GetMessages 获取消息列表
func (s *MessageStore) GetMessages(limit, offset int, eventID, messageType string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, message_type, event_id, product_id, sport_id, routing_key, 
		       xml_content, timestamp, received_at, created_at
		FROM uof_messages
		WHERE ($1 = '' OR event_id = $1)
		  AND ($2 = '' OR message_type = $2)
		ORDER BY received_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.db.Query(query, eventID, messageType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []map[string]interface{}
	for rows.Next() {
		var (
			id          int64
			msgType     string
			evtID       sql.NullString
			prodID      sql.NullInt64
			sptID       sql.NullString
			routingKey  string
			xmlContent  string
			timestamp   sql.NullInt64
			receivedAt  time.Time
			createdAt   time.Time
		)

		if err := rows.Scan(&id, &msgType, &evtID, &prodID, &sptID, &routingKey, &xmlContent, &timestamp, &receivedAt, &createdAt); err != nil {
			return nil, err
		}

		msg := map[string]interface{}{
			"id":           id,
			"message_type": msgType,
			"routing_key":  routingKey,
			"xml_content":  xmlContent,
			"received_at":  receivedAt,
			"created_at":   createdAt,
		}

		if evtID.Valid {
			msg["event_id"] = evtID.String
		}
		if prodID.Valid {
			msg["product_id"] = prodID.Int64
		}
		if sptID.Valid {
			msg["sport_id"] = sptID.String
		}
		if timestamp.Valid {
			msg["timestamp"] = timestamp.Int64
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// GetTrackedEvents 获取跟踪的赛事列表
func (s *MessageStore) GetTrackedEvents() ([]map[string]interface{}, error) {
	query := `
		SELECT id, event_id, sport_id, status, message_count, last_message_at, created_at, updated_at
		FROM tracked_events
		WHERE status = 'active'
		ORDER BY last_message_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []map[string]interface{}
	for rows.Next() {
		var (
			id            int64
			eventID       string
			sportID       sql.NullString
			status        string
			messageCount  int
			lastMessageAt sql.NullTime
			createdAt     time.Time
			updatedAt     time.Time
		)

		if err := rows.Scan(&id, &eventID, &sportID, &status, &messageCount, &lastMessageAt, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		event := map[string]interface{}{
			"id":            id,
			"event_id":      eventID,
			"status":        status,
			"message_count": messageCount,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		}

		if sportID.Valid {
			event["sport_id"] = sportID.String
		}
		if lastMessageAt.Valid {
			event["last_message_at"] = lastMessageAt.Time
		}

		events = append(events, event)
	}

	return events, nil
}

// GetEventMessages 获取特定赛事的所有消息
func (s *MessageStore) GetEventMessages(eventID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, message_type, routing_key, xml_content, timestamp, received_at
		FROM uof_messages
		WHERE event_id = $1
		ORDER BY received_at ASC
	`

	rows, err := s.db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []map[string]interface{}
	for rows.Next() {
		var (
			id         int64
			msgType    string
			routingKey string
			xmlContent string
			timestamp  sql.NullInt64
			receivedAt time.Time
		)

		if err := rows.Scan(&id, &msgType, &routingKey, &xmlContent, &timestamp, &receivedAt); err != nil {
			return nil, err
		}

		msg := map[string]interface{}{
			"id":           id,
			"message_type": msgType,
			"routing_key":  routingKey,
			"xml_content":  xmlContent,
			"received_at":  receivedAt,
		}

		if timestamp.Valid {
			msg["timestamp"] = timestamp.Int64
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

