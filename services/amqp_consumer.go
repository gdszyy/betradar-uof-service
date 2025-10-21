package services

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/streadway/amqp"

	"uof-service/config"
)

// MessageBroadcaster 接口用于广播消息，避免循环依赖
type MessageBroadcaster interface {
	Broadcast(msg interface{})
}

type AMQPConsumer struct {
	config          *config.Config
	messageStore    *MessageStore
	broadcaster     MessageBroadcaster
	recoveryManager *RecoveryManager
	conn            *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
}

func NewAMQPConsumer(cfg *config.Config, store *MessageStore, broadcaster MessageBroadcaster) *AMQPConsumer {
	return &AMQPConsumer{
		config:          cfg,
		messageStore:    store,
		broadcaster:     broadcaster,
		recoveryManager: NewRecoveryManager(cfg, store),
		done:            make(chan bool),
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
	log.Printf("Connecting to AMQP (vhost: %s)...", virtualHost)

	// 使用amqp.DialConfig更精确地控制连接参数，与Python的pika.ConnectionParameters类似
	log.Printf("Resolving host: %s", c.config.MessagingHost)
	
	// TLS配置 - 与Python代码一致，禁用证书验证
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,  // 等同Python的verify_mode=CERT_NONE
	}
	
	// AMQP配置
	config := amqp.Config{
		Vhost:      virtualHost,  // 直接设置，不编码
		Heartbeat:  60 * time.Second,  // 与Python一致
		Locale:     "en_US",
		TLSClientConfig: tlsConfig,  // TLS配置
	}
	
	// 构建AMQP URL - 不包含vhost（通过Config设置）
	amqpURL := fmt.Sprintf("amqps://%s:@%s",
		c.config.AccessToken,  // 不编码token，让库处理
		c.config.MessagingHost,
	)
	
	log.Printf("AMQP URL: amqps://[token]:@%s", c.config.MessagingHost)
	log.Printf("Attempting AMQP connection with DialConfig...")
	log.Printf("This may take up to 30 seconds...")
	
	conn, err := amqp.DialConfig(amqpURL, config)
	
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
	
	// 自动触发恢复（如果启用）
	if c.config.AutoRecovery {
		log.Println("Auto recovery is enabled, triggering full recovery...")
		go func() {
			// 等待几秒确保AMQP连接稳定
			time.Sleep(3 * time.Second)
			if err := c.recoveryManager.TriggerFullRecovery(); err != nil {
				log.Printf("Auto recovery failed: %v", err)
			} else {
				log.Println("Auto recovery completed successfully")
			}
		}()
	}

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
	case "bet_cancel":
		c.handleBetCancel(eventID, productID, xmlContent, timestamp)
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
	// 循环读取token直到找到第一个StartElement(跳过XML声明等)
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

	// 解析odds_change消息获取市场数量
	type OddsChange struct {
		Odds struct {
			Markets []struct {
				ID     string `xml:"id,attr"`
				Status int    `xml:"status,attr"`
				Outcomes []struct {
					ID     string  `xml:"id,attr"`
					Odds   float64 `xml:"odds,attr"`
					Active int     `xml:"active,attr"`
				} `xml:"outcome"`
			} `xml:"market"`
		} `xml:"odds"`
		SportEventStatus struct {
			Status        string `xml:"status,attr"`
			MatchStatus   int    `xml:"match_status,attr"`
			HomeScore     int    `xml:"home_score,attr"`
			AwayScore     int    `xml:"away_score,attr"`
		} `xml:"sport_event_status"`
	}

	var oddsChange OddsChange
	if err := xml.Unmarshal([]byte(xmlContent), &oddsChange); err != nil {
		log.Printf("Failed to parse odds_change: %v", err)
		return
	}

	marketsCount := len(oddsChange.Odds.Markets)
	log.Printf("Odds change for event %s: %d markets, status=%s", 
		eventID, marketsCount, oddsChange.SportEventStatus.Status)

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

	// 解析bet_stop消息
	type BetStop struct {
		MarketStatus int    `xml:"market_status,attr"`
		Groups       string `xml:"groups,attr"`
	}

	var betStop BetStop
	if err := xml.Unmarshal([]byte(xmlContent), &betStop); err != nil {
		log.Printf("Failed to parse bet_stop: %v", err)
	} else {
		log.Printf("Bet stop for event %s: market_status=%d, groups=%s", 
			eventID, betStop.MarketStatus, betStop.Groups)
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

	// 解析bet_settlement消息
	type BetSettlement struct {
		Certainty int `xml:"certainty,attr"`
		Outcomes struct {
			Markets []struct {
				ID string `xml:"id,attr"`
				Outcomes []struct {
					ID     string `xml:"id,attr"`
					Result int    `xml:"result,attr"`
				} `xml:"outcome"`
			} `xml:"market"`
		} `xml:"outcomes"`
	}

	var settlement BetSettlement
	if err := xml.Unmarshal([]byte(xmlContent), &settlement); err != nil {
		log.Printf("Failed to parse bet_settlement: %v", err)
	} else {
		marketsCount := len(settlement.Outcomes.Markets)
		log.Printf("Bet settlement for event %s: %d markets, certainty=%d", 
			eventID, marketsCount, settlement.Certainty)
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



// handleBetCancel 处理投注取消消息
func (c *AMQPConsumer) handleBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	// 解析bet_cancel消息
	type BetCancel struct {
		StartTime int64  `xml:"start_time,attr"`
		EndTime   int64  `xml:"end_time,attr"`
		Markets   []struct {
			ID string `xml:"id,attr"`
		} `xml:"market"`
	}

	var betCancel BetCancel
	if err := xml.Unmarshal([]byte(xmlContent), &betCancel); err != nil {
		log.Printf("Failed to parse bet_cancel: %v", err)
		return
	}

	marketsCount := len(betCancel.Markets)
	log.Printf("Bet cancel for event %s: %d markets cancelled", eventID, marketsCount)

	// 存储到数据库（使用通用的SaveMessage已经存储了，这里可以添加额外处理）
	c.messageStore.UpdateTrackedEvent(eventID)
}

// handleFixtureChange 处理赛程变化消息
func (c *AMQPConsumer) handleFixtureChange(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" {
		return
	}

	// 解析fixture_change消息
	type FixtureChange struct {
		StartTime     int64  `xml:"start_time,attr"`
		NextLiveTime  int64  `xml:"next_live_time,attr"`
		ChangeType    int    `xml:"change_type,attr"`
	}

	var fixtureChange FixtureChange
	if err := xml.Unmarshal([]byte(xmlContent), &fixtureChange); err != nil {
		log.Printf("Failed to parse fixture_change: %v", err)
		return
	}

	if fixtureChange.StartTime > 0 {
		startTimeStr := time.UnixMilli(fixtureChange.StartTime).Format(time.RFC3339)
		log.Printf("Fixture change for event %s: new start time %s", eventID, startTimeStr)
	}

	c.messageStore.UpdateTrackedEvent(eventID)
}

// handleRollbackBetSettlement 处理撤销投注结算消息
func (c *AMQPConsumer) handleRollbackBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	log.Printf("Rollback bet settlement for event %s", eventID)
	c.messageStore.UpdateTrackedEvent(eventID)
}

