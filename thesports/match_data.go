package thesports

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// MatchListOptions represents options for listing matches
type MatchListOptions struct {
	CompetitionID int
	SeasonID      int
	TeamID        int
	Date          time.Time
	DateFrom      time.Time
	DateTo        time.Time
	Status        string // not_started, live, finished
	Page          int
	PageSize      int
}

// GetMatches retrieves a list of matches
func (c *Client) GetMatches(opts MatchListOptions) ([]Match, error) {
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

	body, err := c.get("/v1/football/matches", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int     `json:"code"`
		Msg  string  `json:"msg"`
		Data []Match `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetMatch retrieves detailed information about a specific match
func (c *Client) GetMatch(matchID int) (*Match, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data Match  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetMatchLineups retrieves lineups for a specific match
func (c *Client) GetMatchLineups(matchID int) (*MatchLineups, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/lineups", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int          `json:"code"`
		Msg  string       `json:"msg"`
		Data MatchLineups `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetMatchStatistics retrieves statistics for a specific match
func (c *Client) GetMatchStatistics(matchID int) (*MatchStats, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/statistics", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int        `json:"code"`
		Msg  string     `json:"msg"`
		Data MatchStats `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetMatchIncidents retrieves incidents/events for a specific match
func (c *Client) GetMatchIncidents(matchID int) ([]Incident, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/incidents", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int        `json:"code"`
		Msg  string     `json:"msg"`
		Data []Incident `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetMatchLive retrieves live data for a specific match
func (c *Client) GetMatchLive(matchID int) (*MatchLive, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/live", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int       `json:"code"`
		Msg  string    `json:"msg"`
		Data MatchLive `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetMatchCommentary retrieves commentary for a specific match
func (c *Client) GetMatchCommentary(matchID int) ([]Commentary, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/commentary", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int          `json:"code"`
		Msg  string       `json:"msg"`
		Data []Commentary `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetMatchH2H retrieves head-to-head data for a specific match
func (c *Client) GetMatchH2H(matchID int) (*H2H, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/h2h", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data H2H    `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetMatchOdds retrieves odds for a specific match
func (c *Client) GetMatchOdds(matchID int) ([]Odds, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/odds", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data []Odds `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetMatchPrediction retrieves prediction for a specific match
func (c *Client) GetMatchPrediction(matchID int) (*Prediction, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/prediction", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int        `json:"code"`
		Msg  string     `json:"msg"`
		Data Prediction `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetMatchVideos retrieves videos for a specific match
func (c *Client) GetMatchVideos(matchID int) ([]MatchVideo, error) {
	endpoint := fmt.Sprintf("/v1/football/matches/%d/videos", matchID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int          `json:"code"`
		Msg  string       `json:"msg"`
		Data []MatchVideo `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetLiveMatches retrieves all currently live matches
func (c *Client) GetLiveMatches() ([]Match, error) {
	return c.GetMatches(MatchListOptions{
		Status: "live",
	})
}

// GetTodayMatches retrieves all matches for today
func (c *Client) GetTodayMatches() ([]Match, error) {
	return c.GetMatches(MatchListOptions{
		Date: time.Now(),
	})
}

