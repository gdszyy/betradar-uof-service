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
	log.Println("[StartupBooking] 🚀 Starting automatic booking on service startup...")
	
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
	log.Printf("[StartupBooking] 📊 Found %d live matches", result.TotalLive)
	
	if result.TotalLive == 0 {
		log.Println("[StartupBooking] ℹ️  No live matches found, skipping booking")
		s.sendStartupReport(result)
		return result, nil
	}
	
	// 2. 筛选可订阅的比赛
	bookableMatches := s.filterBookableMatches(liveMatches)
	result.Bookable = len(bookableMatches)
	log.Printf("[StartupBooking] 🎯 Found %d bookable matches", result.Bookable)
	
	if result.Bookable == 0 {
		log.Println("[StartupBooking] ℹ️  No bookable matches found")
		s.sendStartupReport(result)
		return result, nil
	}
	
	// 3. 订阅所有可订阅的比赛
	log.Printf("[StartupBooking] 📝 Booking %d matches...", result.Bookable)
	
	for _, match := range bookableMatches {
		if err := s.bookMatch(match.ID); err != nil {
			log.Printf("[StartupBooking] ❌ Failed to book %s: %v", match.ID, err)
			result.Failed++
			result.FailedMatches[match.ID] = err.Error()
		} else {
			log.Printf("[StartupBooking] ✅ Successfully booked %s", match.ID)
			result.Success++
			result.BookedMatches = append(result.BookedMatches, match.ID)
		}
		
		// 避免请求过快
		time.Sleep(500 * time.Millisecond)
	}
	
	log.Printf("[StartupBooking] 📈 Booking completed: %d success, %d failed out of %d bookable", 
		result.Success, result.Failed, result.Bookable)
	
	// 4. 验证订阅状态
	if result.Success > 0 {
		log.Println("[StartupBooking] 🔍 Verifying subscriptions...")
		time.Sleep(2 * time.Second) // 等待订阅生效
		
		verified := s.verifySubscriptions(result.BookedMatches)
		result.AlreadyBooked = verified
		log.Printf("[StartupBooking] ✅ Verified %d subscriptions", verified)
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
	
	return nil
}

// verifySubscriptions 验证订阅状态
func (s *StartupBookingService) verifySubscriptions(matchIDs []string) int {
	// 查询已订阅的比赛
	url := fmt.Sprintf("%s/liveodds/booking-calendar/events/booked.xml", s.config.APIBaseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[StartupBooking] ⚠️  Failed to create verification request: %v", err)
		return 0
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[StartupBooking] ⚠️  Failed to verify subscriptions: %v", err)
		return 0
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[StartupBooking] ⚠️  Failed to read verification response: %v", err)
		return 0
	}
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("[StartupBooking] ⚠️  Verification API returned status %d", resp.StatusCode)
		return 0
	}
	
	type BookingCalendar struct {
		SportEvents []SportEvent `xml:"sport_event"`
	}
	
	var calendar BookingCalendar
	if err := xml.Unmarshal(body, &calendar); err != nil {
		log.Printf("[StartupBooking] ⚠️  Failed to parse verification response: %v", err)
		return 0
	}
	
	// 统计匹配的订阅
	bookedMap := make(map[string]bool)
	for _, event := range calendar.SportEvents {
		bookedMap[event.ID] = true
	}
	
	verified := 0
	for _, matchID := range matchIDs {
		if bookedMap[matchID] {
			verified++
			log.Printf("[StartupBooking] ✅ Verified subscription: %s", matchID)
		} else {
			log.Printf("[StartupBooking] ⚠️  Subscription not found: %s", matchID)
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

