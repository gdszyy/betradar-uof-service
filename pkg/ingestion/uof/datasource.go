package uof

import (
	"context"
	"fmt"
	"sync"
	"time"

	"uof-service/config"
	"uof-service/pkg/common"
	"uof-service/pkg/ingestion"
	"uof-service/pkg/models"
	"uof-service/services"
)

// UOFDataSource UOF 数据源实现
type UOFDataSource struct {
	config       *config.Config
	logger       common.Logger
	consumer     *services.AMQPConsumer
	eventChan    chan *models.Event
	connected    bool
	mu           sync.RWMutex
}

// NewUOFDataSource 创建 UOF 数据源
func NewUOFDataSource(cfg *config.Config, logger common.Logger) *UOFDataSource {
	return &UOFDataSource{
		config:    cfg,
		logger:    logger,
		eventChan: make(chan *models.Event, 1000),
		connected: false,
	}
}

// Connect 连接到 UOF
func (ds *UOFDataSource) Connect(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.connected {
		return common.ErrAlreadyConnected
	}

	ds.logger.Info("Connecting to UOF data source...")

	// 创建消息存储
	messageStore := services.NewMessageStore()

	// 创建广播器（暂时使用空实现）
	broadcaster := &emptyBroadcaster{}

	// 创建 AMQP 消费者
	ds.consumer = services.NewAMQPConsumer(ds.config, messageStore, broadcaster)

	// 启动消费者
	go func() {
		if err := ds.consumer.Start(); err != nil {
			ds.logger.Error("AMQP consumer error: %v", err)
		}
	}()

	// 等待连接建立
	time.Sleep(2 * time.Second)

	ds.connected = true
	ds.logger.Info("Connected to UOF data source successfully")

	return nil
}

// Disconnect 断开连接
func (ds *UOFDataSource) Disconnect() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if !ds.connected {
		return common.ErrNotConnected
	}

	ds.logger.Info("Disconnecting from UOF data source...")

	if ds.consumer != nil {
		ds.consumer.Stop()
	}

	ds.connected = false
	ds.logger.Info("Disconnected from UOF data source")

	return nil
}

// IsConnected 检查连接状态
func (ds *UOFDataSource) IsConnected() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.connected
}

// GetName 获取数据源名称
func (ds *UOFDataSource) GetName() string {
	return "UOF"
}

// GetType 获取数据源类型
func (ds *UOFDataSource) GetType() ingestion.SourceType {
	return ingestion.SourceTypeUOF
}

// GetEventChannel 获取事件通道
func (ds *UOFDataSource) GetEventChannel() <-chan *models.Event {
	return ds.eventChan
}

// Subscribe 订阅事件
func (ds *UOFDataSource) Subscribe(ctx context.Context, filter ingestion.EventFilter) error {
	if !ds.IsConnected() {
		return common.ErrNotConnected
	}

	ds.logger.Info("Subscribing to events with filter: %+v", filter)
	// UOF 通过 routing keys 自动订阅，这里只是记录
	return nil
}

// Unsubscribe 取消订阅
func (ds *UOFDataSource) Unsubscribe(ctx context.Context, filter ingestion.EventFilter) error {
	if !ds.IsConnected() {
		return common.ErrNotConnected
	}

	ds.logger.Info("Unsubscribing from events with filter: %+v", filter)
	// UOF 取消订阅需要发送特定消息
	return nil
}

// GetMatches 获取比赛列表
func (ds *UOFDataSource) GetMatches(ctx context.Context, filter ingestion.MatchFilter) ([]*models.Match, error) {
	if !ds.IsConnected() {
		return nil, common.ErrNotConnected
	}

	// UOF 不直接提供比赛列表查询，需要通过其他 API
	return nil, fmt.Errorf("GetMatches not implemented for UOF")
}

// GetMatch 获取单个比赛
func (ds *UOFDataSource) GetMatch(ctx context.Context, matchID string) (*models.Match, error) {
	if !ds.IsConnected() {
		return nil, common.ErrNotConnected
	}

	// UOF 不直接提供比赛查询，需要通过其他 API
	return nil, fmt.Errorf("GetMatch not implemented for UOF")
}

// SubscribeMatch 订阅比赛
func (ds *UOFDataSource) SubscribeMatch(ctx context.Context, matchID string) error {
	if !ds.IsConnected() {
		return common.ErrNotConnected
	}

	ds.logger.Info("Subscribing to match: %s", matchID)
	// UOF 订阅比赛需要调用 Booking API
	return nil
}

// UnsubscribeMatch 取消订阅比赛
func (ds *UOFDataSource) UnsubscribeMatch(ctx context.Context, matchID string) error {
	if !ds.IsConnected() {
		return common.ErrNotConnected
	}

	ds.logger.Info("Unsubscribing from match: %s", matchID)
	// UOF 取消订阅需要发送 matchstop 消息
	return nil
}

// GetOdds 获取赔率
func (ds *UOFDataSource) GetOdds(ctx context.Context, matchID string) ([]*models.Odds, error) {
	if !ds.IsConnected() {
		return nil, common.ErrNotConnected
	}

	// UOF 不直接提供赔率查询，赔率通过 AMQP 推送
	return nil, fmt.Errorf("GetOdds not implemented for UOF")
}

// SubscribeOdds 订阅赔率
func (ds *UOFDataSource) SubscribeOdds(ctx context.Context, matchID string) error {
	if !ds.IsConnected() {
		return common.ErrNotConnected
	}

	ds.logger.Info("Subscribing to odds for match: %s", matchID)
	// UOF 赔率订阅通过 routing keys 自动完成
	return nil
}

// UnsubscribeOdds 取消订阅赔率
func (ds *UOFDataSource) UnsubscribeOdds(ctx context.Context, matchID string) error {
	if !ds.IsConnected() {
		return common.ErrNotConnected
	}

	ds.logger.Info("Unsubscribing from odds for match: %s", matchID)
	return nil
}

// emptyBroadcaster 空广播器实现
type emptyBroadcaster struct{}

func (b *emptyBroadcaster) Broadcast(msg interface{}) {
	// 暂时不做任何事情
}

// 确保 UOFDataSource 实现了所有必需的接口
var _ ingestion.DataSource = (*UOFDataSource)(nil)
var _ ingestion.EventDataSource = (*UOFDataSource)(nil)
var _ ingestion.MatchDataSource = (*UOFDataSource)(nil)
var _ ingestion.OddsDataSource = (*UOFDataSource)(nil)

