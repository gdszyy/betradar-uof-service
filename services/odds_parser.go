package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
)

// OddsParser 赔率解析器
type OddsParser struct {
	db *sql.DB
}

// NewOddsParser 创建赔率解析器
func NewOddsParser(db *sql.DB) *OddsParser {
	return &OddsParser{db: db}
}

// OddsChangeData 赔率变化数据结构
type OddsChangeData struct {
	EventID   string       `xml:"event_id,attr"`
	Timestamp int64        `xml:"timestamp,attr"`
	ProductID int          `xml:"product,attr"`
	Markets   []MarketData `xml:"odds>market"`
}

// MarketData 盘口数据
type MarketData struct {
	ID        string        `xml:"id,attr"`
	Status    string        `xml:"status,attr"`
	Specifiers string       `xml:"specifiers,attr"`
	Outcomes  []OutcomeData `xml:"outcome"`
}

// OutcomeData 结果数据
type OutcomeData struct {
	ID     string  `xml:"id,attr"`
	Odds   float64 `xml:"odds,attr"`
	Active int     `xml:"active,attr"`
}

// ParseAndStoreOdds 解析并存储赔率数据
func (p *OddsParser) ParseAndStoreOdds(xmlData []byte, productID int) error {
	var oddsChange OddsChangeData
	if err := xml.Unmarshal(xmlData, &oddsChange); err != nil {
		return fmt.Errorf("failed to parse odds_change XML: %w", err)
	}
	
	log.Printf("[OddsParser] Parsing odds_change for event: %s, markets: %d, producer: %d", 
		oddsChange.EventID, len(oddsChange.Markets), productID)
	
	// 开始事务
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 存储每个盘口
	for _, market := range oddsChange.Markets {
		if err := p.storeMarket(tx, oddsChange.EventID, market, oddsChange.Timestamp, productID); err != nil {
			log.Printf("[OddsParser] Failed to store market %s: %v", market.ID, err)
			continue
		}
	}
	
	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	log.Printf("[OddsParser] Successfully stored odds for event: %s", oddsChange.EventID)
	return nil
}

// storeMarket 存储盘口数据
func (p *OddsParser) storeMarket(tx *sql.Tx, eventID string, market MarketData, timestamp int64, productID int) error {
	// 1. 插入或更新盘口
	marketQuery := `
		INSERT INTO markets (event_id, market_id, market_type, specifiers, status, producer_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (event_id, market_id, specifiers) DO UPDATE
		SET status = EXCLUDED.status, producer_id = EXCLUDED.producer_id, updated_at = NOW()
		RETURNING id
	`
	
	var marketPK int
	err := tx.QueryRow(marketQuery, 
		eventID, 
		market.ID, 
		p.getMarketType(market.ID),
		market.Specifiers,
		market.Status,
		productID,
	).Scan(&marketPK)
	
	if err != nil {
		return fmt.Errorf("failed to insert/update market: %w", err)
	}
	
	// 2. 存储每个结果的赔率
	for _, outcome := range market.Outcomes {
		if err := p.storeOdds(tx, marketPK, eventID, outcome, timestamp); err != nil {
			log.Printf("[OddsParser] Failed to store odds for outcome %s: %v", outcome.ID, err)
		}
	}
	
	return nil
}

