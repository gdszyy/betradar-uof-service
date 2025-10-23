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
type OddsChangeMessage struct {
	XMLName    xml.Name        `xml:"odds_change"`
	EventID    string          `xml:"event_id,attr"`
	ProductID  int             `xml:"product,attr"`
	Timestamp  int64           `xml:"timestamp,attr"`
	SportEvent SportEventInfo  `xml:"sport_event"`
	Odds       OddsInfo        `xml:"odds"`
}

// SportEventInfo 赛事信息
type SportEventInfo struct {
	ID              string          `xml:"id,attr"`
	Scheduled       int64           `xml:"scheduled,attr"`
	StartTime       int64           `xml:"start_time,attr"`
	Status          string          `xml:"status,attr"`
	MatchStatus     string          `xml:"match_status,attr"`
	HomeScore       *int            `xml:"home_score,attr"`
	AwayScore       *int            `xml:"away_score,attr"`
	Competitors     []OddsCompetitor    `xml:"competitors>competitor"`
	SportEventStatus *SportEventStatus `xml:"sport_event_status"`
}

// SportEventStatus 赛事状态(包含比分信息)
type SportEventStatus struct {
	Status          string      `xml:"status,attr"`
	MatchStatus     string      `xml:"match_status,attr"`
	HomeScore       *int        `xml:"home_score,attr"`
	AwayScore       *int        `xml:"away_score,attr"`
	Clock           *ClockInfo  `xml:"clock"`
	PeriodScores    []PeriodScore `xml:"period_scores>period_score"`
}

// ClockInfo 比赛时钟信息
type ClockInfo struct {
	MatchTime       string `xml:"match_time,attr"`
	StoppageTime    string `xml:"stoppage_time,attr"`
	StoppageTimeAnnounced string `xml:"stoppage_time_announced,attr"`
}

// PeriodScore 分段比分
type PeriodScore struct {
	HomeScore int    `xml:"home_score,attr"`
	AwayScore int    `xml:"away_score,attr"`
	Type      string `xml:"type,attr"` // regular_period, overtime, penalties
	Number    int    `xml:"number,attr"`
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

	// 提取比分信息
	var homeScore, awayScore *int
	var matchStatus, status string
	var matchTime string

	// 优先从 sport_event_status 获取比分
	if oddsChange.SportEvent.SportEventStatus != nil {
		ses := oddsChange.SportEvent.SportEventStatus
		homeScore = ses.HomeScore
		awayScore = ses.AwayScore
		matchStatus = ses.MatchStatus
		status = ses.Status

		if ses.Clock != nil {
			matchTime = ses.Clock.MatchTime
		}
	}

	// 如果 sport_event_status 没有比分,从 sport_event 获取
	if homeScore == nil && oddsChange.SportEvent.HomeScore != nil {
		homeScore = oddsChange.SportEvent.HomeScore
	}
	if awayScore == nil && oddsChange.SportEvent.AwayScore != nil {
		awayScore = oddsChange.SportEvent.AwayScore
	}
	if matchStatus == "" {
		matchStatus = oddsChange.SportEvent.MatchStatus
	}
	if status == "" {
		status = oddsChange.SportEvent.Status
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
	// 更新 ld_matches 表
	query := `
		INSERT INTO ld_matches (
			match_id, t1_score, t2_score, match_status, match_time,
			home_team_id, away_team_id, t1_name, t2_name,
			last_event_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (match_id) DO UPDATE SET
			t1_score = COALESCE(EXCLUDED.t1_score, ld_matches.t1_score),
			t2_score = COALESCE(EXCLUDED.t2_score, ld_matches.t2_score),
			match_status = COALESCE(NULLIF(EXCLUDED.match_status, ''), ld_matches.match_status),
			match_time = COALESCE(NULLIF(EXCLUDED.match_time, ''), ld_matches.match_time),
			home_team_id = COALESCE(NULLIF(EXCLUDED.home_team_id, ''), ld_matches.home_team_id),
			away_team_id = COALESCE(NULLIF(EXCLUDED.away_team_id, ''), ld_matches.away_team_id),
			t1_name = COALESCE(NULLIF(EXCLUDED.t1_name, ''), ld_matches.t1_name),
			t2_name = COALESCE(NULLIF(EXCLUDED.t2_name, ''), ld_matches.t2_name),
			last_event_at = EXCLUDED.last_event_at,
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
		eventID, t1Score, t2Score, finalStatus, matchTime,
		homeTeamID, awayTeamID, homeTeamName, awayTeamName,
		now, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert ld_matches: %w", err)
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

