package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// FixtureParser Fixture 消息解析器
type FixtureParser struct {
	db               *sql.DB
	srnMappingService *SRNMappingService
	logger           *log.Logger
	apiBaseURL       string
	accessToken      string
}

// FixtureMessage Fixture 消息结构
type FixtureMessage struct {
	XMLName      xml.Name     `xml:"fixture"`
	EventID      string       `xml:"event_id,attr"`
	ProductID    int          `xml:"product,attr"`
	Timestamp    int64        `xml:"timestamp,attr"`
	ScheduledTime int64       `xml:"scheduled,attr"`
	Sport        SportInfo    `xml:"sport"`
	Tournament   TournamentInfo `xml:"tournament"`
	Competitors  []FixtureCompetitor `xml:"competitors>competitor"`
	Status       string       `xml:"status,attr"`
	StartTime    int64        `xml:"start_time,attr"`
	NextLiveTime int64        `xml:"next_live_time,attr"`
}

// SportInfo 体育信息
type SportInfo struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// TournamentInfo 锦标赛信息
type TournamentInfo struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// FixtureCompetitor 参赛队伍
type FixtureCompetitor struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr"`
	Qualifier string `xml:"qualifier,attr"` // home/away
}

// NewFixtureParser 创建 Fixture 解析器
func NewFixtureParser(db *sql.DB, srnMappingService *SRNMappingService, apiBaseURL, accessToken string) *FixtureParser {
	return &FixtureParser{
		db:               db,
		srnMappingService: srnMappingService,
		logger:           log.New(os.Stdout, "", log.LstdFlags),
		apiBaseURL:       apiBaseURL,
		accessToken:      accessToken,
	}
}

// ParseAndStore 解析并存储 Fixture 消息
func (p *FixtureParser) ParseAndStore(xmlContent string) error {
	var fixture FixtureMessage
	if err := xml.Unmarshal([]byte(xmlContent), &fixture); err != nil {
		return fmt.Errorf("failed to parse fixture message: %w", err)
	}

	p.logger.Printf("Parsing fixture for event: %s", fixture.EventID)

	// 获取 SRN ID
	srnID, err := p.srnMappingService.GetSRNID(fixture.EventID)
	if err != nil {
		p.logger.Printf("Warning: failed to get SRN ID for %s: %v", fixture.EventID, err)
		// 继续处理,SRN ID 不是必需的
	}

	// 提取主客队信息
	var homeTeamID, homeTeamName, awayTeamID, awayTeamName string
	for _, comp := range fixture.Competitors {
		if comp.Qualifier == "home" {
			homeTeamID = comp.ID
			homeTeamName = comp.Name
		} else if comp.Qualifier == "away" {
			awayTeamID = comp.ID
			awayTeamName = comp.Name
		}
	}

	// 转换时间戳
	var scheduleTime *time.Time
	if fixture.ScheduledTime > 0 {
		t := time.UnixMilli(fixture.ScheduledTime)
		scheduleTime = &t
	} else if fixture.StartTime > 0 {
		t := time.UnixMilli(fixture.StartTime)
		scheduleTime = &t
	}

	// 存储到数据库
	if err := p.storeFixtureData(
		fixture.EventID,
		srnID,
		fixture.Sport.ID,
		scheduleTime,
		homeTeamID,
		homeTeamName,
		awayTeamID,
		awayTeamName,
		fixture.Status,
	); err != nil {
		return fmt.Errorf("failed to store fixture data: %w", err)
	}

	p.logger.Printf("Stored fixture data for event %s: home=%s, away=%s, scheduled=%v",
		fixture.EventID, homeTeamName, awayTeamName, scheduleTime)

	return nil
}

