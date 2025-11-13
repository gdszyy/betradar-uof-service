package services

import (
	"encoding/xml"

	"uof-service/config"
	"fmt" // 修复 fmt 未导入的错误
	"strconv" // 修复 strconv 未导入的错误
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
	// 深度解析和数据增强
	switch messageType {
	case "odds_change":
		return p.extractOddsChangeData(xmlContent)
	case "bet_stop":
		return p.extractBetStopData(xmlContent)
	case "fixture_change":
		return p.extractFixtureChangeData(xmlContent)
	case "bet_settlement":
		return p.extractBetSettlementData(xmlContent)
	default:
		// 对于其他消息，只返回原始 XML 内容
		return map[string]interface{}{
			"xml_content": xmlContent,
		}
	}
}

// extractOddsChangeData 提取并增强 odds_change 消息数据
func (p *MessageProcessor) extractOddsChangeData(xmlContent string) interface{} {
	var oddsChange OddsChangeMessage
	if err := xml.Unmarshal([]byte(xmlContent), &oddsChange); err != nil {
		logger.Errorf("Failed to parse odds_change for broadcast: %v", err)
		return map[string]interface{}{"xml_content": xmlContent}
	}

		// 检查 marketDescService 是否存在
		if p.marketDescService == nil {
			logger.Printf("marketDescService is nil in MessageProcessor") // 修复 logger.Error 调用错误
			return map[string]interface{}{"xml_content": xmlContent}
		}

	// 提取比分和状态信息
	var homeScore, awayScore *int
	var matchStatus, status string
	if oddsChange.SportEventStatus != nil {
		ses := oddsChange.SportEventStatus
		homeScore = ses.HomeScore
		awayScore = ses.AwayScore
		matchStatus = ses.MatchStatus
		status = ses.Status
	}

	// 提取队伍名称
	var homeTeamName, awayTeamName string
	for _, comp := range oddsChange.SportEvent.Competitors {
		if comp.Qualifier == "home" {
			homeTeamName = comp.Name
		} else if comp.Qualifier == "away" {
			awayTeamName = comp.Name
		}
	}

	// 提取市场和赔率信息 (简化，只提取关键信息)
		markets := make([]map[string]interface{}, 0)
		for _, market := range oddsChange.Odds.Markets {
			// 构造 ReplacementContext
			ctx := &ReplacementContext{
				Specifiers: market.Specifier,
				// HomeTeamName 和 AwayTeamName 可以在这里添加，但为了简化，暂时只用 Specifiers
			}
			
			// 修复 GetMarketName 参数错误: 需要 string 类型的 marketID, specifiers, 和 ctx
			// 假设 market.ID 是 int，需要转换为 string
			marketIDStr := strconv.Itoa(market.ID)
			marketName := p.marketDescService.GetMarketName(marketIDStr, market.Specifier, ctx)
			
			outcomes := make([]map[string]interface{}, 0)
			for _, outcome := range market.Outcomes {
				// 修复 outcome.Name undefined 错误: 移除 Name 字段，因为它不在 OddsChangeMessage.Outcome 结构体中
				// 结果名称需要通过 GetOutcomeName 获取，但为了简化，暂时只返回 ID 和 Odds
				outcomes = append(outcomes, map[string]interface{}{
					"id": outcome.ID,
					"odds": outcome.Odds,
					"active": outcome.Active,
				})
			}

			markets = append(markets, map[string]interface{}{
				"id": market.ID,
				"specifier": market.Specifier,
				"name": marketName,
				"status": market.Status,
				"outcomes": outcomes,
			})
		}

	return map[string]interface{}{
		"event_id": oddsChange.EventID,
		"product_id": oddsChange.ProductID,
		"timestamp": oddsChange.Timestamp,
		"home_score": homeScore,
		"away_score": awayScore,
		"match_status": matchStatus,
		"status": status,
		"home_team_name": homeTeamName,
		"away_team_name": awayTeamName,
		"markets": markets,
	}
}

