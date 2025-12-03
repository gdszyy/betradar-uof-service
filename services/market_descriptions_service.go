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

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ReplacementContext 变量替换所需的上下文信息
type ReplacementContext struct {
	HomeTeamName string
	AwayTeamName string
	Specifiers   string // 原始 specifiers 字符串
}

// MarketDescriptionsService 市场描述服务
type MarketDescriptionsService struct {
	token          string
	apiBaseURL     string
	db             *sql.DB // 可选的数据库连接
	playersService *PlayersService // 球员信息服务
	markets        map[string]*MarketDescription
	outcomes       map[string]map[string]*OutcomeDescription // marketID -> outcomeID -> outcome
	mappings       map[string]map[string]string              // marketID -> outcomeID (URN) -> product_outcome_name
	marketMappings map[string][]Mapping                      // marketID -> []Mapping
	mu             sync.RWMutex
	lastUpdated    time.Time
}

// MarketDescription 市场描述
type MarketDescription struct {
	ID         string                   `xml:"id,attr"`
	Name       string                   `xml:"name,attr"`
	Groups     string                   `xml:"groups,attr"`
	Outcomes   []OutcomeDescription     `xml:"outcomes>outcome"`
	Specifiers []SpecifierDescription   `xml:"specifiers>specifier"`
	Mappings   []Mapping                `xml:"mappings>mapping"`
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

// Mapping 映射关系
type Mapping struct {
	ProductID  string           `xml:"product_id,attr"`
	ProductIDs string           `xml:"product_ids,attr"`
	SportID    string           `xml:"sport_id,attr"`
	MarketID   string           `xml:"market_id,attr"`
	Outcomes   []MappingOutcome `xml:"mapping_outcome"`
}

// MappingOutcome 映射结果
type MappingOutcome struct {
	OutcomeID        string `xml:"outcome_id,attr"`
	ProductOutcomeID string `xml:"product_outcome_id,attr"`
	ProductOutcomeName string `xml:"product_outcome_name,attr"`
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
		mappings:       make(map[string]map[string]string),
		marketMappings: make(map[string][]Mapping),
	}
}

// SetDatabase 设置数据库连接 (可选)
func (s *MarketDescriptionsService) SetDatabase(db *sql.DB) {
	s.db = db
}

// SetPlayersService 设置球员信息服务 (可选)
func (s *MarketDescriptionsService) SetPlayersService(playersService *PlayersService) {
	s.playersService = playersService
}

// Start 启动服务并加载市场描述
func (s *MarketDescriptionsService) Start() error {
	logger.Println("[MarketDescService] Starting Market Descriptions Service...")

	// 如果有数据库,优先从数据库加载
	if s.db != nil {
		err := s.loadFromDatabase()
		if err == nil {
			logger.Printf("[MarketDescService] ✅ Loaded %d markets from database cache", len(s.markets))

			// 启动定期刷新 (24小时)
			go s.refreshLoop()

			// 启动后台任务处理 Variant Market
			logger.Println("[MarketDescService] Starting asynchronous processing of all variant markets...")
			go s.processAllVariantMarketsAsync()

			return nil
		}
		logger.Printf("[MarketDescService] ⚠️  Failed to load from database, falling back to API: %v", err)
	}

	// 从 API 加载
	if err := s.loadMarketDescriptions(); err != nil {
		return fmt.Errorf("failed to load market descriptions: %w", err)
	}

	// 启动定期刷新 (24小时)
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

	// 加载 mappings
	mappingRows, err := s.db.Query(`
		SELECT market_id, outcome_id, product_outcome_name
		FROM mapping_outcomes
		ORDER BY market_id, outcome_id
	`)
	if err != nil {
		return fmt.Errorf("failed to query mappings: %w", err)
	}
	defer mappingRows.Close()

	mappingCount := 0
	for mappingRows.Next() {
		var marketID, outcomeID, productOutcomeName string

		if err := mappingRows.Scan(&marketID, &outcomeID, &productOutcomeName); err != nil {
			continue
		}

		if s.mappings[marketID] == nil {
			s.mappings[marketID] = make(map[string]string)
		}

		s.mappings[marketID][outcomeID] = productOutcomeName
		mappingCount++
	}

	if marketCount == 0 {
		return fmt.Errorf("no markets found in database")
	}

	s.lastUpdated = time.Now()
	logger.Printf("[MarketDescService] Loaded %d markets, %d outcomes, and %d mappings from database", marketCount, outcomeCount, mappingCount)

	return nil
}

