package processing

import (
	"context"
	"uof-service/pkg/models"
)

// DataProcessor 数据处理器接口
type DataProcessor interface {
	// Process 处理数据
	Process(ctx context.Context, event *models.Event) error
	
	// GetName 获取处理器名称
	GetName() string
}

// DataValidator 数据验证器接口
type DataValidator interface {
	// Validate 验证数据
	Validate(ctx context.Context, event *models.Event) error
	
	// GetName 获取验证器名称
	GetName() string
}

// DataStorage 数据存储接口
type DataStorage interface {
	// SaveEvent 保存事件
	SaveEvent(ctx context.Context, event *models.Event) error
	
	// SaveMatch 保存比赛
	SaveMatch(ctx context.Context, match *models.Match) error
	
	// SaveOdds 保存赔率
	SaveOdds(ctx context.Context, odds *models.Odds) error
	
	// GetEvent 获取事件
	GetEvent(ctx context.Context, eventID string) (*models.Event, error)
	
	// GetMatch 获取比赛
	GetMatch(ctx context.Context, matchID string) (*models.Match, error)
	
	// GetOdds 获取赔率
	GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error)
	
	// QueryEvents 查询事件
	QueryEvents(ctx context.Context, filter EventQuery) ([]*models.Event, error)
	
	// QueryMatches 查询比赛
	QueryMatches(ctx context.Context, filter MatchQuery) ([]*models.Match, error)
}

// EventDispatcher 事件分发器接口
type EventDispatcher interface {
	// Dispatch 分发事件
	Dispatch(ctx context.Context, event *models.Event) error
	
	// Subscribe 订阅事件
	Subscribe(eventType string, handler EventHandler) error
	
	// Unsubscribe 取消订阅
	Unsubscribe(eventType string, handler EventHandler) error
}

// EventHandler 事件处理器
type EventHandler func(ctx context.Context, event *models.Event) error

// ProcessingPipeline 处理管道接口
type ProcessingPipeline interface {
	// AddProcessor 添加处理器
	AddProcessor(processor DataProcessor) error
	
	// AddValidator 添加验证器
	AddValidator(validator DataValidator) error
	
	// Process 处理数据
	Process(ctx context.Context, event *models.Event) error
}

// EventQuery 事件查询
type EventQuery struct {
	EventTypes []string
	MatchIDs   []string
	SportIDs   []string
	Sources    []string
	StartTime  *string
	EndTime    *string
	Limit      int
	Offset     int
}

// MatchQuery 比赛查询
type MatchQuery struct {
	SportIDs  []string
	Status    []string
	Sources   []string
	StartDate *string
	EndDate   *string
	Limit     int
	Offset    int
}

