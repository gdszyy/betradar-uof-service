package services

import (
	"encoding/xml"
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
	// UOF Fixture API 使用全球 API 端点
	if apiBaseURL == "" {
		apiBaseURL = "https://global.api.betradar.com/v1"
	}
	log.Printf("[FixtureService] Using API: %s", apiBaseURL)
	return &FixtureService{
		apiToken: apiToken,
		baseURL:  apiBaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FixtureData Fixture XML 数据结构
// 根元素是 <fixtures_fixture>，包含一个 <fixture> 子元素
type FixtureData struct {
	XMLName xml.Name `xml:"fixtures_fixture"`
	Fixture struct {
		SportEvent struct {
			ID        string `xml:"id,attr"`
			StartTime string `xml:"start_time,attr"`
			Sport     struct {
				ID   string `xml:"id,attr"`
				Name string `xml:"name,attr"`
			} `xml:"sport"`
			Tournament struct {
				ID   string `xml:"id,attr"`
				Name string `xml:"name,attr"`
			} `xml:"tournament"`
			Competitors struct {
				Competitor []struct {
					ID           string `xml:"id,attr"`
					Name         string `xml:"name,attr"`
					Qualifier    string `xml:"qualifier,attr"` // "home" or "away"
					Abbreviation string `xml:"abbreviation,attr"`
				} `xml:"competitor"`
			} `xml:"competitors"`
		} `xml:"sport_event"`
	} `xml:"fixture"`
}

// FetchFixture 获取赛事 Fixture 信息
func (s *FixtureService) FetchFixture(eventID string) (*FixtureData, error) {
	// UOF Fixture API 端点：使用 .xml 格式，不使用 api_token 查询参数
	url := fmt.Sprintf("%s/sports/en/sport_events/%s/fixture.xml",
		s.baseURL, eventID)
	
	log.Printf("[FixtureService] Fetching fixture for event: %s", eventID)
	
	// 创建请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// UOF API 要求使用 x-access-token 请求头
	req.Header.Set("x-access-token", s.apiToken)
	
	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch fixture: %w", err)
	}
	defer resp.Body.Close()
	
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fixture API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// 读取响应体用于调试
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// 解析 XML 响应
	var fixture FixtureData
	if err := xml.Unmarshal(body, &fixture); err != nil {
		// 输出前 500 个字符用于调试
		preview := string(body)
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		log.Printf("[FixtureService] ⚠️  XML parse error. Response preview: %s", preview)
		return nil, fmt.Errorf("failed to decode fixture XML response: %w", err)
	}
	
	log.Printf("[FixtureService] ✅ Fetched fixture for %s: %s", 
		eventID, fixture.Fixture.SportEvent.ID)
	
	return &fixture, nil
}

// GetTeamInfo 从 Fixture 中提取队伍信息
func (f *FixtureData) GetTeamInfo() (homeID, homeName, awayID, awayName, sportID, sportName string) {
	for _, competitor := range f.Fixture.SportEvent.Competitors.Competitor {
		if competitor.Qualifier == "home" {
			homeID = competitor.ID
			homeName = competitor.Name
		} else if competitor.Qualifier == "away" {
			awayID = competitor.ID
			awayName = competitor.Name
		}
	}
	
	sportID = f.Fixture.SportEvent.Sport.ID
	sportName = f.Fixture.SportEvent.Sport.Name
	
	return
}

