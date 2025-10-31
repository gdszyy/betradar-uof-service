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
	
	// 如果缓存中没有,尝试从 API 加载
	if err := s.loadPlayerFromAPI(playerID); err != nil {
		logger.Printf("[PlayersService] ⚠️  Failed to load player %s from API: %v", playerID, err)
		return fmt.Sprintf("Player %s", strings.TrimPrefix(playerID, "sr:player:"))
	}
	
	// 重新查询
	s.mu.RLock()
	name, ok = s.players[playerID]
	s.mu.RUnlock()
	
	if ok {
		return name
	}
	
	return fmt.Sprintf("Player %s", strings.TrimPrefix(playerID, "sr:player:"))
}

// loadPlayerFromAPI 从 API 加载球员信息
func (s *PlayersService) loadPlayerFromAPI(playerID string) error {
	// 构造 URL: /v1/sports/en/players/{player_id}/profile.xml
	apiBase := strings.TrimSuffix(s.apiBaseURL, "/v1")
	url := fmt.Sprintf("%s/v1/sports/en/players/%s/profile.xml", apiBase, playerID)
	
	logger.Printf("[PlayersService] Fetching player profile from: %s", url)
	
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
}

// savePlayer 保存球员信息到数据库和缓存
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

// GetStatus 获取服务状态
func (s *PlayersService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"player_count": len(s.players),
		"last_updated": s.lastUpdated,
	}
}

