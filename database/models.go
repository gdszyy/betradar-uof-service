package database

import (
	"time"
)

// UOFMessage 存储UOF消息
type UOFMessage struct {
	ID          int64     `db:"id"`
	MessageType string    `db:"message_type"`
	EventID     *string   `db:"event_id"`
	ProductID   *int      `db:"product_id"`
	SportID     *string   `db:"sport_id"`
	RoutingKey  string    `db:"routing_key"`
	XMLContent  string    `db:"xml_content"`
	Timestamp   int64     `db:"timestamp"`
	ReceivedAt  time.Time `db:"received_at"`
	CreatedAt   time.Time `db:"created_at"`
}

// TrackedEvent 跟踪的赛事
type TrackedEvent struct {
	ID              int64     `db:"id"`
	EventID         string    `db:"event_id"`
	SportID         *string   `db:"sport_id"`
	Status          string    `db:"status"`
	MessageCount    int       `db:"message_count"`
	LastMessageAt   time.Time `db:"last_message_at"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// OddsChange 赔率变化记录
type OddsChange struct {
	ID              int64     `db:"id"`
	EventID         string    `db:"event_id"`
	ProductID       int       `db:"product_id"`
	Timestamp       int64     `db:"timestamp"`
	OddsChangeReason *string  `db:"odds_change_reason"`
	MarketsCount    int       `db:"markets_count"`
	XMLContent      string    `db:"xml_content"`
	CreatedAt       time.Time `db:"created_at"`
}

// BetStop 投注停止记录
type BetStop struct {
	ID           int64     `db:"id"`
	EventID      string    `db:"event_id"`
	ProductID    int       `db:"product_id"`
	Timestamp    int64     `db:"timestamp"`
	MarketStatus *string   `db:"market_status"`
	XMLContent   string    `db:"xml_content"`
	CreatedAt    time.Time `db:"created_at"`
}

// BetSettlement 投注结算记录
type BetSettlement struct {
	ID         int64     `db:"id"`
	EventID    string    `db:"event_id"`
	ProductID  int       `db:"product_id"`
	Timestamp  int64     `db:"timestamp"`
	Certainty  *int      `db:"certainty"`
	XMLContent string    `db:"xml_content"`
	CreatedAt  time.Time `db:"created_at"`
}

// ProducerStatus 生产者状态
type ProducerStatus struct {
	ID           int64     `db:"id"`
	ProductID    int       `db:"product_id"`
	Status       string    `db:"status"`
	LastAlive    int64     `db:"last_alive"`
	Subscribed   int       `db:"subscribed"`
	UpdatedAt    time.Time `db:"updated_at"`
}


// RecoveryStatus 恢复状态记录
type RecoveryStatus struct {
	ID         int64     `db:"id"`
	RequestID  int       `db:"request_id"`
	ProductID  int       `db:"product_id"`
	NodeID     int       `db:"node_id"`
	Status     string    `db:"status"` // initiated, completed, failed
	Timestamp  int64     `db:"timestamp"`
	CreatedAt  time.Time `db:"created_at"`
	CompletedAt *time.Time `db:"completed_at"`
}

