package services

import (
"uof-service/logger"
	"bytes"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	"uof-service/config"
)

// AutoBookingService 自动订阅服务
type AutoBookingService struct {
	config              *config.Config
	client              *http.Client
	db                  *sql.DB
	larkNotifier        *LarkNotifier
	subscriptionTracker *UOFSubscriptionTracker
}

// NewAutoBookingService 创建自动订阅服务
func NewAutoBookingService(cfg *config.Config, db *sql.DB, notifier *LarkNotifier) *AutoBookingService {
	return &AutoBookingService{
		config:       cfg,
		client:       &http.Client{Timeout: 30 * time.Second},
		db:           db,
		larkNotifier: notifier,
	}
}

// SetSubscriptionManager removed - no longer using subscription manager

// BookMatch 订阅单个比赛
func (s *AutoBookingService) BookMatch(matchID string) error {
	// API: POST /liveodds/booking-calendar/events/{id}/book
	url := fmt.Sprintf("%s/liveodds/booking-calendar/events/%s/book", s.config.APIBaseURL, matchID)
	
	logger.Printf("[AutoBooking] 📝 Booking match: %s", matchID)
	logger.Printf("[AutoBooking] 📤 API URL: %s", url)
	
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	req.Header.Set("Content-Type", "application/xml")
	
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("booking failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	logger.Printf("[AutoBooking] ✅ Match booked successfully: %s", matchID)
	logger.Printf("[AutoBooking] Response: %s", string(body))
	
	// 更新数据库中的订阅状态
	if s.db != nil {
		_, err = s.db.Exec(
			"UPDATE tracked_events SET subscribed = true, live_odds = 'booked', updated_at = $1 WHERE event_id = $2",
			time.Now(), matchID,
		)
		if err != nil {
			logger.Printf("[AutoBooking] ⚠️  Failed to update subscribed status: %v", err)
		}
	}
	
	return nil
}

// BookAllBookableMatches 查询并自动订阅所有可订阅的比赛
func (s *AutoBookingService) BookAllBookableMatches() (int, int, error) {
		logger.Println("[AutoBooking] 🔍 Querying live schedule for bookable matches...")
	
	// Subscription manager removed - cleanup handled elsewhere
	
	// 查询当前直播赛程
	url := fmt.Sprintf("%s/sports/en/schedules/live/schedule.xml", s.config.APIBaseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query schedule: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("schedule query failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析 XML
	type SportEvent struct {
		ID       string `xml:"id,attr"`
		LiveOdds string `xml:"liveodds,attr"`
		Status   string `xml:"status,attr"`
	}
	
	type Schedule struct {
		SportEvents []SportEvent `xml:"sport_event"`
	}
	
	var schedule Schedule
	if err := xml.Unmarshal(body, &schedule); err != nil {
		return 0, 0, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	logger.Printf("[AutoBooking] 📊 Found %d live matches", len(schedule.SportEvents))
	
	// 统计和订阅
	bookableCount := 0
	successCount := 0
	failedCount := 0
	
	var bookableMatches []string
	
	for _, event := range schedule.SportEvents {
		if event.LiveOdds == "bookable" {
			bookableCount++
			bookableMatches = append(bookableMatches, event.ID)
		}
	}
	
	logger.Printf("[AutoBooking] 🎯 Found %d bookable matches", bookableCount)
	
	if bookableCount == 0 {
		logger.Println("[AutoBooking] ℹ️  No bookable matches found")
		return 0, 0, nil
	}
	
	logger.Printf("[AutoBooking] 🚀 Auto-booking enabled: will subscribe all %d bookable matches", bookableCount)
	
	// 订阅所有 bookable 比赛
	for _, matchID := range bookableMatches {
		if err := s.BookMatch(matchID); err != nil {
			logger.Printf("[AutoBooking] ❌ Failed to book %s: %v", matchID, err)
			failedCount++
		} else {
			successCount++
		}
		
		// 避免请求过快
		time.Sleep(500 * time.Millisecond)
	}
	
	logger.Printf("[AutoBooking] 📈 Booking summary: %d success, %d failed out of %d bookable", 
		successCount, failedCount, bookableCount)
	
	// 发送飞书通知
	if s.larkNotifier != nil {
		s.sendBookingReport(bookableCount, successCount, failedCount)
	}
	
	return bookableCount, successCount, nil
}

// sendBookingReport 发送订阅报告到飞书
func (s *AutoBookingService) sendBookingReport(bookable, success, failed int) {
	var buffer bytes.Buffer
	
	buffer.WriteString("📊 **自动订阅报告**\n\n")
	buffer.WriteString(fmt.Sprintf("🔍 发现可订阅比赛: **%d** 场\n", bookable))
	buffer.WriteString(fmt.Sprintf("✅ 订阅成功: **%d** 场\n", success))
	
	if failed > 0 {
		buffer.WriteString(fmt.Sprintf("❌ 订阅失败: **%d** 场\n", failed))
	}
	
	buffer.WriteString(fmt.Sprintf("\n⏰ 时间: %s", time.Now().Format("2006-01-02 15:04:05")))
	
	s.larkNotifier.SendText(buffer.String())
}

// cleanupEndedMatches and sendCleanupReport removed - subscription manager no longer used

