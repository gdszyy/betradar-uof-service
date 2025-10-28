package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// FixtureService Fixture API 服务
type FixtureService struct {
	apiToken string
	baseURL  string
	client   *http.Client
}

// NewFixtureService 创建 Fixture 服务
func NewFixtureService(apiToken, apiBaseURL string) *FixtureService {
	if apiBaseURL == "" {
		apiBaseURL = "https://stgapi.betradar.com/v1"
	}
	return &FixtureService{
		apiToken: apiToken,
		baseURL:  apiBaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FixtureData Fixture 数据结构
type FixtureData struct {
	SportEvent struct {
		ID        string `json:"id"`
		StartTime string `json:"start_time"`
		Sport     struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"sport"`
		Tournament struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"tournament"`
		Competitors []struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Qualifier  string `json:"qualifier"` // "home" or "away"
			Abbreviation string `json:"abbreviation"`
		} `json:"competitors"`
	} `json:"sport_event"`
}

// FetchFixture 获取赛事 Fixture 信息
func (s *FixtureService) FetchFixture(eventID string) (*FixtureData, error) {
	url := fmt.Sprintf("%s/sports/en/sport_events/%s/fixture.json?api_token=%s",
		s.baseURL, eventID, s.apiToken)
	
	log.Printf("[FixtureService] Fetching fixture for event: %s", eventID)
	
	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fixture: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fixture API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var fixture FixtureData
	if err := json.NewDecoder(resp.Body).Decode(&fixture); err != nil {
		return nil, fmt.Errorf("failed to decode fixture response: %w", err)
	}
	
	log.Printf("[FixtureService] ✅ Fetched fixture for %s: %s", 
		eventID, fixture.SportEvent.ID)
	
	return &fixture, nil
}

// GetTeamInfo 从 Fixture 中提取队伍信息
func (f *FixtureData) GetTeamInfo() (homeID, homeName, awayID, awayName, sportID, sportName string) {
	for _, competitor := range f.SportEvent.Competitors {
		if competitor.Qualifier == "home" {
			homeID = competitor.ID
			homeName = competitor.Name
		} else if competitor.Qualifier == "away" {
			awayID = competitor.ID
			awayName = competitor.Name
		}
	}
	
	sportID = f.SportEvent.Sport.ID
	sportName = f.SportEvent.Sport.Name
	
	return
}

