package ingestion

import (
	"context"
	"uof-service/pkg/models"
)

// DataSource 数据源接口
type DataSource interface {
	// Connect 连接到数据源
	Connect(ctx context.Context) error
	
	// Disconnect 断开连接
	Disconnect() error
	
	// IsConnected 检查连接状态
	IsConnected() bool
	
	// GetName 获取数据源名称
	GetName() string
	
	// GetType 获取数据源类型
	GetType() SourceType
}

// EventDataSource 事件数据源接口
type EventDataSource interface {
	DataSource
	
	// Subscribe 订阅事件
	Subscribe(ctx context.Context, filter EventFilter) error
	
	// Unsubscribe 取消订阅
	Unsubscribe(ctx context.Context, filter EventFilter) error
	
	// GetEventChannel 获取事件通道
	GetEventChannel() <-chan *models.Event
}

// MatchDataSource 比赛数据源接口
type MatchDataSource interface {
	DataSource
	
	// GetMatches 获取比赛列表
	GetMatches(ctx context.Context, filter MatchFilter) ([]*models.Match, error)
	
	// GetMatch 获取单个比赛
	GetMatch(ctx context.Context, matchID string) (*models.Match, error)
	
	// SubscribeMatch 订阅比赛
	SubscribeMatch(ctx context.Context, matchID string) error
	
	// UnsubscribeMatch 取消订阅比赛
	UnsubscribeMatch(ctx context.Context, matchID string) error
}

// OddsDataSource 赔率数据源接口
type OddsDataSource interface {
	DataSource
	
	// GetOdds 获取赔率
	GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error)
	
	// SubscribeOdds 订阅赔率变化
	SubscribeOdds(ctx context.Context, matchID string) error
	
	// GetOddsChannel 获取赔率变化通道
	GetOddsChannel() <-chan *models.Odds
}

// RecoveryManager 数据恢复管理器接口
type RecoveryManager interface {
	// TriggerRecovery 触发数据恢复
	TriggerRecovery(ctx context.Context, req RecoveryRequest) error
	
	// GetRecoveryStatus 获取恢复状态
	GetRecoveryStatus(ctx context.Context) (*RecoveryStatus, error)
}

// ConnectionManager 连接管理器接口
type ConnectionManager interface {
	// RegisterDataSource 注册数据源
	RegisterDataSource(source DataSource) error
	
	// UnregisterDataSource 注销数据源
	UnregisterDataSource(name string) error
	
	// GetDataSource 获取数据源
	GetDataSource(name string) (DataSource, error)
	
	// GetAllDataSources 获取所有数据源
	GetAllDataSources() []DataSource
	
	// ConnectAll 连接所有数据源
	ConnectAll(ctx context.Context) error
	
	// DisconnectAll 断开所有数据源
	DisconnectAll() error
}

// SourceType 数据源类型
type SourceType string

const (
	SourceTypeUOF        SourceType = "uof"
	SourceTypeLiveData   SourceType = "livedata"
	SourceTypeTheSports  SourceType = "thesports"
	SourceTypeMTS        SourceType = "mts"
)

// EventFilter 事件过滤器
type EventFilter struct {
	EventTypes []string
	MatchIDs   []string
	SportIDs   []string
}

// MatchFilter 比赛过滤器
type MatchFilter struct {
	SportIDs   []string
	Status     []string
	StartDate  *string
	EndDate    *string
	Live       *bool
}

// RecoveryRequest 恢复请求
type RecoveryRequest struct {
	Source      string
	AfterHours  int
	Products    []string
	EventID     *string
}

// RecoveryStatus 恢复状态
type RecoveryStatus struct {
	InProgress bool
	StartedAt  string
	Progress   int
	Message    string
}

