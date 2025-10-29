package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FixtureChangesService 处理赛事变更数据的获取
type FixtureChangesService struct {
	apiToken string
	baseURL  string
	client   *http.Client
}

// FixtureChange 赛事变更信息
type FixtureChange struct {
	EventID       string    `xml:"event_id,attr" json:"event_id"`
	UpdateTime    time.Time `xml:"update_time,attr" json:"update_time"`
	ChangeType    string    `xml:"change_type,attr" json:"change_type"`
	NextLiveTime  *int64    `xml:"next_live_time,attr" json:"next_live_time,omitempty"`
}

// FixtureChangesResponse API 响应结构
type FixtureChangesResponse struct {
	XMLName xml.Name        `xml:"fixture_changes"`
	Changes []FixtureChange `xml:"fixture_change"`
}

// NewFixtureChangesService 创建 FixtureChangesService 实例
func NewFixtureChangesService(apiToken, apiBaseURL string) *FixtureChangesService {
	if apiBaseURL == "" {
		apiBaseURL = "https://global.api.betradar.com/v1"
	}
	return &FixtureChangesService{
		apiToken: apiToken,
		baseURL:  apiBaseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchFixtureChanges 获取指定时间后的赛事变更
// after: Unix timestamp (秒), 获取此时间之后的变更
func (s *FixtureChangesService) FetchFixtureChanges(after int64) ([]FixtureChange, error) {
	url := fmt.Sprintf("%s/sports/en/fixtures/changes.xml?after=%d", s.baseURL, after)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	
	logger.Printf("[FixtureChanges] Fetching changes after timestamp %d", after)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fixture changes: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	var fixtureChanges FixtureChangesResponse
	if err := xml.Unmarshal(body, &fixtureChanges); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}
	
	logger.Printf("[FixtureChanges] Retrieved %d fixture changes", len(fixtureChanges.Changes))
	
	return fixtureChanges.Changes, nil
}

// FetchFixtureChangesSince 获取指定时间段内的赛事变更
func (s *FixtureChangesService) FetchFixtureChangesSince(duration time.Duration) ([]FixtureChange, error) {
	after := time.Now().Add(-duration).Unix()
	return s.FetchFixtureChanges(after)
}