// storeFixtureData 存储 Fixture 数据到数据库
func (p *FixtureParser) storeFixtureData(
	eventID, srnID, sportID string,
	scheduleTime *time.Time,
	homeTeamID, homeTeamName, awayTeamID, awayTeamName, status string,
) error {
	// 使用 UPSERT 更新或插入 tracked_events
query := `INSERT INTO tracked_events (event_id, srn_id, sport_id, schedule_time, home_team_id, home_team_name, away_team_id, away_team_name, match_status, subscribed, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $11) ON CONFLICT (event_id) DO UPDATE SET srn_id = COALESCE(NULLIF(EXCLUDED.srn_id, ''), tracked_events.srn_id), sport_id = COALESCE(NULLIF(EXCLUDED.sport_id, ''), tracked_events.sport_id), schedule_time = COALESCE(EXCLUDED.schedule_time, tracked_events.schedule_time), home_team_id = CASE WHEN EXCLUDED.home_team_id = '' THEN tracked_events.home_team_id ELSE EXCLUDED.home_team_id END, home_team_name = CASE WHEN EXCLUDED.home_team_name = '' THEN tracked_events.home_team_name ELSE EXCLUDED.home_team_name END, away_team_id = CASE WHEN EXCLUDED.away_team_id = '' THEN tracked_events.away_team_id ELSE EXCLUDED.away_team_id END, away_team_name = CASE WHEN EXCLUDED.away_team_name = '' THEN tracked_events.away_team_name ELSE EXCLUDED.away_team_name END, match_status = CASE WHEN EXCLUDED.match_status = '' THEN tracked_events.match_status ELSE EXCLUDED.match_status END, updated_at = EXCLUDED.updated_at`

	// p.logger.Printf("[DEBUG] SQL Query: %s, Args: event_id=%v, srn_id=%v, sport_id=%v, schedule_time=%v, home_team_id=%v, home_team_name=%v, away_team_id=%v, away_team_name=%v, status=%v", CleanSQLQuery(query), eventID, srnID, sportID, scheduleTime, homeTeamID, homeTeamName, awayTeamID, awayTeamName, status)
		_, err := p.db.Exec(
			query,
			eventID, srnID, sportID, scheduleTime,
			homeTeamID, homeTeamName,
			awayTeamID, awayTeamName,
			status,
			time.Now(), time.Now(),
		)
	if err != nil {
		return fmt.Errorf("failed to upsert tracked_events: %w", err)
	}

	return nil
}

// ParseFixtureChange 解析 fixture_change 消息
func (p *FixtureParser) ParseFixtureChange(eventID string, xmlContent string) error {
	type FixtureChange struct {
		StartTime    int64 `xml:"start_time,attr"`
		NextLiveTime int64 `xml:"next_live_time,attr"`
		ChangeType   int   `xml:"change_type,attr"`
		ProductID    int   `xml:"product,attr"`
	}

	var fixtureChange FixtureChange
	if err := xml.Unmarshal([]byte(xmlContent), &fixtureChange); err != nil {
		return fmt.Errorf("failed to parse fixture_change: %w", err)
	}

	// 日志在处理完成后输出

	// 特殊处理: change_type=5 表示 live coverage 被取消
	if fixtureChange.ChangeType == 5 {
		p.logger.Printf("[fixture_change] 比赛 %s 的直播覆盖已取消", eventID)
		// 更新状态标记
		query := `UPDATE tracked_events SET match_status = 'coverage_dropped', updated_at = $1 WHERE event_id = $2`
		p.db.Exec(query, time.Now(), eventID)
	}

	// 官方建议: 无论 change_type 是什么,都应该调用 Fixture API 获取完整信息
	// 这样可以确保所有属性都是最新的
	if err := p.fetchAndUpdateFixture(eventID); err != nil {
		// API 失败日志已简化
		
		// 如果 API 调用失败,回退到只更新 start_time
		if fixtureChange.StartTime > 0 {
			scheduleTime := time.UnixMilli(fixtureChange.StartTime)
			query := `UPDATE tracked_events SET schedule_time = $1, updated_at = $2 WHERE event_id = $3`
			if _, err := p.db.Exec(query, scheduleTime, time.Now(), eventID); err != nil {
				return fmt.Errorf("failed to update schedule_time: %w", err)
			}
			p.logger.Printf("[fixture_change] 比赛 %s 的开赛时间变更为 %s", eventID, scheduleTime.Format("2006-01-02 15:04"))
		}
		return nil
	}

	p.logger.Printf("[fixture_change] 比赛 %s 的赛事信息已更新", eventID)
	return nil
}



// fetchAndUpdateFixture 从 API 获取完整的 Fixture 信息并更新
func (p *FixtureParser) fetchAndUpdateFixture(eventID string) error {
	// 构造 API URL
	url := fmt.Sprintf("%s/sports/en/sports_events/%s/fixture.xml", p.apiBaseURL, eventID)
	// API 请求日志已简化
	
	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// 添加认证 header
	req.Header.Set("x-access-token", p.accessToken)
	
	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// 解析并存储 Fixture 数据
	if err := p.ParseAndStore(string(body)); err != nil {
		return fmt.Errorf("failed to parse and store fixture: %w", err)
	}
	
	// 成功日志在上层输出
	return nil
}

