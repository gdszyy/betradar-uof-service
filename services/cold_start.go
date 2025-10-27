package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	
	"uof-service/config"
)

// ColdStart 冷启动服务
type ColdStart struct {
	config       *config.Config
	db           *sql.DB
	client       *http.Client
	larkNotifier *LarkNotifier
	logger       *log.Logger
}

// ScheduleData 日程数据
type ScheduleData struct {
	XMLName     xml.Name      `xml:"schedule"`
	SportEvents []ColdStartEvent  `xml:"sport_event"`
}

// ColdStartEvent 赛事
type ColdStartEvent struct {
	ID          string       `xml:"id,attr"`
	Scheduled   string       `xml:"scheduled,attr"`
	StartTime   string       `xml:"start_time,attr"`
	LiveOdds    string       `xml:"liveodds,attr"`
	Sport       SportData    `xml:"sport"`
	Competitors []ColdStartCompetitor `xml:"competitors>competitor"`
}

// SportData 运动数据
type SportData struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// ColdStartCompetitor 参赛者
type ColdStartCompetitor struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	Qualifier string `xml:"qualifier,attr"`
}

// MatchInfo 比赛信息
type MatchInfo struct {
	EventID       string
	SportID       string
	ScheduleTime  *time.Time
	HomeTeamID    string
	HomeTeamName  string
	AwayTeamID    string
	AwayTeamName  string
}

// ValidationReport 验证报告
type ValidationReport struct {
	TotalMatches      int
	CompleteMatches   int
	IncompleteMatches int
	MissingSportID    int
	MissingSchedule   int
	MissingHomeTeam   int
	MissingAwayTeam   int
	SampleIncomplete  []string
}

// NewColdStart 创建冷启动服务
func NewColdStart(cfg *config.Config, db *sql.DB, larkNotifier *LarkNotifier) *ColdStart {
	return &ColdStart{
		config:       cfg,
		db:           db,
		client:       &http.Client{Timeout: 30 * time.Second},
		larkNotifier: larkNotifier,
		logger:       log.New(log.Writer(), "[ColdStart] ", log.LstdFlags),
	}
}

// Run 执行冷启动
func (c *ColdStart) Run() error {
	c.logger.Println("🚀 Starting cold start initialization...")
	startTime := time.Now()
	
	// 1. 获取比赛列表
	matches, err := c.fetchMatches()
	if err != nil {
		return fmt.Errorf("failed to fetch matches: %w", err)
	}
	c.logger.Printf("✅ Fetched %d matches", len(matches))
	
	// 2. 存储到数据库
	stored := c.storeMatches(matches)
	c.logger.Printf("✅ Stored %d matches to database", stored)
	
	// 3. 验证数据质量
	report := c.validateData(matches)
	c.printReport(report)
	
	// 4. 发送通知
	duration := time.Since(startTime)
	c.sendNotification(len(matches), stored, report, duration)
	
	c.logger.Printf("🎉 Cold start completed in %v", duration)
	return nil
}

// fetchMatches 获取比赛列表
func (c *ColdStart) fetchMatches() ([]MatchInfo, error) {
	var allMatches []MatchInfo
	
	// 获取今天和明天的比赛
	dates := []string{
		"live",
		time.Now().Format("2006-01-02"),
		time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
	}
	
	for _, date := range dates {
		matches, err := c.fetchSchedule(date)
		if err != nil {
			c.logger.Printf("⚠️  Failed to fetch %s: %v", date, err)
			continue
		}
		allMatches = append(allMatches, matches...)
		c.logger.Printf("📅 Fetched %d matches for %s", len(matches), date)
	}
	
	// 去重
	unique := c.deduplicate(allMatches)
	c.logger.Printf("✅ Total unique matches: %d", len(unique))
	
	return unique, nil
}

