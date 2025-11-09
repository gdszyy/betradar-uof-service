package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"uof-service/logger"
)

// PlayerProfile 球员信息
type PlayerProfile struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	Nationality string `xml:"nationality,attr"`
	DateOfBirth string `xml:"date_of_birth,attr"`
}

// PlayerProfileResponse API 响应
type PlayerProfileResponse struct {
	XMLName xml.Name      `xml:"player_profile"`
	Player  PlayerProfile `xml:"player"`
}

// PlayerInfo 球员信息 (从 Fixture API 中提取)
type PlayerInfo struct {
	ID   string
	Name string
}

// PlayersService 球员信息服务
type PlayersService struct {
	db          *sql.DB
	apiBaseURL  string
	token       string
	players     map[string]string // player_id -> player_name
	mu          sync.RWMutex
	lastUpdated time.Time
}

// NewPlayersService 创建球员信息服务
func NewPlayersService(token string, apiBaseURL string, db *sql.DB) *PlayersService {
	return &PlayersService{
		db:         db,
		apiBaseURL: apiBaseURL,
		token:      token,
		players:    make(map[string]string),
	}
}

// Start 启动服务
func (s *PlayersService) Start() error {
	logger.Println("[PlayersService] Starting Players Service...")
	
	// 从数据库加载球员数据
	if err := s.loadFromDatabase(); err != nil {
		logger.Printf("[PlayersService] ⚠️  Failed to load from database: %v", err)
	}
	
	logger.Printf("[PlayersService] ✅ Loaded %d players from database", len(s.players))
	logger.Println("[PlayersService] ✅ Players service started")
	
	return nil
}

