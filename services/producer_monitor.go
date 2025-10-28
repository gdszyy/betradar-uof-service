package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// ProducerMonitor 监控 UOF Producer 健康状态
type ProducerMonitor struct {
	db       *sql.DB
	notifier *LarkNotifier
	ticker   *time.Ticker
	done     chan bool
}

// NewProducerMonitor 创建 Producer 监控器
func NewProducerMonitor(db *sql.DB, notifier *LarkNotifier) *ProducerMonitor {
	return &ProducerMonitor{
		db:       db,
		notifier: notifier,
		done:     make(chan bool),
	}
}

// Start 启动监控
func (pm *ProducerMonitor) Start() {
	pm.ticker = time.NewTicker(5 * time.Second)
	
	log.Println("✅ Producer monitor started (checking every 5 seconds)")
	
	go func() {
		for {
			select {
			case <-pm.ticker.C:
				pm.checkProducers()
			case <-pm.done:
				return
			}
		}
	}()
}

// Stop 停止监控
func (pm *ProducerMonitor) Stop() {
	if pm.ticker != nil {
		pm.ticker.Stop()
	}
	close(pm.done)
	log.Println("Producer monitor stopped")
}

// checkProducers 检查所有 Producer 的健康状态
func (pm *ProducerMonitor) checkProducers() {
	query := `
		SELECT product_id, last_alive, subscribed
		FROM producer_status
		WHERE last_alive IS NOT NULL
	`
	
	rows, err := pm.db.Query(query)
	if err != nil {
		log.Printf("[ProducerMonitor] Failed to query producer status: %v", err)
		return
	}
	defer rows.Close()
	
	now := time.Now()
	for rows.Next() {
		var producerID int
		var lastAlive int64  // Unix timestamp in milliseconds
		var subscribed int
		
		if err := rows.Scan(&producerID, &lastAlive, &subscribed); err != nil {
			log.Printf("[ProducerMonitor] Failed to scan producer status: %v", err)
			continue
		}
		
		// 转换为 time.Time (毫秒转秒)
		lastAliveAt := time.Unix(lastAlive/1000, (lastAlive%1000)*1000000)
		
		// 检查是否超过 20 秒没有收到 alive 消息
		timeSinceLastAlive := now.Sub(lastAliveAt)
		if timeSinceLastAlive > 20*time.Second {
			log.Printf("[ProducerMonitor] ⚠️  Producer %d is DOWN (last alive: %v ago)", 
				producerID, timeSinceLastAlive.Round(time.Second))
			
			// 发送告警通知
			pm.sendProducerDownAlert(producerID, timeSinceLastAlive)
		}
	}
}

// sendProducerDownAlert 发送 Producer 下线告警
func (pm *ProducerMonitor) sendProducerDownAlert(producerID int, downTime time.Duration) {
	message := fmt.Sprintf("🚨 UOF Producer %d is DOWN\n\n"+
		"Last alive: %v ago\n"+
		"All markets from this producer should be suspended.",
		producerID,
		downTime.Round(time.Second))
	
	if pm.notifier != nil {
		pm.notifier.SendText(message)
	}
}

// GetProducerStatus 获取所有 Producer 的健康状态
func (pm *ProducerMonitor) GetProducerStatus() ([]ProducerStatus, error) {
	query := `
		SELECT product_id, last_alive, subscribed
		FROM producer_status
		WHERE last_alive IS NOT NULL
		ORDER BY product_id
	`
	
	rows, err := pm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var statuses []ProducerStatus
	now := time.Now()
	
	for rows.Next() {
		var status ProducerStatus
		var lastAlive int64  // Unix timestamp in milliseconds
		
		if err := rows.Scan(&status.ProducerID, &lastAlive, &status.Subscribed); err != nil {
			continue
		}
		
		// 转换为 time.Time (毫秒转秒)
		lastAliveAt := time.Unix(lastAlive/1000, (lastAlive%1000)*1000000)
		
		status.LastAliveAt = lastAliveAt.Format(time.RFC3339)
		status.SecondsSinceLastAlive = int(now.Sub(lastAliveAt).Seconds())
		status.IsHealthy = status.SecondsSinceLastAlive <= 20
		
		statuses = append(statuses, status)
	}
	
	return statuses, nil
}

// ProducerStatus Producer 状态信息
type ProducerStatus struct {
	ProducerID            int    `json:"producer_id"`
	LastAliveAt           string `json:"last_alive_at"`
	SecondsSinceLastAlive int    `json:"seconds_since_last_alive"`
	IsHealthy             bool   `json:"is_healthy"`
	Subscribed            bool   `json:"subscribed"`
}

// CanAcceptBets 检查是否可以接受投注
func (pm *ProducerMonitor) CanAcceptBets() (bool, string) {
	statuses, err := pm.GetProducerStatus()
	if err != nil {
		return false, fmt.Sprintf("Failed to check producer status: %v", err)
	}
	
	// 检查所有 Producer 是否健康
	for _, status := range statuses {
		if !status.IsHealthy {
			return false, fmt.Sprintf("Producer %d is down (%d seconds since last alive)", 
				status.ProducerID, status.SecondsSinceLastAlive)
		}
	}
	
	return true, "All producers are healthy"
}

