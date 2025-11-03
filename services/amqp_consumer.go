package services

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"uof-service/logger"
	"time"

	"github.com/streadway/amqp"

	"uof-service/config"
)

// MessageBroadcaster 接口用于广播消息，避免循环依赖
type MessageBroadcaster interface {
	Broadcast(msg interface{})
}

type AMQPConsumer struct {
	config               *config.Config
	messageStore         *MessageStore
	broadcaster          MessageBroadcaster
	recoveryManager      *RecoveryManager
	notifier             *LarkNotifier
	statsTracker         *MessageStatsTracker
	matchMonitor         *MatchMonitor
	fixtureParser        *FixtureParser
	oddsChangeParser     *OddsChangeParser
	oddsParser           *OddsParser
	betSettlementParser         *BetSettlementParser
	betStopProcessor            *BetStopProcessor
	betCancelProcessor          *BetCancelProcessor
	rollbackBetSettlementProc   *RollbackBetSettlementProcessor
	rollbackBetCancelProc       *RollbackBetCancelProcessor
	srnMappingService           *SRNMappingService
	fixtureService              *FixtureService
	marketDescService           *MarketDescriptionsService
	conn                 *amqp.Connection
	channel              *amqp.Channel
	done                 chan bool
}

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
			config:               cfg,
			messageStore:         store,
			broadcaster:          broadcaster,
			recoveryManager:      NewRecoveryManager(cfg, store),
			notifier:             notifier,
			statsTracker:         statsTracker,
			fixtureParser:        fixtureParser,
			oddsChangeParser:     oddsChangeParser,
		oddsParser:                 oddsParser,
		betSettlementParser:        betSettlementParser,
		betStopProcessor:           betStopProcessor,
		betCancelProcessor:         betCancelProcessor,
		rollbackBetSettlementProc:  rollbackBetSettlementProc,
		rollbackBetCancelProc:      rollbackBetCancelProc,
		srnMappingService:          srnMappingService,
		fixtureService:             fixtureService,
		marketDescService:          marketDescService,
			done:                 make(chan bool),
		}
}

