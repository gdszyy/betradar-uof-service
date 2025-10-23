package thesports

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Basketball-specific models

// BasketballMatch represents a basketball match
type BasketballMatch struct {
	ID              int                  `json:"id"`
	CompetitionID   int                  `json:"competition_id"`
	SeasonID        int                  `json:"season_id"`
	StageID         int                  `json:"stage_id"`
	Round           int                  `json:"round"`
	HomeTeamID      int                  `json:"home_team_id"`
	AwayTeamID      int                  `json:"away_team_id"`
	HomeTeam        *Team                `json:"home_team"`
	AwayTeam        *Team                `json:"away_team"`
	VenueID         int                  `json:"venue_id"`
	StartTime       time.Time            `json:"start_time"`
	Status          string               `json:"status"`
	HomeScore       int                  `json:"home_score"`
	AwayScore       int                  `json:"away_score"`
	Quarter         int                  `json:"quarter"`         // 当前节数 (1-4)
	Overtime        int                  `json:"overtime"`        // 加时赛数
	QuarterScores   *BasketballQuarters  `json:"quarter_scores"`  // 各节比分
	Statistics      *BasketballStats     `json:"statistics"`
	Incidents       []BasketballIncident `json:"incidents"`
	Lineups         *BasketballLineups   `json:"lineups"`
	Trends          *BasketballTrends    `json:"trends"`          // 实时走势
	ShootingPoints  []ShootingPoint      `json:"shooting_points"` // 实时投篮点位
}

// BasketballQuarters represents quarter-by-quarter scores
type BasketballQuarters struct {
	HomeQ1 int `json:"home_q1"` // 主队第1节
	AwayQ1 int `json:"away_q1"` // 客队第1节
	HomeQ2 int `json:"home_q2"` // 主队第2节
	AwayQ2 int `json:"away_q2"` // 客队第2节
	HomeQ3 int `json:"home_q3"` // 主队第3节
	AwayQ3 int `json:"away_q3"` // 客队第3节
	HomeQ4 int `json:"home_q4"` // 主队第4节
	AwayQ4 int `json:"away_q4"` // 客队第4节
	HomeOT int `json:"home_ot"` // 主队加时
	AwayOT int `json:"away_ot"` // 客队加时
}

// BasketballStats represents basketball match statistics
type BasketballStats struct {
	HomeTeam BasketballTeamStats `json:"home_team"`
	AwayTeam BasketballTeamStats `json:"away_team"`
}

// BasketballTeamStats represents team statistics
type BasketballTeamStats struct {
	Points              int     `json:"points"`                // 得分
	FieldGoalsMade      int     `json:"field_goals_made"`      // 投篮命中
	FieldGoalsAttempted int     `json:"field_goals_attempted"` // 投篮出手
	FieldGoalPct        float64 `json:"field_goal_pct"`        // 投篮命中率
	ThreePointsMade     int     `json:"three_points_made"`     // 三分命中
	ThreePointsAttempted int    `json:"three_points_attempted"` // 三分出手
	ThreePointPct       float64 `json:"three_point_pct"`       // 三分命中率
	FreeThrowsMade      int     `json:"free_throws_made"`      // 罚球命中
	FreeThrowsAttempted int     `json:"free_throws_attempted"` // 罚球出手
	FreeThrowPct        float64 `json:"free_throw_pct"`        // 罚球命中率
	Rebounds            int     `json:"rebounds"`              // 篮板
	OffensiveRebounds   int     `json:"offensive_rebounds"`    // 进攻篮板
	DefensiveRebounds   int     `json:"defensive_rebounds"`    // 防守篮板
	Assists             int     `json:"assists"`               // 助攻
	Steals              int     `json:"steals"`                // 抢断
	Blocks              int     `json:"blocks"`                // 盖帽
	Turnovers           int     `json:"turnovers"`             // 失误
	Fouls               int     `json:"fouls"`                 // 犯规
	TimeoutsRemaining   int     `json:"timeouts_remaining"`    // 剩余暂停
}

// BasketballIncident represents a basketball match incident
type BasketballIncident struct {
	ID          int       `json:"id"`
	MatchID     int       `json:"match_id"`
	Type        string    `json:"type"` // made_shot, missed_shot, free_throw, rebound, assist, steal, block, turnover, foul, timeout, substitution
	Time        string    `json:"time"` // 比赛时间
	Quarter     int       `json:"quarter"`
	TeamID      int       `json:"team_id"`
	PlayerID    int       `json:"player_id"`
	Player      *Player   `json:"player"`
	Points      int       `json:"points"`      // 得分 (1/2/3)
	Description string    `json:"description"`
	HomeScore   int       `json:"home_score"`
	AwayScore   int       `json:"away_score"`
	X           float64   `json:"x"` // 投篮位置 X 坐标
	Y           float64   `json:"y"` // 投篮位置 Y 坐标
}

// BasketballLineups represents basketball match lineups
type BasketballLineups struct {
	HomeTeam BasketballTeamLineup `json:"home_team"`
	AwayTeam BasketballTeamLineup `json:"away_team"`
}