// fetchSchedule 获取日程
func (c *ColdStart) fetchSchedule(date string) ([]MatchInfo, error) {
	url := fmt.Sprintf("%s/sports/en/schedules/%s/schedule.xml", c.config.APIBaseURL, date)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// 使用 HTTP Header 认证
	req.Header.Set("x-access-token", c.config.AccessToken)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var schedule ScheduleData
	if err := xml.Unmarshal(body, &schedule); err != nil {
		return nil, err
	}
	
	// 转换为 MatchInfo
	matches := []MatchInfo{}
	for _, event := range schedule.SportEvents {
		match := c.parseEvent(event)
		matches = append(matches, match)
	}
	
	return matches, nil
}

// parseEvent 解析赛事
func (c *ColdStart) parseEvent(event ColdStartEvent) MatchInfo {
	match := MatchInfo{
		EventID: event.ID,
		SportID: event.Sport.ID,
	}
	
	// 如果 sport_id 为空，从 event_id 推断
	if match.SportID == "" {
		match.SportID = c.inferSportID(event.ID)
	}
	
	// 解析时间
	if event.Scheduled != "" {
		if t, err := time.Parse(time.RFC3339, event.Scheduled); err == nil {
			match.ScheduleTime = &t
		}
	} else if event.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, event.StartTime); err == nil {
			match.ScheduleTime = &t
		}
	}
	
	// 解析球队
	for _, comp := range event.Competitors {
		if comp.Qualifier == "home" {
			match.HomeTeamID = comp.ID
			match.HomeTeamName = comp.Name
		} else if comp.Qualifier == "away" {
			match.AwayTeamID = comp.ID
			match.AwayTeamName = comp.Name
		}
	}
	
	return match
}

// deduplicate 去重
func (c *ColdStart) deduplicate(matches []MatchInfo) []MatchInfo {
	seen := make(map[string]bool)
	unique := []MatchInfo{}
	
	for _, match := range matches {
		if !seen[match.EventID] {
			seen[match.EventID] = true
			unique = append(unique, match)
		}
	}
	
	return unique
}

// storeMatches 存储比赛
func (c *ColdStart) storeMatches(matches []MatchInfo) int {
	query := `
		INSERT INTO tracked_events (
			event_id, sport_id, schedule_time,
			home_team_id, home_team_name,
			away_team_id, away_team_name,
			status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'scheduled', $8, $9)
			ON CONFLICT (event_id) DO UPDATE SET
				sport_id = EXCLUDED.sport_id,
				schedule_time = EXCLUDED.schedule_time,
				home_team_id = EXCLUDED.home_team_id,
				home_team_name = EXCLUDED.home_team_name,
				away_team_id = EXCLUDED.away_team_id,
				away_team_name = EXCLUDED.away_team_name,
				updated_at = EXCLUDED.updated_at
	`
	
	stored := 0
	for _, match := range matches {
		_, err := c.db.Exec(query,
			match.EventID,
			match.SportID,
			match.ScheduleTime,
			match.HomeTeamID,
			match.HomeTeamName,
			match.AwayTeamID,
			match.AwayTeamName,
			time.Now(),
			time.Now(),
		)
		
		if err != nil {
			c.logger.Printf("⚠️  Failed to store %s: %v", match.EventID, err)
			continue
		}
		stored++
	}
	
	return stored
}

// validateData 验证数据
func (c *ColdStart) validateData(matches []MatchInfo) ValidationReport {
	report := ValidationReport{
		TotalMatches:     len(matches),
		SampleIncomplete: []string{},
	}
	
	for _, match := range matches {
		complete := true
		missing := []string{}
		
		if match.SportID == "" {
			report.MissingSportID++
			missing = append(missing, "sport_id")
			complete = false
		}
		
		if match.ScheduleTime == nil {
			report.MissingSchedule++
			missing = append(missing, "schedule_time")
			complete = false
		}
		
		if match.HomeTeamID == "" || match.HomeTeamName == "" {
			report.MissingHomeTeam++
			missing = append(missing, "home_team")
			complete = false
		}
		
		if match.AwayTeamID == "" || match.AwayTeamName == "" {
			report.MissingAwayTeam++
			missing = append(missing, "away_team")
			complete = false
		}
		
		if complete {
			report.CompleteMatches++
		} else {
			report.IncompleteMatches++
			if len(report.SampleIncomplete) < 10 {
				report.SampleIncomplete = append(report.SampleIncomplete,
					fmt.Sprintf("%s: missing %v", match.EventID, missing))
			}
		}
	}
	
	return report
}