func (c *AMQPConsumer) Start() error {
	// 获取bookmaker信息
	bookmakerId, virtualHost, err := c.getBookmakerInfo()
	if err != nil {
		return fmt.Errorf("failed to get bookmaker info: %w", err)
	}

	logger.Printf("Bookmaker ID: %s", bookmakerId)
	logger.Printf("Virtual Host: %s", virtualHost)
	logger.Printf("Connecting to AMQP (vhost: %s)...", virtualHost)

	// 使用amqp.DialConfig更精确地控制连接参数，与Python的pika.ConnectionParameters类似
	logger.Printf("Resolving host: %s", c.config.MessagingHost)
	
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
	
	logger.Printf("AMQP URL: amqps://[token]:@%s", c.config.MessagingHost)
	logger.Println("Attempting AMQP connection with DialConfig...")
	logger.Println("This may take up to 30 seconds...")
	
	conn, err := amqp.DialConfig(amqpURL, config)
	
	if err != nil {
		logger.Errorf("Connection failed: %v", err)
		logger.Errorf("Possible causes:")
		logger.Errorf("  1. Network firewall blocking port 5671")
		logger.Errorf("  2. Railway IP not whitelisted by Betradar")
		logger.Errorf("  3. AMQP server unreachable from this location")
		return fmt.Errorf("failed to connect to AMQP: %w", err)
	}
	c.conn = conn

	logger.Println("Connected to AMQP server")

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

	logger.Printf("Queue declared: %s", queue.Name)

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
		logger.Printf("Bound to routing key: %s", routingKey)
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

	logger.Println("Started consuming messages")
	
	// 发送服务启动通知
	go c.notifier.NotifyServiceStart(bookmakerId, c.config.RecoveryProducts)
	
	// 启动消息统计
	go c.statsTracker.StartPeriodicReport()
		
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

func (c *AMQPConsumer) Stop() {
	logger.Println("Stopping AMQP consumer...")
	
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
	
	// 统计消息
	if messageType != "" {
		c.statsTracker.Record(messageType)
	}

	// 存储到数据库
	if err := c.messageStore.SaveMessage(messageType, eventID, productID, sportID, routingKey, xmlContent, timestamp); err != nil {
		logger.Errorf("Failed to save message: %v", err)
	}

	// 广播到WebSocket客户端 (包含基本数据)
	if c.broadcaster != nil {
		// 根据消息类型添加额外数据
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
		logger.Errorf("Failed to parse alive message: %v", err)
		return
	}

	// 更新生产者状态
	if err := c.messageStore.UpdateProducerStatus(alive.ProductID, alive.Timestamp, alive.Subscribed); err != nil {
		logger.Errorf("Failed to update producer status: %v", err)
	}
	
	// 检测订阅取消 (subscribed=0)
	if alive.Subscribed == 0 {
			logger.Printf("[AliveMessage] ⚠️  Producer %d subscription cancelled! All markets should be suspended.", alive.ProductID)
		
		// 发送告警通知
		if c.notifier != nil {
			message := fmt.Sprintf("🚨 UOF Subscription Cancelled\n\n"+
				"Producer %d subscription has been cancelled.\n"+
				"All markets from this producer should be suspended.",
				alive.ProductID)
			c.notifier.SendText(message)
		}
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
		logger.Errorf("Failed to parse odds_change: %v", err)
		return
	}

	marketsCount := len(oddsChange.Odds.Markets)
	// 日志在 OddsChangeParser 中输出

	if err := c.messageStore.SaveOddsChange(eventID, *productID, timestamp, xmlContent, marketsCount); err != nil {
		logger.Printf("Failed to save odds change: %v", err)
	}

	// 更新跟踪的赛事
	c.messageStore.UpdateTrackedEvent(eventID)
	
	// 如果是 Producer 1 的 odds_change，自动将该比赛标记为已订阅
	if *productID == 1 {
		if err := c.messageStore.SetEventSubscribed(eventID, true); err != nil {
			logger.Errorf("[OddsChange] Failed to set event %s as subscribed: %v", eventID, err)
		} else {
			logger.Printf("[OddsChange] ✅ Event %s marked as subscribed (Producer 1)", eventID)
		}
	}
	
	// 检查是否有队伍信息，如果没有则自动获取
	hasTeamInfo, err := c.messageStore.HasTeamInfo(eventID)
	if err != nil {
		logger.Errorf("[OddsChange] Failed to check team info for %s: %v", eventID, err)
	} else if !hasTeamInfo {
		logger.Printf("[OddsChange] Event %s missing team info, fetching fixture...", eventID)
		go c.fetchAndStoreFixture(eventID) // 异步获取，不阻塞消息处理
	}
	
	// 使用 OddsChangeParser 解析比分和比赛信息
	if err := c.oddsChangeParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to parse odds_change data: %v", err)
	}
	
	// 使用 OddsParser 解析和存储赔率数据
	if err := c.oddsParser.ParseAndStoreOdds([]byte(xmlContent), *productID); err != nil {
		logger.Errorf("Failed to parse and store odds: %v", err)
	}
}

func (c *AMQPConsumer) handleBetStop(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	// 日志在 BetStopProcessor 中输出

	// 使用 BetStopProcessor 处理并更新 market status
	if err := c.betStopProcessor.ProcessBetStop(xmlContent); err != nil {
		logger.Errorf("Failed to process bet stop: %v", err)
	}

	// 仍然保存 XML 原文到 bet_stops 表 (备份)
	if err := c.messageStore.SaveBetStop(eventID, *productID, timestamp, xmlContent); err != nil {
		logger.Errorf("Failed to save bet stop XML: %v", err)
	}

	c.messageStore.UpdateTrackedEvent(eventID)
}

func (c *AMQPConsumer) handleBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	// 日志在 BetSettlementParser 中输出

	// 使用 BetSettlementParser 解析并存储
	if err := c.betSettlementParser.ParseAndStore(xmlContent); err != nil {
		logger.Errorf("Failed to parse and store bet settlement: %v", err)
	}

	// 仍然保存 XML 原文到 bet_settlements 表 (备份)
	if err := c.messageStore.SaveBetSettlement(eventID, *productID, timestamp, xmlContent); err != nil {
		logger.Errorf("Failed to save bet settlement XML: %v", err)
	}

	c.messageStore.UpdateTrackedEvent(eventID)
}

