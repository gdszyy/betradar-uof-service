package services

import (
	"database/sql"
	"fmt"
)

type MarketQueryService struct {
	db *sql.DB
}

func NewMarketQueryService(db *sql.DB) *MarketQueryService {
	return &MarketQueryService{db: db}
}

type MarketInfo struct {
	MarketID    string        `json:"sr_market_id"`
	Specifiers  string        `json:"specifiers"`
	Status      int           `json:"status"`
	MarketName  string        `json:"market_name"`
	Outcomes    []OutcomeInfo `json:"outcomes"`
}

type OutcomeInfo struct {
	OutcomeID   string  `json:"outcome_id"`
	OutcomeName string  `json:"outcome_name"`
	Odds        float64 `json:"odds"`
	Active      bool    `json:"active"`
}

func (s *MarketQueryService) GetEventMarkets(eventID string) ([]MarketInfo, error) {
	// 查询所有市场
	marketsQuery := `
		SELECT 
			m.sr_market_id,
			COALESCE(m.specifiers, '') as specifiers,
			m.status,
			COALESCE(m.market_name, '') as market_name
		FROM markets m
		WHERE m.event_id = $1
		ORDER BY m.sr_market_id, m.specifiers
	`

	rows, err := s.db.Query(marketsQuery, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query markets: %w", err)
	}
	defer rows.Close()

	var markets []MarketInfo
	for rows.Next() {
		var market MarketInfo
		if err := rows.Scan(&market.MarketID, &market.Specifiers, &market.Status, &market.MarketName); err != nil {
			return nil, fmt.Errorf("failed to scan market: %w", err)
		}

		// 查询该市场的所有 outcomes
		outcomesQuery := `
			SELECT 
				o.outcome_id,
				COALESCE(o.outcome_name, '') as outcome_name,
				o.odds_value,
				o.active
			FROM odds o
			JOIN markets m ON o.market_id = m.id
			WHERE m.event_id = $1 
				AND m.sr_market_id = $2 
				AND m.specifiers = $3
			ORDER BY o.outcome_id
		`

		outcomeRows, err := s.db.Query(outcomesQuery, eventID, market.MarketID, market.Specifiers)
		if err != nil {
			return nil, fmt.Errorf("failed to query outcomes: %w", err)
		}

		var outcomes []OutcomeInfo
		for outcomeRows.Next() {
			var outcome OutcomeInfo
			if err := outcomeRows.Scan(&outcome.OutcomeID, &outcome.OutcomeName, &outcome.Odds, &outcome.Active); err != nil {
				outcomeRows.Close()
				return nil, fmt.Errorf("failed to scan outcome: %w", err)
			}
			outcomes = append(outcomes, outcome)
		}
		outcomeRows.Close()

		market.Outcomes = outcomes
		markets = append(markets, market)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating markets: %w", err)
	}

	return markets, nil
}

