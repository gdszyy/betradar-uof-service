package services

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

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
	// 获取bookmaker信息
	bookmakerId, virtualHost, err := c.getBookmakerInfo()
	if err != nil {
		return fmt.Errorf("failed to get bookmaker info: %w", err)
	}

	log.Printf("Bookmaker ID: %s", bookmakerId)
	log.Printf("Virtual Host: %s", virtualHost)
	
	// 构建AMQP URL
	// 格式: amqps://username:password@host:port/vhost
	// 注意：不要手动编码virtual host，amqp.DialTLS会自动处理
	// 直接使用原始的virtual host路径，与Python代码一致
	amqpURL := fmt.Sprintf("amqps://%s:@%s%s",
		url.QueryEscape(c.config.AccessToken),
		c.config.MessagingHost,
		virtualHost,  // 直接使用，不编码
	)
	
	log.Printf("AMQP URL format: amqps://[token]:@%s%s", c.config.MessagingHost, virtualHost)
	
	log.Printf("Connecting to AMQP (vhost: %s)...", virtualHost)

	// 连接到AMQP
	log.Printf("Resolving host: %s", c.config.MessagingHost)
	
	// TLS配置 - 与Python代码一致，禁用证书验证
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,  // 等同Python的verify_mode=CERT_NONE
	}

	log.Printf("Attempting AMQP connection...")
	log.Printf("This may take up to 30 seconds...")
	
	conn, err := amqp.DialTLS(amqpURL, tlsConfig)
	
	if err != nil {
		log.Printf("Connection failed: %v", err)
		log.Printf("Possible causes:")
		log.Printf("  1. Network firewall blocking port 5671")
		log.Printf("  2. Railway IP not whitelisted by Betradar")
		log.Printf("  3. AMQP server unreachable from this location")
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
	decoder := xml.NewDecoder(bytes.NewReader([]byte(xmlContent)))
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

func (c *AMQPConsumer) getBookmakerInfo() (bookmakerId, virtualHost string, err error) {
	// 调用Betradar API获取bookmaker_id
	// API端点: GET /users/whoami.xml
	url := c.config.APIBaseURL + "/users/whoami.xml"
	log.Printf("Calling API: %s", url)
	log.Printf("Token length: %d characters", len(c.config.AccessToken))
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	req.Header.Set("x-access-token", c.config.AccessToken)
	log.Printf("Request headers: %v", req.Header)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("API Error Response: Status=%d, Body=%s", resp.StatusCode, string(body))
		return "", "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析XML响应
	type WhoAmIResponse struct {
		BookmakerID string `xml:"bookmaker_id,attr"`
		VirtualHost string `xml:"virtual_host,attr"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}

	var response WhoAmIResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return "", "", fmt.Errorf("failed to parse XML: %w", err)
	}

	if response.BookmakerID == "" {
		return "", "", fmt.Errorf("bookmaker_id not found in response")
	}

	if response.VirtualHost == "" {
		return "", "", fmt.Errorf("virtual_host not found in response")
	}

	return response.BookmakerID, response.VirtualHost, nil
}

