package services

import (
	"crypto/tls"
	"fmt"
	"time"
	"uof-service/logger"
	
	"github.com/streadway/amqp"
)

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	MaxRetries     int           // 最大重试次数 (0 = 无限重试)
	InitialDelay   time.Duration // 初始延迟
	MaxDelay       time.Duration // 最大延迟
	BackoffFactor  float64       // 退避因子
}

// DefaultReconnectConfig 默认重连配置
func DefaultReconnectConfig() *ReconnectConfig {
	return &ReconnectConfig{
		MaxRetries:    0,                // 无限重试
		InitialDelay:  1 * time.Second,  // 1秒
		MaxDelay:      60 * time.Second, // 60秒
		BackoffFactor: 2.0,              // 指数退避
	}
}

// StartWithReconnect 启动 AMQP 消费者并支持自动重连
func (c *AMQPConsumer) StartWithReconnect() error {
	reconnectConfig := DefaultReconnectConfig()
	
	logger.Println("[AMQP] Starting consumer with auto-reconnect enabled")
	
	// 首次连接
	if err := c.connectAndConsume(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}
	
	// 监控连接状态并自动重连
	go c.monitorConnection(reconnectConfig)
	
	return nil
}

// connectAndConsume 连接并开始消费
func (c *AMQPConsumer) connectAndConsume() error {
	// 获取 bookmaker 信息
	bookmakerId, virtualHost, err := c.getBookmakerInfo()
	if err != nil {
		return fmt.Errorf("failed to get bookmaker info: %w", err)
	}
	
	logger.Printf("[AMQP] Bookmaker ID: %s, Virtual Host: %s", bookmakerId, virtualHost)
	
	// 建立连接
	if err := c.connect(virtualHost); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	
	// 设置通道
	if err := c.setupChannel(); err != nil {
		return fmt.Errorf("failed to setup channel: %w", err)
	}
	
	// 开始消费
	if err := c.startConsuming(); err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}
	
	// 发送启动通知
	go c.notifier.NotifyServiceStart(bookmakerId, c.config.RecoveryProducts)
	
	// 启动消息统计
	go c.statsTracker.StartPeriodicReport()
	
	// 自动触发恢复
	if c.config.AutoRecovery {
		logger.Println("[AMQP] Auto recovery enabled, triggering full recovery...")
		go func() {
			time.Sleep(3 * time.Second)
			if err := c.recoveryManager.TriggerFullRecovery(); err != nil {
				logger.Errorf("[AMQP] Auto recovery failed: %v", err)
			} else {
				logger.Println("[AMQP] Auto recovery completed successfully")
			}
		}()
	}
	
	return nil
}

// connect 建立 AMQP 连接
func (c *AMQPConsumer) connect(virtualHost string) error {
	logger.Printf("[AMQP] Connecting to %s (vhost: %s)...", c.config.MessagingHost, virtualHost)
	
	// TLS 配置
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	
	// AMQP 配置
	config := amqp.Config{
		Vhost:           virtualHost,
		Heartbeat:       60 * time.Second,
		Locale:          "en_US",
		TLSClientConfig: tlsConfig,
	}
	
	// 构建 AMQP URL
	amqpURL := fmt.Sprintf("amqps://%s:@%s",
		c.config.AccessToken,
		c.config.MessagingHost,
	)
	
	// 连接
	conn, err := amqp.DialConfig(amqpURL, config)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	
	c.conn = conn
	logger.Println("[AMQP] ✅ Connected to AMQP server")
	
	return nil
}

// setupChannel 设置通道和队列
func (c *AMQPConsumer) setupChannel() error {
	// 创建通道
	channel, err := c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create channel: %w", err)
	}
	c.channel = channel
	
	// 设置 QoS
	if err := channel.Qos(100, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}
	
	// 声明队列
	queue, err := channel.QueueDeclare(
		"",    // name (auto-generated)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	
	logger.Printf("[AMQP] Queue declared: %s", queue.Name)
	
	// 绑定 routing keys
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
		logger.Printf("[AMQP] Bound to routing key: %s", routingKey)
	}
	
	return nil
}

// startConsuming 开始消费消息
func (c *AMQPConsumer) startConsuming() error {
	// 获取当前队列 (已在 setupChannel 中声明)
	// 这里需要重新获取队列信息
	queues, err := c.channel.QueueInspect("")
	if err != nil {
		// 如果无法获取,重新声明
		queue, err := c.channel.QueueDeclare(
			"",    // name (auto-generated)
			false, // durable
			true,  // delete when unused
			true,  // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue: %w", err)
		}
		queues = queue
	}
	
	// 开始消费
	msgs, err := c.channel.Consume(
		queues.Name,
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
	
	logger.Println("[AMQP] ✅ Started consuming messages")
	
	// 处理消息
	go c.handleMessages(msgs)
	
	return nil
}

// monitorConnection 监控连接状态并自动重连
func (c *AMQPConsumer) monitorConnection(config *ReconnectConfig) {
	retryCount := 0
	currentDelay := config.InitialDelay
	
	for {
		// 监听连接关闭事件
		closeErr := <-c.conn.NotifyClose(make(chan *amqp.Error))
		
		if closeErr == nil {
			// 正常关闭
			logger.Println("[AMQP] Connection closed normally")
			return
		}
		
		// 连接断开
		logger.Errorf("[AMQP] ⚠️  Connection lost: %v", closeErr)
		
		// 检查是否达到最大重试次数
		if config.MaxRetries > 0 && retryCount >= config.MaxRetries {
			logger.Errorf("[AMQP] ❌ Max retries (%d) reached, giving up", config.MaxRetries)
			c.notifier.NotifyError("AMQP Connection", 
				fmt.Sprintf("Max retries reached after %d attempts", retryCount))
			return
		}
		
		// 等待后重连
		retryCount++
		logger.Printf("[AMQP] 🔄 Reconnecting in %v (attempt %d)...", currentDelay, retryCount)
		time.Sleep(currentDelay)
		
		// 尝试重连
		if err := c.reconnect(); err != nil {
			logger.Errorf("[AMQP] ❌ Reconnect failed: %v", err)
			
			// 增加延迟 (指数退避)
			currentDelay = time.Duration(float64(currentDelay) * config.BackoffFactor)
			if currentDelay > config.MaxDelay {
				currentDelay = config.MaxDelay
			}
			
			continue
		}
		
		// 重连成功
		logger.Println("[AMQP] ✅ Reconnected successfully")
		c.notifier.NotifyInfo("AMQP Connection", 
			fmt.Sprintf("Reconnected after %d attempts", retryCount))
		
		// 重置重试计数和延迟
		retryCount = 0
		currentDelay = config.InitialDelay
	}
}

// reconnect 重新连接
func (c *AMQPConsumer) reconnect() error {
	// 清理旧连接
	if c.channel != nil {
		c.channel.Close()
		c.channel = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	
	// 重新连接
	return c.connectAndConsume()
}

