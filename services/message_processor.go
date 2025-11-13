package services

import (
	"encoding/xml"

	"uof-service/config"
	"uof-service/logger"
)

// MessageProcessor 负责从 Broker 消费特定 Topic 的消息，并执行业务逻辑
type MessageProcessor struct {
	config                    *config.Config
	broker                    MessageBroker
	broadcaster               MessageBroadcaster
	messageStore              *MessageStore
	
	// 业务处理器依赖 (从 AMQPConsumer 迁移过来)
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

// NewMessageProcessor 创建 MessageProcessor 实例
func NewMessageProcessor(cfg *config.Config, store *MessageStore, broker MessageBroker, broadcaster MessageBroadcaster, marketDescService *MarketDescriptionsService) *MessageProcessor {
	// 初始化解析器 (与原 AMQPConsumer 的初始化逻辑一致)
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

	return &MessageProcessor{
		config:                    cfg,
		broker:                    broker,
		broadcaster:               broadcaster,
		messageStore:              store,
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

// StartConsumer 启动一个消费者，订阅指定的 Topic 并开始处理消息
func (p *MessageProcessor) StartConsumer(messageType string) error {
	topic := GetTopicName(messageType)
	msgs, err := p.broker.Consume(topic)
	if err != nil {
		return err
	}

	logger.Printf("MessageProcessor started for topic: %s", topic)

	go p.handleMessages(msgs)

	return nil
}

// handleMessages 循环处理来自 Broker 的消息
func (p *MessageProcessor) handleMessages(msgs <-chan BrokerMessage) {
	for msg := range msgs {
		p.processMessage(msg)
	}
}

// processMessage 处理单条 Broker 消息
func (p *MessageProcessor) processMessage(msg BrokerMessage) {
	xmlContent := string(msg.Value)
	
	// 提取消息类型 (从 Topic 名称中获取)
	messageType := msg.Topic[len("uof-message-"):]

	// 解析消息基本信息
	type BaseMessage struct {
		EventID   string `xml:"event_id,attr"`
		ProductID int    `xml:"product,attr"`
		Timestamp int64  `xml:"timestamp,attr"`
	}

	var base BaseMessage
	xml.Unmarshal(msg.Value, &base)
	
	eventID := base.EventID
	productID := &base.ProductID
	timestamp := base.Timestamp

	// 广播到WebSocket客户端 (从 AMQPConsumer 迁移过来)
	if p.broadcaster != nil {
		data := p.extractMessageData(messageType, xmlContent)
		p.broadcaster.Broadcast(map[string]interface{}{
			"type":         "message",
			"message_type": messageType,
			"event_id":     eventID,
			"product_id":   productID,
			"timestamp":    timestamp,
			"data":         data,
		})
	}

	// 处理特定消息类型 (从 AMQPConsumer 迁移过来)
	switch messageType {
	case "odds_change":
		p.handleOddsChange(eventID, productID, xmlContent, timestamp)
	case "bet_stop":
		p.handleBetStop(eventID, productID, xmlContent, timestamp)
	case "bet_settlement":
		p.handleBetSettlement(eventID, productID, xmlContent, timestamp)
	case "bet_cancel":
		p.handleBetCancel(eventID, productID, xmlContent, timestamp)
	case "fixture":
		p.handleFixture(eventID, productID, xmlContent, timestamp)
	case "fixture_change":
		p.handleFixtureChange(eventID, productID, xmlContent, timestamp)
	case "rollback_bet_settlement":
		p.handleRollbackBetSettlement(eventID, productID, xmlContent, timestamp)
	case "rollback_bet_cancel":
		p.handleRollbackBetCancel(eventID, productID, xmlContent, timestamp)
	default:
		logger.Printf("[MessageProcessor] Unhandled message type: %s", messageType)
	}
}

// extractMessageData 提取用于广播的附加数据 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) extractMessageData(messageType, xmlContent string) interface{} {
	// TODO: 实际实现应从 xmlContent 中提取数据
	return nil
}

// handleOddsChange 处理 odds_change 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleOddsChange(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.oddsChangeParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to handle odds_change: %v", err)
	}
}

// handleBetStop 处理 bet_stop 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleBetStop(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.betStopProcessor.ProcessBetStop(xmlContent); err != nil {
		logger.Errorf("Failed to handle bet_stop: %v", err)
	}
}

// handleBetSettlement 处理 bet_settlement 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.betSettlementParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to handle bet_settlement: %v", err)
	}
}

// handleBetCancel 处理 bet_cancel 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.betCancelProcessor.ProcessBetCancel(xmlContent); err != nil {
		logger.Errorf("Failed to handle bet_cancel: %v", err)
	}
}

// handleFixture 处理 fixture 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleFixture(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.fixtureParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to handle fixture: %v", err)
	}
}

// handleFixtureChange 处理 fixture_change 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleFixtureChange(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.fixtureParser.ParseFixtureChange(eventID, xmlContent); err != nil {
		logger.Errorf("Failed to handle fixture_change: %v", err)
	}
}

// handleRollbackBetSettlement 处理 rollback_bet_settlement 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleRollbackBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.rollbackBetSettlementProc.ProcessRollbackBetSettlement(xmlContent); err != nil {
		logger.Errorf("Failed to handle rollback_bet_settlement: %v", err)
	}
}

// handleRollbackBetCancel 处理 rollback_bet_cancel 消息 (从 AMQPConsumer 迁移过来)
func (p *MessageProcessor) handleRollbackBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if err := p.rollbackBetCancelProc.ProcessRollbackBetCancel(xmlContent); err != nil {
		logger.Errorf("Failed to handle rollback_bet_cancel: %v", err)
	}
}
