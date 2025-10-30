package services

import (
"os"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
)

// RollbackBetCancelProcessor Rollback Bet Cancel 消息处理器
type RollbackBetCancelProcessor struct {
	db     *sql.DB
	logger *log.Logger
}

// RollbackBetCancelMessage Rollback Bet Cancel 消息结构
type RollbackBetCancelMessage struct {
	XMLName   xml.Name `xml:"rollback_bet_cancel"`
	EventID   string   `xml:"event_id,attr"`
	ProductID int      `xml:"product,attr"`
	Timestamp int64    `xml:"timestamp,attr"`
	Market    []struct {
		ID         string `xml:"id,attr"`
		Specifiers string `xml:"specifiers,attr"`
	} `xml:"market"`
}

// NewRollbackBetCancelProcessor 创建 Rollback Bet Cancel 处理器
func NewRollbackBetCancelProcessor(db *sql.DB) *RollbackBetCancelProcessor {
	return &RollbackBetCancelProcessor{
		db:     db,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// ProcessRollbackBetCancel 处理 Rollback Bet Cancel 消息
func (p *RollbackBetCancelProcessor) ProcessRollbackBetCancel(xmlContent string) error {
	var rollback RollbackBetCancelMessage
	if err := xml.Unmarshal([]byte(xmlContent), &rollback); err != nil {
		return fmt.Errorf("failed to parse rollback_bet_cancel message: %w", err)
	}

	// 开始事务
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 遍历所有市场
	for _, market := range rollback.Market {
		// 1. 删除 bet_cancels 表中的取消记录
		deleteQuery := `
			DELETE FROM bet_cancels
			WHERE event_id = $1 AND market_id = $2 AND specifiers = $3 AND producer_id = $4
		`
		_, err := tx.Exec(deleteQuery, rollback.EventID, market.ID, market.Specifiers, rollback.ProductID)
		if err != nil {
			p.logger.Printf("Warning: failed to delete cancel record: %v", err)
		}

		// 2. 恢复 market 的 status 为 1 (Active)
		updateQuery := `
			UPDATE markets 
			SET status = 1, updated_at = NOW()
			WHERE event_id = $1 AND market_id = $2 AND specifiers = $3
		`
		_, err = tx.Exec(updateQuery, rollback.EventID, market.ID, market.Specifiers)
		if err != nil {
			p.logger.Printf("Warning: failed to restore market status to active: %v", err)
		}

		// 3. 记录到 rollback_bet_cancels 表
		insertQuery := `
			INSERT INTO rollback_bet_cancels (
				event_id, producer_id, timestamp,
				market_id, specifiers,
				created_at
			) VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (event_id, market_id, specifiers, producer_id) 
			DO UPDATE SET
				timestamp = EXCLUDED.timestamp,
				created_at = NOW()
		`
		_, err = tx.Exec(
			insertQuery,
			rollback.EventID,
			rollback.ProductID,
			rollback.Timestamp,
			market.ID,
			market.Specifiers,
		)
		if err != nil {
			return fmt.Errorf("failed to insert rollback_bet_cancel: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 输出自然语言日志
	p.logger.Printf("[rollback_bet_cancel] 比赛 %s 的 %d个市场取消已回滚",
		rollback.EventID, len(rollback.Market))

	return nil
}

