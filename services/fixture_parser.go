package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"time"
)

// FixtureParser Fixture 消息解析器
type FixtureParser struct {
	db               *sql.DB
	srnMappingService *SRNMappingService
	logger           *log.Logger
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
func NewFixtureParser(db *sql.DB, srnMappingService *SRNMappingService) *FixtureParser {
	return &FixtureParser{
		db:               db,
		srnMappingService: srnMappingService,
		logger:           log.New(log.Writer(), "[FixtureParser] ", log.LstdFlags),
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
	eventID, srnID string,
	scheduleTime *time.Time,
	homeTeamID, homeTeamName, awayTeamID, awayTeamName, status string,
) error {
	// 使用 UPSERT 更新或插入 tracked_events
	query := `
		INSERT INTO tracked_events (
			event_id, srn_id, schedule_time,
			home_team_id, home_team_name,
			away_team_id, away_team_name,
			match_status, subscribed,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, $10)
		ON CONFLICT (event_id) DO UPDATE SET
			srn_id = COALESCE(NULLIF(EXCLUDED.srn_id, ''), tracked_events.srn_id),
			schedule_time = COALESCE(EXCLUDED.schedule_time, tracked_events.schedule_time),
			home_team_id = COALESCE(NULLIF(EXCLUDED.home_team_id, ''), tracked_events.home_team_id),
			home_team_name = COALESCE(NULLIF(EXCLUDED.home_team_name, ''), tracked_events.home_team_name),
			away_team_id = COALESCE(NULLIF(EXCLUDED.away_team_id, ''), tracked_events.away_team_id),
			away_team_name = COALESCE(NULLIF(EXCLUDED.away_team_name, ''), tracked_events.away_team_name),
			match_status = COALESCE(NULLIF(EXCLUDED.match_status, ''), tracked_events.match_status),
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.Exec(
		query,
		eventID, srnID, scheduleTime,
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
	}

	var fixtureChange FixtureChange
	if err := xml.Unmarshal([]byte(xmlContent), &fixtureChange); err != nil {
		return fmt.Errorf("failed to parse fixture_change: %w", err)
	}

	p.logger.Printf("Parsing fixture_change for event: %s", eventID)

	// 更新 schedule_time
	if fixtureChange.StartTime > 0 {
		scheduleTime := time.UnixMilli(fixtureChange.StartTime)
		query := `
			UPDATE tracked_events 
			SET schedule_time = $1, updated_at = $2
			WHERE event_id = $3
		`
		if _, err := p.db.Exec(query, scheduleTime, time.Now(), eventID); err != nil {
			return fmt.Errorf("failed to update schedule_time: %w", err)
		}

		p.logger.Printf("Updated schedule_time for event %s: %s", eventID, scheduleTime.Format(time.RFC3339))
	}

	return nil
}

