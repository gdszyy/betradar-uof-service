package services

import (
	"database/sql"
	"encoding/json"
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
	db          *sql.DB // 可选的数据库连接
	markets     map[string]*MarketDescription
	outcomes    map[string]map[string]*OutcomeDescription // marketID -> outcomeID -> outcome
	mu          sync.RWMutex
	lastUpdated time.Time
}

// MarketDescription 市场描述
type MarketDescription struct {
	ID         string                   `xml:"id,attr"`
	Name       string                   `xml:"name,attr"`
	Groups     string                   `xml:"groups,attr"`
	Outcomes   []OutcomeDescription     `xml:"outcomes>outcome"`
	Specifiers []SpecifierDescription   `xml:"specifiers>specifier"`
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

// SetDatabase 设置数据库连接 (可选)
func (s *MarketDescriptionsService) SetDatabase(db *sql.DB) {
	s.db = db
}

// Start 启动服务并加载市场描述
func (s *MarketDescriptionsService) Start() error {
	logger.Println("[MarketDescService] Starting Market Descriptions Service...")
	
	// 如果有数据库,优先从数据库加载
	if s.db != nil {
		err := s.loadFromDatabase()
		if err == nil {
			logger.Printf("[MarketDescService] ✅ Loaded %d markets from database cache", len(s.markets))
			
			// 启动定期刷新 (每24小时)
			go s.refreshLoop()
			return nil
		}
		logger.Printf("[MarketDescService] ⚠️  Failed to load from database, falling back to API: %v", err)
	}
	
	// 从 API 加载
	if err := s.loadMarketDescriptions(); err != nil {
		return fmt.Errorf("failed to load market descriptions: %w", err)
	}
	
	// 启动定期刷新 (每24小时)
	go s.refreshLoop()
	
	return nil
}

// loadFromDatabase 从数据库加载缓存
func (s *MarketDescriptionsService) loadFromDatabase() error {
	if s.db == nil {
		return fmt.Errorf("database not available")
	}
	
	// 加载 markets
	marketRows, err := s.db.Query(`
		SELECT market_id, market_name, groups, specifiers
		FROM market_descriptions
		ORDER BY market_id
	`)
	if err != nil {
		return fmt.Errorf("failed to query markets: %w", err)
	}
	defer marketRows.Close()
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	marketCount := 0
	for marketRows.Next() {
		var marketID, marketName, groups string
		var specifiersJSON sql.NullString
		
		if err := marketRows.Scan(&marketID, &marketName, &groups, &specifiersJSON); err != nil {
			continue
		}
		
		market := &MarketDescription{
			ID:     marketID,
			Name:   marketName,
			Groups: groups,
		}
		
		// 解析 specifiers
		if specifiersJSON.Valid {
			json.Unmarshal([]byte(specifiersJSON.String), &market.Specifiers)
		}
		
		s.markets[marketID] = market
		marketCount++
	}
	
	// 加载 outcomes
	outcomeRows, err := s.db.Query(`
		SELECT market_id, outcome_id, outcome_name
		FROM outcome_descriptions
		ORDER BY market_id, outcome_id
	`)
	if err != nil {
		return fmt.Errorf("failed to query outcomes: %w", err)
	}
	defer outcomeRows.Close()
	
	outcomeCount := 0
	for outcomeRows.Next() {
		var marketID, outcomeID, outcomeName string
		
		if err := outcomeRows.Scan(&marketID, &outcomeID, &outcomeName); err != nil {
			continue
		}
		
		if s.outcomes[marketID] == nil {
			s.outcomes[marketID] = make(map[string]*OutcomeDescription)
		}
		
		s.outcomes[marketID][outcomeID] = &OutcomeDescription{
			ID:   outcomeID,
			Name: outcomeName,
		}
		outcomeCount++
	}
	
	if marketCount == 0 {
		return fmt.Errorf("no markets found in database")
	}
	
	s.lastUpdated = time.Now()
	logger.Printf("[MarketDescService] Loaded %d markets and %d outcomes from database", marketCount, outcomeCount)
	
	return nil
}

// saveToDatabase 保存到数据库
func (s *MarketDescriptionsService) saveToDatabase() error {
	if s.db == nil {
		return nil // 数据库不可用,跳过
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 清空旧数据
	if _, err := tx.Exec("DELETE FROM outcome_descriptions"); err != nil {
		return fmt.Errorf("failed to clear outcomes: %w", err)
	}
	if _, err := tx.Exec("DELETE FROM market_descriptions"); err != nil {
		return fmt.Errorf("failed to clear markets: %w", err)
	}
	
	// 插入 markets
	marketStmt, err := tx.Prepare(`
		INSERT INTO market_descriptions (market_id, market_name, groups, specifiers)
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare market statement: %w", err)
	}
	defer marketStmt.Close()
	
	marketCount := 0
	for _, market := range s.markets {
		specifiersJSON, _ := json.Marshal(market.Specifiers)
		if _, err := marketStmt.Exec(market.ID, market.Name, market.Groups, string(specifiersJSON)); err != nil {
			logger.Printf("[MarketDescService] ⚠️  Failed to insert market %s: %v", market.ID, err)
			continue
		}
		marketCount++
	}
	
	// 插入 outcomes
	outcomeStmt, err := tx.Prepare(`
		INSERT INTO outcome_descriptions (market_id, outcome_id, outcome_name)
		VALUES ($1, $2, $3)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare outcome statement: %w", err)
	}
	defer outcomeStmt.Close()
	
	outcomeCount := 0
	for marketID, outcomes := range s.outcomes {
		for _, outcome := range outcomes {
			if _, err := outcomeStmt.Exec(marketID, outcome.ID, outcome.Name); err != nil {
				logger.Printf("[MarketDescService] ⚠️  Failed to insert outcome %s/%s: %v", marketID, outcome.ID, err)
				continue
			}
			outcomeCount++
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	logger.Printf("[MarketDescService] ✅ Saved %d markets and %d outcomes to database", marketCount, outcomeCount)
	return nil
}

// loadMarketDescriptions 从 API 加载市场描述
func (s *MarketDescriptionsService) loadMarketDescriptions() error {
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
		return fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	var response MarketDescriptionsResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse XML: %w", err)
	}
	
	s.mu.Lock()
	s.markets = make(map[string]*MarketDescription)
	s.outcomes = make(map[string]map[string]*OutcomeDescription)
	
	for i := range response.Markets {
		market := &response.Markets[i]
		s.markets[market.ID] = market
		
		// 索引 outcomes
		s.outcomes[market.ID] = make(map[string]*OutcomeDescription)
		for j := range market.Outcomes {
			outcome := &market.Outcomes[j]
			s.outcomes[market.ID][outcome.ID] = outcome
		}
	}
	
	s.lastUpdated = time.Now()
	s.mu.Unlock()
	
	logger.Printf("[MarketDescService] ✅ Loaded %d market descriptions from API", len(s.markets))
	
	// 保存到数据库 (如果可用)
	if err := s.saveToDatabase(); err != nil {
		logger.Printf("[MarketDescService] ⚠️  Failed to save to database: %v", err)
	}
	
	return nil
}

// refreshLoop 定期刷新
func (s *MarketDescriptionsService) refreshLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		logger.Println("[MarketDescService] Refreshing market descriptions...")
		if err := s.loadMarketDescriptions(); err != nil {
			logger.Printf("[MarketDescService] ⚠️  Failed to refresh: %v", err)
		}
	}
}

// GetMarketName 获取市场名称
func (s *MarketDescriptionsService) GetMarketName(marketID string, specifiers string, ctx *ReplacementContext) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	market, ok := s.markets[marketID]
	if !ok {
		return fmt.Sprintf("Market %s", marketID)
	}
	
	name := market.Name
	
	// 替换变量
	if ctx != nil {
		name = strings.ReplaceAll(name, "$competitor1", ctx.HomeTeamName)
		name = strings.ReplaceAll(name, "$competitor2", ctx.AwayTeamName)
		name = strings.ReplaceAll(name, "{$competitor1}", ctx.HomeTeamName)
		name = strings.ReplaceAll(name, "{$competitor2}", ctx.AwayTeamName)
	}
	
	// 替换 specifiers
	if specifiers != "" {
		pairs := strings.Split(specifiers, "|")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				name = strings.ReplaceAll(name, "{"+key+"}", value)
				name = strings.ReplaceAll(name, "{+"+key+"}", "+"+value)
				name = strings.ReplaceAll(name, "{-"+key+"}", "-"+value)
			}
		}
	}
	
	return name
}