// BasketballTeamLineup represents a team's lineup
type BasketballTeamLineup struct {
	StartingFive []BasketballPlayerLineup `json:"starting_five"` // 首发五人
	Bench        []BasketballPlayerLineup `json:"bench"`         // 替补
}

// BasketballPlayerLineup represents a player in the lineup
type BasketballPlayerLineup struct {
	PlayerID     int     `json:"player_id"`
	Player       *Player `json:"player"`
	ShirtNumber  int     `json:"shirt_number"`
	Position     string  `json:"position"` // PG, SG, SF, PF, C
	IsStarter    bool    `json:"is_starter"`
	MinutesPlayed int    `json:"minutes_played"`
	Points       int     `json:"points"`
	Rebounds     int     `json:"rebounds"`
	Assists      int     `json:"assists"`
}

// BasketballTrends represents real-time match trends
type BasketballTrends struct {
	ScoreTrend []TrendPoint `json:"score_trend"` // 比分走势
}

// TrendPoint represents a point in the trend
type TrendPoint struct {
	Time      string `json:"time"`
	HomeScore int    `json:"home_score"`
	AwayScore int    `json:"away_score"`
}

// ShootingPoint represents a shooting point on the court
type ShootingPoint struct {
	PlayerID int     `json:"player_id"`
	TeamID   int     `json:"team_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Made     bool    `json:"made"`
	Points   int     `json:"points"` // 1, 2, or 3
	Quarter  int     `json:"quarter"`
}

// Basketball API Methods

// GetBasketballCompetitions retrieves all basketball competitions
func (c *Client) GetBasketballCompetitions(countryID int) ([]Competition, error) {
	params := url.Values{}
	if countryID > 0 {
		params.Set("country_id", strconv.Itoa(countryID))
	}

	body, err := c.get("/v1/basketball/competitions", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int           `json:"code"`
		Msg  string        `json:"msg"`
		Data []Competition `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetBasketballMatches retrieves basketball matches
func (c *Client) GetBasketballMatches(opts MatchListOptions) ([]BasketballMatch, error) {
	params := url.Values{}
	
	if opts.CompetitionID > 0 {
		params.Set("competition_id", strconv.Itoa(opts.CompetitionID))
	}
	if opts.SeasonID > 0 {
		params.Set("season_id", strconv.Itoa(opts.SeasonID))
	}
	if opts.TeamID > 0 {
		params.Set("team_id", strconv.Itoa(opts.TeamID))
	}
	if !opts.Date.IsZero() {
		params.Set("date", opts.Date.Format("2006-01-02"))
	}
	if !opts.DateFrom.IsZero() {
		params.Set("date_from", opts.DateFrom.Format("2006-01-02"))
	}
	if !opts.DateTo.IsZero() {
		params.Set("date_to", opts.DateTo.Format("2006-01-02"))
	}
	if opts.Status != "" {
		params.Set("status", opts.Status)
	}
	if opts.Page > 0 {
		params.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.PageSize > 0 {
		params.Set("page_size", strconv.Itoa(opts.PageSize))
	}

	body, err := c.get("/v1/basketball/matches", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int                `json:"code"`
		Msg  string             `json:"msg"`
		Data []BasketballMatch  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetBasketballTodayMatches retrieves today's basketball matches
func (c *Client) GetBasketballTodayMatches() ([]BasketballMatch, error) {
	return c.GetBasketballMatches(MatchListOptions{
		Date: time.Now(),
	})
}

// GetBasketballLiveMatches retrieves live basketball matches
func (c *Client) GetBasketballLiveMatches() ([]BasketballMatch, error) {
	return c.GetBasketballMatches(MatchListOptions{
		Status: "live",
	})
}

// GetBasketballMatch retrieves a specific basketball match
func (c *Client) GetBasketballMatch(matchID int) (*BasketballMatch, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/basketball/match", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int              `json:"code"`
		Msg  string           `json:"msg"`
		Data BasketballMatch  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetBasketballMatchStatistics retrieves basketball match statistics
func (c *Client) GetBasketballMatchStatistics(matchID int) (*BasketballStats, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/basketball/match/statistics", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int              `json:"code"`
		Msg  string           `json:"msg"`
		Data BasketballStats  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetBasketballMatchIncidents retrieves basketball match incidents
func (c *Client) GetBasketballMatchIncidents(matchID int) ([]BasketballIncident, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/basketball/match/incidents", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int                   `json:"code"`
		Msg  string                `json:"msg"`
		Data []BasketballIncident  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetBasketballMatchLineups retrieves basketball match lineups
func (c *Client) GetBasketballMatchLineups(matchID int) (*BasketballLineups, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/basketball/match/lineups", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int                `json:"code"`
		Msg  string             `json:"msg"`
		Data BasketballLineups  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetBasketballMatchTrends retrieves real-time match trends
func (c *Client) GetBasketballMatchTrends(matchID int) (*BasketballTrends, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/basketball/match/trends", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data BasketballTrends  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetBasketballMatchShootingPoints retrieves real-time shooting points
func (c *Client) GetBasketballMatchShootingPoints(matchID int) ([]ShootingPoint, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/basketball/match/shooting_points", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data []ShootingPoint `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

