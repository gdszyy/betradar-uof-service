package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// SRNMappingService SRN ID 映射服务
type SRNMappingService struct {
	apiToken   string
	apiBaseURL string
	db         *sql.DB
	cache      map[string]string // event_id -> srn_id
	mu         sync.RWMutex
	logger     *log.Logger
}

// SRNMappingResponse API 响应结构
type SRNMappingResponse struct {
	Event struct {
		ID  string `json:"id"`
		URN string `json:"urn"`
	} `json:"event"`
	Mappings []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"mappings"`
}

// NewSRNMappingService 创建 SRN 映射服务
func NewSRNMappingService(apiToken, apiBaseURL string, db *sql.DB) *SRNMappingService {
	if apiBaseURL == "" {
		apiBaseURL = "https://stgapi.betradar.com/v1"
	}
	return &SRNMappingService{
		apiToken:   apiToken,
		apiBaseURL: apiBaseURL,
		db:         db,
		cache:      make(map[string]string),
		logger:     log.New(log.Writer(), "[SRNMapping] ", log.LstdFlags),
	}
}

// GetSRNID 获取事件的 SRN ID
func (s *SRNMappingService) GetSRNID(eventID string) (string, error) {
	// 先检查缓存
	s.mu.RLock()
	if srnID, ok := s.cache[eventID]; ok {
		s.mu.RUnlock()
		return srnID, nil
	}
	s.mu.RUnlock()

	// 从 API 获取
	srnID, err := s.fetchSRNIDFromAPI(eventID)
	if err != nil {
		return "", err
	}

	// 更新缓存
	s.mu.Lock()
	s.cache[eventID] = srnID
	s.mu.Unlock()

	// 存储到数据库
	if err := s.storeSRNMapping(eventID, srnID); err != nil {
		s.logger.Printf("Failed to store SRN mapping: %v", err)
	}

	return srnID, nil
}

// fetchSRNIDFromAPI 从 API 获取 SRN ID
func (s *SRNMappingService) fetchSRNIDFromAPI(eventID string) (string, error) {
	// UOF API endpoint for event mappings
	url := fmt.Sprintf("%s/sports/en/sport_events/sr:match:%s/mappings.json?api_token=%s",
		s.apiBaseURL, eventID, s.apiToken)

	s.logger.Printf("Fetching SRN mapping for event: %s", eventID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SRN mapping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var mappingResp SRNMappingResponse
	if err := json.NewDecoder(resp.Body).Decode(&mappingResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// 查找 SRN mapping
	for _, mapping := range mappingResp.Mappings {
		if mapping.Type == "srn" || mapping.Type == "sportradar" {
			s.logger.Printf("Found SRN mapping for %s: %s", eventID, mapping.Value)
			return mapping.Value, nil
		}
	}

	return "", fmt.Errorf("no SRN mapping found for event: %s", eventID)
}

// storeSRNMapping 存储 SRN 映射到数据库
func (s *SRNMappingService) storeSRNMapping(eventID, srnID string) error {
	query := `
		UPDATE tracked_events 
		SET srn_id = $1, updated_at = $2
		WHERE event_id = $3
	`
	_, err := s.db.Exec(query, srnID, time.Now(), eventID)
	if err != nil {
		return fmt.Errorf("failed to store SRN mapping: %w", err)
	}

	s.logger.Printf("Stored SRN mapping: %s -> %s", eventID, srnID)
	return nil
}

// LoadCacheFromDB 从数据库加载缓存
func (s *SRNMappingService) LoadCacheFromDB() error {
	query := `SELECT event_id, srn_id FROM tracked_events WHERE srn_id IS NOT NULL AND srn_id != ''`
	rows, err := s.db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to load SRN mappings: %w", err)
	}
	defer rows.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for rows.Next() {
		var eventID, srnID string
		if err := rows.Scan(&eventID, &srnID); err != nil {
			s.logger.Printf("Failed to scan row: %v", err)
			continue
		}
		s.cache[eventID] = srnID
		count++
	}

	s.logger.Printf("Loaded %d SRN mappings from database", count)
	return nil
}

// GetCachedSRNID 从缓存获取 SRN ID (不触发 API 调用)
func (s *SRNMappingService) GetCachedSRNID(eventID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	srnID, ok := s.cache[eventID]
	return srnID, ok
}

// ClearCache 清空缓存
func (s *SRNMappingService) ClearCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]string)
	s.logger.Println("Cache cleared")
}

