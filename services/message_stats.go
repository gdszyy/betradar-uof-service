package services

import (
"uof-service/logger"
	"fmt"
	"sync"
	"time"
)

// MessageStatsTracker 消息统计追踪器
type MessageStatsTracker struct {
	mu           sync.RWMutex
	stats        map[string]int
	totalCount   int
	lastReported time.Time
	notifier     *LarkNotifier
	interval     time.Duration
	firstReport  bool
}

// NewMessageStatsTracker 创建消息统计追踪器
func NewMessageStatsTracker(notifier *LarkNotifier, interval time.Duration) *MessageStatsTracker {
	return &MessageStatsTracker{
		stats:        make(map[string]int),
		lastReported: time.Now(),
		notifier:     notifier,
		interval:     interval,
		firstReport:  true,
	}
}

// Record 记录一条消息
func (t *MessageStatsTracker) Record(messageType string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.stats[messageType]++
	t.totalCount++
}

// CheckAndReport 检查是否需要报告
func (t *MessageStatsTracker) CheckAndReport() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	now := time.Now()
	elapsed := now.Sub(t.lastReported)
	
	// 第一次立即报告,之后按间隔报告
	if t.firstReport || elapsed >= t.interval {
		if t.totalCount > 0 {
			// 复制当前统计
			statsCopy := make(map[string]int)
			for k, v := range t.stats {
				statsCopy[k] = v
			}
			
			period := "启动至今"
			if !t.firstReport {
				period = fmt.Sprintf("过去 %.0f 分钟", elapsed.Minutes())
			}
			
			// 异步发送通知
			go func() {
				if err := t.notifier.NotifyMessageStats(statsCopy, t.totalCount, period); err != nil {
					logger.Printf("[MessageStats] Failed to send notification: %v", err)
				}
			}()
			
			// 重置统计(除了第一次)
			if !t.firstReport {
				t.stats = make(map[string]int)
				t.totalCount = 0
			}
			
			t.lastReported = now
			t.firstReport = false
		}
	}
}

// StartPeriodicReport 启动定期报告
func (t *MessageStatsTracker) StartPeriodicReport() {
	ticker := time.NewTicker(30 * time.Second) // 每30秒检查一次
	defer ticker.Stop()
	
	for range ticker.C {
		t.CheckAndReport()
	}
}