// saveToDatabase 保存到数据库 - 使用分步提交策略
func (s *MarketDescriptionsService) saveToDatabase() error {
	if s.db == nil {
		return nil // 数据库不可用,跳过
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// 清空旧数据
	logger.Println("[MarketDescService] Clearing old data from database...")
	if _, err := s.db.Exec("DELETE FROM mapping_outcomes"); err != nil {
		return fmt.Errorf("failed to clear mapping_outcomes: %w", err)
	}
	if _, err := s.db.Exec("DELETE FROM outcome_descriptions"); err != nil {
		return fmt.Errorf("failed to clear outcomes: %w", err)
	}
	if _, err := s.db.Exec("DELETE FROM market_descriptions"); err != nil {
		return fmt.Errorf("failed to clear markets: %w", err)
	}

	// 使用分步提交策略：每个表每次提交一次
	logger.Println("[MarketDescService] Saving markets...")
	marketCount, err := s.saveMarketsToDatabase()
	if err != nil {
		logger.Printf("[MarketDescService] ⚠️  Failed to save markets: %v", err)
		return err
	}

	logger.Println("[MarketDescService] Saving outcomes...")
	outcomeCount, err := s.saveOutcomesToDatabase()
	if err != nil {
		logger.Printf("[MarketDescService] ⚠️  Failed to save outcomes: %v", err)
		return err
	}

	logger.Println("[MarketDescService] Saving mappings...")
	mappingCount, err := s.saveMappingsToDatabase()
	if err != nil {
		logger.Printf("[MarketDescService] ⚠️  Failed to save mappings: %v", err)
		return err
	}

	logger.Printf("[MarketDescService] ✅ Saved %d markets, %d outcomes, and %d mappings to database", marketCount, outcomeCount, mappingCount)
	return nil
}

// saveMarketsToDatabase 保存市场到数据库
func (s *MarketDescriptionsService) saveMarketsToDatabase() (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	marketStmt, err := tx.Prepare(`
		INSERT INTO market_descriptions (market_id, market_name, groups, specifiers)
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare market statement: %w", err)
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

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit markets transaction: %w", err)
	}

	return marketCount, nil
}

// saveOutcomesToDatabase 保存结果到数据库
func (s *MarketDescriptionsService) saveOutcomesToDatabase() (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	outcomeStmt, err := tx.Prepare(`
		INSERT INTO outcome_descriptions (market_id, outcome_id, outcome_name)
		VALUES ($1, $2, $3)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare outcome statement: %w", err)
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
		return 0, fmt.Errorf("failed to commit outcomes transaction: %w", err)
	}

	return outcomeCount, nil
}

// saveMappingsToDatabase 保存映射到数据库
func (s *MarketDescriptionsService) saveMappingsToDatabase() (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	mappingStmt, err := tx.Prepare(`
		INSERT INTO mapping_outcomes (market_id, outcome_id, product_outcome_name, product_id, sport_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (market_id, outcome_id, product_id, sport_id) DO UPDATE
		SET product_outcome_name = EXCLUDED.product_outcome_name
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare mapping statement: %w", err)
	}
	defer mappingStmt.Close()

	mappingCount := 0
	for marketID, mappingList := range s.marketMappings {
		for _, mapping := range mappingList {
			for _, mappingOutcome := range mapping.Outcomes {
				if _, err := mappingStmt.Exec(
					marketID,
					mappingOutcome.OutcomeID,
					mappingOutcome.ProductOutcomeName,
					mapping.ProductID,
					mapping.SportID,
				); err != nil {
					logger.Printf("[MarketDescService] ⚠️  Failed to insert mapping %s/%s: %v", marketID, mappingOutcome.OutcomeID, err)
					continue
				}
				mappingCount++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit mappings transaction: %w", err)
	}

	return mappingCount, nil
}

// loadMarketDescriptions 从 API 加载市场描述
func (s *MarketDescriptionsService) loadMarketDescriptions() error {
	// 构造 URL,如果 apiBaseURL 已经包含 /v1 则不重复添加
	apiBase := strings.TrimSuffix(s.apiBaseURL, "/v1")
	url := fmt.Sprintf("%s/v1/descriptions/en/markets.xml?include_mappings=true", apiBase)

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
	s.mappings = make(map[string]map[string]string)
	s.marketMappings = make(map[string][]Mapping)

	for i := range response.Markets {
		market := &response.Markets[i]
		s.markets[market.ID] = market

		// 索引 outcomes
		s.outcomes[market.ID] = make(map[string]*OutcomeDescription)
		for j := range market.Outcomes {
			outcome := &market.Outcomes[j]
			s.outcomes[market.ID][outcome.ID] = outcome
		}

		// 索引 mappings
		s.mappings[market.ID] = make(map[string]string)
		for _, mapping := range market.Mappings {
			for _, mappingOutcome := range mapping.Outcomes {
				// 使用 URN 作为 key
				s.mappings[market.ID][mappingOutcome.OutcomeID] = mappingOutcome.ProductOutcomeName
			}
		}

		// 保存完整的 Mapping 信息
		s.marketMappings[market.ID] = market.Mappings
	}

	s.lastUpdated = time.Now()
	s.mu.Unlock()

	// 统计 mappings 数量
	totalMappings := 0
	for _, outcomes := range s.mappings {
		totalMappings += len(outcomes)
	}

	logger.Printf("[MarketDescService] ✅ Loaded %d market descriptions from API", len(s.markets))
	logger.Printf("[MarketDescService] ✅ Parsed %d total mapping outcomes", totalMappings)

	// 保存到数据库 (如果可用)
	logger.Println("[MarketDescService] Saving initial market descriptions to database...")
	if err := s.saveToDatabase(); err != nil {
		logger.Printf("[MarketDescService] ⚠️  Failed to save initial data to database: %v", err)
		// Do not return an error here. We want the service to start even if the initial save fails.
	}

	// Asynchronously process all variant markets in the background
	logger.Println("[MarketDescService] Starting asynchronous processing of all variant markets...")
	go s.processAllVariantMarketsAsync()

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

	if market, ok := s.markets[marketID]; ok {
		name := market.Name
		// 替换变量
		if ctx != nil {
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
					name = strings.ReplaceAll(name, "{!"+key+"}", value)
				}
			}
		}
		return name
	}

	logger.Printf("[MarketDescService] ⚠️  Market not found: %s", marketID)
	return marketID
}

// GetOutcomeName 获取结果名称
func (s *MarketDescriptionsService) GetOutcomeName(marketID, outcomeID, specifiers string, ctx *ReplacementContext) string {
	s.mu.RLock()

	// 第一优先级: 从 outcomes 中查找
	if outcomes, ok := s.outcomes[marketID]; ok {
		if outcome, ok := outcomes[outcomeID]; ok {
			s.mu.RUnlock()
			name := outcome.Name
			// 替换变量
			if ctx != nil {
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
						name = strings.ReplaceAll(name, "{!"+key+"}", value)
					}
				}
			}
			return name
		}
	}

	// The logic for dynamically fetching variants has been moved to the background process
	// to prevent blocking. This function will now only return data from the cache.

	s.mu.RUnlock()

	// 第三优先级: 从 mappings 中查询 (仅用于特殊情况的降级)
	if mappings, ok := s.mappings[marketID]; ok {
		if productOutcomeName, ok := mappings[outcomeID]; ok {
			// 如果 product_outcome_name 是简单的数字或字母,可能需要进一步处理
			// 这里我们记录一个警告,表明使用了降级方案
			logger.Printf("[MarketDescService] ℹ️  Using mapping fallback for marketID=%s, outcomeID=%s, name=%s", marketID, outcomeID, productOutcomeName)
			return productOutcomeName
		}
	}

	// 检查是否是球员市场 (outcomeID 是球员 URN)
	if strings.HasPrefix(outcomeID, "sr:player:") {
		// 尝试从 PlayersService 获取球员姓名
		if s.playersService != nil {
			// 解锁以调用 playersService
			playerName := s.playersService.GetPlayerName(outcomeID)

			// GetPlayerName 总是返回一个值,如果找不到会返回 "Player {id}"
			// 我们直接返回这个值
			return playerName
		}
		// 如果找不到球员信息,返回球员 ID (不输出警告,因为这是正常情况)
		return outcomeID
	}

	// 对于非球员市场,输出警告日志
	logger.Printf("[MarketDescService] ⚠️  Outcome name not found: marketID=%s, outcomeID=%s, specifiers=%s", marketID, outcomeID, specifiers)
	return outcomeID
}

// VariantDescription 动态盘口描述
type VariantDescription struct {
	XMLName xml.Name `xml:"variant_description"`
	Variant struct {
		ID       string                   `xml:"id,attr"`
		Outcomes []OutcomeDescription     `xml:"outcomes>outcome"`
		Mappings []Mapping                `xml:"mappings>mapping"`
	} `xml:"variant"`
}

// fetchAndCacheVariant 动态加载并缓存 variant 描述
func (s *MarketDescriptionsService) fetchAndCacheVariant(marketID, outcomeID, variant string) (string, error) {
	// 构造 URL
	// 根据 Sportradar 文档，正确的路径是 /variants/（复数）而不是 /variant/（单数）
	apiBase := strings.TrimSuffix(s.apiBaseURL, "/v1")
	url := fmt.Sprintf("%s/v1/descriptions/en/markets/%s/variants/%s?include_mappings=true", apiBase, marketID, variant)

	logger.Printf("[MarketDescService] ⚡️ Dynamically fetching variant description from: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-access-token", s.token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch variant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d for variant %s", resp.StatusCode, variant)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	logger.Printf("[MarketDescService] API response body length: %d bytes", len(body))

	var variantDesc VariantDescription
	if err := xml.Unmarshal(body, &variantDesc); err != nil {
		logger.Printf("[MarketDescService] XML parsing error: %v", err)
		return "", fmt.Errorf("failed to parse variant XML: %w", err)
	}

	logger.Printf("[MarketDescService] Parsed variant: outcomes=%d, mappings=%d", len(variantDesc.Variant.Outcomes), len(variantDesc.Variant.Mappings))

s.mu.Lock()
		defer s.mu.Unlock()

		foundName := ""

		// 优先使用 <outcomes> 中的标准结果描述
		if len(variantDesc.Variant.Outcomes) > 0 {
			tx, err := s.db.Begin()
			if err != nil {
				return "", fmt.Errorf("failed to begin transaction: %w", err)
			}
			defer tx.Rollback() // Defer rollback in case of error

			stmt, err := tx.Prepare(`
				INSERT INTO outcome_descriptions (market_id, outcome_id, outcome_name, is_variant, variant_urn, updated_at)
				VALUES ($1, $2, $3, $4, $5, NOW())
				ON CONFLICT (market_id, outcome_id) DO UPDATE
				SET outcome_name = EXCLUDED.outcome_name, is_variant = EXCLUDED.is_variant, variant_urn = EXCLUDED.variant_urn, updated_at = NOW();
			`)
			if err != nil {
				return "", fmt.Errorf("failed to prepare statement: %w", err)
			}
			defer stmt.Close()

			// 使用 <outcomes> 中的标准结果描述
			for _, outcome := range variantDesc.Variant.Outcomes {
				// 写入数据库
				if _, err := stmt.Exec(marketID, outcome.ID, outcome.Name, true, variant); err != nil {
					logger.Printf("[MarketDescService] ⚠️  Failed to save variant outcome to DB: %v", err)
					continue
				}

				// 写入内存缓存
				if s.outcomes[marketID] == nil {
					s.outcomes[marketID] = make(map[string]*OutcomeDescription)
				}
				s.outcomes[marketID][outcome.ID] = &OutcomeDescription{ID: outcome.ID, Name: outcome.Name}

				if outcome.ID == outcomeID {
					foundName = outcome.Name
				}
			}

			if err := tx.Commit(); err != nil {
				return "", fmt.Errorf("failed to commit transaction: %w", err)
			}
		} else if len(variantDesc.Variant.Mappings) > 0 {
			// 备用：如果没有 <outcomes>，使用 <mappings> 中的 product_outcome_name
			tx, err := s.db.Begin()
			if err != nil {
				return "", fmt.Errorf("failed to begin transaction: %w", err)
			}
			defer tx.Rollback()

			stmt, err := tx.Prepare(`
				INSERT INTO outcome_descriptions (market_id, outcome_id, outcome_name, is_variant, variant_urn, updated_at)
				VALUES ($1, $2, $3, $4, $5, NOW())
				ON CONFLICT (market_id, outcome_id) DO UPDATE
				SET outcome_name = EXCLUDED.outcome_name, is_variant = EXCLUDED.is_variant, variant_urn = EXCLUDED.variant_urn, updated_at = NOW();
			`)
			if err != nil {
				return "", fmt.Errorf("failed to prepare statement: %w", err)
			}
			defer stmt.Close()

			// 使用 <mappings> 中的 product_outcome_name 作为备用
			for _, mapping := range variantDesc.Variant.Mappings {
				for _, o := range mapping.Outcomes {
					// 写入数据库
					if _, err := stmt.Exec(marketID, o.OutcomeID, o.ProductOutcomeName, true, variant); err != nil {
						logger.Printf("[MarketDescService] ⚠️  Failed to save variant outcome to DB: %v", err)
						continue
					}

					// 写入内存缓存
					if s.outcomes[marketID] == nil {
						s.outcomes[marketID] = make(map[string]*OutcomeDescription)
					}
					s.outcomes[marketID][o.OutcomeID] = &OutcomeDescription{ID: o.OutcomeID, Name: o.ProductOutcomeName}

					if o.OutcomeID == outcomeID {
						foundName = o.ProductOutcomeName
					}
				}
			}

			if err := tx.Commit(); err != nil {
				return "", fmt.Errorf("failed to commit transaction: %w", err)
			}
		}

		if foundName != "" {
			return foundName, nil
		}

		return "", fmt.Errorf("outcome %s not found in variant %s", outcomeID, variant)
}

// GetOutcomeNameTemplate 获取 outcome 的名称模板
func (s *MarketDescriptionsService) GetOutcomeNameTemplate(marketID, outcomeID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 从 outcomes 中查找
	if outcomes, ok := s.outcomes[marketID]; ok {
		if outcome, ok := outcomes[outcomeID]; ok {
			return outcome.Name, nil
		}
	}

	// 如果找不到，返回错误
	return "", fmt.Errorf("outcome template not found for marketID=%s, outcomeID=%s", marketID, outcomeID)
}

// GetMarketNameTemplate 获取 market 的名称模板
func (s *MarketDescriptionsService) GetMarketNameTemplate(marketID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 从 markets 中查找
	if market, ok := s.markets[marketID]; ok {
		return market.Name, nil
	}

	// 如果找不到，返回错误
	return "", fmt.Errorf("market template not found for marketID=%s", marketID)
}

// processAllVariantMarketsAsync 异步处理所有变体市场
func (s *MarketDescriptionsService) processAllVariantMarketsAsync() {
	logger.Println("[MarketDescService] Background task started: processing all variant markets.")

	if s.db == nil {
		logger.Println("[MarketDescService] Database not available, skipping variant market processing")
		return
	}

	// 给服务一些时间启动
	time.Sleep(5 * time.Second)

	// Query all variant markets that need to be fetched
	// Note: Only process sr: variants (Sportradar standard markets)
	// pre: variants (player props) are not supported by the /variant/ API endpoint
	logger.Println("[MarketDescService] Querying database for variant markets...")
rows, err := s.db.Query(`
				SELECT DISTINCT m.sr_market_id, o.outcome_id, m.specifiers
				FROM odds o
				JOIN markets m ON o.market_id = m.id
				WHERE m.specifiers LIKE 'variant=sr:%'
				AND NOT EXISTS (
					SELECT 1 FROM outcome_descriptions od
					WHERE od.market_id = CAST(m.sr_market_id AS VARCHAR)
					AND od.outcome_id = o.outcome_id
				)
				LIMIT 1000
		`)
	if err != nil {
		logger.Printf("[MarketDescService] ⚠️  Failed to query variant markets: %v", err)
		return
	}
	defer rows.Close()

	type VariantMarket struct {
		MarketID   string
		OutcomeID  string
		Specifiers string
	}

	var variants []VariantMarket
	for rows.Next() {
		var marketID, outcomeID, specifiers string
		if err := rows.Scan(&marketID, &outcomeID, &specifiers); err != nil {
			continue
		}
		variants = append(variants, VariantMarket{
			MarketID:   marketID,
			OutcomeID:  outcomeID,
			Specifiers: specifiers,
		})
	}

	if len(variants) == 0 {
		logger.Println("[MarketDescService] No variant markets found to process. This may indicate:")
		logger.Println("  1. All variant market outcomes already have names cached")
		logger.Println("  2. No markets in the database have specifiers with 'variant='")
		logger.Println("  3. The odds table is empty or not yet populated")
		return
	}

	logger.Printf("[MarketDescService] Found %d sr: variant markets to process (pre: variants are not supported by the API)", len(variants))

	// 处理每个变体市场
		processedCount := 0
		failedCount := 0
		for i, variant := range variants {
			// 从 specifiers 中提取 variant URN
			variantURN := s.extractVariantURN(variant.Specifiers)
			if variantURN == "" {
				logger.Printf("[MarketDescService] ⚠️  Failed to extract variant URN from specifiers: %s", variant.Specifiers)
				failedCount++
				continue
			}

			logger.Printf("[MarketDescService] [%d/%d] Processing sr: variant marketID=%s, outcomeID=%s, variantURN=%s", i+1, len(variants), variant.MarketID, variant.OutcomeID, variantURN)

			// 获取并缓存变体描述
			if _, err := s.fetchAndCacheVariant(variant.MarketID, variant.OutcomeID, variantURN); err != nil {
				logger.Printf("[MarketDescService] ❌ Failed to fetch variant %s/%s: %v", variant.MarketID, variantURN, err)
				failedCount++
				continue
			}

			logger.Printf("[MarketDescService] ✓ Successfully cached variant %s/%s", variant.MarketID, variantURN)
			processedCount++

			// 为了避免过度请求 API，每处理 10 个变体后休息 1 秒
			if processedCount%10 == 0 {
				time.Sleep(1 * time.Second)
			}
		}

		logger.Printf("[MarketDescService] ✅ Variant market processing completed: %d succeeded, %d failed out of %d total", processedCount, failedCount, len(variants))
}

// extractVariantURN 从 specifiers 中提取 variant URN
func (s *MarketDescriptionsService) extractVariantURN(specifiers string) string {
	// specifiers 格式: "variant=sr:point_range:6+" 或 "variant=sr:exact_goals:3+"
	pairs := strings.Split(specifiers, "|")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 && parts[0] == "variant" {
			return parts[1]
		}
	}
	return ""
}

// GetStatus 获取服务状态
func (s *MarketDescriptionsService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"markets_loaded":  len(s.markets),
		"outcomes_loaded": len(s.outcomes),
		"mappings_loaded": len(s.mappings),
		"last_updated":    s.lastUpdated,
	}
}

// ForceRefresh 强制刷新市场描述
func (s *MarketDescriptionsService) ForceRefresh() {
	logger.Println("[MarketDescService] Force refresh initiated...")
	if err := s.loadMarketDescriptions(); err != nil {
		logger.Printf("[MarketDescService] ⚠️  Force refresh failed: %v", err)
	}
}

// UpdateExistingMarkets 更新数据库中的现有市场和结果
func (s *MarketDescriptionsService) UpdateExistingMarkets() (int, int, error) {
	if s.db == nil {
		return 0, 0, fmt.Errorf("database not available")
	}

	logger.Println("[MarketDescService] Starting bulk update of existing markets and outcomes...")

	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 更新 markets 表
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

	marketStmt, err := tx.Prepare("UPDATE markets SET market_name = $1 WHERE id = $2")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare market update statement: %w", err)
	}
	defer marketStmt.Close()

	marketUpdateCount := 0
	for marketRows.Next() {
		var id int
		var marketID, specifiers, homeTeam, awayTeam string
		if err := marketRows.Scan(&id, &marketID, &specifiers, &homeTeam, &awayTeam); err != nil {
			continue
		}

		ctx := &ReplacementContext{HomeTeamName: homeTeam, AwayTeamName: awayTeam}
		marketName := s.GetMarketName(marketID, specifiers, ctx)

		if _, err := marketStmt.Exec(marketName, id); err != nil {
			// log and continue
		}
		marketUpdateCount++
	}
	logger.Printf("[MarketDescService] Updated %d market names", marketUpdateCount)

	// 更新 odds 表
	outcomeRows, err := tx.Query(`
		SELECT o.id, m.market_id, o.outcome_id, m.specifiers, COALESCE(m.home_team_name, ''), COALESCE(m.away_team_name, '')
		FROM odds o
		JOIN markets m ON o.market_id = m.id
		WHERE o.outcome_name IS NULL OR o.outcome_name = ''
		LIMIT 50000
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query outcomes: %w", err)
	}
	defer outcomeRows.Close()

	outcomeStmt, err := tx.Prepare("UPDATE odds SET outcome_name = $1 WHERE id = $2")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare outcome update statement: %w", err)
	}
	defer outcomeStmt.Close()

	outcomeUpdateCount := 0
	for outcomeRows.Next() {
		var id int
		var marketID, outcomeID, specifiers, homeTeam, awayTeam string
		if err := outcomeRows.Scan(&id, &marketID, &outcomeID, &specifiers, &homeTeam, &awayTeam); err != nil {
			continue
		}

		ctx := &ReplacementContext{HomeTeamName: homeTeam, AwayTeamName: awayTeam}
		outcomeName := s.GetOutcomeName(marketID, outcomeID, specifiers, ctx)

		if _, err := outcomeStmt.Exec(outcomeName, id); err != nil {
			// log and continue
		}
		outcomeUpdateCount++
	}
	logger.Printf("[MarketDescService] Updated %d outcome names", outcomeUpdateCount)

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return marketUpdateCount, outcomeUpdateCount, nil
}