func (c *AMQPConsumer) getBookmakerInfo() (bookmakerId, virtualHost string, err error) {
	// 调用Betradar API获取bookmaker_id
	// API端点: GET /users/whoami.xml
	url := c.config.APIBaseURL + "/users/whoami.xml"
	logger.Printf("Calling API: %s", url)
	logger.Printf("Token length: %d characters", len(c.config.AccessToken))
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	req.Header.Set("x-access-token", c.config.AccessToken)
	logger.Printf("Request headers: %v", req.Header)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			logger.Errorf("API Error Response: Status=%d, Body=%s", resp.StatusCode, string(body))
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
	
	// SetStatsTracker 设置消息统计追踪器
	func (c *AMQPConsumer) SetStatsTracker(tracker *MessageStatsTracker) {
		c.statsTracker = tracker
	}
	
	// GetChannel 获取AMQP通道
	func (c *AMQPConsumer) GetChannel() *amqp.Channel {
		return c.channel
	}
	
	// handleBetCancel 处理投注取消消息
func (c *AMQPConsumer) handleBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	// 日志在 BetCancelProcessor 中输出

	// 使用 BetCancelProcessor 处理并更新 market status
	if err := c.betCancelProcessor.ProcessBetCancel(xmlContent); err != nil {
		logger.Errorf("Failed to process bet cancel: %v", err)
	}

	c.messageStore.UpdateTrackedEvent(eventID)
}

// handleFixture 处理 fixture 消息
func (c *AMQPConsumer) handleFixture(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" {
		return
	}
	
		// 日志在 FixtureParser 中输出
		
		// 使用 FixtureParser 解析完整的 fixture 消息
		if err := c.fixtureParser.ParseAndStore(xmlContent); err != nil {
			logger.Errorf("Failed to parse fixture data: %v", err)
		}
	
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
			logger.Errorf("Failed to parse fixture_change: %v", err)
			return
		}

	// 日志在 FixtureParser 中输出

	c.messageStore.UpdateTrackedEvent(eventID)
	
		// 使用 FixtureParser 解析赛程变化
		if err := c.fixtureParser.ParseFixtureChange(eventID, xmlContent); err != nil {
			logger.Errorf("Failed to parse fixture_change data: %v", err)
		}
}

// handleRollbackBetSettlement 处理撤销投注结算消息
func (c *AMQPConsumer) handleRollbackBetSettlement(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	// 日志在 RollbackBetSettlementProcessor 中输出

	// 使用 RollbackBetSettlementProcessor 处理并恢复 market status
	if err := c.rollbackBetSettlementProc.ProcessRollbackBetSettlement(xmlContent); err != nil {
		logger.Errorf("Failed to process rollback bet settlement: %v", err)
	}

	c.messageStore.UpdateTrackedEvent(eventID)
}

// handleRollbackBetCancel 处理撤销投注取消消息
func (c *AMQPConsumer) handleRollbackBetCancel(eventID string, productID *int, xmlContent string, timestamp int64) {
	if eventID == "" || productID == nil {
		return
	}

	// 日志在 RollbackBetCancelProcessor 中输出

	// 使用 RollbackBetCancelProcessor 处理并恢复 market status
	if err := c.rollbackBetCancelProc.ProcessRollbackBetCancel(xmlContent); err != nil {
		logger.Errorf("Failed to process rollback bet cancel: %v", err)
	}

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
		logger.Printf("Failed to parse snapshot_complete: %v", err)
		return
	}

	// 更新恢复状态
	if snapshot.RequestID > 0 {
		if err := c.messageStore.UpdateRecoveryCompleted(snapshot.RequestID, snapshot.Product, snapshot.Timestamp); err != nil {
			logger.Printf("Failed to update recovery status: %v", err)
			if c.notifier != nil {
				c.notifier.NotifyError("Recovery", fmt.Sprintf("Failed to update recovery status: %v", err))
			}
		} else {
			logger.Printf("[snapshot_complete] Producer %d 的数据恢复已完成 (request_id=%d)", snapshot.Product, snapshot.RequestID)
			if c.notifier != nil {
				c.notifier.NotifyRecoveryComplete(snapshot.Product, int64(snapshot.RequestID))
			}
		}
	}
}



