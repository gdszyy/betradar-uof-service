package services

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// ProcessBetStop 处理 Bet Stop 消息并更新 market status
func (p *BetStopProcessor) ProcessBetStop(xmlContent string) error {
	var betStop BetStopMessage
	if err := xml.Unmarshal([]byte(xmlContent), &betStop); err != nil {
		return fmt.Errorf("failed to parse bet_stop message: %w", err)
	}

	// 日志在更新后输出

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
	// Market Status 枚举:
	//   1 = Active, -1 = Suspended, 0 = Inactive, -3 = Settled, -4 = Cancelled, -2 = Handed over
	targetStatus := -1 // 默认: -1 = Suspended

	if betStop.MarketStatus != nil {
		targetStatus = *betStop.MarketStatus
	}

	// 根据 groups 字段更新不同的市场
	if betStop.Groups == "all" || betStop.Groups == "" {
		// 更新该赛事的所有市场
<<<<<<< HEAD
		query := `
				UPDATE markets 
				SET status = $1, updated_at = NOW()
				WHERE event_id::text = $2
		`
			p.logger.Printf("[DEBUG] SQL Query: %s", query)
			p.logger.Printf("[DEBUG] SQL Args: %v", []interface{}{targetStatus, betStop.EventID})
			
			result, err := p.db.Exec(query, targetStatus, betStop.EventID)
			if err != nil {
				return fmt.Errorf("failed to update all markets: %w", err)
			}
=======
			eventID, err := ExtractEventIDFromURN(betStop.EventID)
			if err != nil {
				return fmt.Errorf("failed to extract event ID from URN: %w", err)
			}
			query := `
				UPDATE markets 
				SET status = $1, updated_at = NOW()
				WHERE event_id = $2
			`
			result, err := p.db.Exec(query, targetStatus, eventID)
		if err != nil {
			return fmt.Errorf("failed to update all markets: %w", err)
		}
>>>>>>> 20d4911 (Fix: bet_stop_processor.go event_id type mismatch and add ExtractEventIDFromURN)

		rowsAffected, _ := result.RowsAffected()
		p.logger.Printf("[bet_stop] 比赛 %s 的所有市场已暂停 (%d个市场)",
			betStop.EventID, rowsAffected)

	} else {
		// 更新特定市场组
		// groups 格式: "1|2|3" 表示 market group IDs
		// 注意: 这需要 market_descriptions 中的 group 信息
		
		// 简化实现: 如果 groups 不是 "all",暂时也更新所有市场
		// TODO: 实现基于 market group 的筛选
<<<<<<< HEAD
		query := `
				UPDATE markets 
				SET status = $1, updated_at = NOW()
				WHERE event_id::text = $2
		`
			p.logger.Printf("[DEBUG] SQL Query: %s", query)
			p.logger.Printf("[DEBUG] SQL Args: %v", []interface{}{targetStatus, betStop.EventID})
			
			result, err := p.db.Exec(query, targetStatus, betStop.EventID)
			if err != nil {
				return fmt.Errorf("failed to update markets by groups: %w", err)
			}
=======
			eventID, err := ExtractEventIDFromURN(betStop.EventID)
			if err != nil {
				return fmt.Errorf("failed to extract event ID from URN: %w", err)
			}
			query := `
				UPDATE markets 
				SET status = $1, updated_at = NOW()
				WHERE event_id = $2
			`
			result, err := p.db.Exec(query, targetStatus, eventID)
		if err != nil {
			return fmt.Errorf("failed to update markets by groups: %w", err)
		}
>>>>>>> 20d4911 (Fix: bet_stop_processor.go event_id type mismatch and add ExtractEventIDFromURN)

		rowsAffected, _ := result.RowsAffected()
		p.logger.Printf("[bet_stop] 比赛 %s 的市场组 %s 已暂停 (%d个市场)",
			betStop.EventID, betStop.Groups, rowsAffected)
	}

	return nil
}