// inferSportID 从 event_id推断sport_id
func (c *ColdStart) inferSportID(eventID string) string {
	// 根据 event_id 的格式推断 sport_id
	// sr:match:xxx -> 足球 (sr:sport:1)
	// sr:stage:xxx -> 足球 (sr:sport:1)
	// sr:simple_tournament:xxx -> 网球等其他运动
	
	if strings.HasPrefix(eventID, "sr:match:") {
		// 大多数 match 是足球
		return "sr:sport:1"
	} else if strings.HasPrefix(eventID, "sr:stage:") {
		// stage 通常也是足球
		return "sr:sport:1"
	} else if strings.HasPrefix(eventID, "sr:simple_tournament:") {
		// 网球等其他运动
		// 默认也设为足球，需要时可以调整
		return "sr:sport:1"
	}
	
	// 默认返回足球
	return "sr:sport:1"
}

// printReport 打印报告
func (c *ColdStart) printReport(report ValidationReport) {
	c.logger.Println(strings.Repeat("=", 80))
	c.logger.Println("📊 DATA QUALITY VALIDATION REPORT")
	c.logger.Println(strings.Repeat("=", 80))
	
	c.logger.Printf("Total Matches: %d", report.TotalMatches)
	c.logger.Printf("✅ Complete Matches: %d (%.2f%%)",
		report.CompleteMatches,
		float64(report.CompleteMatches)/float64(report.TotalMatches)*100)
	c.logger.Printf("⚠️  Incomplete Matches: %d (%.2f%%)",
		report.IncompleteMatches,
		float64(report.IncompleteMatches)/float64(report.TotalMatches)*100)
	c.logger.Println("")
	
	c.logger.Println("Missing Fields Statistics:")
	c.logger.Printf("  sport_id missing: %d matches", report.MissingSportID)
	c.logger.Printf("  schedule_time missing: %d matches", report.MissingSchedule)
	c.logger.Printf("  home_team missing: %d matches", report.MissingHomeTeam)
	c.logger.Printf("  away_team missing: %d matches", report.MissingAwayTeam)
	c.logger.Println("")
	
	if len(report.SampleIncomplete) > 0 {
		c.logger.Println("Sample Incomplete Matches (first 10):")
		for _, sample := range report.SampleIncomplete {
			c.logger.Printf("  %s", sample)
		}
		c.logger.Println("")
	}
	
	c.logger.Println(strings.Repeat("=", 80))
}

// sendNotification 发送通知
func (c *ColdStart) sendNotification(total, stored int, report ValidationReport, duration time.Duration) {
	if c.larkNotifier == nil {
		return
	}
	
	message := fmt.Sprintf(
		"🚀 **Cold Start Initialization Completed**\n\n"+
			"**Statistics:**\n"+
			"- Total Matches Found: %d\n"+
			"- Stored to Database: %d\n"+
			"- Complete Data: %d (%.2f%%)\n"+
			"- Incomplete Data: %d (%.2f%%)\n"+
			"- Duration: %v\n\n"+
			"**Missing Fields:**\n"+
			"- sport_id: %d\n"+
			"- schedule_time: %d\n"+
			"- home_team: %d\n"+
			"- away_team: %d\n\n"+
			"**Status:** ✅ Success",
		total,
		stored,
		report.CompleteMatches,
		float64(report.CompleteMatches)/float64(stored)*100,
		report.IncompleteMatches,
		float64(report.IncompleteMatches)/float64(stored)*100,
		duration,
		report.MissingSportID,
		report.MissingSchedule,
		report.MissingHomeTeam,
		report.MissingAwayTeam,
	)
	
	// 发送飞书通知
	if err := c.larkNotifier.SendText(message); err != nil {
		c.logger.Printf("⚠️  Failed to send notification: %v", err)
	}
}

