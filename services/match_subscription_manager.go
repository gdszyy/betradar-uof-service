package services

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MatchSubscriptionManager 管理比赛订阅生命周期
type MatchSubscriptionManager struct {
	ldClient     *LDClient
	larkNotifier *LarkNotifier
	
	// 已订阅的比赛及其状态
	subscriptions map[string]*MatchSubscription
	mu            sync.RWMutex
	
	// 自动清理配置
	autoCleanup       bool
	cleanupInterval   time.Duration
	cleanupAfterEnded time.Duration // 比赛结束后多久清理
	
	done chan struct{}
}

// MatchSubscription 比赛订阅信息
type MatchSubscription struct {
	MatchID       string
	SubscribedAt  time.Time
	LastEventAt   time.Time
	Status        string // live, ended, closed
	EventCount    int
	AutoUnsubscribe bool
}

// NewMatchSubscriptionManager 创建订阅管理器
func NewMatchSubscriptionManager(ldClient *LDClient, larkNotifier *LarkNotifier) *MatchSubscriptionManager {
	return &MatchSubscriptionManager{
		ldClient:          ldClient,
		larkNotifier:      larkNotifier,
		subscriptions:     make(map[string]*MatchSubscription),
		autoCleanup:       true,
		cleanupInterval:   5 * time.Minute,
		cleanupAfterEnded: 10 * time.Minute, // 比赛结束10分钟后取消订阅
		done:              make(chan struct{}),
	}
}

// Start 启动订阅管理器
func (m *MatchSubscriptionManager) Start() {
	if !m.autoCleanup {
		log.Println("[SubscriptionManager] Auto cleanup is disabled")
		return
	}
	
	log.Printf("[SubscriptionManager] 🚀 Started with cleanup interval: %v", m.cleanupInterval)
	
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.cleanupEndedMatches()
		case <-m.done:
			log.Println("[SubscriptionManager] Stopped")
			return
		}
	}
}

// Stop 停止订阅管理器
func (m *MatchSubscriptionManager) Stop() {
	close(m.done)
}

// AddSubscription 添加订阅记录
func (m *MatchSubscriptionManager) AddSubscription(matchID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.subscriptions[matchID]; exists {
		log.Printf("[SubscriptionManager] Match %s already subscribed", matchID)
		return
	}
	
	m.subscriptions[matchID] = &MatchSubscription{
		MatchID:         matchID,
		SubscribedAt:    time.Now(),
		LastEventAt:     time.Now(),
		Status:          "live",
		EventCount:      0,
		AutoUnsubscribe: true,
	}
	
	log.Printf("[SubscriptionManager] ✅ Added subscription for match %s", matchID)
}

// UpdateMatchStatus 更新比赛状态
func (m *MatchSubscriptionManager) UpdateMatchStatus(matchID, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	sub, exists := m.subscriptions[matchID]
	if !exists {
		return
	}
	
	oldStatus := sub.Status
	sub.Status = status
	sub.LastEventAt = time.Now()
	sub.EventCount++
	
	// 如果比赛状态变为结束
	if (status == "ended" || status == "closed") && oldStatus != status {
		log.Printf("[SubscriptionManager] 🏁 Match %s ended (status: %s)", matchID, status)
		
		// 发送飞书通知
		if m.larkNotifier != nil {
			m.larkNotifier.SendText(fmt.Sprintf(
				"🏁 **比赛结束**\n\n"+
					"比赛ID: %s\n"+
					"状态: %s\n"+
					"事件数: %d\n"+
					"订阅时长: %v\n"+
					"将在 %v 后自动取消订阅",
				matchID,
				status,
				sub.EventCount,
				time.Since(sub.SubscribedAt).Round(time.Second),
				m.cleanupAfterEnded,
			))
		}
	}
}

// RecordEvent 记录事件
func (m *MatchSubscriptionManager) RecordEvent(matchID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if sub, exists := m.subscriptions[matchID]; exists {
		sub.LastEventAt = time.Now()
		sub.EventCount++
	}
}

// UnsubscribeMatch 取消订阅单个比赛
func (m *MatchSubscriptionManager) UnsubscribeMatch(matchID string) error {
	m.mu.Lock()
	sub, exists := m.subscriptions[matchID]
	m.mu.Unlock()
	
	if !exists {
		return fmt.Errorf("match %s not subscribed", matchID)
	}
	
	// 发送取消订阅消息
	msg := fmt.Sprintf(`<matchstop matchid="%s"/>`, matchID)
	
	log.Printf("[SubscriptionManager] 🛑 Unsubscribing from match: %s", matchID)
	
	if err := m.ldClient.sendMessage(msg); err != nil {
		log.Printf("[SubscriptionManager] ❌ Failed to unsubscribe match %s: %v", matchID, err)
		return err
	}
	
	// 从订阅列表中移除
	m.mu.Lock()
	delete(m.subscriptions, matchID)
	m.mu.Unlock()
	
	log.Printf("[SubscriptionManager] ✅ Unsubscribed from match %s (events: %d, duration: %v)",
		matchID, sub.EventCount, time.Since(sub.SubscribedAt).Round(time.Second))
	
	return nil
}

