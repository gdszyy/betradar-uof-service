package models

import "time"

// Match 统一的比赛模型
type Match struct {
	ID          string    `json:"id"`
	SportID     string    `json:"sport_id"`
	SportName   string    `json:"sport_name"`
	HomeTeam    Team      `json:"home_team"`
	AwayTeam    Team      `json:"away_team"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time"`
	Score       Score     `json:"score"`
	Statistics  Statistics `json:"statistics,omitempty"`
	Source      string    `json:"source"` // uof, ld, thesports
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Team 队伍信息
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Score 比分信息
type Score struct {
	Home int `json:"home"`
	Away int `json:"away"`
}

// Statistics 比赛统计
type Statistics struct {
	Possession      *int `json:"possession,omitempty"`
	Shots           *int `json:"shots,omitempty"`
	ShotsOnTarget   *int `json:"shots_on_target,omitempty"`
	Corners         *int `json:"corners,omitempty"`
	YellowCards     *int `json:"yellow_cards,omitempty"`
	RedCards        *int `json:"red_cards,omitempty"`
}

// MatchStatus 比赛状态
type MatchStatus string

const (
	MatchStatusNotStarted MatchStatus = "not_started"
	MatchStatusLive       MatchStatus = "live"
	MatchStatusEnded      MatchStatus = "ended"
	MatchStatusClosed     MatchStatus = "closed"
	MatchStatusCancelled  MatchStatus = "cancelled"
	MatchStatusPostponed  MatchStatus = "postponed"
)

