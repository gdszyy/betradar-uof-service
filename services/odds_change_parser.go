package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// OddsChangeParser Odds Change 消息解析器
type OddsChangeParser struct {
	db              *sql.DB
	logger          *log.Logger
	teamsService    *TeamsService
	logoFetcher     *LogoFetcherService
	fixtureParser   *FixtureParser // 用于获取完整的 Fixture 信息
}

// OddsChangeMessage Odds Change 消息结构
// 根据官方文档: sport_event_status 是 odds_change 的直接子元素
type OddsChangeMessage struct {
	XMLName          xml.Name          `xml:"odds_change"`
	EventID          string            `xml:"event_id,attr"`
	ProductID        int               `xml:"product,attr"`
	Timestamp        int64             `xml:"timestamp,attr"`
	SportEvent       SportEventInfo    `xml:"sport_event"`
	SportEventStatus *SportEventStatus `xml:"sport_event_status"`
	Odds             OddsInfo          `xml:"odds"`
}

// SportEventInfo 赛事基本信息
type SportEventInfo struct {
	ID          string           `xml:"id,attr"`
	Scheduled   int64            `xml:"scheduled,attr"`
	StartTime   int64            `xml:"start_time,attr"`
	Competitors []OddsCompetitor `xml:"competitors>competitor"`
}

// SportEventStatus 赛事状态(包含比分信息)
// 这是 odds_change 的直接子元素,不是嵌套在 sport_event 下
type SportEventStatus struct {
	Status       string        `xml:"status,attr"`
	MatchStatus  string        `xml:"match_status,attr"`
	HomeScore    *int          `xml:"home_score,attr"`
	AwayScore    *int          `xml:"away_score,attr"`
	Clock        *ClockInfo    `xml:"clock"`
	PeriodScores []PeriodScore `xml:"period_scores>period_score"`
	Statistics   *Statistics   `xml:"statistics"`
}

// ClockInfo 比赛时钟信息
type ClockInfo struct {
	MatchTime             string `xml:"match_time,attr"`
	StoppageTime          string `xml:"stoppage_time,attr"`
	StoppageTimeAnnounced string `xml:"stoppage_time_announced,attr"`
}

// PeriodScore 分段比分
type PeriodScore struct {
	HomeScore int    `xml:"home_score,attr"`
	AwayScore int    `xml:"away_score,attr"`
	Type      string `xml:"type,attr"` // regular_period, overtime, penalties
	Number    int    `xml:"number,attr"`
}

// Statistics 比赛统计信息
type Statistics struct {
	YellowCards    *TeamStats `xml:"yellow_cards"`
	RedCards       *TeamStats `xml:"red_cards"`
	YellowRedCards *TeamStats `xml:"yellow_red_cards"`
	Corners        *TeamStats `xml:"corners"`
}

// TeamStats 双方统计数据
type TeamStats struct {
	Home int `xml:"home,attr"`
	Away int `xml:"away,attr"`
}

// OddsInfo 赔率信息
type OddsInfo struct {
	Markets []Market `xml:"market"`
}

// Market 市场信息
type Market struct {
	ID        int       `xml:"id,attr"`
	Specifier string    `xml:"specifiers,attr"`
	Status   int       `xml:"status,attr"`
	Outcomes []Outcome `xml:"outcome"`
}

// Outcome 结果信息
type Outcome struct {
	ID     string  `xml:"id,attr"`
	Odds   float64 `xml:"odds,attr"`
	Active int     `xml:"active,attr"`
}

// NewOddsChangeParser 创建 Odds Change 解析器
func NewOddsChangeParser(db *sql.DB, teamsService *TeamsService, logoFetcher *LogoFetcherService, fixtureParser *FixtureParser) *OddsChangeParser {
	return &OddsChangeParser{
		db:            db,
		logger:        log.New(os.Stdout, "", log.LstdFlags),
		teamsService:  teamsService,
		logoFetcher:   logoFetcher,
		fixtureParser: fixtureParser,
	}
}

