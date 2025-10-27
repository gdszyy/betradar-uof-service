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

// ScheduleSportEvent schedule API 返回的比赛结构
type ScheduleSportEvent struct {
	ID        string `xml:"id,attr"`
	Scheduled string `xml:"scheduled,attr"`
	LiveOdds  string `xml:"liveodds,attr"` // booked, bookable, buyable, unavailable
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
	for _, matchID := range bookedMatches {
		bookedMap[matchID] = true
	}
	
	// 3. 获取数据库中所有标记为 subscribed = true 的比赛
	rows, err := s.db.Query("SELECT event_id FROM tracked_events WHERE subscribed = true")
	if err != nil {
		return nil, fmt.Errorf("failed to query database: %w", err)
	}
	defer rows.Close()
	
	var dbBookedMatches []string
	for rows.Next() {
		var eventID string
		if err := rows.Scan(&eventID); err != nil {
			continue
		}
		dbBookedMatches = append(dbBookedMatches, eventID)
	}
	
	log.Printf("[SubscriptionSync] 📊 Found %d subscribed matches in database", len(dbBookedMatches))
	
	// 4. 将数据库中已订阅但 API 中未订阅的比赛设置为 false
	for _, eventID := range dbBookedMatches {
		if !bookedMap[eventID] {
			_, err := s.db.Exec(
				"UPDATE tracked_events SET subscribed = false, updated_at = $1 WHERE event_id = $2",
				time.Now(), eventID,
			)
			if err != nil {
				log.Printf("[SubscriptionSync] ⚠️  Failed to update %s to false: %v", eventID, err)
				result.Failed++
			} else {
				result.UpdatedToFalse++
			}
		}
	}
	
	// 5. 将 API 中已订阅的比赛设置为 true
	for matchID := range bookedMap {
		res, err := s.db.Exec(
			"UPDATE tracked_events SET subscribed = true, updated_at = $1 WHERE event_id = $2",
			time.Now(), matchID,
		)
		
		if err != nil {
			log.Printf("[SubscriptionSync] ⚠️  Failed to update %s to true: %v", matchID, err)
			result.Failed++
			continue
		}
		
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected > 0 {
			result.UpdatedToTrue++
		} else {
			result.NotFound++
			log.Printf("[SubscriptionSync] ⚠️  Event not found in database: %s", matchID)
		}
	}
	
	log.Printf("[SubscriptionSync] 📈 Sync completed: %d updated to true, %d updated to false, %d not found, %d failed", 
		result.UpdatedToTrue, result.UpdatedToFalse, result.NotFound, result.Failed)
	
	return result, nil
}

// queryBetradarBookedMatches 从 Betradar API 查询已订阅的比赛
// 使用 schedule API,过滤 liveodds="booked" 的比赛
func (s *SubscriptionSyncService) queryBetradarBookedMatches() ([]string, error) {
	// 查询今天和明天的 schedule
	today := time.Now().UTC().Format("2006-01-02")
	tomorrow := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02")
	
	var allBookedMatches []string
	
	// 查询今天的 schedule
	todayMatches, err := s.queryScheduleForDate(today)
	if err != nil {
		log.Printf("[SubscriptionSync] ⚠️  Failed to query schedule for %s: %v", today, err)
	} else {
		allBookedMatches = append(allBookedMatches, todayMatches...)
	}
	
	// 查询明天的 schedule
	tomorrowMatches, err := s.queryScheduleForDate(tomorrow)
	if err != nil {
		log.Printf("[SubscriptionSync] ⚠️  Failed to query schedule for %s: %v", tomorrow, err)
	} else {
		allBookedMatches = append(allBookedMatches, tomorrowMatches...)
	}
	
	// 查询 live schedule
	liveMatches, err := s.queryLiveSchedule()
	if err != nil {
		log.Printf("[SubscriptionSync] ⚠️  Failed to query live schedule: %v", err)
	} else {
		allBookedMatches = append(allBookedMatches, liveMatches...)
	}
	
	if len(allBookedMatches) == 0 {
		return nil, fmt.Errorf("no booked matches found in any schedule")
	}
	
	// 去重
	uniqueMatches := make(map[string]bool)
	for _, matchID := range allBookedMatches {
		uniqueMatches[matchID] = true
	}
	
	result := make([]string, 0, len(uniqueMatches))
	for matchID := range uniqueMatches {
		result = append(result, matchID)
	}
	
	return result, nil
}

// queryScheduleForDate 查询指定日期的 schedule
func (s *SubscriptionSyncService) queryScheduleForDate(date string) ([]string, error) {
	url := fmt.Sprintf("%s/sports/en/schedules/%s/schedule.xml", s.config.APIBaseURL, date)
	return s.queryScheduleFromURL(url)
}

// queryLiveSchedule 查询 live schedule
func (s *SubscriptionSyncService) queryLiveSchedule() ([]string, error) {
	url := fmt.Sprintf("%s/sports/en/schedules/live/schedule.xml", s.config.APIBaseURL)
	return s.queryScheduleFromURL(url)
}

// queryScheduleFromURL 从指定 URL 查询 schedule 并过滤 booked 比赛
func (s *SubscriptionSyncService) queryScheduleFromURL(url string) ([]string, error) {
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
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	type Schedule struct {
		SportEvents []ScheduleSportEvent `xml:"sport_event"`
	}
	
	var schedule Schedule
	if err := xml.Unmarshal(body, &schedule); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	// 过滤 liveodds="booked" 的比赛
	var bookedMatches []string
	for _, event := range schedule.SportEvents {
		if event.LiveOdds == "booked" {
			bookedMatches = append(bookedMatches, event.ID)
		}
	}
	
	log.Printf("[SubscriptionSync] ✅ Found %d booked matches in %s", len(bookedMatches), url)
	
	return bookedMatches, nil
}

