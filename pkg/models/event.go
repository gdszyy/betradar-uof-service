package models

import "time"

// Event 统一的事件模型
type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Source      string                 `json:"source"` // uof, ld, thesports
	Timestamp   time.Time              `json:"timestamp"`
	MatchID     string                 `json:"match_id"`
	SportID     string                 `json:"sport_id"`
	Data        map[string]interface{} `json:"data"`
	RawData     []byte                 `json:"raw_data,omitempty"`
	ProcessedAt time.Time              `json:"processed_at,omitempty"`
}

// EventType 事件类型
type EventType string

const (
	EventTypeOddsChange     EventType = "odds_change"
	EventTypeBetStop        EventType = "bet_stop"
	EventTypeBetSettlement  EventType = "bet_settlement"
	EventTypeMatchStatus    EventType = "match_status"
	EventTypeMatchInfo      EventType = "match_info"
	EventTypeScore          EventType = "score"
	EventTypeStatistics     EventType = "statistics"
)

// EventSource 事件来源
type EventSource string

const (
	EventSourceUOF        EventSource = "uof"
	EventSourceLiveData   EventSource = "livedata"
	EventSourceTheSports  EventSource = "thesports"
)