// ParseAndStore 解析并存储 Odds Change 消息
func (p *OddsChangeParser) ParseAndStore(xmlContent string) error {
	var oddsChange OddsChangeMessage
	if err := xml.Unmarshal([]byte(xmlContent), &oddsChange); err != nil {
		return fmt.Errorf("failed to parse odds_change message: %w", err)
	}

	// 日志在处理完成后输出

	// 提取比分和状态信息
	var homeScore, awayScore *int
	var matchStatus, status string
	var matchTime string

	// 从 sport_event_status 获取比分和状态
	if oddsChange.SportEventStatus != nil {
		ses := oddsChange.SportEventStatus
		homeScore = ses.HomeScore
		awayScore = ses.AwayScore
		matchStatus = ses.MatchStatus
		status = ses.Status

		if ses.Clock != nil {
			matchTime = ses.Clock.MatchTime
		}

		// 提取日志已移除
	}

	// 提取主客队信息
	var homeTeamID, homeTeamName, awayTeamID, awayTeamName string
	for _, comp := range oddsChange.SportEvent.Competitors {
		if comp.Qualifier == "home" {
			homeTeamID = comp.ID
			homeTeamName = comp.Name
		} else if comp.Qualifier == "away" {
			awayTeamID = comp.ID
			awayTeamName = comp.Name
		}
	}

	// 处理队伍信息：检查并创建队伍记录，如果是新队伍则安排 Logo 获取
	if p.teamsService != nil && p.logoFetcher != nil {
		// 处理主队
		if homeTeamID != "" && homeTeamName != "" {
			p.processTeam(homeTeamID, homeTeamName, oddsChange.EventID)
		}
		// 处理客队
		if awayTeamID != "" && awayTeamName != "" {
			p.processTeam(awayTeamID, awayTeamName, oddsChange.EventID)
		}
	}

		// 存储到数据库
		statusOrder := p.getStatusOrder(status)
		if err := p.storeOddsChangeData(
			oddsChange.EventID,
			homeScore,
			awayScore,
			matchStatus,
			status,
			matchTime,
			homeTeamID,
			homeTeamName,
			awayTeamID,
			awayTeamName,
			statusOrder,
		); err != nil {
		return fmt.Errorf("failed to store odds_change data: %w", err)
	}

	// 统计市场和结果数量
	marketCount := len(oddsChange.Odds.Markets)
	outcomeCount := 0
	for _, market := range oddsChange.Odds.Markets {
		outcomeCount += len(market.Outcomes)
	}
	
	// 格式化日志
	logParts := []string{
		fmt.Sprintf("%d个市场", marketCount),
		fmt.Sprintf("%d个结果", outcomeCount),
	}
	
	// 添加比分
	if homeScore != nil && awayScore != nil {
		logParts = append(logParts, fmt.Sprintf("比分 %d-%d", *homeScore, *awayScore))
	}
	
	// 添加比赛阶段
	if matchStatus != "" {
		matchStatusNames := map[string]string{
			"0": "未开始",
			"1": "上半场",
			"2": "中场",
			"3": "下半场",
			"4": "加时",
			"5": "点球",
			"6": "已结束",
			"7": "已取消",
		}
		if name, ok := matchStatusNames[matchStatus]; ok {
			logParts = append(logParts, name)
		} else {
			logParts = append(logParts, fmt.Sprintf("阶段%s", matchStatus))
		}
	}
	
	p.logger.Printf("[odds_change] 比赛 %s: %s",
		oddsChange.EventID, strings.Join(logParts, ", "))

	// 检查是否需要获取 Fixture 信息
	if p.fixtureParser != nil && p.needsFixtureInfo(oddsChange.EventID) {
		// 异步获取 Fixture 信息，不阻塞当前消息处理
		go func(eventID string) {
			if err := p.fixtureParser.FetchAndUpdateFixture(eventID); err != nil {
				p.logger.Printf("[odds_change] ⚠️  Failed to fetch fixture for %s: %v", eventID, err)
			} else {
				p.logger.Printf("[odds_change] ✓ Fetched fixture info for %s", eventID)
			}
		}(oddsChange.EventID)
	}

	return nil
}

// getStatusOrder 为比赛状态分配一个数值，用于确保状态的单向更新
func (p *OddsChangeParser) getStatusOrder(status string) int {
	switch status {
	case "closed":
		return 50
	case "ended":
		return 40
	case "live":
		return 30
	case "suspended", "interrupted", "delayed":
		return 20
	case "not_started", "postponed", "cancelled", "abandoned":
		return 10
	default:
		return 0
	}
}

