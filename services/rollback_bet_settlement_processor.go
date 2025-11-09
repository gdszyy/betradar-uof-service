package services

import (
	"os"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
)

// RollbackBetSettlementProcessor Rollback Bet Settlement 消息处理器
type RollbackBetSettlementProcessor struct {
	db     *sql.DB
	logger *log.Logger
}

// RollbackBetSettlementMessage Rollback Bet Settlement 消息结构
type RollbackBetSettlementMessage struct {
	XMLName   xml.Name `xml:"rollback_bet_settlement"`
	EventID   string   `xml:"event_id,attr"`
	ProductID int      `xml:"product,attr"`
	Timestamp int64    `xml:"timestamp,attr"`
	Market    []struct {
		ID         string `xml:"id,attr"`
		Specifiers string `xml:"specifiers,attr"`
	} `xml:"market"`
}

// NewRollbackBetSettlementProcessor 创建 Rollback Bet Settlement 处理器
func NewRollbackBetSettlementProcessor(db *sql.DB) *RollbackBetSettlementProcessor {
	return &RollbackBetSettlementProcessor{
		db:     db,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// ProcessRollbackBetSettlement 处理 Rollback Bet Settlement 消息
func (p *RollbackBetSettlementProcessor) ProcessRollbackBetSettlement(xmlContent string) error {
	var rollback RollbackBetSettlementMessage
	if err := xml.Unmarshal([]byte(xmlContent), &rollback); err != nil {
		return fmt.Errorf("failed to parse rollback_bet_settlement message: %w", err)
	}

	// 开始事务
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 遍历所有市场
	for _, market := range rollback.Market {
		// 1. 删除 bet_settlements 表中的结算记录
		// 原始查询: DELETE FROM bet_settlements WHERE event_id = $1 AND sr_market_id = $2 AND specifiers = $3 AND producer_id = $4
		// 修正: producer_id 应该使用 product_id，且参数顺序调整
		deleteQuery := `
			DELETE FROM bet_settlements
			WHERE event_id = $1 AND product_id = $2 AND sr_market_id = $3 AND specifiers = $4
		`
		// 原始参数: rollback.EventID, market.ID, market.Specifiers, rollback.ProductID
		// 修正参数: rollback.EventID, rollback.ProductID, market.ID, market.Specifiers
		_, err := tx.Exec(deleteQuery, rollback.EventID, rollback.ProductID, market.ID, market.Specifiers)
		if err != nil {
			p.logger.Printf("Warning: failed to delete settlement record: %v", err)
		}

		// 2. 恢复 market 的 status 为 1 (Active)
		// 原始查询: UPDATE markets SET status = 1, updated_at = NOW() WHERE event_id = $1 AND sr_market_id = $2 AND specifiers = $3
		// 修正: markets 表需要 producer_id (即 product_id) 作为 WHERE 条件
		updateQuery := `
			UPDATE markets 
			SET status = 1, updated_at = NOW()
			WHERE event_id = $1 AND sr_market_id = $2 AND specifiers = $3 AND producer_id = $4
		`
		// 原始参数: rollback.EventID, market.ID, market.Specifiers
		// 修正参数: rollback.EventID, market.ID, market.Specifiers, rollback.ProductID
		_, err = tx.Exec(updateQuery, rollback.EventID, market.ID, market.Specifiers, rollback.ProductID)
		if err != nil {
			p.logger.Printf("Warning: failed to restore market status to active: %v", err)
		}

		// 3. 记录到 rollback_bet_settlements 表
		insertQuery := `
			INSERT INTO rollback_bet_settlements (
				event_id, producer_id, product_id, timestamp,
				sr_market_id, specifiers, market_count,
				xml_content
			) VALUES ($1, $2, $3, $4, $5, $6, 0, '')
			ON CONFLICT (event_id, sr_market_id, specifiers, producer_id) 
			DO UPDATE SET
				timestamp = EXCLUDED.timestamp,
				updated_at = NOW()
		`
		// 原始参数: rollback.EventID, rollback.ProductID, rollback.Timestamp, market.ID, market.Specifiers
		// 修正参数: rollback.EventID, rollback.ProductID, rollback.ProductID, rollback.Timestamp, market.ID, market.Specifiers
		_, err = tx.Exec(
			insertQuery,
			rollback.EventID,
			rollback.ProductID,
			rollback.ProductID, // product_id
			rollback.Timestamp,
			market.ID,
			market.Specifiers,
		)
		if err != nil {
			return fmt.Errorf("failed to insert rollback_bet_settlement: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 输出自然语言日志
	p.logger.Printf("[rollback_bet_settlement] 比赛 %s 的 %d个市场结算已回滚",
		rollback.EventID, len(rollback.Market))

	return nil
}
