package services

import (
"uof-service/logger"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	"uof-service/config"
)

// StartupBookingService 启动时自动订阅服务
type StartupBookingService struct {
	config       *config.Config
	db           *sql.DB
	client       *http.Client
	larkNotifier *LarkNotifier
}

// NewStartupBookingService 创建启动订阅服务
func NewStartupBookingService(cfg *config.Config, db *sql.DB, notifier *LarkNotifier) *StartupBookingService {
	return &StartupBookingService{
		config:       cfg,
		db:           db,
		client:       &http.Client{Timeout: 30 * time.Second},
		larkNotifier: notifier,
	}
}

// BookingResult 订阅结果
type BookingResult struct {
	TotalLive     int
	Bookable      int
	Success       int
	Failed        int
	AlreadyBooked int
	BookedMatches []string
	FailedMatches map[string]string // matchID -> error message
}

// ExecuteStartupBooking 执行启动时自动订阅
func (s *StartupBookingService) ExecuteStartupBooking() (*BookingResult, error) {
	logger.Println("[StartupBooking] 🚀 Starting automatic booking on service startup...")
	
	result := &BookingResult{
		BookedMatches: []string{},
		FailedMatches: make(map[string]string),
	}
	
	// 1. 查询当前直播赛程
	liveMatches, err := s.queryLiveSchedule()
	if err != nil {
		return nil, fmt.Errorf("failed to query live schedule: %w", err)
	}
	
	result.TotalLive = len(liveMatches)
	logger.Printf("[StartupBooking] 📊 Found %d live matches", result.TotalLive)
	
	if result.TotalLive == 0 {
		logger.Println("[StartupBooking] ℹ️  No live matches found, skipping booking")
		s.sendStartupReport(result)
		return result, nil
	}
	
	// 2. 筛选可订阅的比赛
	bookableMatches := s.filterBookableMatches(liveMatches)
	result.Bookable = len(bookableMatches)
	logger.Printf("[StartupBooking] 🎯 Found %d bookable matches", result.Bookable)
	
	if result.Bookable == 0 {
		logger.Println("[StartupBooking] ℹ️  No bookable matches found")
		s.sendStartupReport(result)
		return result, nil
	}
	
	// 3. 订阅所有可订阅的比赛
	logger.Printf("[StartupBooking] 📝 Booking %d matches...", result.Bookable)
	
	for _, match := range bookableMatches {
		if err := s.bookMatch(match.ID); err != nil {
			logger.Printf("[StartupBooking] ❌ Failed to book %s: %v", match.ID, err)
			result.Failed++
			result.FailedMatches[match.ID] = err.Error()
		} else {
			logger.Printf("[StartupBooking] ✅ Successfully booked %s", match.ID)
			result.Success++
			result.BookedMatches = append(result.BookedMatches, match.ID)
		}
		
		// 避免请求过快
		time.Sleep(500 * time.Millisecond)
	}
	
	logger.Printf("[StartupBooking] 📈 Booking completed: %d success, %d failed out of %d bookable", 
		result.Success, result.Failed, result.Bookable)
	
	// 4. 验证订阅状态
	if result.Success > 0 {
		logger.Println("[StartupBooking] 🔍 Verifying subscriptions...")
		time.Sleep(2 * time.Second) // 等待订阅生效
		
		verified := s.verifySubscriptions(result.BookedMatches)
		result.AlreadyBooked = verified
		logger.Printf("[StartupBooking] ✅ Verified %d subscriptions", verified)
	}
	
	// 5. 发送飞书通知
	s.sendStartupReport(result)
	
	return result, nil
}

// queryLiveSchedule 查询当前直播赛程
func (s *StartupBookingService) queryLiveSchedule() ([]SportEvent, error) {
	url := fmt.Sprintf("%s/sports/en/schedules/live/schedule.xml", s.config.APIBaseURL)
	
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
	
	type Schedule struct {
		SportEvents []SportEvent `xml:"sport_event"`
	}
	
	var schedule Schedule
	if err := xml.Unmarshal(body, &schedule); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	return schedule.SportEvents, nil
}

// filterBookableMatches 筛选可订阅的比赛
func (s *StartupBookingService) filterBookableMatches(matches []SportEvent) []SportEvent {
	var bookable []SportEvent
	
	for _, match := range matches {
		if match.LiveOdds == "bookable" {
			bookable = append(bookable, match)
		}
	}
	
	return bookable
}