// GetOutcomeName 获取结果名称
func (s *MarketDescriptionsService) GetOutcomeName(marketID string, outcomeID string, specifiers string, ctx *ReplacementContext) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	outcomes, ok := s.outcomes[marketID]
	if !ok {
		return fmt.Sprintf("Outcome %s", outcomeID)
	}
	
	outcome, ok := outcomes[outcomeID]
	if !ok {
		return fmt.Sprintf("Outcome %s", outcomeID)
	}
	
	name := outcome.Name
	
	// 替换变量
	if ctx != nil {
		name = strings.ReplaceAll(name, "$competitor1", ctx.HomeTeamName)
		name = strings.ReplaceAll(name, "$competitor2", ctx.AwayTeamName)
		name = strings.ReplaceAll(name, "{$competitor1}", ctx.HomeTeamName)
		name = strings.ReplaceAll(name, "{$competitor2}", ctx.AwayTeamName)
	}
	
	// 替换 specifiers
	if specifiers != "" {
		pairs := strings.Split(specifiers, "|")
		for _, pair := range pairs {
			parts := strings.Split(pair, "=")
			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				name = strings.ReplaceAll(name, "{"+key+"}", value)
				name = strings.ReplaceAll(name, "{+"+key+"}", "+"+value)
				name = strings.ReplaceAll(name, "{-"+key+"}", "-"+value)
			}
		}
	}
	
	return name
}