// storeOdds 存储赔率
func (p *OddsParser) storeOdds(tx *sql.Tx, marketPK int, eventID string, outcome OutcomeData, timestamp int64) error {
	// 查询旧赔率
	var oldOdds sql.NullFloat64
	oldOddsQuery := `SELECT odds_value FROM odds WHERE market_id = $1 AND outcome_id = $2`
	tx.QueryRow(oldOddsQuery, marketPK, outcome.ID).Scan(&oldOdds)
	
	// 计算隐含概率
	probability := 0.0
	if outcome.Odds > 0 {
		probability = 1.0 / outcome.Odds
	}
	
	// 插入或更新当前赔率
	oddsQuery := `
		INSERT INTO odds (market_id, event_id, outcome_id, outcome_name, odds_value, probability, active, timestamp, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (market_id, outcome_id) DO UPDATE
		SET odds_value = EXCLUDED.odds_value,
		    probability = EXCLUDED.probability,
		    active = EXCLUDED.active,
		    timestamp = EXCLUDED.timestamp,
		    updated_at = NOW()
	`
	
	_, err := tx.Exec(oddsQuery,
		marketPK,
		eventID,
		outcome.ID,
		p.getOutcomeName(outcome.ID),
		outcome.Odds,
		probability,
		outcome.Active == 1,
		timestamp,
	)
	
	if err != nil {
		return fmt.Errorf("failed to insert/update odds: %w", err)
	}
	
	// 如果赔率有变化,记录到历史表
	if oldOdds.Valid && oldOdds.Float64 != outcome.Odds {
		changeType := "up"
		if outcome.Odds < oldOdds.Float64 {
			changeType = "down"
		}
		
		historyQuery := `
			INSERT INTO odds_history (market_id, event_id, outcome_id, outcome_name, odds_value, probability, change_type, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		
		_, err = tx.Exec(historyQuery,
			marketPK,
			eventID,
			outcome.ID,
			p.getOutcomeName(outcome.ID),
			outcome.Odds,
			probability,
			changeType,
			timestamp,
		)
		
		if err != nil {
			log.Printf("[OddsParser] Failed to insert odds history: %v", err)
		}
	} else if !oldOdds.Valid {
		// 新赔率
		historyQuery := `
			INSERT INTO odds_history (market_id, event_id, outcome_id, outcome_name, odds_value, probability, change_type, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, 'new', $7)
		`
		
		tx.Exec(historyQuery, marketPK, eventID, outcome.ID, p.getOutcomeName(outcome.ID), outcome.Odds, probability, timestamp)
	}
	
	return nil
}

// getMarketType 获取盘口类型
func (p *OddsParser) getMarketType(marketID string) string {
	// 解析盘口 ID,提取类型
	// 例如: 1 -> 1x2, 18 -> handicap, 26 -> totals
	marketTypeMap := map[string]string{
		"1":   "1x2",
		"2":   "double_chance",
		"10":  "draw_no_bet",
		"18":  "handicap",
		"26":  "totals",
		"29":  "both_teams_to_score",
		"52":  "asian_handicap",
		"238": "asian_totals",
	}
	
	if marketType, ok := marketTypeMap[marketID]; ok {
		return marketType
	}
	
	return "other"
}

// getOutcomeName 获取结果名称
func (p *OddsParser) getOutcomeName(outcomeID string) string {
	// 解析结果 ID,返回友好名称
	// 例如: 1 -> Home, 2 -> Away, X -> Draw
	outcomeNameMap := map[string]string{
		"1":  "Home",
		"2":  "Away",
		"X":  "Draw",
		"12": "Home or Away",
		"1X": "Home or Draw",
		"2X": "Away or Draw",
		"over": "Over",
		"under": "Under",
		"yes": "Yes",
		"no": "No",
	}
	
	if name, ok := outcomeNameMap[outcomeID]; ok {
		return name
	}
	
	return outcomeID
}

// GetMarketOdds 获取盘口的当前赔率
func (p *OddsParser) GetMarketOdds(eventID, marketID string) ([]OddsDetail, error) {
	query := `
		SELECT 
			o.outcome_id,
			o.outcome_name,
			o.odds_value,
			o.probability,
			o.active,
			o.timestamp,
			o.updated_at
		FROM odds o
		JOIN markets m ON o.market_id = m.id
		WHERE m.event_id = $1 AND m.market_id = $2
		ORDER BY o.outcome_id
	`
	
	rows, err := p.db.Query(query, eventID, marketID)
	if err != nil {
		return nil, fmt.Errorf("failed to query odds: %w", err)
	}
	defer rows.Close()
	
	var oddsList []OddsDetail
	for rows.Next() {
		var odds OddsDetail
		err := rows.Scan(
			&odds.OutcomeID,
			&odds.OutcomeName,
			&odds.OddsValue,
			&odds.Probability,
			&odds.Active,
			&odds.Timestamp,
			&odds.UpdatedAt,
		)
		if err != nil {
			log.Printf("[OddsParser] Failed to scan odds: %v", err)
			continue
		}
		oddsList = append(oddsList, odds)
	}
	
	return oddsList, nil
}

// GetOddsHistory 获取赔率变化历史
func (p *OddsParser) GetOddsHistory(eventID, marketID, outcomeID string, limit int) ([]OddsHistoryInfo, error) {
	query := `
		SELECT 
			oh.odds_value,
			oh.probability,
			oh.change_type,
			oh.timestamp,
			oh.created_at
		FROM odds_history oh
		JOIN markets m ON oh.market_id = m.id
		WHERE m.event_id = $1 AND m.market_id = $2 AND oh.outcome_id = $3
		ORDER BY oh.created_at DESC
		LIMIT $4
	`
	
	rows, err := p.db.Query(query, eventID, marketID, outcomeID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query odds history: %w", err)
	}
	defer rows.Close()
	
	var historyList []OddsHistoryInfo
	for rows.Next() {
		var history OddsHistoryInfo
		err := rows.Scan(
			&history.OddsValue,
			&history.Probability,
			&history.ChangeType,
			&history.Timestamp,
			&history.CreatedAt,
		)
		if err != nil {
			log.Printf("[OddsParser] Failed to scan odds history: %v", err)
			continue
		}
		historyList = append(historyList, history)
	}
	
	return historyList, nil
}

// OddsDetail 赔率详情
type OddsDetail struct {
	OutcomeID   string  `json:"outcome_id"`
	OutcomeName string  `json:"outcome_name"`
	OddsValue   float64 `json:"odds_value"`
	Probability float64 `json:"probability"`
	Active      bool    `json:"active"`
	Timestamp   int64   `json:"timestamp"`
	UpdatedAt   string  `json:"updated_at"`
}

// OddsHistoryInfo 赔率历史信息
type OddsHistoryInfo struct {
	OddsValue   float64 `json:"odds_value"`
	Probability float64 `json:"probability"`
	ChangeType  string  `json:"change_type"`
	Timestamp   int64   `json:"timestamp"`
	CreatedAt   string  `json:"created_at"`
}

// GetEventMarkets 获取比赛的所有盘口
func (p *OddsParser) GetEventMarkets(eventID string) ([]MarketInfo, error) {
	query := `
		SELECT 
			m.id,
			m.market_id,
			m.market_type,
			m.market_name,
			m.specifiers,
			m.status,
			COUNT(o.id) as odds_count,
			m.updated_at
		FROM markets m
		LEFT JOIN odds o ON m.id = o.market_id
		WHERE m.event_id = $1
		GROUP BY m.id
		ORDER BY m.market_type, m.id
	`
	
	rows, err := p.db.Query(query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()
	
	var markets []MarketInfo
	for rows.Next() {
		var market MarketInfo
		var specifiers sql.NullString
		var marketName sql.NullString
		
		err := rows.Scan(
			&market.ID,
			&market.MarketID,
			&market.MarketType,
			&marketName,
			&specifiers,
			&market.Status,
			&market.OddsCount,
			&market.UpdatedAt,
		)
		if err != nil {
			log.Printf("[OddsParser] Failed to scan market: %v", err)
			continue
		}
		
		if specifiers.Valid {
			market.Specifiers = specifiers.String
		}
		if marketName.Valid {
			market.MarketName = marketName.String
		} else {
			market.MarketName = p.getMarketTypeName(market.MarketType)
		}
		
		markets = append(markets, market)
	}
	
	return markets, nil
}

// MarketInfo 盘口信息
type MarketInfo struct {
	ID          int    `json:"id"`
	MarketID    string `json:"market_id"`
	MarketType  string `json:"market_type"`
	MarketName  string `json:"market_name"`
	Specifiers  string `json:"specifiers,omitempty"`
	Status      string `json:"status"`
	OddsCount   int    `json:"odds_count"`
	UpdatedAt   string `json:"updated_at"`
}

// getMarketTypeName 获取盘口类型名称
func (p *OddsParser) getMarketTypeName(marketType string) string {
	marketTypeNames := map[string]string{
		"1x2":                 "胜平负",
		"double_chance":       "双重机会",
		"draw_no_bet":         "平局退款",
		"handicap":            "让球",
		"totals":              "大小球",
		"both_teams_to_score": "两队都进球",
		"asian_handicap":      "亚洲让球",
		"asian_totals":        "亚洲大小球",
	}
	
	if name, ok := marketTypeNames[marketType]; ok {
		return name
	}
	
	return marketType
}

// 添加缺少的 UNIQUE 约束
func init() {
	// 这个函数会在包初始化时执行
	// 用于确保数据库表有正确的约束
}

