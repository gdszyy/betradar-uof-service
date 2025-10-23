package thesports

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Esports-specific models

// EsportsGame represents an esports game type
type EsportsGame struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`       // LOL, DOTA2, CSGO, etc.
	ShortName string `json:"short_name"`
	Logo      string `json:"logo"`
}

// EsportsMatch represents an esports match
type EsportsMatch struct {
	ID              int                 `json:"id"`
	GameID          int                 `json:"game_id"`
	Game            *EsportsGame        `json:"game"`
	CompetitionID   int                 `json:"competition_id"`
	Competition     *Competition        `json:"competition"`
	SeasonID        int                 `json:"season_id"`
	StageID         int                 `json:"stage_id"`
	Round           int                 `json:"round"`
	HomeTeamID      int                 `json:"home_team_id"`
	AwayTeamID      int                 `json:"away_team_id"`
	HomeTeam        *EsportsTeam        `json:"home_team"`
	AwayTeam        *EsportsTeam        `json:"away_team"`
	StartTime       time.Time           `json:"start_time"`
	Status          string              `json:"status"`
	HomeScore       int                 `json:"home_score"` // 赢得的局数
	AwayScore       int                 `json:"away_score"` // 赢得的局数
	BestOf          int                 `json:"best_of"`    // BO1, BO3, BO5
	Games           []EsportsGameDetail `json:"games"`      // 各局详情
	Statistics      *EsportsStats       `json:"statistics"`
	LiveData        *EsportsLiveData    `json:"live_data"`
}

// EsportsTeam represents an esports team
type EsportsTeam struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	Logo      string `json:"logo"`
	CountryID int    `json:"country_id"`
}

// EsportsGameDetail represents a single game in a match
type EsportsGameDetail struct {
	GameNumber  int                  `json:"game_number"` // 第几局
	Winner      string               `json:"winner"`      // home, away
	Duration    int                  `json:"duration"`    // 游戏时长（秒）
	HomeScore   int                  `json:"home_score"`  // 游戏内得分
	AwayScore   int                  `json:"away_score"`  // 游戏内得分
	Map         string               `json:"map"`         // 地图名称（CSGO, Valorant）
	Side        string               `json:"side"`        // 阵营（LOL, DOTA2）
	Statistics  map[string]interface{} `json:"statistics"`
	Events      []EsportsGameEvent   `json:"events"`
}

// EsportsStats represents esports match statistics
type EsportsStats struct {
	HomeTeam EsportsTeamStats `json:"home_team"`
	AwayTeam EsportsTeamStats `json:"away_team"`
}

// EsportsTeamStats represents team statistics (varies by game)
type EsportsTeamStats struct {
	// Common stats
	Kills   int `json:"kills"`
	Deaths  int `json:"deaths"`
	Assists int `json:"assists"`
	
	// MOBA specific (LOL, DOTA2)
	Gold          int     `json:"gold"`
	Towers        int     `json:"towers"`
	Dragons       int     `json:"dragons"`        // LOL
	Barons        int     `json:"barons"`         // LOL
	Roshans       int     `json:"roshans"`        // DOTA2
	
	// FPS specific (CSGO, Valorant)
	Rounds        int     `json:"rounds"`
	FirstKills    int     `json:"first_kills"`
	Headshots     int     `json:"headshots"`
	HeadshotPct   float64 `json:"headshot_pct"`
	
	// Player stats
	Players       []EsportsPlayerStats `json:"players"`
}

// EsportsPlayerStats represents player statistics
type EsportsPlayerStats struct {
	PlayerID    int     `json:"player_id"`
	Player      *Player `json:"player"`
	Kills       int     `json:"kills"`
	Deaths      int     `json:"deaths"`
	Assists     int     `json:"assists"`
	KDA         float64 `json:"kda"` // (K+A)/D
	
	// MOBA specific
	Champion    string  `json:"champion"`    // LOL
	Hero        string  `json:"hero"`        // DOTA2
	Gold        int     `json:"gold"`
	CS          int     `json:"cs"`          // Creep Score
	Damage      int     `json:"damage"`
	
	// FPS specific
	Headshots   int     `json:"headshots"`
	ADR         float64 `json:"adr"`         // Average Damage per Round
	Rating      float64 `json:"rating"`
}

// EsportsGameEvent represents an in-game event
type EsportsGameEvent struct {
	Type        string    `json:"type"` // kill, death, tower, dragon, baron, round_end, etc.
	Time        int       `json:"time"` // 游戏内时间（秒）
	TeamID      int       `json:"team_id"`
	PlayerID    int       `json:"player_id"`
	Player      *Player   `json:"player"`
	TargetID    int       `json:"target_id"`    // 被击杀的玩家
	Target      *Player   `json:"target"`
	Description string    `json:"description"`
	X           float64   `json:"x"` // 位置坐标
	Y           float64   `json:"y"`
}

// EsportsLiveData represents real-time game data
type EsportsLiveData struct {
	GameNumber    int                  `json:"game_number"`
	GameTime      int                  `json:"game_time"` // 当前游戏时间（秒）
	Status        string               `json:"status"`    // in_progress, paused, finished
	HomeTeamData  *EsportsLiveTeamData `json:"home_team_data"`
	AwayTeamData  *EsportsLiveTeamData `json:"away_team_data"`
	RecentEvents  []EsportsGameEvent   `json:"recent_events"`
}

// EsportsLiveTeamData represents real-time team data
type EsportsLiveTeamData struct {
	Kills       int                    `json:"kills"`
	Gold        int                    `json:"gold"`
	Towers      int                    `json:"towers"`
	Dragons     int                    `json:"dragons"`
	Barons      int                    `json:"barons"`
	Players     []EsportsLivePlayerData `json:"players"`
}

// EsportsLivePlayerData represents real-time player data
type EsportsLivePlayerData struct {
	PlayerID    int     `json:"player_id"`
	Kills       int     `json:"kills"`
	Deaths      int     `json:"deaths"`
	Assists     int     `json:"assists"`
	Gold        int     `json:"gold"`
	CS          int     `json:"cs"`
	Level       int     `json:"level"`
	Items       []int   `json:"items"` // 装备 ID 列表
}

// Esports API Methods

// GetEsportsGames retrieves all supported esports games
func (c *Client) GetEsportsGames() ([]EsportsGame, error) {
	body, err := c.get("/v1/esports/games", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int           `json:"code"`
		Msg  string        `json:"msg"`
		Data []EsportsGame `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetEsportsCompetitions retrieves esports competitions
