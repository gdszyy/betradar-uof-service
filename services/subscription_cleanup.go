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

// SubscriptionCleanupService 订阅清理服务
type SubscriptionCleanupService struct {
	config       *config.Config
	db           *sql.DB
	client       *http.Client
	larkNotifier *LarkNotifier
	mapper       *SRMapper
}

// NewSubscriptionCleanupService 创建订阅清理服务
func NewSubscriptionCleanupService(cfg *config.Config, db *sql.DB, notifier *LarkNotifier) *SubscriptionCleanupService {
	return &SubscriptionCleanupService{
		config:       cfg,
		db:           db,
		client:       &http.Client{Timeout: 30 * time.Second},
		larkNotifier: notifier,
		mapper:       NewSRMapper(),
	}
}

// CleanupResult 清理结果
type CleanupResult struct {
	TotalBooked    int
	EndedMatches   int
	Unbooked       int
	Failed         int
	UnbookedList   []string
	FailedList     map[string]string
}

// ExecuteCleanup 执行清理
func (s *SubscriptionCleanupService) ExecuteCleanup() (*CleanupResult, error) {
	log.Println("[SubscriptionCleanup] 🧹 Starting subscription cleanup...")
	
	result := &CleanupResult{
		UnbookedList: []string{},
		FailedList:   make(map[string]string),
	}
	
	// 1. 查询所有已订阅的比赛
	bookedMatches, err := s.queryBookedMatches()
	if err != nil {
		return nil, fmt.Errorf("failed to query booked matches: %w", err)
	}
	
	result.TotalBooked = len(bookedMatches)
	log.Printf("[SubscriptionCleanup] 📊 Found %d booked matches", result.TotalBooked)
	
	if result.TotalBooked == 0 {
		log.Println("[SubscriptionCleanup] ℹ️  No booked matches to cleanup")
		return result, nil
	}
	
	// 2. 检查每个比赛的状态
	endedMatches := s.findEndedMatches(bookedMatches)
	result.EndedMatches = len(endedMatches)
	log.Printf("[SubscriptionCleanup] 🎯 Found %d ended matches to unbook", result.EndedMatches)
	
	if result.EndedMatches == 0 {
		log.Println("[SubscriptionCleanup] ℹ️  No ended matches to unbook")
		s.sendCleanupReport(result)
		return result, nil
	}
	
	// 3. 取消订阅已结束的比赛
	log.Printf("[SubscriptionCleanup] 🚀 Unbooking %d ended matches...", result.EndedMatches)
	
	for _, match := range endedMatches {
		if err := s.unbookMatch(match.ID); err != nil {
			log.Printf("[SubscriptionCleanup] ❌ Failed to unbook %s: %v", match.ID, err)
			result.Failed++
			result.FailedList[match.ID] = err.Error()
		} else {
			log.Printf("[SubscriptionCleanup] ✅ Successfully unbooked %s", match.ID)
			result.Unbooked++
			result.UnbookedList = append(result.UnbookedList, match.ID)
		}
		
		// 避免请求过快
		time.Sleep(500 * time.Millisecond)
	}
	
	log.Printf("[SubscriptionCleanup] 📈 Cleanup completed: %d unbooked, %d failed out of %d ended", 
		result.Unbooked, result.Failed, result.EndedMatches)
	
	// 4. 发送飞书通知
	s.sendCleanupReport(result)
	
	return result, nil
}

// queryBookedMatches 查询已订阅的比赛
func (s *SubscriptionCleanupService) queryBookedMatches() ([]BookedMatch, error) {
	url := fmt.Sprintf("%s/liveodds/booking-calendar/events/booked.xml", s.config.APIBaseURL)
	
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

// BookedMatch 已订阅的比赛
type BookedMatch struct {
	ID        string `xml:"id,attr"`
	Scheduled string `xml:"scheduled,attr"`
	Status    string `xml:"status,attr"`
	LiveOdds  string `xml:"liveodds,attr"`
}

// findEndedMatches 查找已结束的比赛
func (s *SubscriptionCleanupService) findEndedMatches(matches []BookedMatch) []BookedMatch {
	var endedMatches []BookedMatch
	
	for _, match := range matches {
		// 从数据库查询比赛状态
		var matchStatus sql.NullString
		query := `SELECT match_status FROM tracked_events WHERE event_id = $1`
		err := s.db.QueryRow(query, match.ID).Scan(&matchStatus)
		
		if err == nil && matchStatus.Valid {
			// 使用映射器判断是否已结束
			if s.mapper.IsMatchEnded(matchStatus.String) {
				log.Printf("[SubscriptionCleanup] 🔍 Found ended match: %s (status: %s)", 
					match.ID, matchStatus.String)
				endedMatches = append(endedMatches, match)
			}
		}
	}
	
	return endedMatches
}

// unbookMatch 取消订阅单个比赛
func (s *SubscriptionCleanupService) unbookMatch(matchID string) error {
	// API: DELETE /liveodds/booking-calendar/events/{id}/unbook
	url := fmt.Sprintf("%s/liveodds/booking-calendar/events/%s/unbook", s.config.APIBaseURL, matchID)
	
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// sendCleanupReport 发送清理报告
func (s *SubscriptionCleanupService) sendCleanupReport(result *CleanupResult) {
	if s.larkNotifier == nil {
		return
	}
	
	var msg string
	msg += "🧹 **订阅清理报告**\n\n"
	msg += fmt.Sprintf("📊 已订阅比赛总数: **%d** 场\n", result.TotalBooked)
	msg += fmt.Sprintf("🎯 发现已结束比赛: **%d** 场\n", result.EndedMatches)
	
	if result.EndedMatches > 0 {
		msg += fmt.Sprintf("✅ 取消订阅成功: **%d** 场\n", result.Unbooked)
		
		if result.Failed > 0 {
			msg += fmt.Sprintf("❌ 取消订阅失败: **%d** 场\n", result.Failed)
		}
		
		// 列出取消订阅的比赛
		if len(result.UnbookedList) > 0 && len(result.UnbookedList) <= 10 {
			msg += "\n**已取消订阅:**\n"
			for _, matchID := range result.UnbookedList {
				msg += fmt.Sprintf("- %s\n", matchID)
			}
		}
		
		// 列出失败的比赛
		if len(result.FailedList) > 0 && len(result.FailedList) <= 5 {
			msg += "\n**取消失败:**\n"
			for matchID, errMsg := range result.FailedList {
				msg += fmt.Sprintf("- %s: %s\n", matchID, errMsg)
			}
		}
	}
	
	msg += fmt.Sprintf("\n⏰ 时间: %s", time.Now().Format("2006-01-02 15:04:05"))
	
	s.larkNotifier.SendText(msg)
}

