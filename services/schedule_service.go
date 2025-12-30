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

// ScheduleService 赛程服务
type ScheduleService struct {
	db          *sql.DB
	apiBaseURL  string
	accessToken string
	client      *http.Client
}

// TournamentScheduleResponse API 响应
type TournamentScheduleResponse struct {
	XMLName xml.Name `xml:"schedule"`
	SportEvents []struct {
		ID       string `xml:"id,attr"`
		LiveOdds string `xml:"liveodds,attr"`
	} `xml:"sport_event"`
}

// SportEventSummaryResponse API 响应
type SportEventSummaryResponse struct {
	XMLName xml.Name `xml:"match_summary"`
	SportEvent struct {
		ID string `xml:"id,attr"`
	} `xml:"sport_event"`
	Lineups struct {
		Players []struct {
			ID string `xml:"id,attr"`
			Name string `xml:"name,attr"`
		} `xml:"player"`
	} `xml:"lineups"`
}

// NewScheduleService 创建赛程服务
func NewScheduleService(db *sql.DB, accessToken, apiBaseURL string) *ScheduleService {
	return &ScheduleService{
		db:          db,
		apiBaseURL:  apiBaseURL,
		accessToken: accessToken,
		client:      &http.Client{Timeout: 60 * time.Second},
	}
}

// Start 启动赛程服务
func (s *ScheduleService) Start() error {
	logger.Println("[Schedule] Starting schedule service...")

		// 启动时立即执行一次
		if _, err := s.FetchUpcomingSchedule(); err != nil {
			logger.Errorf("[Schedule] ❌ Failed to fetch upcoming schedule: %v", err)
			return err
		}

	// 每天凌晨 1 点执行一次
	go s.scheduleDailyFetch()

	logger.Println("[Schedule] ✅ Schedule service started (daily at 1:00 AM)")
	return nil
}

// FetchUpcomingSchedule 获取未来的赛程（使用 pre/schedule 端点）
func (s *ScheduleService) FetchUpcomingSchedule() ([]string, error) {
	// 使用正确的 API 端点: /schedules/pre/schedule.xml
	// 参数: start=0, limit=100 （获取前 100 场未来比赛）
	url := fmt.Sprintf("%s/sports/en/schedules/pre/schedule.xml?start=0&limit=100", s.apiBaseURL)

	logger.Printf("[Schedule] 📥 Fetching upcoming schedule from: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-access-token", s.accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 解析 XML
	var schedule TournamentScheduleResponse

	if err := xml.Unmarshal(body, &schedule); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	eventIDs := make([]string, len(schedule.SportEvents))
	for i, event := range schedule.SportEvents {
		eventIDs[i] = event.ID
		// 更新数据库中的 live_odds 状态
		if s.db != nil {
			_, err := s.db.Exec(
				"UPDATE tracked_events SET live_odds = $1, updated_at = $2 WHERE event_id = $3",
				event.LiveOdds, time.Now(), event.ID,
			)
			if err != nil {
				logger.Errorf("[Schedule] Failed to update live_odds for %s: %v", event.ID, err)
			}
		}
	}

	logger.Printf("[Schedule] ✅ Fetched %d sport events", len(eventIDs))

	return eventIDs, nil
}

// FetchSportEventSummary 获取比赛阵容信息
func (s *ScheduleService) FetchSportEventSummary(eventID string) ([]PlayerInfo, error) {
	// 构造 URL: /v1/sports/en/sport_events/{event_id}/summary.xml
	url := fmt.Sprintf("%s/sports/en/sport_events/%s/summary.xml", s.apiBaseURL, eventID)

	logger.Printf("[Schedule] 📥 Fetching summary for event: %s", eventID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-access-token", s.accessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 404 可能是因为比赛没有阵容信息,不作为错误处理
		if resp.StatusCode == http.StatusNotFound {
			logger.Printf("[Schedule] ⚠️  Summary not found for event %s (404)", eventID)
			return nil, nil
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var summary SportEventSummaryResponse
	if err := xml.Unmarshal(body, &summary); err != nil {
		return nil, fmt.Errorf("failed to parse XML for event %s: %w", eventID, err)
	}

	players := make([]PlayerInfo, len(summary.Lineups.Players))
	for i, p := range summary.Lineups.Players {
		players[i] = PlayerInfo{
			ID:   fmt.Sprintf("sr:player:%s", p.ID), // 补全 URN
			Name: p.Name,
		}
	}

	logger.Printf("[Schedule] ✅ Fetched %d players for event %s", len(players), eventID)

		return players, nil
	}
	
	// scheduleDailyFetch 每天定时执行
	func (s *ScheduleService) scheduleDailyFetch() {
		// 计算到下一个凌晨 1 点的时间
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 1, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			// 如果已经过了今天的 1 点，设置为明天 1 点
			nextRun = nextRun.Add(24 * time.Hour)
		}
	
		// 等待到第一次执行时间
		initialDelay := time.Until(nextRun)
		logger.Printf("[Schedule] Next fetch scheduled at %s (in %s)", nextRun.Format("2006-01-02 15:04:05"), initialDelay.Round(time.Minute))
		time.Sleep(initialDelay)
	
		// 执行第一次
		if _, err := s.FetchUpcomingSchedule(); err != nil {
			logger.Errorf("[Schedule] ❌ Daily fetch failed: %v", err)
		}
	
		// 之后每 24 小时执行一次
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
	
		for range ticker.C {
			if _, err := s.FetchUpcomingSchedule(); err != nil {
				logger.Errorf("[Schedule] ❌ Daily fetch failed: %v", err)
			}
		}
	}
	