// GetStatus 获取服务状态
func (s *MarketDescriptionsService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"market_count":   len(s.markets),
		"last_updated":   s.lastUpdated,
		"database_enabled": s.db != nil,
	}
}

// ForceRefresh 强制刷新
func (s *MarketDescriptionsService) ForceRefresh() error {
	return s.loadMarketDescriptions()
}

// UpdateExistingMarkets 批量更新已存在的 markets 和 outcomes 表中的 name 字段
func (s *MarketDescriptionsService) UpdateExistingMarkets() (int, int, error) {
	if s.db == nil {
		return 0, 0, fmt.Errorf("database not available")
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	logger.Println("[MarketDescService] Starting bulk update of existing markets and outcomes...")
	
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 更新 markets 表 (分批处理)
	marketRows, err := tx.Query(`
		SELECT id, market_id, specifiers, COALESCE(home_team_name, ''), COALESCE(away_team_name, '')
		FROM markets
		WHERE market_name IS NULL OR market_name = ''
		LIMIT 10000
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query markets: %w", err)
	}
	defer marketRows.Close()
	
	updatedMarkets := 0
	for marketRows.Next() {
		var id int
		var marketID, specifiers, homeTeamName, awayTeamName string
		
		if err := marketRows.Scan(&id, &marketID, &specifiers, &homeTeamName, &awayTeamName); err != nil {
			continue
		}
		
		ctx := &ReplacementContext{
			HomeTeamName: homeTeamName,
			AwayTeamName: awayTeamName,
			Specifiers:   specifiers,
		}
		
		marketName := s.GetMarketName(marketID, specifiers, ctx)
		
		_, err := tx.Exec(`UPDATE markets SET market_name = $1 WHERE id = $2`, marketName, id)
		if err != nil {
			logger.Printf("[MarketDescService] ⚠️  Failed to update market %s: %v", marketID, err)
			continue
		}
		updatedMarkets++
	}
	
	// 更新 outcomes 表 (分批处理)
	outcomeRows, err := tx.Query(`
		SELECT id, market_id, outcome_id, specifiers
		FROM outcomes
		WHERE outcome_name IS NULL OR outcome_name = ''
		LIMIT 50000
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query outcomes: %w", err)
	}
	defer outcomeRows.Close()
	
	updatedOutcomes := 0
	for outcomeRows.Next() {
		var id int
		var marketID, outcomeID, specifiers string
		
		if err := outcomeRows.Scan(&id, &marketID, &outcomeID, &specifiers); err != nil {
			continue
		}
		
		// 大部分 outcome 名称不需要 team name
		ctx := &ReplacementContext{
			Specifiers: specifiers,
		}
		
		outcomeName := s.GetOutcomeName(marketID, outcomeID, specifiers, ctx)
		
		_, err := tx.Exec(`UPDATE outcomes SET outcome_name = $1 WHERE id = $2`, outcomeName, id)
		if err != nil {
			logger.Printf("[MarketDescService] ⚠️  Failed to update outcome %s/%s: %v", marketID, outcomeID, err)
			continue
		}
		updatedOutcomes++
	}
	
	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	logger.Printf("[MarketDescService] ✅ Bulk update completed: %d markets, %d outcomes updated", updatedMarkets, updatedOutcomes)
	
	return updatedMarkets, updatedOutcomes, nil
}

