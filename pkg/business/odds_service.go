package business

import (
	"context"
	"fmt"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
	"uof-service/pkg/processing"
)

// DefaultOddsService 默认赔率管理服务实现
type DefaultOddsService struct {
	logger  common.Logger
	storage processing.DataStorage
}

// NewOddsService 创建赔率管理服务
func NewOddsService(logger common.Logger, storage processing.DataStorage) OddsService {
	return &DefaultOddsService{
		logger:  logger,
		storage: storage,
	}
}

// GetOdds 获取比赛赔率
func (s *DefaultOddsService) GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error) {
	s.logger.Debug("Getting odds for match: %s", matchID)

	odds, err := s.storage.GetOdds(ctx, matchID)
	if err != nil {
		s.logger.Error("Failed to get odds: %v", err)
		return nil, err
	}

	s.logger.Debug("Retrieved %d odds for match: %s", len(odds), matchID)
	return odds, nil
}

// GetOddsByMarket 获取指定市场的赔率
func (s *DefaultOddsService) GetOddsByMarket(ctx context.Context, matchID, marketID string) (*models.Odds, error) {
	s.logger.Debug("Getting odds for match: %s, market: %s", matchID, marketID)

	allOdds, err := s.storage.GetOdds(ctx, matchID)
	if err != nil {
		return nil, err
	}

	for _, odds := range allOdds {
		if odds.MarketID == marketID {
			return odds, nil
		}
	}

	return nil, common.ErrNotFound
}

// UpdateOdds 更新赔率
func (s *DefaultOddsService) UpdateOdds(ctx context.Context, odds *models.Odds) error {
	s.logger.Debug("Updating odds: %s", odds.ID)

	// 设置更新时间
	odds.UpdatedAt = time.Now()

	if err := s.storage.SaveOdds(ctx, odds); err != nil {
		s.logger.Error("Failed to update odds: %v", err)
		return err
	}

	s.logger.Debug("Odds updated successfully: %s", odds.ID)
	return nil
}

// GetActiveMarkets 获取活跃市场
func (s *DefaultOddsService) GetActiveMarkets(ctx context.Context, matchID string) ([]string, error) {
	s.logger.Debug("Getting active markets for match: %s", matchID)

	allOdds, err := s.storage.GetOdds(ctx, matchID)
	if err != nil {
		return nil, err
	}

	// 收集活跃市场
	marketsMap := make(map[string]bool)
	for _, odds := range allOdds {
		if odds.Status == models.OddsStatusActive {
			marketsMap[odds.MarketID] = true
		}
	}

	markets := make([]string, 0, len(marketsMap))
	for market := range marketsMap {
		markets = append(markets, market)
	}

	s.logger.Debug("Found %d active markets for match: %s", len(markets), matchID)
	return markets, nil
}

// GetOddsHistory 获取赔率历史
func (s *DefaultOddsService) GetOddsHistory(ctx context.Context, matchID, marketID string, limit int) ([]*models.Odds, error) {
	s.logger.Debug("Getting odds history for match: %s, market: %s, limit: %d", matchID, marketID, limit)

	allOdds, err := s.storage.GetOdds(ctx, matchID)
	if err != nil {
		return nil, err
	}

	// 过滤指定市场的赔率
	var history []*models.Odds
	for _, odds := range allOdds {
		if odds.MarketID == marketID {
			history = append(history, odds)
			if limit > 0 && len(history) >= limit {
				break
			}
		}
	}

	s.logger.Debug("Retrieved %d odds history records", len(history))
	return history, nil
}

// CompareOdds 比较赔率变化
func (s *DefaultOddsService) CompareOdds(ctx context.Context, oldOdds, newOdds *models.Odds) (*OddsComparison, error) {
	s.logger.Debug("Comparing odds: %s vs %s", oldOdds.ID, newOdds.ID)

	if oldOdds.MarketID != newOdds.MarketID {
		return nil, fmt.Errorf("cannot compare odds from different markets")
	}

	comparison := &OddsComparison{
		MarketID:  oldOdds.MarketID,
		OldOdds:   oldOdds,
		NewOdds:   newOdds,
		Changes:   make([]OutcomeChange, 0),
		Timestamp: time.Now(),
	}

	// 比较每个结果的赔率变化
	oldOutcomes := make(map[string]*models.Outcome)
	for _, outcome := range oldOdds.Outcomes {
		oldOutcomes[outcome.ID] = &outcome
	}

	for _, newOutcome := range newOdds.Outcomes {
		oldOutcome, exists := oldOutcomes[newOutcome.ID]
		if !exists {
			// 新增的结果
			comparison.Changes = append(comparison.Changes, OutcomeChange{
				OutcomeID:   newOutcome.ID,
				OutcomeName: newOutcome.Name,
				OldValue:    0,
				NewValue:    newOutcome.Odds,
				ChangeType:  "added",
			})
			continue
		}

		// 比较赔率变化
		if oldOutcome.Odds != newOutcome.Odds {
			changeType := "increased"
			if newOutcome.Odds < oldOutcome.Odds {
				changeType = "decreased"
			}

			comparison.Changes = append(comparison.Changes, OutcomeChange{
				OutcomeID:   newOutcome.ID,
				OutcomeName: newOutcome.Name,
				OldValue:    oldOutcome.Odds,
				NewValue:    newOutcome.Odds,
				ChangeType:  changeType,
			})
		}
	}

	s.logger.Debug("Found %d odds changes", len(comparison.Changes))
	return comparison, nil
}

// CalculateMargin 计算市场利润率
func (s *DefaultOddsService) CalculateMargin(ctx context.Context, odds *models.Odds) (float64, error) {
	s.logger.Debug("Calculating margin for odds: %s", odds.ID)

	if len(odds.Outcomes) == 0 {
		return 0, fmt.Errorf("no outcomes to calculate margin")
	}

	// 计算隐含概率总和
	totalProbability := 0.0
	for _, outcome := range odds.Outcomes {
		if outcome.Odds > 0 {
			totalProbability += 1.0 / outcome.Odds
		}
	}

	// 利润率 = (隐含概率总和 - 1) * 100%
	margin := (totalProbability - 1.0) * 100.0

	s.logger.Debug("Calculated margin: %.2f%%", margin)
	return margin, nil
}

// OddsComparison 赔率比较结果
type OddsComparison struct {
	MarketID  string
	OldOdds   *models.Odds
	NewOdds   *models.Odds
	Changes   []OutcomeChange
	Timestamp time.Time
}

// OutcomeChange 结果变化
type OutcomeChange struct {
	OutcomeID   string
	OutcomeName string
	OldValue    float64
	NewValue    float64
	ChangeType  string // "increased", "decreased", "added", "removed"
}

