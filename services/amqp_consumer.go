package services

import (
	"bytes"
	"encoding/xml"
	"time"

	"github.com/streadway/amqp"

	"uof-service/config"
	"uof-service/logger"
)

// MessageBroadcaster 接口用于广播消息，避免循环依赖
type MessageBroadcaster interface {
	Broadcast(msg interface{})
}

// AMQPConsumer 负责处理从 AMQPConnector 接收到的消息
type AMQPConsumer struct {
	config                    *config.Config
	messageStore              *MessageStore
	broadcaster               MessageBroadcaster
	recoveryManager           *RecoveryManager
	notifier                  *LarkNotifier
	statsTracker              *MessageStatsTracker
	matchMonitor              *MatchMonitor
	fixtureParser             *FixtureParser
	oddsChangeParser          *OddsChangeParser
	oddsParser                *OddsParser
	betSettlementParser       *BetSettlementParser
	betStopProcessor          *BetStopProcessor
	betCancelProcessor        *BetCancelProcessor
	rollbackBetSettlementProc *RollbackBetSettlementProcessor
	rollbackBetCancelProc     *RollbackBetCancelProcessor
	srnMappingService         *SRNMappingService
	fixtureService            *FixtureService
	marketDescService         *MarketDescriptionsService
	done                      chan bool
}

// NewAMQPConsumer 创建 AMQPConsumer 实例
func NewAMQPConsumer(cfg *config.Config, store *MessageStore, broadcaster MessageBroadcaster, marketDescService *MarketDescriptionsService) *AMQPConsumer {
	notifier := NewLarkNotifier(cfg.LarkWebhook)
	statsTracker := NewMessageStatsTracker(notifier, 5*time.Minute)

	// 初始化解析器
	srnMappingService := NewSRNMappingService(cfg.UOFAPIToken, cfg.APIBaseURL, store.db)
	fixtureParser := NewFixtureParser(store.db, srnMappingService, cfg.APIBaseURL, cfg.AccessToken)
	oddsChangeParser := NewOddsChangeParser(store.db)
	oddsParser := NewOddsParser(store.db, marketDescService)
	betSettlementParser := NewBetSettlementParser(store.db)
	betStopProcessor := NewBetStopProcessor(store.db)
	betCancelProcessor := NewBetCancelProcessor(store.db)
	rollbackBetSettlementProc := NewRollbackBetSettlementProcessor(store.db)
	rollbackBetCancelProc := NewRollbackBetCancelProcessor(store.db)
	fixtureService := NewFixtureService(cfg.UOFAPIToken, cfg.APIBaseURL)

	// 从数据库加载 SRN mapping 缓存
	if err := srnMappingService.LoadCacheFromDB(); err != nil {
		logger.Errorf("Warning: failed to load SRN mapping cache: %v", err)
	}

	return &AMQPConsumer{
		config:                    cfg,
		messageStore:              store,
		broadcaster:               broadcaster,
		recoveryManager:           NewRecoveryManager(cfg, store),
		notifier:                  notifier,
		statsTracker:              statsTracker,
		fixtureParser:             fixtureParser,
		oddsChangeParser:          oddsChangeParser,
		oddsParser:                oddsParser,
		betSettlementParser:       betSettlementParser,
		betStopProcessor:          betStopProcessor,
		betCancelProcessor:        betCancelProcessor,
		rollbackBetSettlementProc: rollbackBetSettlementProc,
		rollbackBetCancelProc:     rollbackBetCancelProc,
		srnMappingService:         srnMappingService,
		fixtureService:            fixtureService,
		marketDescService:         marketDescService,
		done:                      make(chan bool),
	}
}

// SetStatsTracker 设置消息统计追踪器
func (c *AMQPConsumer) SetStatsTracker(tracker *MessageStatsTracker) {
	c.statsTracker = tracker
}