// bookMatch 订阅单个比赛
func (s *StartupBookingService) bookMatch(matchID string) error {
	url := fmt.Sprintf("%s/liveodds/booking-calendar/events/%s/book", s.config.APIBaseURL, matchID)
	
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
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	
	// 更新数据库订阅状态
	_, err = s.db.Exec(
		"UPDATE tracked_events SET subscribed = true, live_odds = 'booked', updated_at = $1 WHERE event_id = $2",
		time.Now(), matchID,
	)
	if err != nil {
		logger.Printf("[StartupBooking] ⚠️  Failed to update database for %s: %v", matchID, err)
		// 不返回错误，因为 API 订阅已经成功
	}
	
	return nil
}

// verifySubscriptions 验证订阅状态
func (s *StartupBookingService) verifySubscriptions(matchIDs []string) int {
	// 查询已订阅的比赛
	// 尝试多个可能的 API 路径
	urls := []string{
		fmt.Sprintf("%s/liveodds/booking-calendar/events/booked.xml", s.config.APIBaseURL),
		fmt.Sprintf("%s/liveodds/booking-calendar/booked.xml", s.config.APIBaseURL),
	}
	
	for _, url := range urls {
		verified := s.verifySubscriptionsFromURL(url, matchIDs)
		if verified >= 0 {
			return verified
		}
	}
	
	logger.Println("[StartupBooking] ⚠️  All verification API endpoints failed")
	return 0
}

// verifySubscriptionsFromURL 从指定 URL 验证订阅状态
func (s *StartupBookingService) verifySubscriptionsFromURL(url string, matchIDs []string) int {
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Printf("[StartupBooking] ⚠️  Failed to create verification request: %v", err)
		return -1
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		logger.Printf("[StartupBooking] ⚠️  Failed to verify subscriptions: %v", err)
		return -1
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("[StartupBooking] ⚠️  Failed to read verification response: %v", err)
		return -1
	}
	
	if resp.StatusCode != http.StatusOK {
		logger.Printf("[StartupBooking] ⚠️  Verification API %s returned status %d", url, resp.StatusCode)
		return -1
	}
	
	type BookingCalendar struct {
		SportEvents []SportEvent `xml:"sport_event"`
	}
	
	var calendar BookingCalendar
	if err := xml.Unmarshal(body, &calendar); err != nil {
		logger.Printf("[StartupBooking] ⚠️  Failed to parse verification response from %s: %v", url, err)
		return -1
	}
	
	logger.Printf("[StartupBooking] ✅ Successfully queried %s, found %d booked matches", url, len(calendar.SportEvents))
	
	// 统计匹配的订阅
	bookedMap := make(map[string]bool)
	for _, event := range calendar.SportEvents {
		bookedMap[event.ID] = true
	}
	
	verified := 0
	for _, matchID := range matchIDs {
		if bookedMap[matchID] {
			verified++
			logger.Printf("[StartupBooking] ✅ Verified subscription: %s", matchID)
		} else {
			logger.Printf("[StartupBooking] ⚠️  Subscription not found: %s", matchID)
		}
	}
	
	return verified
}

// sendStartupReport 发送启动订阅报告
func (s *StartupBookingService) sendStartupReport(result *BookingResult) {
	if s.larkNotifier == nil {
		return
	}
	
	var msg string
	msg += "🚀 **服务启动自动订阅报告**\n\n"
	msg += fmt.Sprintf("📊 直播比赛总数: **%d** 场\n", result.TotalLive)
	msg += fmt.Sprintf("🎯 可订阅比赛: **%d** 场\n", result.Bookable)
	
	if result.Bookable > 0 {
		msg += fmt.Sprintf("✅ 订阅成功: **%d** 场\n", result.Success)
		
		if result.Failed > 0 {
			msg += fmt.Sprintf("❌ 订阅失败: **%d** 场\n", result.Failed)
		}
		
		if result.AlreadyBooked > 0 {
			msg += fmt.Sprintf("🔍 已验证订阅: **%d** 场\n", result.AlreadyBooked)
		}
		
		// 列出成功订阅的比赛
		if len(result.BookedMatches) > 0 && len(result.BookedMatches) <= 10 {
			msg += "\n**已订阅比赛:**\n"
			for _, matchID := range result.BookedMatches {
				msg += fmt.Sprintf("- %s\n", matchID)
			}
		}
		
		// 列出失败的比赛
		if len(result.FailedMatches) > 0 && len(result.FailedMatches) <= 5 {
			msg += "\n**订阅失败:**\n"
			for matchID, errMsg := range result.FailedMatches {
				msg += fmt.Sprintf("- %s: %s\n", matchID, errMsg)
			}
		}
	}
	
	msg += fmt.Sprintf("\n⏰ 时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	
	s.larkNotifier.SendText(msg)
}

