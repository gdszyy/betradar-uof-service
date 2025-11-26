package web

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// MarketInfo 盘口信息
type MarketInfo struct {
	ID            int64         `json:"id"`
	EventID       string        `json:"event_id"`
	MarketID      string        `json:"market_id"`
	Specifier     string        `json:"specifier,omitempty"`
	Status        string        `json:"status"`
	CashoutStatus string        `json:"cashout_status,omitempty"`
	BetStatus     string        `json:"bet_status,omitempty"`
	Outcomes      []OutcomeInfo `json:"outcomes,omitempty"`
}

// OutcomeInfo 赔率信息
type OutcomeInfo struct {
	ID        int64   `json:"id"`
	MarketID  int64   `json:"market_id"`
	OutcomeID string  `json:"outcome_id"`
	Odds      float64 `json:"odds"`
	Status    string  `json:"status"`
	BetStatus string  `json:"bet_status,omitempty"`
}

// getMarketsForEvent 获取指定赛事和盘口类型的盘口和赔率信息
func (s *Server) getMarketsForEvent(eventID string, marketIDs []string) ([]MarketInfo, error) {
	if len(marketIDs) == 0 {
		return []MarketInfo{}, nil
	}

	// 1. 构建查询
	placeholders := make([]string, len(marketIDs))
	args := make([]interface{}, len(marketIDs)+1)
	args[0] = eventID
	for i, id := range marketIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT 
			m.id, m.event_id, m.sr_market_id, 
			COALESCE(m.specifier, '') as specifier,
			COALESCE(m.status, '') as status,
			COALESCE(m.cashout_status, '') as cashout_status,
			COALESCE(m.bet_status, '') as bet_status,
			COALESCE(o.id, 0) as outcome_id_int,
			COALESCE(o.market_id, 0) as outcome_market_id,
			COALESCE(o.outcome_id, '') as outcome_id_str,
			COALESCE(o.odds, 0) as odds,
			COALESCE(o.status, '') as outcome_status,
			COALESCE(o.bet_status, '') as outcome_bet_status
		FROM markets m
		LEFT JOIN outcomes o ON m.id = o.market_id
		WHERE m.event_id = $1 AND m.sr_market_id IN (%s)
		ORDER BY m.id, o.id
	`, strings.Join(placeholders, ","))

	// 2. 查询数据
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query markets and outcomes for event %s: %w", eventID, err)
	}
	defer rows.Close()

	// 3. 构建嵌套结构
	marketMap := make(map[int64]*MarketInfo)
	var marketOrder []int64 // 保持顺序

	for rows.Next() {
		var marketID int64
		var eventIDStr, srMarketID, specifier, status, cashoutStatus, betStatus string
		var outcomeIDInt, outcomeMarketID int64
		var outcomeIDStr, outcomeStatus, outcomeBetStatus string
		var odds float64

		if err := rows.Scan(
			&marketID, &eventIDStr, &srMarketID, &specifier, &status, &cashoutStatus, &betStatus,
			&outcomeIDInt, &outcomeMarketID, &outcomeIDStr, &odds, &outcomeStatus, &outcomeBetStatus,
		); err != nil {
			log.Printf("[API] Error scanning market and outcome detail for event %s: %v", eventID, err)
			continue
		}

		// 如果 market 不存在，创建新的 MarketInfo
		market, exists := marketMap[marketID]
		if !exists {
			market = &MarketInfo{
				ID:            marketID,
				EventID:       eventIDStr,
				MarketID:      srMarketID,
				Specifier:     specifier,
				Status:        status,
				CashoutStatus: cashoutStatus,
				BetStatus:     betStatus,
				Outcomes:      []OutcomeInfo{},
			}
			marketMap[marketID] = market
			marketOrder = append(marketOrder, marketID)
		}

		// 如果有 outcome 数据，添加到 market
		if outcomeIDInt > 0 && outcomeIDStr != "" {
			outcome := OutcomeInfo{
				ID:        outcomeIDInt,
				MarketID:  outcomeMarketID,
				OutcomeID: outcomeIDStr,
				Odds:      odds,
				Status:    outcomeStatus,
				BetStatus: outcomeBetStatus,
			}
			market.Outcomes = append(market.Outcomes, outcome)
		}
	}

	// 4. 按顺序返回 markets
	markets := make([]MarketInfo, 0, len(marketOrder))
	for _, marketID := range marketOrder {
		if market, ok := marketMap[marketID]; ok {
			markets = append(markets, *market)
		}
	}

	return markets, nil
}