// fetchAndStoreFixture 获取并存储赛事的 Fixture 信息
func (c *AMQPConsumer) fetchAndStoreFixture(eventID string) {
	if c.fixtureService == nil {
		logger.Printf("[FixtureFetch] ⚠️  FixtureService not initialized")
		return
	}
	
	// 获取 Fixture 信息
	fixture, err := c.fixtureService.FetchFixture(eventID)
	if err != nil {
		logger.Printf("[FixtureFetch] ❌ Failed to fetch fixture for %s: %v", eventID, err)
		return
	}
	
	// 提取队伍信息、运动类型和状态
	homeID, homeName, awayID, awayName, sportID, sportName, status := fixture.GetTeamInfo()
	
	// 更新数据库
	if err := c.messageStore.UpdateEventTeamInfo(eventID, homeID, homeName, awayID, awayName, sportID, sportName, status); err != nil {
		logger.Printf("[FixtureFetch] ❌ Failed to update team info for %s: %v", eventID, err)
		return
	}
	
	logger.Printf("[FixtureFetch] ✅ Updated team info for %s: %s vs %s (sport: %s, status: %s)", eventID, homeName, awayName, sportName, status)
}



// extractMessageData 从 XML 中提取关键数据用于 WebSocket 广播
func (c *AMQPConsumer) extractMessageData(messageType string, xmlContent string) map[string]interface{} {
	data := make(map[string]interface{})
	
	switch messageType {
	case "odds_change":
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
		if err := xml.Unmarshal([]byte(xmlContent), &oddsChange); err == nil {
			data["markets_count"] = len(oddsChange.Odds.Markets)
			data["status"] = oddsChange.SportEventStatus.Status
			data["match_status"] = oddsChange.SportEventStatus.MatchStatus
			data["home_score"] = oddsChange.SportEventStatus.HomeScore
			data["away_score"] = oddsChange.SportEventStatus.AwayScore
			
			// 添加部分盘口信息 (最多前 5 个)
			markets := []map[string]interface{}{}
			for i, market := range oddsChange.Odds.Markets {
				if i >= 5 {
					break
				}
				marketData := map[string]interface{}{
					"id":     market.ID,
					"status": market.Status,
				}
				outcomes := []map[string]interface{}{}
				for _, outcome := range market.Outcomes {
					outcomes = append(outcomes, map[string]interface{}{
						"id":     outcome.ID,
						"odds":   outcome.Odds,
						"active": outcome.Active,
					})
				}
				marketData["outcomes"] = outcomes
				markets = append(markets, marketData)
			}
			data["markets"] = markets
		}
		
	case "bet_stop":
		type BetStop struct {
			Groups string `xml:"groups,attr"`
			MarketStatus int `xml:"market_status,attr"`
		}
		var betStop BetStop
		if err := xml.Unmarshal([]byte(xmlContent), &betStop); err == nil {
			data["groups"] = betStop.Groups
			data["market_status"] = betStop.MarketStatus
		}
		
	case "bet_settlement":
		type BetSettlement struct {
			Outcomes struct {
				Markets []struct {
					ID string `xml:"id,attr"`
					Outcomes []struct {
						ID     string  `xml:"id,attr"`
						Result int     `xml:"result,attr"`
					} `xml:"outcome"`
				} `xml:"market"`
			} `xml:"outcomes"`
		}
		var settlement BetSettlement
		if err := xml.Unmarshal([]byte(xmlContent), &settlement); err == nil {
			data["markets_count"] = len(settlement.Outcomes.Markets)
		}
		
	case "fixture_change":
		type FixtureChange struct {
			ChangeType int   `xml:"change_type,attr"`
			StartTime  int64 `xml:"start_time,attr"`
		}
		var fixtureChange FixtureChange
		if err := xml.Unmarshal([]byte(xmlContent), &fixtureChange); err == nil {
			data["change_type"] = fixtureChange.ChangeType
			data["start_time"] = fixtureChange.StartTime
		}
		
	case "bet_cancel":
		type BetCancel struct {
			Markets []struct {
				ID string `xml:"id,attr"`
			} `xml:"market"`
		}
		var betCancel BetCancel
		if err := xml.Unmarshal([]byte(xmlContent), &betCancel); err == nil {
			data["markets_count"] = len(betCancel.Markets)
		}
	}
	
	return data
}

