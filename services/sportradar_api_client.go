package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// SportradarAPIClient Sportradar API 客户端
type SportradarAPIClient struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
	
	// 缓存
	sportsCache      *SportsList
	sportsCacheMutex sync.RWMutex
	sportsCacheTime  time.Time
	
	tournamentsCache      map[string]*TournamentsList // key: sport_id
	tournamentsCacheMutex sync.RWMutex
	tournamentsCacheTime  map[string]time.Time
}

// NewSportradarAPIClient 创建 Sportradar API 客户端
func NewSportradarAPIClient(baseURL, accessToken string) *SportradarAPIClient {
	return &SportradarAPIClient{
		baseURL:              baseURL,
		accessToken:          accessToken,
		httpClient:           &http.Client{Timeout: 30 * time.Second},
		tournamentsCache:     make(map[string]*TournamentsList),
		tournamentsCacheTime: make(map[string]time.Time),
	}
}

// APISport 体育类型（API响应）
type APISport struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// SportsList 体育类型列表
type SportsList struct {
	XMLName xml.Name   `xml:"sports"`
	Sports  []APISport `xml:"sport"`
}

// APITournament 联赛/赛事（API响应）
type APITournament struct {
	ID       string `xml:"id,attr"`
	Name     string `xml:"name,attr"`
	Category struct {
		ID   string `xml:"id,attr"`
		Name string `xml:"name,attr"`
	} `xml:"category"`
}

// TournamentsList 联赛列表
type TournamentsList struct {
	XMLName     xml.Name        `xml:"sport_tournaments"`
	Sport       APISport        `xml:"sport"`
	Tournaments []APITournament `xml:"tournaments>tournament"`
}

// GetAllSports 获取所有体育类型
func (c *SportradarAPIClient) GetAllSports() (*SportsList, error) {
	// 检查缓存 (缓存 1 小时)
	c.sportsCacheMutex.RLock()
	if c.sportsCache != nil && time.Since(c.sportsCacheTime) < time.Hour {
		defer c.sportsCacheMutex.RUnlock()
		log.Printf("[SportradarAPI] Returning cached sports list")
		return c.sportsCache, nil
	}
	c.sportsCacheMutex.RUnlock()
	
	// 构建 URL（不包含 api_key）
	url := fmt.Sprintf("%s/sports/en/sports.xml", c.baseURL)
	
	// 记录请求的 URL
	log.Printf("[SportradarAPI] Calling external URL address: %s", url)
	
	// 创建请求并添加认证 Header
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("x-access-token", c.accessToken)
	
	// 发起请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sports: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析 XML
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// 记录返回的 XML
	log.Printf("[SportradarAPI] External URL returned XML: %s", string(body))
	
	var sportsList SportsList
	if err := xml.Unmarshal(body, &sportsList); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	log.Printf("[SportradarAPI] Fetched %d sports", len(sportsList.Sports))
	
	// 更新缓存
	c.sportsCacheMutex.Lock()
	c.sportsCache = &sportsList
	c.sportsCacheTime = time.Now()
	c.sportsCacheMutex.Unlock()
	
	return &sportsList, nil
}

// GetTournamentsBySport 获取指定体育类型的联赛列表
func (c *SportradarAPIClient) GetTournamentsBySport(sportID string) (*TournamentsList, error) {
	// 检查缓存 (缓存 30 分钟)
	c.tournamentsCacheMutex.RLock()
	if cached, ok := c.tournamentsCache[sportID]; ok {
		if cacheTime, ok := c.tournamentsCacheTime[sportID]; ok {
			if time.Since(cacheTime) < 30*time.Minute {
				defer c.tournamentsCacheMutex.RUnlock()
				log.Printf("[SportradarAPI] Returning cached tournaments for sport %s", sportID)
				return cached, nil
			}
		}
	}
	c.tournamentsCacheMutex.RUnlock()
	
	// 构建 URL（不包含 api_key）
	url := fmt.Sprintf("%s/sports/en/sports/%s/tournaments.xml", c.baseURL, sportID)
	
	// 记录请求的 URL
	log.Printf("[SportradarAPI] Calling external URL address: %s", url)
	
	// 创建请求并添加认证 Header
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("x-access-token", c.accessToken)
	
	// 发起请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tournaments: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// 解析 XML
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// 记录返回的 XML
	log.Printf("[SportradarAPI] External URL returned XML: %s", string(body))
	
	var tournamentsList TournamentsList
	if err := xml.Unmarshal(body, &tournamentsList); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}
	
	log.Printf("[SportradarAPI] Fetched %d tournaments for sport %s", len(tournamentsList.Tournaments), sportID)
	
	// 更新缓存
	c.tournamentsCacheMutex.Lock()
	c.tournamentsCache[sportID] = &tournamentsList
	c.tournamentsCacheTime[sportID] = time.Now()
	c.tournamentsCacheMutex.Unlock()
	
	return &tournamentsList, nil
}

// GetAllTournaments 获取所有体育类型的联赛列表
func (c *SportradarAPIClient) GetAllTournaments() (map[string]*TournamentsList, error) {
	// 先获取所有体育类型
	sportsList, err := c.GetAllSports()
	if err != nil {
		return nil, fmt.Errorf("failed to get sports list: %w", err)
	}
	
	// 获取每个体育类型的联赛列表
	allTournaments := make(map[string]*TournamentsList)
	
	for _, sport := range sportsList.Sports {
		// GetAllTournaments 依赖 GetTournamentsBySport，而 GetTournamentsBySport 已经添加了日志
		tournaments, err := c.GetTournamentsBySport(sport.ID)
		if err != nil {
			log.Printf("[SportradarAPI] Failed to get tournaments for sport %s: %v", sport.ID, err)
			continue
		}
		
		allTournaments[sport.ID] = tournaments
	}
	
	return allTournaments, nil
}
