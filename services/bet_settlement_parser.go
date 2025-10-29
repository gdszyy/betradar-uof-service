package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
)

// BetSettlementParser Bet Settlement 消息解析器
type BetSettlementParser struct {
	db     *sql.DB
	logger *log.Logger
}

// BetSettlementMessage Bet Settlement 消息结构
type BetSettlementMessage struct {
	XMLName   xml.Name `xml:"bet_settlement"`
	EventID   string   `xml:"event_id,attr"`
	ProductID int      `xml:"product,attr"`
	Timestamp int64    `xml:"timestamp,attr"`
	Certainty int      `xml:"certainty,attr"`
	Outcomes  struct {
		Markets []SettlementMarket `xml:"market"`
	} `xml:"outcomes"`
}

// SettlementMarket 结算市场
type SettlementMarket struct {
	ID         string             `xml:"id,attr"`
	Specifiers string             `xml:"specifiers,attr"`
	VoidFactor *float64           `xml:"void_factor,attr"` // 可选
	Outcomes   []SettlementOutcome `xml:"outcome"`
}

// SettlementOutcome 结算结果
type SettlementOutcome struct {
	ID             string   `xml:"id,attr"`
	Result         int      `xml:"result,attr"` // -1=void, 0=lose, 1=win, 0.5=half-win
	VoidFactor     *float64 `xml:"void_factor,attr"` // 可选,outcome 级别
	DeadHeatFactor *float64 `xml:"dead_heat_factor,attr"` // 可选
}

// NewBetSettlementParser 创建 Bet Settlement 解析器
func NewBetSettlementParser(db *sql.DB) *BetSettlementParser {
	return &BetSettlementParser{
		db:     db,
		logger: log.New(log.Writer(), "[BetSettlementParser] ", log.LstdFlags),
	}
}

// ParseAndStore 解析并存储 Bet Settlement 消息
func (p *BetSettlementParser) ParseAndStore(xmlContent string) error {
	var settlement BetSettlementMessage
	if err := xml.Unmarshal([]byte(xmlContent), &settlement); err != nil {
		return fmt.Errorf("failed to parse bet_settlement message: %w", err)
	}

	p.logger.Printf("Parsing bet_settlement for event: %s (certainty=%d, markets=%d)",
		settlement.EventID, settlement.Certainty, len(settlement.Outcomes.Markets))

	// 开始事务
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 遍历所有市场
	for _, market := range settlement.Outcomes.Markets {
		// 遍历所有结果
		for _, outcome := range market.Outcomes {
			// 确定最终的 void_factor (outcome 级别优先于 market 级别)
			var finalVoidFactor *float64
			if outcome.VoidFactor != nil {
				finalVoidFactor = outcome.VoidFactor
			} else if market.VoidFactor != nil {
				finalVoidFactor = market.VoidFactor
			}

			// 存储到数据库
			query := `
				INSERT INTO bet_settlements (
					event_id, producer_id, timestamp, certainty,
					market_id, specifiers, void_factor,
					outcome_id, result, dead_heat_factor,
					created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
				ON CONFLICT (event_id, market_id, specifiers, outcome_id, producer_id) 
				DO UPDATE SET
					certainty = EXCLUDED.certainty,
					void_factor = EXCLUDED.void_factor,
					result = EXCLUDED.result,
					dead_heat_factor = EXCLUDED.dead_heat_factor,
					timestamp = EXCLUDED.timestamp,
					created_at = NOW()
			`

			_, err := tx.Exec(
				query,
				settlement.EventID,
				settlement.ProductID,
				settlement.Timestamp,
				settlement.Certainty,
				market.ID,
				market.Specifiers,
				finalVoidFactor,
				outcome.ID,
				outcome.Result,
				outcome.DeadHeatFactor,
			)
			if err != nil {
				return fmt.Errorf("failed to insert bet_settlement: %w", err)
			}

			p.logger.Printf("Stored settlement: event=%s, market=%s, specifiers=%s, outcome=%s, result=%d, void_factor=%v",
				settlement.EventID, market.ID, market.Specifiers, outcome.ID, outcome.Result, finalVoidFactor)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	p.logger.Printf("Successfully stored bet_settlement for event %s: %d markets",
		settlement.EventID, len(settlement.Outcomes.Markets))

	return nil
}