// handleRollbackBetCancel 处理撤销投注取消消息
func (c *AMQPConsumer) handleRollbackBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	log.Printf("Rollback bet cancel for event %s", eventID)
	c.messageStore.UpdateTrackedEvent(eventID)
}

// handleSnapshotComplete 处理快照完成消息
func (c *AMQPConsumer) handleSnapshotComplete(xmlContent string) {
	// 解析snapshot_complete消息
	type SnapshotComplete struct {
		RequestID int    `xml:"request_id,attr"`
		Product   int    `xml:"product,attr"`
		Timestamp int64  `xml:"timestamp,attr"`
	}

	var snapshot SnapshotComplete
	if err := xml.Unmarshal([]byte(xmlContent), &snapshot); err != nil {
		log.Printf("Failed to parse snapshot_complete: %v", err)
		return
	}

	log.Printf("✅ Snapshot complete: product=%d, request_id=%d, timestamp=%d", snapshot.Product, snapshot.RequestID, snapshot.Timestamp)
	
	// 更新恢复状态
	if snapshot.RequestID > 0 {
		if err := c.messageStore.UpdateRecoveryCompleted(snapshot.RequestID, snapshot.Product, snapshot.Timestamp); err != nil {
			log.Printf("Failed to update recovery status: %v", err)
		} else {
			log.Printf("Recovery request %d for product %d marked as completed", snapshot.RequestID, snapshot.Product)
		}
	}
}