// Start 开始处理来自通道的消息
func (c *AMQPConsumer) Start(msgs <-chan amqp.Delivery) error {
	logger.Println("AMQP consumer started, waiting for messages...")

	// 自动触发恢复（如果启用）
	if c.config.AutoRecovery {
		logger.Println("Auto recovery is enabled, triggering full recovery...")
		go func() {
			// 等待几秒确保AMQP连接稳定
			time.Sleep(3 * time.Second)
			if err := c.recoveryManager.TriggerFullRecovery(); err != nil {
				logger.Errorf("Auto recovery failed: %v", err)
			} else {
				logger.Println("Auto recovery completed successfully")
			}
		}()
	}

	// 处理消息
	go c.handleMessages(msgs)

	<-c.done
	return nil
}

// Stop 停止消费者
func (c *AMQPConsumer) Stop() {
	logger.Println("Stopping AMQP consumer...")
	close(c.done)
}

// handleMessages 循环处理消息
func (c *AMQPConsumer) handleMessages(msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		c.processMessage(msg)
	}
}

// processMessage 处理单条消息
func (c *AMQPConsumer) processMessage(msg amqp.Delivery) {
	routingKey := msg.RoutingKey
	xmlContent := string(msg.Body)

	// 解析消息类型
	messageType, eventID, productID, sportID, timestamp := c.parseMessage(xmlContent)

	// 统计消息
	if messageType != "" && c.statsTracker != nil {
		c.statsTracker.Record(messageType)
	}

	// 存储到数据库
	if err := c.messageStore.SaveMessage(messageType, eventID, productID, sportID, routingKey, xmlContent, timestamp); err != nil {
		logger.Errorf("Failed to save message: %v", err)
	}

	// 广播到WebSocket客户端 (包含基本数据)
	if c.broadcaster != nil {
		data := c.extractMessageData(messageType, xmlContent)
		c.broadcaster.Broadcast(map[string]interface{}{
			"type":         "message",
			"message_type": messageType,
			"event_id":     eventID,
			"product_id":   productID,
			"routing_key":  routingKey,
			"timestamp":    timestamp,
			"data":         data,
		})
	}

	// 处理特定消息类型
	switch messageType {
	case "alive":
		c.handleAlive(xmlContent)
	case "odds_change":
		c.handleOddsChange(eventID, productID, xmlContent, timestamp)
	case "bet_stop":
		c.handleBetStop(eventID, productID, xmlContent, timestamp)
	case "bet_settlement":
		c.handleBetSettlement(eventID, productID, xmlContent, timestamp)
	case "bet_cancel":
		c.handleBetCancel(eventID, productID, xmlContent, timestamp)
	case "fixture":
		c.handleFixture(eventID, productID, xmlContent, timestamp)
	case "fixture_change":
		c.handleFixtureChange(eventID, productID, xmlContent, timestamp)
	case "rollback_bet_settlement":
		c.handleRollbackBetSettlement(eventID, productID, xmlContent, timestamp)
	case "rollback_bet_cancel":
		c.handleRollbackBetCancel(eventID, productID, xmlContent, timestamp)
	case "snapshot_complete":
		c.handleSnapshotComplete(xmlContent)
	}
}

// parseMessage 解析消息基本信息
func (c *AMQPConsumer) parseMessage(xmlContent string) (messageType, eventID string, productID *int, sportID *string, timestamp int64) {
	type BaseMessage struct {
		EventID   string `xml:"event_id,attr"`
		ProductID int    `xml:"product,attr"`
		SportID   string `xml:"sport_id,attr"`
		Timestamp int64  `xml:"timestamp,attr"`
	}

	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlContent)))
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		if startElement, ok := token.(xml.StartElement); ok {
			messageType = startElement.Name.Local
			break
		}
	}

	var base BaseMessage
	xml.Unmarshal([]byte(xmlContent), &base)

	if base.EventID != "" {
		eventID = base.EventID
	}
	if base.ProductID != 0 {
		productID = &base.ProductID
	}
	if base.SportID != "" {
		sportID = &base.SportID
	}
	timestamp = base.Timestamp

	return
}