// extractBetStopData 提取并增强 bet_stop 消息数据
func (p *MessageProcessor) extractBetStopData(xmlContent string) interface{} {
	var betStop BetStopMessage
	if err := xml.Unmarshal([]byte(xmlContent), &betStop); err != nil {
		logger.Errorf("Failed to parse bet_stop for broadcast: %v", err)
		return map[string]interface{}{"xml_content": xmlContent}
	}

	return map[string]interface{}{
		"event_id": betStop.EventID,
		"product_id": betStop.ProductID,
		"timestamp": betStop.Timestamp,
		"market_status": betStop.MarketStatus,
		"groups": betStop.Groups,
		"reason": "Betting Suspended", // 补充业务描述
	}
}

// extractFixtureChangeData 提取并增强 fixture_change 消息数据
func (p *MessageProcessor) extractFixtureChangeData(xmlContent string) interface{} {
	type FixtureChange struct {
		StartTime    int64 `xml:"start_time,attr"`
		NextLiveTime int64 `xml:"next_live_time,attr"`
		ChangeType   int   `xml:"change_type,attr"`
		ProductID    int   `xml:"product,attr"`
	}
	var fixtureChange FixtureChange
	if err := xml.Unmarshal([]byte(xmlContent), &fixtureChange); err != nil {
		logger.Errorf("Failed to parse fixture_change for broadcast: %v", err)
		return map[string]interface{}{"xml_content": xmlContent}
	}

	// 尝试获取最新的赛事信息（假设 fixtureService 提供了 GetTrackedEventInfo 方法）
	// 由于没有看到 GetTrackedEventInfo，我们只返回 change 消息的关键信息
	
	changeDescription := ""
	switch fixtureChange.ChangeType {
	case 0:
		changeDescription = "New Fixture"
	case 1:
		changeDescription = "Start Time Change"
	case 2:
		changeDescription = "Coverage Change"
	case 3:
		changeDescription = "Coverage Added"
	case 4:
		changeDescription = "Coverage Removed"
	case 5:
		changeDescription = "Live Coverage Dropped"
	default:
		changeDescription = fmt.Sprintf("Unknown Change Type (%d)", fixtureChange.ChangeType)
	}

	return map[string]interface{}{
		"product_id": fixtureChange.ProductID,
		"timestamp": fixtureChange.StartTime, // 使用 start_time 作为时间戳
		"change_type": fixtureChange.ChangeType,
		"change_description": changeDescription,
		"new_start_time": fixtureChange.StartTime,
	}
}

// extractBetSettlementData 提取并增强 bet_settlement 消息数据
func (p *MessageProcessor) extractBetSettlementData(xmlContent string) interface{} {
	// 简化处理，只提取关键信息
	type BetSettlement struct {
		EventID string `xml:"event_id,attr"`
		ProductID int `xml:"product,attr"`
		Timestamp int64 `xml:"timestamp,attr"`
		Markets []struct {
			ID string `xml:"id,attr"`
			Specifier string `xml:"specifiers,attr"` // 新增 specifier 字段
			Outcomes []struct {
				ID string `xml:"id,attr"`
				Status string `xml:"status,attr"` // Won, Lost, Half_Won, Half_Lost, Void
			} `xml:"outcome"`
		} `xml:"market"`
	}
	var settlement BetSettlement
	if err := xml.Unmarshal([]byte(xmlContent), &settlement); err != nil {
		logger.Errorf("Failed to parse bet_settlement for broadcast: %v", err)
		return map[string]interface{}{"xml_content": xmlContent}
	}

	markets := make([]map[string]interface{}, 0)
		for _, market := range settlement.Markets {
			// 构造 ReplacementContext
			ctx := &ReplacementContext{
				Specifiers: market.Specifier,
			}
			
			// 修复 GetMarketName 参数错误: 需要 string 类型的 marketID, specifiers, 和 ctx
			marketName := p.marketDescService.GetMarketName(market.ID, market.Specifier, ctx)
			
			outcomes := make([]map[string]interface{}, 0)
			for _, outcome := range market.Outcomes {
				outcomes = append(outcomes, map[string]interface{}{
					"id": outcome.ID,
					"status": outcome.Status,
				})
			}

			markets = append(markets, map[string]interface{}{
				"id": market.ID,
				"specifier": market.Specifier,
				"name": marketName,
				"outcomes": outcomes,
			})
		}

	return map[string]interface{}{
		"event_id": settlement.EventID,
		"product_id": settlement.ProductID,
		"timestamp": settlement.Timestamp,
		"markets": markets,
	}
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
