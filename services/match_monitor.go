package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
	
	"uof-service/config"
)

// ScheduleResponse Schedule API响应
type ScheduleResponse struct {
	XMLName     xml.Name     `xml:"schedule"`
	GeneratedAt string       `xml:"generated_at,attr"`
	SportEvents []SportEvent `xml:"sport_event"`
}

type SportEvent struct {
	ID           string      `xml:"id,attr"`
	Scheduled    string      `xml:"scheduled,attr"`
	Status       string      `xml:"status,attr"`
	LiveOdds     string      `xml:"liveodds,attr"`
	NextLiveTime string      `xml:"next_live_time,attr"`
	Tournament   Tournament  `xml:"tournament"`
	Competitors  Competitors `xml:"competitors"`
}

type Tournament struct {
	ID       string   `xml:"id,attr"`
	Name     string   `xml:"name,attr"`
	Sport    Sport    `xml:"sport"`
	Category Category `xml:"category"`
}

type Sport struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type Category struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type Competitors struct {
	Competitor []Competitor `xml:"competitor"`
}

type Competitor struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	Qualifier string `xml:"qualifier,attr"`
}

// MatchMonitor 比赛订阅监控
type MatchMonitor struct {
	config *config.Config
	client *http.Client
}

// NewMatchMonitor 创建比赛监控
func NewMatchMonitor(cfg *config.Config, _ interface{}) *MatchMonitor {
	return &MatchMonitor{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// QueryBookedMatches 查询已订阅的比赛 (使用 REST API)
func (m *MatchMonitor) QueryBookedMatches() (*ScheduleResponse, error) {
	logger.Printf("📋 Querying live matches via REST API...")
	
	// 使用 Live Schedule API
	url := fmt.Sprintf("%s/sports/en/schedules/live/schedule.xml", m.config.APIBaseURL)
	
	logger.Printf("📤 Calling API: %s", url)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// 添加认证头
	req.Header.Set("x-access-token", m.config.AccessToken)
	
	// 发送请求
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	logger.Printf("📥 Received response (%d bytes)", resp.ContentLength)
	
	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var schedule ScheduleResponse
	if err := xml.Unmarshal(body, &schedule); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	return &schedule, nil
}

// AnalyzeBookedMatches 分析已订阅的比赛
func (m *MatchMonitor) AnalyzeBookedMatches(schedule *ScheduleResponse) {
	logger.Println("\n" + "═══════════════════════════════════════════════════════════")
	logger.Println("📊 BOOKED MATCHES ANALYSIS")
	logger.Println("═══════════════════════════════════════════════════════════")
	
	totalMatches := len(schedule.SportEvents)
	bookedCount := 0
	bookableCount := 0
	notAvailableCount := 0
	liveCount := 0
	notStartedCount := 0
	
	bookedMatches := []SportEvent{}
	
	// 统计
	for _, event := range schedule.SportEvents {
		switch event.LiveOdds {
		case "booked":
			bookedCount++
			bookedMatches = append(bookedMatches, event)
			
			// 判断是否live
			if event.Status == "live" || event.Status == "started" {
				liveCount++
			} else {
				notStartedCount++
			}
		case "bookable":
			bookableCount++
		case "not_available":
			notAvailableCount++
		}
	}
	
	logger.Printf("📈 Summary:")
	logger.Printf("  Total live matches: %d", totalMatches)
	logger.Printf("  Booked matches: %d", bookedCount)
	logger.Printf("    - Live/Started: %d", liveCount)
	logger.Printf("    - Not started: %d", notStartedCount)
	logger.Printf("  Bookable (not booked): %d", bookableCount)
	logger.Printf("  Not available: %d", notAvailableCount)
	
	// 显示已订阅的比赛
	if bookedCount > 0 {
		logger.Println("\n🎯 Booked Matches:")
		logger.Printf("%-20s %-15s %-30s %-30s %s", "Match ID", "Status", "Home", "Away", "Sport")
		logger.Println("─────────────────────────────────────────────────────────────────────────────────────────────────────────")
		
		for _, event := range bookedMatches {
			homeName := ""
			awayName := ""
			if len(event.Competitors.Competitor) >= 2 {
				for _, comp := range event.Competitors.Competitor {
					if comp.Qualifier == "home" {
						homeName = comp.Name
					} else if comp.Qualifier == "away" {
						awayName = comp.Name
					}
				}
			}
			
			logger.Printf("%-20s %-15s %-30s %-30s %s",
				truncate(event.ID, 20),
				event.Status,
				truncate(homeName, 30),
				truncate(awayName, 30),
				event.Tournament.Sport.Name,
			)
		}
	} else {
		logger.Println("\n⚠️  WARNING: No booked matches found!")
		logger.Println("   This explains why you're not receiving odds_change messages.")
		logger.Println("   You need to subscribe to matches to receive odds updates.")
		
		if bookableCount > 0 {
			logger.Printf("\n💡 TIP: There are %d bookable matches available.", bookableCount)
			logger.Println("   Use the booking API to subscribe to matches:")
			logger.Println("   POST /liveodds/booking-calendar/events/{match_id}/book")
		}
	}
	
	if bookedCount > 0 && liveCount == 0 {
		logger.Println("\n⚠️  NOTE: No live matches currently.")
		logger.Println("   Odds_change messages are typically sent for live matches.")
		logger.Println("   Pre-match odds updates are less frequent.")
	}
	
	logger.Println("═══════════════════════════════════════════════════════════\n")
}

// CheckAndReport 检查并报告
func (m *MatchMonitor) CheckAndReport() {
	schedule, err := m.QueryBookedMatches()
	if err != nil {
		logger.Printf("❌ Failed to query booked matches: %v", err)
		return
	}
	
	m.AnalyzeBookedMatches(schedule)
}

// CheckAndReportWithNotifier 检查并报告(带通知)
func (m *MatchMonitor) CheckAndReportWithNotifier(notifier *LarkNotifier) {
	schedule, err := m.QueryBookedMatches()
	if err != nil {
		logger.Printf("❌ Failed to query booked matches: %v", err)
		if notifier != nil {
			notifier.NotifyError("MatchMonitor", fmt.Sprintf("Failed to query: %v", err))
		}
		return
	}
	
	m.AnalyzeBookedMatches(schedule)
	
	// 发送通知
	if notifier != nil {
		totalMatches := len(schedule.SportEvents)
		bookedCount := 0
		liveCount := 0
		notStartedCount := 0
		
		for _, event := range schedule.SportEvents {
			if event.LiveOdds == "booked" {
				bookedCount++
				if event.Status == "live" || event.Status == "started" {
					liveCount++
				} else {
					notStartedCount++
				}
			}
		}
		
		notifier.NotifyMatchMonitor(totalMatches, bookedCount, notStartedCount, liveCount)
	}
}

// MonitorPeriodically 定期监控
func (m *MatchMonitor) MonitorPeriodically(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// 立即执行一次
	m.CheckAndReport()
	
	// 定期执行
	for range ticker.C {
		m.CheckAndReport()
	}
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

