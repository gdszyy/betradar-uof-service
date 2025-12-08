package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
	"uof-service/config"
	"uof-service/logger"
)

// MatchStatusService 比赛阶段服务
// 从API获取match_status的名称映射
type MatchStatusService struct {
	config      *config.Config
	client      *http.Client
	statusMap   map[string]MatchStatusDesc // code -> description
	mu          sync.RWMutex
	lastUpdated time.Time
}

// MatchStatusDesc 比赛阶段描述
type MatchStatusDesc struct {
	ID          string `xml:"id,attr"`
	Description string `xml:"description,attr"`
}

// MatchStatusDescriptions API响应结构
type MatchStatusDescriptions struct {
	XMLName      xml.Name          `xml:"match_status_descriptions"`
	MatchStatuses []MatchStatusDesc `xml:"match_status"`
}

// NewMatchStatusService 创建match_status服务
func NewMatchStatusService(cfg *config.Config) *MatchStatusService {
	return &MatchStatusService{
		config:    cfg,
		client:    &http.Client{Timeout: 30 * time.Second},
		statusMap: make(map[string]MatchStatusDesc),
	}
}

// Start 启动服务并加载match_status映射
func (s *MatchStatusService) Start() error {
	logger.Println("[MatchStatusService] 🚀 Starting match status service...")
	
	// 首次加载
	if err := s.loadMatchStatusDescriptions(); err != nil {
		logger.Errorf("[MatchStatusService] ⚠️  Failed to load match status descriptions: %v", err)
		// 不返回错误，使用默认映射
	} else {
		logger.Printf("[MatchStatusService] ✅ Loaded %d match status descriptions", len(s.statusMap))
	}
	
	// 启动定期更新（每24小时）
	go s.periodicUpdate()
	
	return nil
}

// periodicUpdate 定期更新match_status映射
func (s *MatchStatusService) periodicUpdate() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		if err := s.loadMatchStatusDescriptions(); err != nil {
			logger.Errorf("[MatchStatusService] ⚠️  Failed to update match status descriptions: %v", err)
		} else {
			logger.Printf("[MatchStatusService] ✅ Updated %d match status descriptions", len(s.statusMap))
		}
	}
}

// loadMatchStatusDescriptions 从API加载match_status描述
func (s *MatchStatusService) loadMatchStatusDescriptions() error {
	// API: /descriptions/{language}/match_status.xml
	url := fmt.Sprintf("%s/descriptions/en/match_status.xml", s.config.APIBaseURL)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.config.AccessToken)
	
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	
	var descriptions MatchStatusDescriptions
	if err := xml.Unmarshal(body, &descriptions); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}
	
	// 更新映射
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.statusMap = make(map[string]MatchStatusDesc)
	for _, desc := range descriptions.MatchStatuses {
		s.statusMap[desc.ID] = desc
	}
	s.lastUpdated = time.Now()
	
	return nil
}

// GetDescription 获取match_status的描述
func (s *MatchStatusService) GetDescription(code string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if desc, ok := s.statusMap[code]; ok {
		return desc.Description
	}
	
	// 如果没有找到，返回默认映射
	return s.getDefaultDescription(code)
}

// GetDescriptionWithFallback 获取match_status的描述（带中文回退）
func (s *MatchStatusService) GetDescriptionWithFallback(code string) (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	enDesc := ""
	if desc, ok := s.statusMap[code]; ok {
		enDesc = desc.Description
	} else {
		enDesc = s.getDefaultDescription(code)
	}
	
	// 同时返回中文描述
	cnDesc := s.getChineseDescription(code)
	
	return enDesc, cnDesc
}

// getDefaultDescription 获取默认的英文描述
func (s *MatchStatusService) getDefaultDescription(code string) string {
	// 使用现有的映射作为回退
	defaultMap := map[string]string{
		"0":   "Not started",
		"1":   "Live",
		"2":   "Suspended",
		"3":   "Ended",
		"4":   "Closed",
		"5":   "Cancelled",
		"6":   "Delayed",
		"7":   "Interrupted",
		"8":   "Postponed",
		"9":   "Abandoned",
		"10":  "Coverage dropped",
		"11":  "About to start",
		"20":  "1st quarter",
		"21":  "2nd quarter",
		"22":  "3rd quarter",
		"23":  "4th quarter",
		"30":  "1st half",
		"31":  "Halftime",
		"32":  "2nd half",
		"40":  "Overtime",
		"50":  "Penalties",
		"60":  "1st period",
		"61":  "2nd period",
		"62":  "3rd period",
		"70":  "Awaiting extra time",
		"80":  "Extra time",
		"100": "Ended after extra time",
		"110": "Ended after penalties",
		"120": "Pause",
		"140": "Awaiting penalties",
		"141": "Golden goal",
	}
	
	if desc, ok := defaultMap[code]; ok {
		return desc
	}
	
	return fmt.Sprintf("Unknown (%s)", code)
}

// getChineseDescription 获取中文描述
func (s *MatchStatusService) getChineseDescription(code string) string {
	// 使用现有的中文映射
	chineseMap := map[string]string{
		"0":   "未开始",
		"1":   "进行中",
		"2":   "暂停",
		"3":   "已结束",
		"4":   "已关闭",
		"5":   "已取消",
		"6":   "延迟",
		"7":   "中断",
		"8":   "推迟",
		"9":   "放弃",
		"10":  "直播已取消",
		"11":  "即将开始",
		"20":  "第1节",
		"21":  "第2节",
		"22":  "第3节",
		"23":  "第4节",
		"30":  "上半场",
		"31":  "中场休息",
		"32":  "下半场",
		"40":  "加时赛",
		"50":  "点球大战",
		"60":  "第1局",
		"61":  "第2局",
		"62":  "第3局",
		"70":  "等待加时赛",
		"80":  "加时赛进行中",
		"100": "加时赛结束",
		"110": "点球大战结束",
		"120": "暂停",
		"140": "等待点球大战",
		"141": "金球",
	}
	
	if desc, ok := chineseMap[code]; ok {
		return desc
	}
	
	return fmt.Sprintf("未知 (%s)", code)
}

// GetAllDescriptions 获取所有match_status描述
func (s *MatchStatusService) GetAllDescriptions() map[string]MatchStatusDesc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 返回副本
	result := make(map[string]MatchStatusDesc)
	for k, v := range s.statusMap {
		result[k] = v
	}
	
	return result
}

// IsLoaded 检查是否已加载
func (s *MatchStatusService) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return len(s.statusMap) > 0
}

// LastUpdated 获取最后更新时间
func (s *MatchStatusService) LastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.lastUpdated
}
