package thesports

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GetCategories retrieves all sport categories
func (c *Client) GetCategories() ([]Category, error) {
	body, err := c.get("/v1/football/categories", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int        `json:"code"`
		Msg  string     `json:"msg"`
		Data []Category `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetCountries retrieves all countries
func (c *Client) GetCountries() ([]Country, error) {
	body, err := c.get("/v1/football/countries", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int       `json:"code"`
		Msg  string    `json:"msg"`
		Data []Country `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetCompetitions retrieves competitions
func (c *Client) GetCompetitions(countryID int) ([]Competition, error) {
	params := url.Values{}
	if countryID > 0 {
		params.Set("country_id", strconv.Itoa(countryID))
	}

	body, err := c.get("/v1/football/competitions", params)
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

// GetCompetition retrieves a specific competition by ID
func (c *Client) GetCompetition(competitionID int) (*Competition, error) {
	endpoint := fmt.Sprintf("/v1/football/competitions/%d", competitionID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data Competition `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetTeam retrieves a specific team by ID
func (c *Client) GetTeam(teamID int) (*Team, error) {
	endpoint := fmt.Sprintf("/v1/football/teams/%d", teamID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data Team   `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetTeams retrieves teams by competition
func (c *Client) GetTeams(competitionID, seasonID int) ([]Team, error) {
	params := url.Values{}
	if competitionID > 0 {
		params.Set("competition_id", strconv.Itoa(competitionID))
	}
	if seasonID > 0 {
		params.Set("season_id", strconv.Itoa(seasonID))
	}

	body, err := c.get("/v1/football/teams", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data []Team `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetPlayer retrieves a specific player by ID
func (c *Client) GetPlayer(playerID int) (*Player, error) {
	endpoint := fmt.Sprintf("/v1/football/players/%d", playerID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data Player `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetCoach retrieves a specific coach by ID
func (c *Client) GetCoach(coachID int) (*Coach, error) {
	endpoint := fmt.Sprintf("/v1/football/coaches/%d", coachID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data Coach  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetReferee retrieves a specific referee by ID
func (c *Client) GetReferee(refereeID int) (*Referee, error) {
	endpoint := fmt.Sprintf("/v1/football/referees/%d", refereeID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int     `json:"code"`
		Msg  string  `json:"msg"`
		Data Referee `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetVenue retrieves a specific venue by ID
func (c *Client) GetVenue(venueID int) (*Venue, error) {
	endpoint := fmt.Sprintf("/v1/football/venues/%d", venueID)
	body, err := c.get(endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data Venue  `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetSeasons retrieves seasons for a competition
func (c *Client) GetSeasons(competitionID int) ([]Season, error) {
	params := url.Values{}
	params.Set("competition_id", strconv.Itoa(competitionID))

	body, err := c.get("/v1/football/seasons", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int      `json:"code"`
		Msg  string   `json:"msg"`
		Data []Season `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

// GetStages retrieves stages for a season
func (c *Client) GetStages(seasonID int) ([]Stage, error) {
	params := url.Values{}
	params.Set("season_id", strconv.Itoa(seasonID))

	body, err := c.get("/v1/football/stages", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int     `json:"code"`
		Msg  string  `json:"msg"`
		Data []Stage `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

