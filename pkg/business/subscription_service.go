package business

import (
	"context"
	"fmt"
	"sync"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/ingestion"
)

// DefaultSubscriptionService 默认订阅管理服务实现
type DefaultSubscriptionService struct {
	logger         common.Logger
	dataSource     ingestion.DataSource
	subscriptions  map[string]*SubscriptionInfo
	mu             sync.RWMutex
}

// NewSubscriptionService 创建订阅管理服务
func NewSubscriptionService(logger common.Logger, dataSource ingestion.DataSource) SubscriptionService {
	return &DefaultSubscriptionService{
		logger:        logger,
		dataSource:    dataSource,
		subscriptions: make(map[string]*SubscriptionInfo),
	}
}

// Subscribe 订阅比赛
func (s *DefaultSubscriptionService) Subscribe(ctx context.Context, matchID string) error {
	s.logger.Info("Subscribing to match: %s", matchID)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已订阅
	if _, exists := s.subscriptions[matchID]; exists {
		s.logger.Warn("Match already subscribed: %s", matchID)
		return nil
	}

	// 订阅数据源
	if err := s.dataSource.Subscribe(ctx, matchID); err != nil {
		s.logger.Error("Failed to subscribe to match: %v", err)
		return err
	}

	// 记录订阅信息
	s.subscriptions[matchID] = &SubscriptionInfo{
		MatchID:       matchID,
		SubscribedAt:  time.Now(),
		EventCount:    0,
		LastEventTime: time.Time{},
		Status:        "active",
	}

	s.logger.Info("Successfully subscribed to match: %s", matchID)
	return nil
}

// Unsubscribe 取消订阅
func (s *DefaultSubscriptionService) Unsubscribe(ctx context.Context, matchID string) error {
	s.logger.Info("Unsubscribing from match: %s", matchID)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已订阅
	if _, exists := s.subscriptions[matchID]; !exists {
		s.logger.Warn("Match not subscribed: %s", matchID)
		return nil
	}

	// 取消数据源订阅
	if err := s.dataSource.Unsubscribe(ctx, matchID); err != nil {
		s.logger.Error("Failed to unsubscribe from match: %v", err)
		return err
	}

	// 删除订阅信息
	delete(s.subscriptions, matchID)

	s.logger.Info("Successfully unsubscribed from match: %s", matchID)
	return nil
}

// GetSubscriptions 获取所有订阅
func (s *DefaultSubscriptionService) GetSubscriptions(ctx context.Context) ([]*SubscriptionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	subscriptions := make([]*SubscriptionInfo, 0, len(s.subscriptions))
	for _, sub := range s.subscriptions {
		subscriptions = append(subscriptions, sub)
	}

	return subscriptions, nil
}

// GetSubscription 获取订阅信息
func (s *DefaultSubscriptionService) GetSubscription(ctx context.Context, matchID string) (*SubscriptionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sub, exists := s.subscriptions[matchID]
	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", matchID)
	}

	return sub, nil
}

// UpdateSubscription 更新订阅信息
func (s *DefaultSubscriptionService) UpdateSubscription(ctx context.Context, matchID string, eventCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, exists := s.subscriptions[matchID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", matchID)
	}

	sub.EventCount = eventCount
	sub.LastEventTime = time.Now()

	return nil
}

// CleanupInactiveSubscriptions 清理不活跃的订阅
func (s *DefaultSubscriptionService) CleanupInactiveSubscriptions(ctx context.Context, inactiveDuration time.Duration) error {
	s.logger.Info("Cleaning up inactive subscriptions (inactive > %v)", inactiveDuration)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	inactiveMatches := make([]string, 0)

	for matchID, sub := range s.subscriptions {
		// 检查是否长时间没有事件
		if !sub.LastEventTime.IsZero() && now.Sub(sub.LastEventTime) > inactiveDuration {
			inactiveMatches = append(inactiveMatches, matchID)
		}
	}

	// 取消不活跃的订阅
	for _, matchID := range inactiveMatches {
		if err := s.dataSource.Unsubscribe(ctx, matchID); err != nil {
			s.logger.Error("Failed to unsubscribe inactive match: %v", err)
			continue
		}

		delete(s.subscriptions, matchID)
		s.logger.Info("Cleaned up inactive subscription: %s", matchID)
	}

	s.logger.Info("Cleaned up %d inactive subscriptions", len(inactiveMatches))
	return nil
}

// GetSubscriptionStats 获取订阅统计
func (s *DefaultSubscriptionService) GetSubscriptionStats(ctx context.Context) (*SubscriptionStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &SubscriptionStats{
		TotalSubscriptions: len(s.subscriptions),
		ActiveSubscriptions: 0,
		TotalEvents:        0,
	}

	now := time.Now()
	for _, sub := range s.subscriptions {
		stats.TotalEvents += sub.EventCount

		// 统计活跃订阅 (最近 1 小时有事件)
		if !sub.LastEventTime.IsZero() && now.Sub(sub.LastEventTime) < time.Hour {
			stats.ActiveSubscriptions++
		}
	}

	return stats, nil
}

// SubscriptionInfo 订阅信息
type SubscriptionInfo struct {
	MatchID       string
	SubscribedAt  time.Time
	EventCount    int
	LastEventTime time.Time
	Status        string
}

// SubscriptionStats 订阅统计
type SubscriptionStats struct {
	TotalSubscriptions  int
	ActiveSubscriptions int
	TotalEvents         int
}

