package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"time"
)

// OddsChangeParser Odds Change 消息解析器
type OddsChangeParser struct {
	db     *sql.DB
	logger *log.Logger
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
	ID       int       `xml:"id,attr"`
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
func NewOddsChangeParser(db *sql.DB) *OddsChangeParser {
	return &OddsChangeParser{
		db:     db,
		logger: log.New(log.Writer(), "[OddsChangeParser] ", log.LstdFlags),
	}
}

// ParseAndStore 解析并存储 Odds Change 消息
func (p *OddsChangeParser) ParseAndStore(xmlContent string) error {
	var oddsChange OddsChangeMessage
	if err := xml.Unmarshal([]byte(xmlContent), &oddsChange); err != nil {
		return fmt.Errorf("failed to parse odds_change message: %w", err)
	}

	p.logger.Printf("Parsing odds_change for event: %s", oddsChange.EventID)

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

		p.logger.Printf("Extracted from sport_event_status: score=%v-%v, status=%s, match_status=%s, time=%s",
			formatScore(homeScore), formatScore(awayScore), status, matchStatus, matchTime)
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

	// 存储到数据库
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
	); err != nil {
		return fmt.Errorf("failed to store odds_change data: %w", err)
	}

	p.logger.Printf("Stored odds_change data for event %s: %v-%v, status=%s, time=%s",
		oddsChange.EventID,
		formatScore(homeScore),
		formatScore(awayScore),
		matchStatus,
		matchTime)

	return nil
}

// storeOddsChangeData 存储 Odds Change 数据到数据库
func (p *OddsChangeParser) storeOddsChangeData(
	eventID string,
	homeScore, awayScore *int,
	matchStatus, status, matchTime string,
	homeTeamID, homeTeamName, awayTeamID, awayTeamName string,
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
	
	// 更新 tracked_events 表 (不再使用 ld_matches)
	query := `
		INSERT INTO tracked_events (
			event_id, home_score, away_score, match_status, match_time, status,
			home_team_id, away_team_id, home_team_name, away_team_name,
			last_message_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (event_id) DO UPDATE SET
			home_score = COALESCE(EXCLUDED.home_score, tracked_events.home_score),
			away_score = COALESCE(EXCLUDED.away_score, tracked_events.away_score),
			match_status = COALESCE(NULLIF(EXCLUDED.match_status, ''), tracked_events.match_status),
			match_time = COALESCE(NULLIF(EXCLUDED.match_time, ''), tracked_events.match_time),
			status = COALESCE(NULLIF(EXCLUDED.status, ''), tracked_events.status),
			home_team_id = COALESCE(NULLIF(EXCLUDED.home_team_id, ''), tracked_events.home_team_id),
			away_team_id = COALESCE(NULLIF(EXCLUDED.away_team_id, ''), tracked_events.away_team_id),
			home_team_name = COALESCE(NULLIF(EXCLUDED.home_team_name, ''), tracked_events.home_team_name),
			away_team_name = COALESCE(NULLIF(EXCLUDED.away_team_name, ''), tracked_events.away_team_name),
			last_message_at = EXCLUDED.last_message_at,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	var t1Score, t2Score int
	if homeScore != nil {
		t1Score = *homeScore
	}
	if awayScore != nil {
		t2Score = *awayScore
	}

	// 使用 status 如果 matchStatus 为空
	finalStatus := matchStatus
	if finalStatus == "" {
		finalStatus = status
	}

	_, err := p.db.Exec(
		query,
		eventID, t1Score, t2Score, finalStatus, matchTime, statusName,
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