// UnsubscribeMatches 批量取消订阅
func (m *MatchSubscriptionManager) UnsubscribeMatches(matchIDs []string) error {
	if len(matchIDs) == 0 {
		return nil
	}
	
	log.Printf("[SubscriptionManager] 🛑 Batch unsubscribing %d matches", len(matchIDs))
	
	// 构建批量取消订阅消息
	msg := "<matchunsubscription>\n"
	for _, matchID := range matchIDs {
		msg += fmt.Sprintf(`  <match matchid="%s"/>`, matchID) + "\n"
	}
	msg += "</matchunsubscription>"
	
	if err := m.ldClient.sendMessage(msg); err != nil {
		log.Printf("[SubscriptionManager] ❌ Failed to batch unsubscribe: %v", err)
		return err
	}
	
	// 从订阅列表中移除
	m.mu.Lock()
	for _, matchID := range matchIDs {
		delete(m.subscriptions, matchID)
	}
	m.mu.Unlock()
	
	log.Printf("[SubscriptionManager] ✅ Batch unsubscribed %d matches", len(matchIDs))
	
	return nil
}

// cleanupEndedMatches 清理已结束的比赛
func (m *MatchSubscriptionManager) cleanupEndedMatches() {
	m.mu.RLock()
	
	var toUnsubscribe []string
	now := time.Now()
	
	for matchID, sub := range m.subscriptions {
		// 检查是否需要自动取消订阅
		if !sub.AutoUnsubscribe {
			continue
		}
		
		// 比赛已结束且超过清理时间
		if (sub.Status == "ended" || sub.Status == "closed") &&
			now.Sub(sub.LastEventAt) > m.cleanupAfterEnded {
			toUnsubscribe = append(toUnsubscribe, matchID)
		}
		
		// 超过24小时没有事件的订阅也清理
		if now.Sub(sub.LastEventAt) > 24*time.Hour {
			log.Printf("[SubscriptionManager] ⚠️  Match %s inactive for 24h, will unsubscribe", matchID)
			toUnsubscribe = append(toUnsubscribe, matchID)
		}
	}
	
	m.mu.RUnlock()
	
	if len(toUnsubscribe) == 0 {
		return
	}
	
	log.Printf("[SubscriptionManager] 🧹 Cleaning up %d ended matches", len(toUnsubscribe))
	
	// 批量取消订阅
	if err := m.UnsubscribeMatches(toUnsubscribe); err != nil {
		log.Printf("[SubscriptionManager] ❌ Cleanup failed: %v", err)
		return
	}
	
	// 发送飞书通知
	if m.larkNotifier != nil {
		m.larkNotifier.SendText(fmt.Sprintf(
			"🧹 **自动清理订阅**\n\n"+
				"已取消 %d 个已结束比赛的订阅\n"+
				"释放订阅名额,避免达到上限",
			len(toUnsubscribe),
		))
	}
}

// GetSubscriptionStats 获取订阅统计
func (m *MatchSubscriptionManager) GetSubscriptionStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total":  len(m.subscriptions),
		"live":   0,
		"ended":  0,
		"closed": 0,
	}
	
	for _, sub := range m.subscriptions {
		switch sub.Status {
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

// GetSubscriptions 获取所有订阅
func (m *MatchSubscriptionManager) GetSubscriptions() []*MatchSubscription {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	subs := make([]*MatchSubscription, 0, len(m.subscriptions))
	for _, sub := range m.subscriptions {
		subs = append(subs, sub)
	}
	
	return subs
}

// SetAutoCleanup 设置自动清理
func (m *MatchSubscriptionManager) SetAutoCleanup(enabled bool) {
	m.autoCleanup = enabled
	log.Printf("[SubscriptionManager] Auto cleanup: %v", enabled)
}

// SetCleanupInterval 设置清理间隔
func (m *MatchSubscriptionManager) SetCleanupInterval(interval time.Duration) {
	m.cleanupInterval = interval
	log.Printf("[SubscriptionManager] Cleanup interval: %v", interval)
}

// SetCleanupAfterEnded 设置比赛结束后清理时间
func (m *MatchSubscriptionManager) SetCleanupAfterEnded(duration time.Duration) {
	m.cleanupAfterEnded = duration
	log.Printf("[SubscriptionManager] Cleanup after ended: %v", duration)
}

