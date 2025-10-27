package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"uof-service/config"
)

// SubscriptionSyncService 订阅同步服务
// 用于同步 Betradar API 的订阅状态到本地数据库
type SubscriptionSyncService struct {
	config *config.Config
	db     *sql.DB
	client *http.Client
}

// NewSubscriptionSyncService 创建订阅同步服务
func NewSubscriptionSyncService(cfg *config.Config, db *sql.DB) *SubscriptionSyncService {
	return &SubscriptionSyncService{
		config: cfg,
		db:     db,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SyncResult 同步结果
type SyncResult struct {
	TotalBooked    int
	UpdatedToTrue  int
	UpdatedToFalse int
	NotFound       int
	Failed         int
}

// ExecuteSync 执行同步
func (s *SubscriptionSyncService) ExecuteSync() (*SyncResult, error) {
	log.Println("[SubscriptionSync] 🔄 Starting subscription synchronization...")
	
	result := &SyncResult{}
	
	// 1. 从 Betradar API 获取所有已订阅的比赛
	bookedMatches, err := s.queryBetradarBookedMatches()
	if err != nil {
		return nil, fmt.Errorf("failed to query Betradar booked matches: %w", err)
	}
	
	result.TotalBooked = len(bookedMatches)
	log.Printf("[SubscriptionSync] 📊 Found %d booked matches from Betradar API", result.TotalBooked)
	
	// 2. 创建已订阅比赛的 map
	bookedMap := make(map[string]bool)
	for _, match := range bookedMatches {
		bookedMap[match.ID] = true
	}
	
	// 3. 更新数据库中所有比赛的订阅状态
	// 首先将所有比赛设置为 subscribed = false
	_, err = s.db.Exec("UPDATE tracked_events SET subscribed = false WHERE subscribed = true")
	if err != nil {
		return nil, fmt.Errorf("failed to reset subscribed status: %w", err)
	}
	
	// 然后将 Betradar API 中已订阅的比赛设置为 subscribed = true
	for eventID := range bookedMap {
		res, err := s.db.Exec(
			"UPDATE tracked_events SET subscribed = true, updated_at = $1 WHERE event_id = $2",
			time.Now(), eventID,
		)
		
		if err != nil {
			log.Printf("[SubscriptionSync] ⚠️  Failed to update %s: %v", eventID, err)
			result.Failed++
			continue
		}
		
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected > 0 {
			result.UpdatedToTrue++
		} else {
			result.NotFound++
			log.Printf("[SubscriptionSync] ⚠️  Event not found in database: %s", eventID)
		}
	}
	
	log.Printf("[SubscriptionSync] 📈 Sync completed: %d updated to true, %d not found, %d failed", 
		result.UpdatedToTrue, result.NotFound, result.Failed)
	
	return result, nil
}

// queryBetradarBookedMatches 从 Betradar API 查询已订阅的比赛
func (s *SubscriptionSyncService) queryBetradarBookedMatches() ([]BookedMatch, error) {
	// 尝试多个可能的 API 路径
	urls := []string{
		fmt.Sprintf("%s/liveodds/booking-calendar/events/booked.xml", s.config.APIBaseURL),
		fmt.Sprintf("%s/liveodds/booking-calendar/booked.xml", s.config.APIBaseURL),
	}
	
	var lastErr error
	for _, url := range urls {
		matches, err := s.queryBookedMatchesFromURL(url)
		if err == nil {
			return matches, nil
		}
		lastErr = err
		log.Printf("[SubscriptionSync] ⚠️  Failed to query %s: %v", url, err)
	}
	
	return nil, fmt.Errorf("all API endpoints failed, last error: %w", lastErr)
}

// queryBookedMatchesFromURL 从指定 URL 查询已订阅的比赛
func (s *SubscriptionSyncService) queryBookedMatchesFromURL(url string) ([]BookedMatch, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	type BookingCalendar struct {
		SportEvents []BookedMatch `xml:"sport_event"`
	}
	
	var calendar BookingCalendar
	if err := xml.Unmarshal(body, &calendar); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	return calendar.SportEvents, nil
}

