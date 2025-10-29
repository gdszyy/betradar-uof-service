package services

import (
	"database/sql"
	"sync"
	"time"
)

// UOFSubscriptionTracker 跟踪 UOF 订阅的比赛状态
type UOFSubscriptionTracker struct {
	db           *sql.DB
	mu           sync.RWMutex
	subscriptions map[string]*UOFSubscription
}

// UOFSubscription UOF 订阅信息
type UOFSubscription struct {
	EventID      string
	BookedAt     time.Time
	LastEventAt  time.Time
	MatchStatus  string // not_started, live, ended, closed
	EventCount   int
}

// NewUOFSubscriptionTracker 创建 UOF 订阅跟踪器
func NewUOFSubscriptionTracker(db *sql.DB) *UOFSubscriptionTracker {
	tracker := &UOFSubscriptionTracker{
		db:            db,
		subscriptions: make(map[string]*UOFSubscription),
	}
	
	// 从数据库加载已订阅的比赛
	tracker.loadFromDatabase()
	
	return tracker
}

// loadFromDatabase 从数据库加载已订阅的比赛
func (t *UOFSubscriptionTracker) loadFromDatabase() {
	// 查询最近 24 小时内有事件的比赛
	query := `
		SELECT DISTINCT event_id, 
		       MIN(timestamp) as first_event,
		       MAX(timestamp) as last_event,
		       COUNT(*) as event_count
		FROM tracked_events
		WHERE timestamp > $1
		GROUP BY event_id
		ORDER BY last_event DESC
	`
	
	since := time.Now().Add(-24 * time.Hour)
	rows, err := t.db.Query(query, since.Unix())
	if err != nil {
		logger.Printf("[UOFTracker] Failed to load subscriptions from database: %v", err)
		return
	}
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		var eventID string
		var firstEvent, lastEvent int64
		var eventCount int
		
		if err := rows.Scan(&eventID, &firstEvent, &lastEvent, &eventCount); err != nil {
			continue
		}
		
		t.subscriptions[eventID] = &UOFSubscription{
			EventID:     eventID,
			BookedAt:    time.Unix(firstEvent, 0),
			LastEventAt: time.Unix(lastEvent, 0),
			EventCount:  eventCount,
			MatchStatus: "unknown",
		}
		count++
	}
	
	logger.Printf("[UOFTracker] Loaded %d active subscriptions from database", count)
}

// AddSubscription 添加订阅记录
func (t *UOFSubscriptionTracker) AddSubscription(eventID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	if _, exists := t.subscriptions[eventID]; exists {
		return
	}
	
	t.subscriptions[eventID] = &UOFSubscription{
		EventID:     eventID,
		BookedAt:    time.Now(),
		LastEventAt: time.Now(),
		MatchStatus: "not_started",
	}
	
	logger.Printf("[UOFTracker] ✅ Added subscription for event %s", eventID)
}

// UpdateMatchStatus 更新比赛状态
func (t *UOFSubscriptionTracker) UpdateMatchStatus(eventID, status string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	sub, exists := t.subscriptions[eventID]
	if !exists {
		// 如果不存在,创建一个
		sub = &UOFSubscription{
			EventID:     eventID,
			BookedAt:    time.Now(),
			LastEventAt: time.Now(),
		}
		t.subscriptions[eventID] = sub
	}
	
	oldStatus := sub.MatchStatus
	sub.MatchStatus = status
	sub.LastEventAt = time.Now()
	sub.EventCount++
	
	if (status == "ended" || status == "closed") && oldStatus != status {
		logger.Printf("[UOFTracker] 🏁 Event %s ended (status: %s)", eventID, status)
	}
}

// RecordEvent 记录事件
func (t *UOFSubscriptionTracker) RecordEvent(eventID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	sub, exists := t.subscriptions[eventID]
	if !exists {
		return
	}
	
	sub.LastEventAt = time.Now()
	sub.EventCount++
}

// GetEndedEvents 获取已结束的事件列表
func (t *UOFSubscriptionTracker) GetEndedEvents(afterDuration time.Duration) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	var endedEvents []string
	now := time.Now()
	
	for eventID, sub := range t.subscriptions {
		// 比赛已结束且超过指定时间
		if (sub.MatchStatus == "ended" || sub.MatchStatus == "closed") &&
			now.Sub(sub.LastEventAt) > afterDuration {
			endedEvents = append(endedEvents, eventID)
		}
	}
	
	return endedEvents
}

// GetSubscriptions 获取所有订阅
func (t *UOFSubscriptionTracker) GetSubscriptions() []*UOFSubscription {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	subs := make([]*UOFSubscription, 0, len(t.subscriptions))
	for _, sub := range t.subscriptions {
		subs = append(subs, sub)
	}
	
	return subs
}

// RemoveSubscription 移除订阅记录
func (t *UOFSubscriptionTracker) RemoveSubscription(eventID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	delete(t.subscriptions, eventID)
	logger.Printf("[UOFTracker] 🗑️  Removed subscription for event %s", eventID)
}

// GetStats 获取统计信息
func (t *UOFSubscriptionTracker) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total":       len(t.subscriptions),
		"not_started": 0,
		"live":        0,
		"ended":       0,
		"closed":      0,
	}
	
	for _, sub := range t.subscriptions {
		switch sub.MatchStatus {
		case "not_started":
			stats["not_started"] = stats["not_started"].(int) + 1
		case "live":
			stats["live"] = stats["live"].(int) + 1
		case "ended":
			stats["ended"] = stats["ended"].(int) + 1
		case "closed":
			stats["closed"] = stats["closed"].(int) + 1
		}
	}
	
	return stats
}

