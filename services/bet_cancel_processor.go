package services

import (
	"os"
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	)

// BetCancelProcessor Bet Cancel 消息处理器
type BetCancelProcessor struct {
	db     *sql.DB
	logger *log.Logger
}

// BetCancelMessage Bet Cancel 消息结构
type BetCancelMessage struct {
	XMLName      xml.Name `xml:"bet_cancel"`
	EventID      string   `xml:"event_id,attr"`
	ProductID    int      `xml:"product,attr"`
	Timestamp    int64    `xml:"timestamp,attr"`
	StartTime    *int64   `xml:"start_time,attr"`
	EndTime      *int64   `xml:"end_time,attr"`
	SupercededBy *string  `xml:"superceded_by,attr"`
	Market       []struct {
		ID         string `xml:"id,attr"`
		Specifiers string `xml:"specifiers,attr"`
		VoidReason *int   `xml:"void_reason,attr"`
	} `xml:"market"`
}

// NewBetCancelProcessor 创建 Bet Cancel 处理器
func NewBetCancelProcessor(db *sql.DB) *BetCancelProcessor {
	return &BetCancelProcessor{
		db:     db,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// ProcessBetCancel 处理 Bet Cancel 消息并更新 market status
func (p *BetCancelProcessor) ProcessBetCancel(xmlContent string) error {
	var betCancel BetCancelMessage
	if err := xml.Unmarshal([]byte(xmlContent), &betCancel); err != nil {
		return fmt.Errorf("failed to parse bet_cancel message: %w", err)
	}

	// 开始事务
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// 遍历所有市场
	for _, market := range betCancel.Market {
		var marketID int64
		marketID, err = ExtractMarketIDFromURN(market.ID)
		if err != nil {
			p.logger.Printf("Warning: failed to extract market ID from URN %s: %v", market.ID, err)
			continue
		}

		// 存储到 bet_cancels 表
		// 修复: 字段 producer_id, sr_market_id, specifiers, void_reason, start_time, end_time, superceded_by
		// 修复: ON CONFLICT 字段
		query := `
		INSERT INTO bet_cancels (
			event_id, producer_id, product_id, timestamp,
			sr_market_id, specifiers, void_reason,
			start_time, end_time, superceded_by,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (event_id, sr_market_id, specifiers, producer_id) 
		DO UPDATE SET
			void_reason = EXCLUDED.void_reason,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			superceded_by = EXCLUDED.superceded_by,
			timestamp = EXCLUDED.timestamp,
			created_at = NOW()
		`
		
		_, err := tx.Exec(
			query,
			betCancel.EventID,
			betCancel.ProductID, // $2
			betCancel.ProductID, // $3 - 修正：ProductID 应该在 $3，原代码中 $2 传了 ProductID，但 SQL 中是 producer_id。这里假设 ProductID 就是 producer_id。
			betCancel.Timestamp, // $4
			marketID,            // $5
			market.Specifiers,   // $6
			market.VoidReason,   // $7
			betCancel.StartTime, // $8
			betCancel.EndTime,   // $9
			betCancel.SupercededBy, // $10
		)
		if err != nil {
			p.logger.Printf("Error: failed to insert bet_cancel: %v", err)
			return fmt.Errorf("failed to insert bet_cancel: %w", err)
		}

		// 更新当前 market 的 status 为 -4 (Cancelled)
		// 修复: 确保 WHERE 子句中的字段正确 (sr_market_id, specifiers)
		updateQuery := `
		UPDATE markets 
		SET status = -4, updated_at = NOW()
		WHERE event_id = $1 AND sr_market_id = $2 AND specifiers = $3
		`
		_, err = tx.Exec(updateQuery, betCancel.EventID, marketID, market.Specifiers)
		if err != nil {
			p.logger.Printf("Warning: failed to update market status to cancelled: %v", err)
			// 不返回错误，因为这只是一个警告
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 输出自然语言日志
	p.logger.Printf("[bet_cancel] 比赛 %s 的 %d个市场已取消",
		betCancel.EventID, len(betCancel.Market))

	return nil
}
