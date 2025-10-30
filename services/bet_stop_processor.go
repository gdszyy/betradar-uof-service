package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
)

// BetStopProcessor Bet Stop 消息处理器
type BetStopProcessor struct {
	db     *sql.DB
	logger *log.Logger
}

// BetStopMessage Bet Stop 消息结构
type BetStopMessage struct {
	XMLName      xml.Name `xml:"bet_stop"`
	EventID      string   `xml:"event_id,attr"`
	ProductID    int      `xml:"product,attr"`
	Timestamp    int64    `xml:"timestamp,attr"`
	MarketStatus *int     `xml:"market_status,attr"` // 可选
	Groups       string   `xml:"groups,attr"`        // "all" 或 "1|2|3"
}

// NewBetStopProcessor 创建 Bet Stop 处理器
func NewBetStopProcessor(db *sql.DB) *BetStopProcessor {
	return &BetStopProcessor{
		db:     db,
		logger: log.New(log.Writer(), "[BetStopProcessor] ", log.LstdFlags),
	}
}

// ProcessBetStop 处理 Bet Stop 消息并更新 market status
func (p *BetStopProcessor) ProcessBetStop(xmlContent string) error {
	var betStop BetStopMessage
	if err := xml.Unmarshal([]byte(xmlContent), &betStop); err != nil {
		return fmt.Errorf("failed to parse bet_stop message: %w", err)
	}

	p.logger.Printf("Processing bet_stop for event: %s (groups=%s, market_status=%v)",
		betStop.EventID, betStop.Groups, betStop.MarketStatus)

	// 根据 groups 更新 market status
	if err := p.updateMarketStatus(betStop); err != nil {
		return fmt.Errorf("failed to update market status: %w", err)
	}

	return nil
}

// updateMarketStatus 更新市场状态
func (p *BetStopProcessor) updateMarketStatus(betStop BetStopMessage) error {
	// 确定要设置的状态值
	// 根据 Betradar 文档:
	// - bet_stop 通常表示市场暂停 (suspended)
	// - market_status 如果存在,使用该值
	targetStatus := 2 // 默认: 2 = suspended

	if betStop.MarketStatus != nil {
		targetStatus = *betStop.MarketStatus
	}

	// 根据 groups 字段更新不同的市场
	if betStop.Groups == "all" || betStop.Groups == "" {
		// 更新该赛事的所有市场
		query := `
			UPDATE markets 
			SET status = $1, updated_at = NOW()
			WHERE event_id = $2
		`
		result, err := p.db.Exec(query, targetStatus, betStop.EventID)
		if err != nil {
			return fmt.Errorf("failed to update all markets: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		p.logger.Printf("✅ Updated %d markets to status=%d for event %s (groups=all)",
			rowsAffected, targetStatus, betStop.EventID)

	} else {
		// 更新特定市场组
		// groups 格式: "1|2|3" 表示 market group IDs
		// 注意: 这需要 market_descriptions 中的 group 信息
		
		// 简化实现: 如果 groups 不是 "all",暂时也更新所有市场
		// TODO: 实现基于 market group 的筛选
		query := `
			UPDATE markets 
			SET status = $1, updated_at = NOW()
			WHERE event_id = $2
		`
		result, err := p.db.Exec(query, targetStatus, betStop.EventID)
		if err != nil {
			return fmt.Errorf("failed to update markets by groups: %w", err)
		}

		rowsAffected, _ := result.RowsAffected()
		p.logger.Printf("✅ Updated %d markets to status=%d for event %s (groups=%s)",
			rowsAffected, targetStatus, betStop.EventID, betStop.Groups)
	}

	return nil
}

