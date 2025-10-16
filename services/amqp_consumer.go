package services

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/streadway/amqp"

	"uof-service/config"
)

// MessageBroadcaster 接口用于广播消息，避免循环依赖
type MessageBroadcaster interface {
	Broadcast(msg interface{})
}

type AMQPConsumer struct {
	config       *config.Config
	messageStore *MessageStore
	broadcaster  MessageBroadcaster
	conn         *amqp.Connection
	channel      *amqp.Channel
	done         chan bool
}

func NewAMQPConsumer(cfg *config.Config, store *MessageStore, broadcaster MessageBroadcaster) *AMQPConsumer {
	return &AMQPConsumer{
		config:       cfg,
		messageStore: store,
		broadcaster:  broadcaster,
		done:         make(chan bool),
	}
}

func (c *AMQPConsumer) Start() error {
	// 获取bookmaker_id
	bookmakerId, err := c.getBookmakerId()
	if err != nil {
		return fmt.Errorf("failed to get bookmaker_id: %w", err)
	}

	log.Printf("Bookmaker ID: %s", bookmakerId)

	// 构建AMQP URL
	virtualHost := fmt.Sprintf("/unifiedfeed/%s", bookmakerId)
	amqpURL := fmt.Sprintf("amqps://%s:@%s%s",
		url.QueryEscape(c.config.AccessToken),
		c.config.MessagingHost,
		virtualHost,
	)

	// 连接到AMQP
	tlsConfig := &tls.Config{
		ServerName: "stgmq.betradar.com",
	}

	conn, err := amqp.DialTLS(amqpURL, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to AMQP: %w", err)
	}
	c.conn = conn

	log.Println("Connected to AMQP server")

	// 创建channel
	channel, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}
	c.channel = channel

	// 设置QoS
	if err := channel.Qos(100, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// 声明队列
	queue, err := channel.QueueDeclare(
		"",    // name (empty for auto-generated)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	log.Printf("Queue declared: %s", queue.Name)

	// 绑定routing keys
	for _, routingKey := range c.config.RoutingKeys {
		if err := channel.QueueBind(
			queue.Name,
			routingKey,
			"unifiedfeed",
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue: %w", err)
		}
		log.Printf("Bound to routing key: %s", routingKey)
	}

	// 开始消费消息
	msgs, err := channel.Consume(
		queue.Name,
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to consume: %w", err)
	}

	log.Println("Started consuming messages")

	// 处理消息
	go c.handleMessages(msgs)

	<-c.done
	return nil
}

func (c *AMQPConsumer) Stop() {
	log.Println("Stopping AMQP consumer...")
	
	if c.channel != nil {
		c.channel.Close()
	}
	
	if c.conn != nil {
		c.conn.Close()
	}
	
	close(c.done)
}

func (c *AMQPConsumer) handleMessages(msgs <-chan amqp.Delivery) {
	for msg := range msgs {
		c.processMessage(msg)
	}
}

func (c *AMQPConsumer) processMessage(msg amqp.Delivery) {
	routingKey := msg.RoutingKey
	xmlContent := string(msg.Body)

	// 解析消息类型
	messageType, eventID, productID, sportID, timestamp := c.parseMessage(xmlContent)

	// 存储到数据库
	if err := c.messageStore.SaveMessage(messageType, eventID, productID, sportID, routingKey, xmlContent, timestamp); err != nil {
		log.Printf("Failed to save message: %v", err)
	}

	// 广播到WebSocket客户端
	if c.broadcaster != nil {
		c.broadcaster.Broadcast(map[string]interface{}{
			"type":         "message",
			"message_type": messageType,
			"event_id":     eventID,
			"product_id":   productID,
			"routing_key":  routingKey,
			"xml":          xmlContent,
			"timestamp":    timestamp,
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
	}
}

func (c *AMQPConsumer) parseMessage(xmlContent string) (messageType, eventID string, productID *int, sportID *string, timestamp int64) {
	// 简单的XML解析获取基本信息
	type BaseMessage struct {
		EventID   string `xml:"event_id,attr"`
		ProductID int    `xml:"product,attr"`
		SportID   string `xml:"sport_id,attr"`
		Timestamp int64  `xml:"timestamp,attr"`
	}

	// 获取根元素名称作为消息类型
	decoder := xml.NewDecoder([]byte(xmlContent))
	token, _ := decoder.Token()
	if startElement, ok := token.(xml.StartElement); ok {
		messageType = startElement.Name.Local
	}

	// 解析基本属性
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

func (c *AMQPConsumer) handleAlive(xmlContent string) {
	type AliveMessage struct {
		ProductID  int `xml:"product,attr"`
		Timestamp  int64 `xml:"timestamp,attr"`
		Subscribed int `xml:"subscribed,attr"`
	}

	var alive AliveMessage
	if err := xml.Unmarshal([]byte(xmlContent), &alive); err != nil {
		log.Printf("Failed to parse alive message: %v", err)
		return
	}

	// 更新生产者状态
	if err := c.messageStore.UpdateProducerStatus(alive.ProductID, alive.Timestamp, alive.Subscribed); err != nil {
		log.Printf("Failed to update producer status: %v", err)
	}
}

func (c *AMQPConsumer) handleOddsChange(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	// 计算市场数量
	marketsCount := 0
	// TODO: 解析XML获取准确的市场数量

	if err := c.messageStore.SaveOddsChange(eventID, *productID, timestamp, xmlContent, marketsCount); err != nil {
		log.Printf("Failed to save odds change: %v", err)
	}

	// 更新跟踪的赛事
	c.messageStore.UpdateTrackedEvent(eventID)
}

func (c *AMQPConsumer) handleBetStop(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	if err := c.messageStore.SaveBetStop(eventID, *productID, timestamp, xmlContent); err != nil {
		log.Printf("Failed to save bet stop: %v", err)
	}

	c.messageStore.UpdateTrackedEvent(eventID)
}

func (c *AMQPConsumer) handleBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	if err := c.messageStore.SaveBetSettlement(eventID, *productID, timestamp, xmlContent); err != nil {
		log.Printf("Failed to save bet settlement: %v", err)
	}

	c.messageStore.UpdateTrackedEvent(eventID)
}

func (c *AMQPConsumer) getBookmakerId() (string, error) {
	// TODO: 调用API获取bookmaker_id
	// 这里先返回一个示例值
	return "12345", nil
}

