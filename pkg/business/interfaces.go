package business

import (
	"context"
	"uof-service/pkg/models"
)

// OddsService 赔率管理服务接口
type OddsService interface {
	// GetOdds 获取赔率
	GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error)
	
	// GetOddsHistory 获取赔率历史
	GetOddsHistory(ctx context.Context, matchID string, marketID string) ([]*models.Odds, error)
	
	// SubscribeOdds 订阅赔率变化
	SubscribeOdds(ctx context.Context, matchID string) error
	
	// UnsubscribeOdds 取消订阅赔率
	UnsubscribeOdds(ctx context.Context, matchID string) error
}

// MatchService 赛事管理服务接口
type MatchService interface {
	// GetMatch 获取比赛信息
	GetMatch(ctx context.Context, matchID string) (*models.Match, error)
	
	// GetMatches 获取比赛列表
	GetMatches(ctx context.Context, filter MatchFilter) ([]*models.Match, error)
	
	// GetLiveMatches 获取直播比赛
	GetLiveMatches(ctx context.Context) ([]*models.Match, error)
	
	// GetTodayMatches 获取今日比赛
	GetTodayMatches(ctx context.Context) ([]*models.Match, error)
	
	// SubscribeMatch 订阅比赛
	SubscribeMatch(ctx context.Context, matchID string) error
	
	// UnsubscribeMatch 取消订阅比赛
	UnsubscribeMatch(ctx context.Context, matchID string) error
}

// SettlementService 结算管理服务接口
type SettlementService interface {
	// GetSettlement 获取结算信息
	GetSettlement(ctx context.Context, matchID string) (*Settlement, error)
	
	// ProcessSettlement 处理结算
	ProcessSettlement(ctx context.Context, settlement *Settlement) error
}

// NotificationService 通知管理服务接口
type NotificationService interface {
	// SendNotification 发送通知
	SendNotification(ctx context.Context, notification *Notification) error
	
	// SendAlert 发送告警
	SendAlert(ctx context.Context, alert *Alert) error
	
	// SendMatchEndNotification 发送比赛结束通知
	SendMatchEndNotification(ctx context.Context, match *models.Match) error
}

// SubscriptionService 订阅管理服务接口
type SubscriptionService interface {
	// GetSubscriptions 获取所有订阅
	GetSubscriptions(ctx context.Context) ([]*Subscription, error)
	
	// GetSubscriptionStats 获取订阅统计
	GetSubscriptionStats(ctx context.Context) (*SubscriptionStats, error)
	
	// UnsubscribeMatch 取消订阅比赛
	UnsubscribeMatch(ctx context.Context, matchID string) error
	
	// UnsubscribeMatches 批量取消订阅
	UnsubscribeMatches(ctx context.Context, matchIDs []string) error
	
	// CleanupEndedMatches 清理已结束比赛
	CleanupEndedMatches(ctx context.Context) error
}

// BookingService 预订管理服务接口
type BookingService interface {
	// BookMatch 预订比赛
	BookMatch(ctx context.Context, matchID string) error
	
	// BookAllBookableMatches 预订所有可预订比赛
	BookAllBookableMatches(ctx context.Context) (*BookingResult, error)
	
	// GetBookableMatches 获取可预订比赛
	GetBookableMatches(ctx context.Context) ([]*BookableMatch, error)
}

// MatchFilter 比赛过滤器
type MatchFilter struct {
	SportIDs  []string
	Status    []string
	StartDate *string
	EndDate   *string
	Live      *bool
}

// Settlement 结算信息
type Settlement struct {
	ID        string
	MatchID   string
	MarketID  string
	OutcomeID string
	Status    string
	Timestamp string
}

// Notification 通知
type Notification struct {
	Type    string
	Title   string
	Message string
	Data    map[string]interface{}
}

// Alert 告警
type Alert struct {
	Level   string
	Title   string
	Message string
	Data    map[string]interface{}
}

// Subscription 订阅信息
type Subscription struct {
	MatchID         string
	SubscribedAt    string
	LastEventAt     string
	Status          string
	EventCount      int
	AutoUnsubscribe bool
}

// SubscriptionStats 订阅统计
type SubscriptionStats struct {
	Total  int
	Live   int
	Ended  int
	Closed int
}

// BookingResult 预订结果
type BookingResult struct {
	TotalMatches  int
	BookedMatches int
	FailedMatches int
	Errors        []string
}

// BookableMatch 可预订比赛
type BookableMatch struct {
	MatchID   string
	SportID   string
	HomeTeam  string
	AwayTeam  string
	StartTime string
	Bookable  bool
}

