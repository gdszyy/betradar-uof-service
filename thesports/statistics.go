package thesports

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// GetPlayerStatistics retrieves statistics for a specific player
func (c *Client) GetPlayerStatistics(playerID, seasonID int) (*PlayerStatistics, error) {
	params := url.Values{}
	params.Set("player_id", strconv.Itoa(playerID))
	if seasonID > 0 {
		params.Set("season_id", strconv.Itoa(seasonID))
	}

	body, err := c.get("/v1/football/players/statistics", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int              `json:"code"`
		Msg  string           `json:"msg"`
		Data PlayerStatistics `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetTeamStatistics retrieves statistics for a specific team
func (c *Client) GetTeamStatistics(teamID, seasonID int) (*TeamStatistics, error) {
	params := url.Values{}
	params.Set("team_id", strconv.Itoa(teamID))
	if seasonID > 0 {
		params.Set("season_id", strconv.Itoa(seasonID))
	}

	body, err := c.get("/v1/football/teams/statistics", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data TeamStatistics `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response.Data, nil
}

// GetBookmakers retrieves all bookmakers
func (c *Client) GetBookmakers() ([]Bookmaker, error) {
	body, err := c.get("/v1/football/bookmakers", nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data []Bookmaker `json:"data"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Data, nil
}