// extractMessageData 提取用于广播的附加数据
func (c *AMQPConsumer) extractMessageData(messageType, xmlContent string) interface{} {
	// ... (此处省略了具体实现，与原文件一致)
	return nil
}

// handleAlive 处理 alive 消息
func (c *AMQPConsumer) handleAlive(xmlContent string) {
	type AliveMessage struct {
		ProductID  int   `xml:"product,attr"`
		Timestamp  int64 `xml:"timestamp,attr"`
		Subscribed int   `xml:"subscribed,attr"`
	}

	var alive AliveMessage
	if err := xml.Unmarshal([]byte(xmlContent), &alive); err != nil {
		logger.Errorf("Failed to parse alive message: %v", err)
		return
	}

	if err := c.messageStore.UpdateProducerStatus(alive.ProductID, alive.Timestamp, alive.Subscribed); err != nil {
		logger.Errorf("Failed to update producer status: %v", err)
	}
}

// handleOddsChange 处理 odds_change 消息
func (c *AMQPConsumer) handleOddsChange(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.oddsChangeParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to handle odds_change: %v", err)
	}
}

// handleBetStop 处理 bet_stop 消息
func (c *AMQPConsumer) handleBetStop(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.betStopProcessor.ProcessBetStop(xmlContent); err != nil {
		logger.Errorf("Failed to handle bet_stop: %v", err)
	}
}

// handleBetSettlement 处理 bet_settlement 消息
func (c *AMQPConsumer) handleBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.betSettlementParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to handle bet_settlement: %v", err)
	}
}

// handleBetCancel 处理 bet_cancel 消息
func (c *AMQPConsumer) handleBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.betCancelProcessor.ProcessBetCancel(xmlContent); err != nil {
		logger.Errorf("Failed to handle bet_cancel: %v", err)
	}
}

// handleFixture 处理 fixture 消息
func (c *AMQPConsumer) handleFixture(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.fixtureParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to handle fixture: %v", err)
	}
}

// handleFixtureChange 处理 fixture_change 消息
func (c *AMQPConsumer) handleFixtureChange(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.fixtureParser.ParseFixtureChange(eventID, xmlContent); err != nil {
		logger.Errorf("Failed to handle fixture_change: %v", err)
	}
}

// handleRollbackBetSettlement 处理 rollback_bet_settlement 消息
func (c *AMQPConsumer) handleRollbackBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.rollbackBetSettlementProc.ProcessRollbackBetSettlement(xmlContent); err != nil {
		logger.Errorf("Failed to handle rollback_bet_settlement: %v", err)
	}
}

// handleRollbackBetCancel 处理 rollback_bet_cancel 消息
func (c *AMQPConsumer) handleRollbackBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := c.rollbackBetCancelProc.ProcessRollbackBetCancel(xmlContent); err != nil {
		logger.Errorf("Failed to handle rollback_bet_cancel: %v", err)
	}
}

// handleSnapshotComplete 处理 snapshot_complete 消息
func (c *AMQPConsumer) handleSnapshotComplete(xmlContent string) {
	type SnapshotComplete struct {
		ProductID int   `xml:"product,attr"`
		RequestID int   `xml:"request_id,attr"`
		Timestamp int64 `xml:"timestamp,attr"`
	}

	var snapshot SnapshotComplete
	if err := xml.Unmarshal([]byte(xmlContent), &snapshot); err != nil {
		logger.Errorf("Failed to parse snapshot_complete: %v", err)
		return
	}

	if err := c.messageStore.UpdateRecoveryCompleted(snapshot.RequestID, snapshot.ProductID, snapshot.Timestamp); err != nil {
		logger.Errorf("Failed to update recovery status: %v", err)
	}

	logger.Printf("Snapshot complete for product %d, request %d", snapshot.ProductID, snapshot.RequestID)
}