func (c *Client) GetEsportsCompetitions(gameID int) ([]Competition, error) {
	params := url.Values{}
	if gameID > 0 {
		params.Set("game_id", strconv.Itoa(gameID))
	}

	body, err := c.get("/v1/esports/competitions", params)
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

// GetEsportsMatches retrieves esports matches
func (c *Client) GetEsportsMatches(opts EsportsMatchOptions) ([]EsportsMatch, error) {
	params := url.Values{}
	
	if opts.GameID > 0 {
		params.Set("game_id", strconv.Itoa(opts.GameID))
	}
	if opts.CompetitionID > 0 {
		params.Set("competition_id", strconv.Itoa(opts.CompetitionID))
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

	body, err := c.get("/v1/esports/matches", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int             `json:"code"`
		Msg  string          `json:"msg"`
		Data []EsportsMatch  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// EsportsMatchOptions represents options for querying esports matches
type EsportsMatchOptions struct {
	GameID        int
	CompetitionID int
	TeamID        int
	Date          time.Time
	DateFrom      time.Time
	DateTo        time.Time
	Status        string // not_started, live, finished
}

// GetEsportsTodayMatches retrieves today's esports matches
func (c *Client) GetEsportsTodayMatches(gameID int) ([]EsportsMatch, error) {
	return c.GetEsportsMatches(EsportsMatchOptions{
		GameID: gameID,
		Date:   time.Now(),
	})
}

// GetEsportsLiveMatches retrieves live esports matches
func (c *Client) GetEsportsLiveMatches(gameID int) ([]EsportsMatch, error) {
	return c.GetEsportsMatches(EsportsMatchOptions{
		GameID: gameID,
		Status: "live",
	})
}

// GetEsportsMatch retrieves a specific esports match
func (c *Client) GetEsportsMatch(matchID int) (*EsportsMatch, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/esports/match", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int           `json:"code"`
		Msg  string        `json:"msg"`
		Data EsportsMatch  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetEsportsMatchStatistics retrieves esports match statistics
func (c *Client) GetEsportsMatchStatistics(matchID int) (*EsportsStats, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/esports/match/statistics", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int           `json:"code"`
		Msg  string        `json:"msg"`
		Data EsportsStats  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetEsportsMatchLive retrieves real-time esports match data
func (c *Client) GetEsportsMatchLive(matchID int) (*EsportsLiveData, error) {
	params := url.Values{}
	params.Set("match_id", strconv.Itoa(matchID))

	body, err := c.get("/v1/esports/match/live", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int              `json:"code"`
		Msg  string           `json:"msg"`
		Data EsportsLiveData  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetEsportsTeam retrieves an esports team
func (c *Client) GetEsportsTeam(teamID int) (*EsportsTeam, error) {
	params := url.Values{}
	params.Set("team_id", strconv.Itoa(teamID))

	body, err := c.get("/v1/esports/team", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int          `json:"code"`
		Msg  string       `json:"msg"`
		Data EsportsTeam  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// Game-specific helper methods

// GetLOLMatches retrieves League of Legends matches
func (c *Client) GetLOLMatches(opts EsportsMatchOptions) ([]EsportsMatch, error) {
	opts.GameID = 1 // Assuming LOL game_id is 1
	return c.GetEsportsMatches(opts)
}

// GetDOTA2Matches retrieves DOTA2 matches
func (c *Client) GetDOTA2Matches(opts EsportsMatchOptions) ([]EsportsMatch, error) {
	opts.GameID = 2 // Assuming DOTA2 game_id is 2
	return c.GetEsportsMatches(opts)
}

// GetCSGOMatches retrieves CS:GO matches
func (c *Client) GetCSGOMatches(opts EsportsMatchOptions) ([]EsportsMatch, error) {
	opts.GameID = 3 // Assuming CSGO game_id is 3
	return c.GetEsportsMatches(opts)
}

