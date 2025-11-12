package services

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/streadway/amqp"
	"uof-service/config"
	"uof-service/logger"
)

// AMQPConnector 负责建立和维护 AMQP 连接，并返回消息通道
type AMQPConnector struct {
	config *config.Config
	conn   *amqp.Connection
	channel *amqp.Channel // 新增
	done   chan bool
}

// NewAMQPConnector 创建 AMQPConnector 实例
func NewAMQPConnector(cfg *config.Config) *AMQPConnector {
	return &AMQPConnector{
		config: cfg,
		done:   make(chan bool),
	}
}

// getBookmakerInfo 从 API 获取 bookmaker ID 和 virtual host
func (c *AMQPConnector) getBookmakerInfo() (string, string, error) {
	url := fmt.Sprintf("%s/users/whoami.xml", c.config.APIBaseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("x-access-token", c.config.AccessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}

		type UserInfo struct {
			XMLName     xml.Name `xml:"bookmaker_details"`
			BookmakerID string   `xml:"bookmaker_id,attr"`
			VirtualHost string   `xml:"virtual_host,attr"`
		}

	var userInfo UserInfo
	if err := xml.Unmarshal(body, &userInfo); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal user info: %w", err)
	}

		return userInfo.BookmakerID, userInfo.VirtualHost, nil
	}
	
	// Start 建立 AMQP 连接，声明队列，并开始消费，返回消息通道
	func (c *AMQPConnector) Start() (<-chan amqp.Delivery, error) {
		// 1. 获取bookmaker信息
		bookmakerId, virtualHost, err := c.getBookmakerInfo()
		if err != nil {
			return nil, fmt.Errorf("failed to get bookmaker info: %w", err)
		}
	
		// 将获取到的信息保存到 config 中，供 consumer 使用
		c.config.BookmakerID = bookmakerId
		c.config.VirtualHost = virtualHost
	
		logger.Printf("Bookmaker ID: %s", bookmakerId)
		logger.Printf("Virtual Host: %s", virtualHost)
		logger.Printf("Connecting to AMQP (vhost: %s)...", virtualHost)

	// 2. TLS配置 - 与Python代码一致，禁用证书验证
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // 等同Python的verify_mode=CERT_NONE
	}

	// 3. AMQP配置
	config := amqp.Config{
		Vhost:           virtualHost,
		Heartbeat:       60 * time.Second,
		Locale:          "en_US",
		TLSClientConfig: tlsConfig,
	}

	// 4. 构建AMQP URL
	amqpURL := fmt.Sprintf("amqps://%s:@%s",
		c.config.AccessToken,
		c.config.MessagingHost,
	)

	logger.Printf("AMQP URL: amqps://[token]:@%s", c.config.MessagingHost)
	logger.Println("Attempting AMQP connection with DialConfig...")
	logger.Println("This may take up to 30 seconds...")

	conn, err := amqp.DialConfig(amqpURL, config)
	if err != nil {
		logger.Errorf("Connection failed: %v", err)
		return nil, fmt.Errorf("failed to connect to AMQP: %w", err)
	}
	c.conn = conn

	logger.Println("Connected to AMQP server")

		// 5. 创建channel
		channel, err := conn.Channel()
		if err != nil {
			return nil, fmt.Errorf("failed to create channel: %w", err)
		}
		c.channel = channel // 保存 channel
	
		// 6. 设置QoS
	if err := channel.Qos(100, 0, false); err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	// 7. 声明队列
	queue, err := channel.QueueDeclare(
		"",    // name (empty for auto-generated)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	logger.Printf("Queue declared: %s", queue.Name)

	// 8. 绑定routing keys
	for _, routingKey := range c.config.RoutingKeys {
		if err := channel.QueueBind(
			queue.Name,
			routingKey,
			"unifiedfeed",
			false,
			nil,
		); err != nil {
			return nil, fmt.Errorf("failed to bind queue: %w", err)
		}
		logger.Printf("Bound to routing key: %s", routingKey)
	}

	// 9. 开始消费消息
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
		return nil, fmt.Errorf("failed to consume: %w", err)
	}

	logger.Println("Started consuming messages")

	return msgs, nil
}

// Stop 关闭连接
func (c *AMQPConnector) Stop() {
	logger.Println("Stopping AMQP connector...")
	if c.conn != nil {
		c.conn.Close()
	}
	close(c.done)
}
