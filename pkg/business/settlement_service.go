package business

import (
	"context"
	"fmt"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
	"uof-service/pkg/processing"
)

// DefaultSettlementService 默认结算管理服务实现
type DefaultSettlementService struct {
	logger  common.Logger
	storage processing.DataStorage
}

// NewSettlementService 创建结算管理服务
func NewSettlementService(logger common.Logger, storage processing.DataStorage) SettlementService {
	return &DefaultSettlementService{
		logger:  logger,
		storage: storage,
	}
}

// ProcessSettlement 处理结算
func (s *DefaultSettlementService) ProcessSettlement(ctx context.Context, settlement *Settlement) error {
	s.logger.Info("Processing settlement for match: %s, market: %s", settlement.MatchID, settlement.MarketID)

	// 验证结算数据
	if err := s.validateSettlement(settlement); err != nil {
		s.logger.Error("Settlement validation failed: %v", err)
		return err
	}

	// 保存结算事件
	event := &models.Event{
		ID:        fmt.Sprintf("settlement_%s_%s_%d", settlement.MatchID, settlement.MarketID, time.Now().Unix()),
		Type:      models.EventTypeSettlement,
		Source:    settlement.Source,
		Timestamp: settlement.Timestamp,
		MatchID:   settlement.MatchID,
		Data: map[string]interface{}{
			"market_id":      settlement.MarketID,
			"winning_outcome": settlement.WinningOutcome,
			"status":         settlement.Status,
		},
		ProcessedAt: time.Now(),
	}

	if err := s.storage.SaveEvent(ctx, event); err != nil {
		s.logger.Error("Failed to save settlement event: %v", err)
		return err
	}

	s.logger.Info("Settlement processed successfully for match: %s", settlement.MatchID)
	return nil
}

// GetSettlements 获取结算记录
func (s *DefaultSettlementService) GetSettlements(ctx context.Context, matchID string) ([]*Settlement, error) {
	s.logger.Debug("Getting settlements for match: %s", matchID)

	filter := processing.EventQueryFilter{
		MatchID:   matchID,
		EventType: models.EventTypeSettlement,
		Limit:     100,
	}

	events, err := s.storage.QueryEvents(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get settlements: %v", err)
		return nil, err
	}

	settlements := make([]*Settlement, 0, len(events))
	for _, event := range events {
		settlement := s.eventToSettlement(event)
		if settlement != nil {
			settlements = append(settlements, settlement)
		}
	}

	s.logger.Debug("Retrieved %d settlements for match: %s", len(settlements), matchID)
	return settlements, nil
}

// GetSettlementByMarket 获取指定市场的结算
func (s *DefaultSettlementService) GetSettlementByMarket(ctx context.Context, matchID, marketID string) (*Settlement, error) {
	s.logger.Debug("Getting settlement for match: %s, market: %s", matchID, marketID)

	settlements, err := s.GetSettlements(ctx, matchID)
	if err != nil {
		return nil, err
	}

	for _, settlement := range settlements {
		if settlement.MarketID == marketID {
			return settlement, nil
		}
	}

	return nil, common.ErrNotFound
}

// CancelSettlement 取消结算
func (s *DefaultSettlementService) CancelSettlement(ctx context.Context, matchID, marketID string, reason string) error {
	s.logger.Info("Canceling settlement for match: %s, market: %s, reason: %s", matchID, marketID, reason)

	// 保存取消结算事件
	event := &models.Event{
		ID:        fmt.Sprintf("cancel_settlement_%s_%s_%d", matchID, marketID, time.Now().Unix()),
		Type:      models.EventTypeBetCancel,
		Source:    models.EventSourceUOF,
		Timestamp: time.Now(),
		MatchID:   matchID,
		Data: map[string]interface{}{
			"market_id": marketID,
			"reason":    reason,
		},
		ProcessedAt: time.Now(),
	}

	if err := s.storage.SaveEvent(ctx, event); err != nil {
		s.logger.Error("Failed to save cancel settlement event: %v", err)
		return err
	}

	s.logger.Info("Settlement canceled successfully for match: %s, market: %s", matchID, marketID)
	return nil
}

// GetPendingSettlements 获取待结算的市场
func (s *DefaultSettlementService) GetPendingSettlements(ctx context.Context) ([]*PendingSettlement, error) {
	s.logger.Debug("Getting pending settlements")

	// 获取所有已结束的比赛
	filter := processing.MatchQueryFilter{
		Status: models.MatchStatusEnded,
		Limit:  100,
	}

	matches, err := s.storage.QueryMatches(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to get ended matches: %v", err)
		return nil, err
	}

	pendingSettlements := make([]*PendingSettlement, 0)

	for _, match := range matches {
		// 获取比赛的所有赔率市场
		odds, err := s.storage.GetOdds(ctx, match.ID)
		if err != nil {
			continue
		}

		// 获取已结算的市场
		settlements, err := s.GetSettlements(ctx, match.ID)
		if err != nil {
			continue
		}

		settledMarkets := make(map[string]bool)
		for _, settlement := range settlements {
			settledMarkets[settlement.MarketID] = true
		}

		// 找出未结算的市场
		for _, odd := range odds {
			if !settledMarkets[odd.MarketID] {
				pendingSettlements = append(pendingSettlements, &PendingSettlement{
					MatchID:   match.ID,
					MarketID:  odd.MarketID,
					StartTime: match.StartTime,
					EndTime:   match.UpdatedAt,
				})
			}
		}
	}

	s.logger.Debug("Found %d pending settlements", len(pendingSettlements))
	return pendingSettlements, nil
}

// validateSettlement 验证结算数据
func (s *DefaultSettlementService) validateSettlement(settlement *Settlement) error {
	if settlement.MatchID == "" {
		return fmt.Errorf("match ID is required")
	}

	if settlement.MarketID == "" {
		return fmt.Errorf("market ID is required")
	}

	if settlement.WinningOutcome == "" {
		return fmt.Errorf("winning outcome is required")
	}

	if settlement.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	return nil
}

// eventToSettlement 将事件转换为结算
func (s *DefaultSettlementService) eventToSettlement(event *models.Event) *Settlement {
	if event.Type != models.EventTypeSettlement {
		return nil
	}

	data, ok := event.Data.(map[string]interface{})
	if !ok {
		return nil
	}

	settlement := &Settlement{
		MatchID:   event.MatchID,
		Source:    event.Source,
		Timestamp: event.Timestamp,
	}

	if marketID, ok := data["market_id"].(string); ok {
		settlement.MarketID = marketID
	}

	if winningOutcome, ok := data["winning_outcome"].(string); ok {
		settlement.WinningOutcome = winningOutcome
	}

	if status, ok := data["status"].(string); ok {
		settlement.Status = status
	}

	return settlement
}

// Settlement 结算信息
type Settlement struct {
	MatchID        string
	MarketID       string
	WinningOutcome string
	Status         string
	Source         string
	Timestamp      time.Time
}

// PendingSettlement 待结算信息
type PendingSettlement struct {
	MatchID   string
	MarketID  string
	StartTime time.Time
	EndTime   time.Time
}