// storeOddsChangeData 存储 Odds Change 数据到数据库
func (p *OddsChangeParser) storeOddsChangeData(
		eventID string,
		homeScore, awayScore *int,
	matchStatus, status, matchTime string,
			homeTeamID, homeTeamName, awayTeamID, awayTeamName string,
			statusOrder int,
		) error {
	// 将 sport_event_status.status 数字映射为状态名称
	statusMap := map[string]string{
		"0": "not_started",
		"1": "live",
		"2": "suspended",
		"3": "ended",      // 比赛结束
		"4": "closed",     // 结果确认
		"5": "cancelled",
		"6": "delayed",
		"7": "interrupted",
		"8": "postponed",
		"9": "abandoned",
	}
	
	statusName := ""
	if name, ok := statusMap[status]; ok {
		statusName = name
	}
	
	// 尝试从 event_id 提取 tournament_id (例如 sr:match:12345 -> sr:tournament:xxx 需要从 API 获取，但我们可以先占位)
	// 实际上，如果 event_id 包含 tournament 信息，我们可以尝试提取
	extractedTournamentID := ""
	if strings.Contains(eventID, "sr:match:") {
		// 某些情况下 event_id 可能包含 tournament 信息，但通常需要通过 API 获取
	}

	// 更新 tracked_events 表 (不再使用 ld_matches)
	query := `INSERT INTO tracked_events (event_id, tournament_id, home_score, away_score, match_status, status, status_order, home_team_id, away_team_id, home_team_name, away_team_name, last_message_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) ON CONFLICT (event_id) DO UPDATE SET home_score = EXCLUDED.home_score, away_score = EXCLUDED.away_score, match_status = CASE WHEN EXCLUDED.match_status = '' THEN tracked_events.match_status ELSE EXCLUDED.match_status END, status = CASE WHEN EXCLUDED.status = '' THEN tracked_events.status ELSE EXCLUDED.status END, status_order = CASE WHEN EXCLUDED.status_order > tracked_events.status_order THEN EXCLUDED.status_order ELSE tracked_events.status_order END, home_team_id = CASE WHEN EXCLUDED.home_team_id = '' THEN tracked_events.home_team_id ELSE EXCLUDED.home_team_id END, away_team_id = CASE WHEN EXCLUDED.away_team_id = '' THEN tracked_events.away_team_id ELSE EXCLUDED.away_team_id END, home_team_name = CASE WHEN EXCLUDED.home_team_name = '' THEN tracked_events.home_team_name ELSE EXCLUDED.home_team_name END, away_team_name = CASE WHEN EXCLUDED.away_team_name = '' THEN tracked_events.away_team_name ELSE EXCLUDED.away_team_name END, last_message_at = EXCLUDED.last_message_at, updated_at = EXCLUDED.updated_at`

	now := time.Now()
	var t1Score, t2Score int
	if homeScore != nil {
		t1Score = *homeScore
	}
		if awayScore != nil {
			t2Score = *awayScore
		}
		
		// 使用 status 如果 matchStatus 为空
		finalStatus := fmt.Sprintf("%s", matchStatus)
		if finalStatus == "" {
			finalStatus = status
		}
		
			statusOrder = p.getStatusOrder(statusName)

	// p.logger.Printf("[DEBUG] SQL Query: %s, Args: event_id=%v, home_score=%v, away_score=%v, match_status=%v, match_time=%v, status=%v", CleanSQLQuery(query), eventID, t1Score, t2Score, finalStatus, matchTime, statusName)
		
		_, err := p.db.Exec(
				query,
						eventID, extractedTournamentID, t1Score, t2Score, finalStatus, statusName, statusOrder,
						homeTeamID, awayTeamID, homeTeamName, awayTeamName,
						now, now, now,
			)
		if err != nil {
			return fmt.Errorf("failed to upsert tracked_events: %w", err)
		}

	return nil
}

// formatScore 格式化比分用于日志输出
func formatScore(score *int) string {
	if score == nil {
		return "?"
	}
	return fmt.Sprintf("%d", *score)
}


// needsFixtureInfo 检查赛事是否需要获取 Fixture 信息
// 如果赛事没有 category_id 或 tournament_id，则需要获取
func (p *OddsChangeParser) needsFixtureInfo(eventID string) bool {
	var categoryID, tournamentID string
	err := p.db.QueryRow(
		"SELECT category_id, tournament_id FROM tracked_events WHERE event_id = $1",
		eventID,
	).Scan(&categoryID, &tournamentID)
	
	// 如果记录不存在或查询失败，需要获取
	if err != nil {
		return true
	}
	
	// 如果 category_id 或 tournament_id 为空，需要获取
	return categoryID == "" || tournamentID == ""
}

// processTeam 处理队伍信息，检查是否为新队伍并安排 Logo 获取
func (p *OddsChangeParser) processTeam(teamID, teamName, eventID string) {
	// 尝试获取或创建队伍记录
	teamInfo := TeamInfo{
		TeamID:   teamID,
		TeamName: teamName,
		// 注意：这里没有 sport_id 等信息，可以后续从其他地方补充
	}
	
	team, isNew, err := p.teamsService.GetOrCreateTeam(teamInfo)
	if err != nil {
		p.logger.Printf("[OddsChangeParser] ⚠️  Failed to process team %s: %v", teamName, err)
		return
	}
	
	// 如果是新队伍，异步安排 Logo 获取
	if isNew {
		p.logger.Printf("[OddsChangeParser] 🆕 New team detected: %s (ID: %s), scheduling logo fetch", teamName, teamID)
		p.logoFetcher.ScheduleLogoFetch(teamID, teamName)
	}
	
	// 如果队伍已存在但 Logo 未获取，也可以尝试重新获取（可选）
	if !isNew && !team.LogoFetched && team.LogoFetchRetryCount < 3 {
		p.logger.Printf("[OddsChangeParser] 🔄 Team %s exists but logo not fetched, retrying", teamName)
		p.logoFetcher.ScheduleLogoFetch(teamID, teamName)
	}
}