// loadFromDatabase 从数据库加载球员数据
func (s *PlayersService) loadFromDatabase() error {
	rows, err := s.db.Query(`
		SELECT player_id, player_name 
		FROM players 
		ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query players: %w", err)
	}
	defer rows.Close()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	count := 0
	for rows.Next() {
		var playerID, playerName string
		if err := rows.Scan(&playerID, &playerName); err != nil {
			continue
		}
		
		s.players[playerID] = playerName
		count++
	}
	
	s.lastUpdated = time.Now()
	
	return nil
}

// GetPlayerName 获取球员名称
func (s *PlayersService) GetPlayerName(playerID string) string {
	s.mu.RLock()
	name, ok := s.players[playerID]
	s.mu.RUnlock()
	
	if ok {
		return name
	}
	
		// 如果缓存中没有，说明预加载失败或者该球员信息不在预加载范围内
	// 缓存未命中，尝试从 API 动态加载
	
	// 动态加载应该使用 loadPlayerFromAPI 的逻辑，但需要返回 PlayerProfile
	// 重新实现动态加载逻辑，避免 loadPlayerFromAPI 的副作用
	
	playerIDNum := strings.TrimPrefix(playerID, "sr:player:")
	
	// 构造 URL: /v1/sports/en/players/{player_id}/profile.xml
	apiBase := strings.TrimSuffix(s.apiBaseURL, "/v1")
	url := fmt.Sprintf("%s/v1/sports/en/players/%s/profile.xml", apiBase, playerID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Printf("[PlayersService] ⚠️  Failed to create request for dynamic load of %s: %v", playerID, err)
		return fmt.Sprintf("Player %s", playerIDNum)
	}
	
	req.Header.Set("x-access-token", s.token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Printf("[PlayersService] ⚠️  Failed to fetch player profile for %s: %v", playerID, err)
		return fmt.Sprintf("Player %s", playerIDNum)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		logger.Printf("[PlayersService] ⚠️  API returned status %d for dynamic load of %s", resp.StatusCode, playerID)
		return fmt.Sprintf("Player %s", playerIDNum)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("[PlayersService] ⚠️  Failed to read response body for %s: %v", playerID, err)
		return fmt.Sprintf("Player %s", playerIDNum)
	}
	
	var profile PlayerProfileResponse
	if err := xml.Unmarshal(body, &profile); err != nil {
		logger.Printf("[PlayersService] ⚠️  Failed to parse XML for %s: %v", playerID, err)
		return fmt.Sprintf("Player %s", playerIDNum)
	}
	
	// 动态加载成功，保存到数据库和缓存
	if err := s.savePlayer(&profile.Player); err != nil {
		logger.Printf("[PlayersService] ⚠️  Failed to save dynamically loaded player %s: %v", playerID, err)
	}
	
	logger.Printf("[PlayersService] ✅ Dynamically loaded player %s: %s", playerID, profile.Player.Name)
	return profile.Player.Name
}

// loadPlayerFromAPI 从 API 加载球员信息
// 该方法现在只在 PreloadPlayers 中使用
func (s *PlayersService) loadPlayerFromAPI(playerID string) error {
	// 检查是否已经存在,避免重复加载
	s.mu.RLock()
	_, ok := s.players[playerID]
	s.mu.RUnlock()
	if ok {
		return nil
	}
	
	// 构造 URL: /v1/sports/en/players/{player_id}/profile.xml
	apiBase := strings.TrimSuffix(s.apiBaseURL, "/v1")
	url := fmt.Sprintf("%s/v1/sports/en/players/%s/profile.xml", apiBase, playerID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.token)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch player profile: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	var profile PlayerProfileResponse
	if err := xml.Unmarshal(body, &profile); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}
	
	// 保存到数据库和缓存
	if err := s.savePlayer(&profile.Player); err != nil {
		return fmt.Errorf("failed to save player: %w", err)
	}
	
	logger.Printf("[PlayersService] ✅ Loaded player: %s (%s)", profile.Player.Name, playerID)
	
	return nil
}// savePlayer 保存球员信息到数据库和缓存
func (s *PlayersService) savePlayer(player *PlayerProfile) error {
	// 解析出生日期
	var dateOfBirth *time.Time
	if player.DateOfBirth != "" {
		if t, err := time.Parse("2006-01-02", player.DateOfBirth); err == nil {
			dateOfBirth = &t
		}
	}
	
	// 保存到数据库
	_, err := s.db.Exec(`
		INSERT INTO players (player_id, player_name, nationality, date_of_birth, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (player_id) DO UPDATE
		SET player_name = EXCLUDED.player_name,
		    nationality = EXCLUDED.nationality,
		    date_of_birth = EXCLUDED.date_of_birth,
		    updated_at = NOW()
	`, player.ID, player.Name, player.Nationality, dateOfBirth)
	
	if err != nil {
		return fmt.Errorf("failed to insert player: %w", err)
	}
	
	// 更新缓存
	s.mu.Lock()
	s.players[player.ID] = player.Name
	s.mu.Unlock()
	
	return nil
}

// PreloadPlayers 批量预加载球员信息
func (s *PlayersService) PreloadPlayers(players []PlayerInfo) {
	// 使用 goroutine 并发加载球员信息
	var wg sync.WaitGroup
	
	// 使用 channel 限制并发数
	concurrencyLimit := 10
	semaphore := make(chan struct{}, concurrencyLimit)
	
	for _, player := range players {
		// 检查是否已存在,避免重复 API 调用
		s.mu.RLock()
		_, ok := s.players[player.ID]
		s.mu.RUnlock()
		
		if ok {
			continue
		}
		
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(playerID string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			// 检查是否已存在,避免并发冲突
			s.mu.RLock()
			_, ok := s.players[playerID]
			s.mu.RUnlock()
			
			if ok {
				return
			}
			
			// 尝试从 API 加载
			if err := s.loadPlayerFromAPI(playerID); err != nil {
				logger.Printf("[PlayersService] ⚠️  Failed to preload player %s from API: %v", playerID, err)
			}
		}(player.ID)
	}
	
	wg.Wait()
	logger.Printf("[PlayersService] ✅ Preload finished. Total players: %d", len(s.players))
}

// GetStatus 获取服务状态
func (s *PlayersService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"player_count": len(s.players),
		"last_updated": s.lastUpdated,
	}
}

