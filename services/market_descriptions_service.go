package services

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"uof-service/logger"
)

// MarketDescriptionsService 市场描述服务
type MarketDescriptionsService struct {
	token       string
	apiBaseURL  string
	markets     map[string]*MarketDescription
	outcomes    map[string]map[string]*OutcomeDescription // marketID -> outcomeID -> outcome
	mu          sync.RWMutex
	logger      *logger.Logger
	lastUpdated time.Time
}

// MarketDescription 市场描述
type MarketDescription struct {
	ID        string                          `xml:"id,attr"`
	Name      string                          `xml:"name,attr"`
	Groups    string                          `xml:"groups,attr"`
	Outcomes  []OutcomeDescription            `xml:"outcomes>outcome"`
	Specifiers []SpecifierDescription         `xml:"specifiers>specifier"`
}

// OutcomeDescription 结果描述
type OutcomeDescription struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// SpecifierDescription 说明符描述
type SpecifierDescription struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

// MarketDescriptionsResponse API响应
type MarketDescriptionsResponse struct {
	XMLName      xml.Name            `xml:"market_descriptions"`
	ResponseCode string              `xml:"response_code,attr"`
	Markets      []MarketDescription `xml:"market"`
}

// NewMarketDescriptionsService 创建市场描述服务
func NewMarketDescriptionsService(token string, apiBaseURL string) *MarketDescriptionsService {
	return &MarketDescriptionsService{
		token:      token,
		apiBaseURL: apiBaseURL,
		markets:    make(map[string]*MarketDescription),
		outcomes:   make(map[string]map[string]*OutcomeDescription),
		logger:     logger.New(),
	}
}

// Start 启动服务并加载市场描述
func (s *MarketDescriptionsService) Start() error {
	s.logger.Info("Starting Market Descriptions Service...")
	
	// 首次加载
	if err := s.loadMarketDescriptions(); err != nil {
		return fmt.Errorf("failed to load market descriptions: %w", err)
	}
	
	// 启动定期刷新 (每24小时)
	go s.refreshLoop()
	
	return nil
}

// loadMarketDescriptions 加载市场描述
func (s *MarketDescriptionsService) loadMarketDescriptions() error {
	url := fmt.Sprintf("%s/v1/descriptions/en/markets.xml", s.apiBaseURL)
	
	s.logger.Info("Fetching market descriptions from: %s", url)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", s.token)
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch market descriptions: %w", err)
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
	
	var response MarketDescriptionsResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}
	
	if response.ResponseCode != "OK" {
		return fmt.Errorf("API returned response_code: %s", response.ResponseCode)
	}
	
	// 更新缓存
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.markets = make(map[string]*MarketDescription)
	s.outcomes = make(map[string]map[string]*OutcomeDescription)
	
	for i := range response.Markets {
		market := &response.Markets[i]
		s.markets[market.ID] = market
		
		// 构建 outcomes 索引
		outcomeMap := make(map[string]*OutcomeDescription)
		for j := range market.Outcomes {
			outcome := &market.Outcomes[j]
			outcomeMap[outcome.ID] = outcome
		}
		s.outcomes[market.ID] = outcomeMap
	}
	
	s.lastUpdated = time.Now()
	
	s.logger.Info("Loaded %d market descriptions", len(s.markets))
	
	return nil
}

// refreshLoop 定期刷新市场描述
func (s *MarketDescriptionsService) refreshLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		s.logger.Info("Refreshing market descriptions...")
		if err := s.loadMarketDescriptions(); err != nil {
			s.logger.Error("Failed to refresh market descriptions: %v", err)
		}
	}
}

// GetMarketName 获取市场名称
func (s *MarketDescriptionsService) GetMarketName(marketID string, specifiers string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	market, ok := s.markets[marketID]
	if !ok {
		return fmt.Sprintf("Market %s", marketID)
	}
	
	name := market.Name
	
	// 替换 specifiers (简化版)
	if specifiers != "" {
		specMap := parseSpecifiers(specifiers)
		name = replaceSpecifiers(name, specMap)
	}
	
	return name
}

// GetOutcomeName 获取结果名称
func (s *MarketDescriptionsService) GetOutcomeName(marketID string, outcomeID string, specifiers string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	outcomeMap, ok := s.outcomes[marketID]
	if !ok {
		return fmt.Sprintf("Outcome %s", outcomeID)
	}
	
	outcome, ok := outcomeMap[outcomeID]
	if !ok {
		return fmt.Sprintf("Outcome %s", outcomeID)
	}
	
	name := outcome.Name
	
	// 替换 specifiers (简化版)
	if specifiers != "" {
		specMap := parseSpecifiers(specifiers)
		name = replaceSpecifiers(name, specMap)
	}
	
	return name
}

// GetMarketCount 获取已加载的市场数量
func (s *MarketDescriptionsService) GetMarketCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.markets)
}

// GetLastUpdated 获取最后更新时间
func (s *MarketDescriptionsService) GetLastUpdated() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdated
}

// parseSpecifiers 解析 specifiers 字符串
// 例如: "total=2.5|variant=sr:exact_goals:2" -> map[string]string{"total": "2.5", "variant": "sr:exact_goals:2"}
func parseSpecifiers(specifiers string) map[string]string {
	result := make(map[string]string)
	
	pairs := strings.Split(specifiers, "|")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	
	return result
}

// replaceSpecifiers 替换市场名称中的 specifiers
// 支持的模板:
// - {X}: 替换为 specifier X 的值
// - {!X}: 替换为 specifier X 的序数形式 (1st, 2nd, 3rd, ...)
// - {+X}: 替换为 specifier X 的值并添加 +/- 符号
// - {-X}: 替换为 specifier X 的负值并添加 +/- 符号
// - {$competitor1}, {$competitor2}: 保留原样 (需要从 event 获取)
func replaceSpecifiers(template string, specifiers map[string]string) string {
	result := template
	
	// 简单替换 {X}
	for key, value := range specifiers {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	
	// TODO: 实现更复杂的模板替换逻辑
	// - {!X}: 序数形式
	// - {+X}, {-X}: 带符号的数字
	// - {$competitor1}, {$competitor2}: 队伍名称
	
	return result
}

