package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	
	"uof-service/logger"
)

// SubscriptionSyncService 定期同步订阅状态
type SubscriptionSyncService struct {
	db              *sql.DB
	accessToken     string
	apiBaseURL      string
	syncInterval    time.Duration
	stopChan        chan struct{}
	running         bool
}

// NewSubscriptionSyncService 创建订阅同步服务
func NewSubscriptionSyncService(db *sql.DB, accessToken string, apiBaseURL string, syncIntervalMinutes int) *SubscriptionSyncService {
	if syncIntervalMinutes <= 0 {
		syncIntervalMinutes = 5 // 默认 5 分钟
	}
	
	return &SubscriptionSyncService{
		db:           db,
		accessToken:  accessToken,
		apiBaseURL:   apiBaseURL,
		syncInterval: time.Duration(syncIntervalMinutes) * time.Minute,
		stopChan:     make(chan struct{}),
	}
}

// Start 启动订阅同步服务
func (s *SubscriptionSyncService) Start() error {
	if s.running {
		return fmt.Errorf("subscription sync service already running")
	}
	
	s.running = true
	
	// 立即执行一次同步
	go func() {
		logger.Println("[SubscriptionSync] Starting subscription sync service...")
		if err := s.syncSubscriptions(); err != nil {
			logger.Errorf("[SubscriptionSync] Initial sync failed: %v", err)
		}
		
		// 定期同步
		ticker := time.NewTicker(s.syncInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				if err := s.syncSubscriptions(); err != nil {
					logger.Errorf("[SubscriptionSync] Sync failed: %v", err)
				}
			case <-s.stopChan:
				logger.Println("[SubscriptionSync] Stopping subscription sync service...")
				return
			}
		}
	}()
	
	return nil
}

// Stop 停止订阅同步服务
func (s *SubscriptionSyncService) Stop() {
	if !s.running {
		return
	}
	
	close(s.stopChan)
	s.running = false
}

// syncSubscriptions 同步订阅状态
func (s *SubscriptionSyncService) syncSubscriptions() error {
	logger.Println("[SubscriptionSync] Fetching subscriptions from Betradar API...")
	
	// 调用 Betradar API 获取订阅列表
	subscriptions, err := s.fetchSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to fetch subscriptions: %w", err)
	}
	
	logger.Printf("[SubscriptionSync] Found %d subscribed events", len(subscriptions))
	
	// 更新数据库
	if err := s.updateSubscriptions(subscriptions); err != nil {
		return fmt.Errorf("failed to update subscriptions: %w", err)
	}
	
	logger.Printf("[SubscriptionSync] ✅ Successfully synced %d subscriptions", len(subscriptions))
	return nil
}

// fetchSubscriptions 从 Betradar API 获取订阅列表
func (s *SubscriptionSyncService) fetchSubscriptions() ([]string, error) {
	// 构建 API URL
	// 注意: Betradar API 的 subscriptions 端点可能需要 user_id
	// 这里使用通用的端点,如果需要 user_id,需要从配置中获取
	url := fmt.Sprintf("%s/v1/users/whoami.xml", s.apiBaseURL)
	
	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 添加 access token
	req.Header.Set("x-access-token", s.accessToken)
	
	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	// 解析 XML
	// 注意: 这里的 XML 结构需要根据实际 API 响应调整
	type WhoAmI struct {
		BookmakerID int `xml:"bookmaker_id,attr"`
	}
	
	var whoami WhoAmI
	if err := xml.Unmarshal(body, &whoami); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	// 获取订阅列表
	// 使用 bookmaker_id 调用订阅列表 API
	subsURL := fmt.Sprintf("%s/v1/users/bookmakers/%d/subscriptions.xml", s.apiBaseURL, whoami.BookmakerID)
	
	req2, err := http.NewRequest("GET", subsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions request: %w", err)
	}
	
	req2.Header.Set("x-access-token", s.accessToken)
	
	resp2, err := client.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("failed to send subscriptions request: %w", err)
	}
	defer resp2.Body.Close()
	
	if resp2.StatusCode != http.StatusOK {
		body2, _ := io.ReadAll(resp2.Body)
		return nil, fmt.Errorf("subscriptions API returned status %d: %s", resp2.StatusCode, string(body2))
	}
	
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read subscriptions response: %w", err)
	}
	
	// 解析订阅列表 XML
	type Subscriptions struct {
		Subscriptions []struct {
			ID string `xml:"id,attr"`
		} `xml:"subscription"`
	}
	
	var subs Subscriptions
	if err := xml.Unmarshal(body2, &subs); err != nil {
		return nil, fmt.Errorf("failed to parse subscriptions XML: %w", err)
	}
	
	// 提取 event_id 列表
	var eventIDs []string
	for _, sub := range subs.Subscriptions {
		if sub.ID != "" {
			eventIDs = append(eventIDs, sub.ID)
		}
	}
	
	return eventIDs, nil
}

// updateSubscriptions 更新数据库中的订阅状态
func (s *SubscriptionSyncService) updateSubscriptions(eventIDs []string) error {
	if len(eventIDs) == 0 {
		logger.Println("[SubscriptionSync] No subscriptions to update")
		return nil
	}
	
	// 开始事务
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 批量更新订阅状态
	for _, eventID := range eventIDs {
		query := `
			INSERT INTO tracked_events (event_id, subscribed, updated_at)
			VALUES ($1, true, $2)
			ON CONFLICT (event_id)
			DO UPDATE SET
				subscribed = true,
				updated_at = $2
		`
		
		if _, err := tx.Exec(query, eventID, time.Now()); err != nil {
			return fmt.Errorf("failed to update event %s: %w", eventID, err)
		}
	}
	
	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

