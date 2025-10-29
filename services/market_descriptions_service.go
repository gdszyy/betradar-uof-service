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

// ReplacementContext 变量替换所需的上下文信息
type ReplacementContext struct {
	HomeTeamName string
	AwayTeamName string
	Specifiers   string // 原始 specifiers 字符串
}

// MarketDescriptionsService 市场描述服务
type MarketDescriptionsService struct {
	token       string
	apiBaseURL  string
	markets     map[string]*MarketDescription
	outcomes    map[string]map[string]*OutcomeDescription // marketID -> outcomeID -> outcome
	mu          sync.RWMutex
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
	}
}

// Start 启动服务并加载市场描述
func (s *MarketDescriptionsService) Start() error {
	logger.Println("[MarketDescService] Starting Market Descriptions Service...")
	
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
	// 尝试去掉 /v1，因为 API Base URL 可能已经包含了
	url := fmt.Sprintf("%s/descriptions/en/markets.xml", s.apiBaseURL)
	
	logger.Printf("[MarketDescService] Fetching market descriptions from: %s", url)
	
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
	
	logger.Printf("[MarketDescService] Loaded %d market descriptions", len(s.markets))
	
	return nil
}

// refreshLoop 定期刷新市场描述
func (s *MarketDescriptionsService) refreshLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		logger.Println("[MarketDescService] Refreshing market descriptions...")
		if err := s.loadMarketDescriptions(); err != nil {
			logger.Errorf("[MarketDescService] Failed to refresh market descriptions: %v", err)
		}
	}
}

// GetMarketName 获取市场名称
func (s *MarketDescriptionsService) GetMarketName(marketID string, ctx ReplacementContext) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	market, ok := s.markets[marketID]
	if !ok {
		return fmt.Sprintf("Market %s", marketID)
	}
	
	name := market.Name
	
	// 替换 specifiers
	name = replaceSpecifiers(name, ctx)
	
	return name
}

// GetOutcomeName 获取结果名称
func (s *MarketDescriptionsService) GetOutcomeName(marketID string, outcomeID string, ctx ReplacementContext) string {
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
	
	// 替换 specifiers
	name = replaceSpecifiers(name, ctx)
	
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
// - {$competitor1},// replaceSpecifiers 替换市场名称中的 specifiers
func replaceSpecifiers(template string, ctx ReplacementContext) string {
	result := template
	
	// 1. 处理 {$competitor1} 和 {$competitor2}
	result = strings.ReplaceAll(result, "{$competitor1}", ctx.HomeTeamName)
	result = strings.ReplaceAll(result, "{$competitor2}", ctx.AwayTeamName)
	
	// 2. 处理 {X} 格式的 specifiers (例如 {hcp})
	if ctx.Specifiers != "" {
		specMap := parseSpecifiers(ctx.Specifiers)
		for key, value := range specMap {
			placeholder := fmt.Sprintf("{%s}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}
	
	// TODO: 更多复杂的替换逻辑 (例如 {!setnr}, {+hcp}, {-hcp})
	
	return result
}

