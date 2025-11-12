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
	// AMQPConsumer 现在作为 Ingestor，只负责将消息转发到 Broker
	broker MessageBroker // 新增：抽象的消息队列接口
	
	// 保留部分核心依赖
	messageStore              *MessageStore
	recoveryManager           *RecoveryManager
	notifier                  *LarkNotifier
	statsTracker              *MessageStatsTracker
	
	// 移除所有业务处理器依赖
	// fixtureParser             *FixtureParser
	// oddsChangeParser          *OddsChangeParser
	// ...

	done                      chan bool
}

// NewAMQPConsumer 创建 AMQPConsumer 实例
// AMQPConsumer 现在作为 Ingestor，只依赖于 Broker 和少量核心服务
func NewAMQPConsumer(cfg *config.Config, store *MessageStore, broker MessageBroker) *AMQPConsumer {
	notifier := NewLarkNotifier(cfg.LarkWebhook)
	statsTracker := NewMessageStatsTracker(notifier, 5*time.Minute)

	return &AMQPConsumer{
		config:          cfg,
		messageStore:    store,
		broker:          broker, // 注入 Broker
		recoveryManager: NewRecoveryManager(cfg, store),
		notifier:        notifier,
		statsTracker:    statsTracker,
		done:            make(chan bool),
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

	// 存储到数据库 (Ingestor 仍然负责存储原始消息)
	if err := c.messageStore.SaveMessage(messageType, eventID, productID, sportID, routingKey, xmlContent, timestamp); err != nil {
		logger.Errorf("Failed to save message: %v", err)
	}

	// -------------------------------------------------------------------
	// 核心修改：将消息转发到 Broker
	// -------------------------------------------------------------------
	if c.broker != nil && messageType != "" {
		topic := GetTopicName(messageType)
		brokerMsg := BrokerMessage{
			Topic: topic,
			Key:   eventID, // 使用 eventID 作为 Key，确保同一赛事的顺序性
			Value: msg.Body, // 发送原始字节，避免二次转换
		}
		if err := c.broker.Produce(brokerMsg); err != nil {
			logger.Errorf("Failed to produce message to broker topic %s: %v", topic, err)
		}
	}
	// -------------------------------------------------------------------
	
	// 仅保留 Ingestor 必须处理的逻辑：alive 和 snapshot_complete
	switch messageType {
	case "alive":
		c.handleAlive(xmlContent)
	case "snapshot_complete":
		c.handleSnapshotComplete(xmlContent)
	}
	
	// 移除 WebSocket 广播逻辑，这应该由 MessageProcessor 或单独的模块处理
	// 移除所有业务处理逻辑 (odds_change, bet_stop, fixture, etc.)
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

// 移除所有业务处理函数，它们将被 MessageProcessor 模块取代

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
