package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
	"uof-service/logger"
)

// LogoFetcherService Logo 获取服务
type LogoFetcherService struct {
	db           *sql.DB
	teamsService *TeamsService
	httpClient   *http.Client
	apiKey       string
	apiBaseURL   string
	maxRetries   int
	batchSize    int
	interval     time.Duration
	stopChan     chan struct{}
}

// NewLogoFetcherService 创建新的 Logo 获取服务实例
func NewLogoFetcherService(db *sql.DB, teamsService *TeamsService) *LogoFetcherService {
	return &LogoFetcherService{
		db:           db,
		teamsService: teamsService,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
			apiKey:     "123", // TheSportsDB 免费 API Key
			apiBaseURL: "https://www.thesportsdb.com/api/v1/json",
			maxRetries: 3,
			batchSize:  50,              // 每次处理 50 个队伍（加速处理）
			interval:   30 * time.Second, // 每 30 秒检查一次（加速处理）
		stopChan:   make(chan struct{}),
	}
}

// TheSportsDBResponse TheSportsDB API 响应结构
type TheSportsDBResponse struct {
	Teams []struct {
		StrBadge string `json:"strBadge"`
		StrLogo  string `json:"strLogo"`
	} `json:"teams"`
}

// Start 启动 Logo 获取服务
func (s *LogoFetcherService) Start() {
	logger.Println("[LogoFetcherService] 🚀 Starting Logo fetcher service...")
	
	// 启动时立即执行一次
	go s.fetchLogosForPendingTeams()
	
	// 定期执行
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				s.fetchLogosForPendingTeams()
			case <-s.stopChan:
				logger.Println("[LogoFetcherService] 🛑 Logo fetcher service stopped")
				return
			}
		}
	}()
}

// Stop 停止 Logo 获取服务
func (s *LogoFetcherService) Stop() {
	close(s.stopChan)
}

// fetchLogosForPendingTeams 为待处理的队伍获取 Logo
func (s *LogoFetcherService) fetchLogosForPendingTeams() {
	logger.Println("[LogoFetcherService] 📥 Fetching logos for pending teams...")
	
	// 获取需要获取 Logo 的队伍列表
	teams, err := s.teamsService.GetTeamsNeedingLogoFetch(s.maxRetries, s.batchSize)
	if err != nil {
		logger.Errorf("[LogoFetcherService] ❌ Failed to get teams needing logo fetch: %v", err)
		return
	}
	
	if len(teams) == 0 {
		logger.Println("[LogoFetcherService] ✅ No teams need logo fetch")
		return
	}
	
	logger.Printf("[LogoFetcherService] 📋 Found %d teams needing logo fetch", len(teams))
	
	// 逐个获取 Logo
	successCount := 0
	failureCount := 0
	
	for _, team := range teams {
		logoURL, err := s.fetchLogoFromTheSportsDB(team.TeamName)
		if err != nil {
			logger.Errorf("[LogoFetcherService] ⚠️  Failed to fetch logo for team %s: %v", team.TeamName, err)
			// 更新失败记录
			s.teamsService.UpdateTeamLogo(team.TeamID, "", false)
			failureCount++
			continue
		}
		
		if logoURL == "" {
			logger.Printf("[LogoFetcherService] ⚠️  No logo found for team %s", team.TeamName)
			// 更新失败记录
			s.teamsService.UpdateTeamLogo(team.TeamID, "", false)
			failureCount++
			continue
		}
		
		// 更新成功记录
		err = s.teamsService.UpdateTeamLogo(team.TeamID, logoURL, true)
		if err != nil {
			logger.Errorf("[LogoFetcherService] ❌ Failed to update logo for team %s: %v", team.TeamName, err)
			failureCount++
			continue
		}
		
		successCount++
		
		// 避免请求过快，添加短暂延迟
		time.Sleep(500 * time.Millisecond)
	}
	
	logger.Printf("[LogoFetcherService] ✅ Logo fetch completed: %d success, %d failure", successCount, failureCount)
}

// fetchLogoFromTheSportsDB 从 TheSportsDB API 获取队伍 Logo
func (s *LogoFetcherService) fetchLogoFromTheSportsDB(teamName string) (string, error) {
	// 构建 API URL
	apiURL := fmt.Sprintf("%s/%s/searchteams.php?t=%s", s.apiBaseURL, s.apiKey, url.QueryEscape(teamName))
	
	// 发送 HTTP 请求
	resp, err := s.httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status code %d", resp.StatusCode)
	}
	
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	
	// 解析 JSON 响应
	var apiResponse TheSportsDBResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}
	
	// 提取 Logo URL
	if len(apiResponse.Teams) == 0 {
		return "", nil // 未找到队伍
	}
	
	// 优先使用 strBadge，如果为空则使用 strLogo
	logoURL := apiResponse.Teams[0].StrBadge
	if logoURL == "" {
		logoURL = apiResponse.Teams[0].StrLogo
	}
	
	return logoURL, nil
}

// FetchLogoForTeam 为单个队伍立即获取 Logo (同步方法)
func (s *LogoFetcherService) FetchLogoForTeam(teamID string, teamName string) error {
	logoURL, err := s.fetchLogoFromTheSportsDB(teamName)
	if err != nil {
		logger.Errorf("[LogoFetcherService] ⚠️  Failed to fetch logo for team %s: %v", teamName, err)
		return s.teamsService.UpdateTeamLogo(teamID, "", false)
	}
	
	if logoURL == "" {
		logger.Printf("[LogoFetcherService] ⚠️  No logo found for team %s", teamName)
		return s.teamsService.UpdateTeamLogo(teamID, "", false)
	}
	
	return s.teamsService.UpdateTeamLogo(teamID, logoURL, true)
}

// ScheduleLogoFetch 异步调度 Logo 获取任务
func (s *LogoFetcherService) ScheduleLogoFetch(teamID string, teamName string) {
	go func() {
		// 添加随机延迟，避免大量请求同时发出
		time.Sleep(time.Duration(1+len(teamID)%5) * time.Second)
		
		err := s.FetchLogoForTeam(teamID, teamName)
		if err != nil {
			logger.Errorf("[LogoFetcherService] ❌ Failed to schedule logo fetch for team %s: %v", teamName, err)
		}
	}()
}
