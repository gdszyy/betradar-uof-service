package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"os"
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
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// ParseAndStore 解析并存储 Bet Settlement 消息
func (p *BetSettlementParser) ParseAndStore(xmlContent string) error {
	var settlement BetSettlementMessage
	if err := xml.Unmarshal([]byte(xmlContent), &settlement); err != nil {
		return fmt.Errorf("failed to parse bet_settlement message: %w", err)
	}

	// 日志在处理完成后输出

	// 开始事务
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

		// 遍历所有市场
		for _, market := range settlement.Outcomes.Markets {
			marketID, err := ExtractMarketIDFromURN(market.ID)
			if err != nil {
				p.logger.Printf("Warning: failed to extract market ID from URN %s: %v", market.ID, err)
				continue
			}

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
						sr_market_id, specifiers, void_factor,
							outcome_id, result, dead_heat_factor,
							created_at
						) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
				ON CONFLICT (event_id, sr_market_id, specifiers, outcome_id, producer_id) 
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
						marketID,
					market.Specifiers,
				finalVoidFactor,
				outcome.ID,
				outcome.Result,
				outcome.DeadHeatFactor,
			)
				if err != nil {
					p.logger.Printf("Error: failed to insert bet_settlement: %v", err)
					return fmt.Errorf("failed to insert bet_settlement: %w", err)
				}
		}

		// 更新当前 market 的 status 为 -3 (Settled)
		updateQuery := `
				UPDATE markets 
				SET status = -3, updated_at = NOW()
					WHERE event_id = $1 AND sr_market_id = $2 AND specifiers = $3
			`
			_, err = tx.Exec(updateQuery, settlement.EventID, marketID, market.Specifiers)
			if err != nil {
				p.logger.Printf("Warning: failed to update market status to settled: %v", err)
				// 不返回错误，因为这只是一个警告
			}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 统计结算结果数量
	outcomeCount := 0
	for _, market := range settlement.Outcomes.Markets {
		outcomeCount += len(market.Outcomes)
	}
	
	// 确定结算类型
	certaintyText := ""
	switch settlement.Certainty {
	case 1:
		certaintyText = "低确定性"
	case 2:
		certaintyText = "高确定性"
	case 3:
		certaintyText = "最高确定性"
	default:
		certaintyText = fmt.Sprintf("certainty=%d", settlement.Certainty)
	}
	
	p.logger.Printf("[bet_settlement] 比赛 %s 的 %d个市场已结算: %d个结果 (%s)",
		settlement.EventID, len(settlement.Outcomes.Markets), outcomeCount, certaintyText)

	return nil
}

