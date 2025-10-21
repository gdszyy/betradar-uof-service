package services

import (
	"log"
	"sync"
	"time"
)

// MessageStats 消息统计
type MessageStats struct {
	mu                sync.RWMutex
	startTime         time.Time
	totalMessages     int64
	messagesByType    map[string]int64
	lastMessageTime   map[string]time.Time
	messagesLastHour  map[string]int64
	hourlyResetTime   time.Time
}

// Monitor 监控服务
type Monitor struct {
	stats            *MessageStats
	alertThresholds  map[string]int // 每小时最小消息数阈值
	checkInterval    time.Duration
	stopChan         chan struct{}
}

// NewMonitor 创建监控服务
func NewMonitor() *Monitor {
	return &Monitor{
		stats: &MessageStats{
			startTime:        time.Now(),
			messagesByType:   make(map[string]int64),
			lastMessageTime:  make(map[string]time.Time),
			messagesLastHour: make(map[string]int64),
			hourlyResetTime:  time.Now(),
		},
		alertThresholds: map[string]int{
			"odds_change":      10,  // 期望每小时至少10条odds_change
			"bet_stop":         5,   // 期望每小时至少5条bet_stop
			"fixture_change":   50,  // 期望每小时至少50条fixture_change
			"alive":            100, // 期望每小时至少100条alive
		},
		checkInterval: 5 * time.Minute,
		stopChan:      make(chan struct{}),
	}
}

// RecordMessage 记录消息
func (m *Monitor) RecordMessage(messageType string) {
	m.stats.mu.Lock()
	defer m.stats.mu.Unlock()
	
	now := time.Now()
	
	// 更新总计数
	m.stats.totalMessages++
	m.stats.messagesByType[messageType]++
	m.stats.lastMessageTime[messageType] = now
	
	// 更新小时计数
	m.stats.messagesLastHour[messageType]++
	
	// 检查是否需要重置小时计数
	if now.Sub(m.stats.hourlyResetTime) >= time.Hour {
		m.resetHourlyStats()
	}
}

// resetHourlyStats 重置小时统计
func (m *Monitor) resetHourlyStats() {
	m.stats.messagesLastHour = make(map[string]int64)
	m.stats.hourlyResetTime = time.Now()
}

// Start 启动监控
func (m *Monitor) Start() {
	log.Println("📊 Starting message monitor...")
	
	// 启动定期检查
	go m.periodicCheck()
	
	// 启动每小时报告
	go m.hourlyReport()
}

// Stop 停止监控
func (m *Monitor) Stop() {
	close(m.stopChan)
}

// periodicCheck 定期检查
func (m *Monitor) periodicCheck() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.checkAlerts()
		case <-m.stopChan:
			return
		}
	}
}

// hourlyReport 每小时报告
func (m *Monitor) hourlyReport() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.printHourlyReport()
		case <-m.stopChan:
			return
		}
	}
}

// checkAlerts 检查告警
func (m *Monitor) checkAlerts() {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()
	
	now := time.Now()
	
	// 检查关键消息类型
	for msgType, threshold := range m.alertThresholds {
		count := m.stats.messagesLastHour[msgType]
		lastTime, exists := m.stats.lastMessageTime[msgType]
		
		// 告警1: 消息数低于阈值
		if count < int64(threshold) {
			if msgType == "odds_change" && count == 0 {
				log.Printf("⚠️  ALERT: No odds_change messages received in the last hour!")
				log.Printf("   Possible reasons:")
				log.Printf("   1. No booked matches (booked=0)")
				log.Printf("   2. No live matches currently")
				log.Printf("   3. Account permission issue")
			} else if count > 0 {
				log.Printf("⚠️  WARNING: %s messages below threshold (received: %d, expected: %d)",
					msgType, count, threshold)
			}
		}
		
		// 告警2: 长时间没有收到消息
		if exists {
			timeSince := now.Sub(lastTime)
			if timeSince > 10*time.Minute {
				log.Printf("⚠️  WARNING: No %s messages for %v", msgType, timeSince.Round(time.Second))
			}
		}
	}
}

// printHourlyReport 打印小时报告
func (m *Monitor) printHourlyReport() {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()
	
	log.Println("\n" + "═══════════════════════════════════════════════════════════")
	log.Println("📊 HOURLY MESSAGE REPORT")
	log.Println("═══════════════════════════════════════════════════════════")
	
	// 运行时间
	uptime := time.Since(m.stats.startTime)
	log.Printf("⏱️  Uptime: %v", uptime.Round(time.Second))
	
	// 总消息数
	log.Printf("📨 Total messages: %d", m.stats.totalMessages)
	
	// 按类型统计
	log.Println("\n📋 Messages by type (last hour):")
	
	// 关键消息类型
	keyTypes := []string{"odds_change", "bet_stop", "bet_settlement", "fixture_change", "alive"}
	for _, msgType := range keyTypes {
		count := m.stats.messagesLastHour[msgType]
		totalCount := m.stats.messagesByType[msgType]
		lastTime, exists := m.stats.lastMessageTime[msgType]
		
		status := "✅"
		if msgType == "odds_change" && count == 0 {
			status = "❌"
		} else if threshold, ok := m.alertThresholds[msgType]; ok && count < int64(threshold) {
			status = "⚠️ "
		}
		
		if exists {
			timeSince := time.Since(lastTime)
			log.Printf("  %s %-20s: %4d (total: %d, last: %v ago)",
				status, msgType, count, totalCount, timeSince.Round(time.Second))
		} else {
			log.Printf("  %s %-20s: %4d (total: %d, never received)",
				status, msgType, count, totalCount)
		}
	}
	
	// 其他消息类型
	log.Println("\n📋 Other message types:")
	for msgType, count := range m.stats.messagesLastHour {
		// 跳过已经显示的关键类型
		isKey := false
		for _, key := range keyTypes {
			if msgType == key {
				isKey = true
				break
			}
		}
		if !isKey {
			totalCount := m.stats.messagesByType[msgType]
			log.Printf("  %-20s: %4d (total: %d)", msgType, count, totalCount)
		}
	}
	
	// 特别关注odds_change
	if m.stats.messagesLastHour["odds_change"] == 0 {
		log.Println("\n⚠️  ODDS_CHANGE ALERT:")
		log.Println("   No odds_change messages received in the last hour!")
		log.Println("   This is likely because:")
		log.Println("   1. No matches are booked (booked=0)")
		log.Println("   2. No live matches currently")
		log.Println("   3. Account doesn't have odds feed permission")
	}
	
	log.Println("═══════════════════════════════════════════════════════════\n")
}

// GetStats 获取统计信息
func (m *Monitor) GetStats() map[string]interface{} {
	m.stats.mu.RLock()
	defer m.stats.mu.RUnlock()
	
	// 复制数据
	messagesByType := make(map[string]int64)
	for k, v := range m.stats.messagesByType {
		messagesByType[k] = v
	}
	
	messagesLastHour := make(map[string]int64)
	for k, v := range m.stats.messagesLastHour {
		messagesLastHour[k] = v
	}
	
	lastMessageTime := make(map[string]string)
	for k, v := range m.stats.lastMessageTime {
		lastMessageTime[k] = v.Format(time.RFC3339)
	}
	
	return map[string]interface{}{
		"uptime_seconds":      time.Since(m.stats.startTime).Seconds(),
		"total_messages":      m.stats.totalMessages,
		"messages_by_type":    messagesByType,
		"messages_last_hour":  messagesLastHour,
		"last_message_time":   lastMessageTime,
		"hourly_reset_time":   m.stats.hourlyResetTime.Format(time.RFC3339),
	}
}

