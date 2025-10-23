package thesports

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const (
	// DefaultMQTTBroker is the default MQTT broker address
	DefaultMQTTBroker = "ssl://mq.thesports.com:443"
	
	// MQTT Quality of Service levels
	QoSAtMostOnce  = 0
	QoSAtLeastOnce = 1
	QoSExactlyOnce = 2
)

// MQTTClient represents an MQTT client for The Sports API
type MQTTClient struct {
	username string
	password string
	broker   string
	client   mqtt.Client
	handlers map[string][]MQTTMessageHandler
	mu       sync.RWMutex
}

// MQTTMessageHandler is a function that handles MQTT messages
type MQTTMessageHandler func(topic string, payload []byte)

// MQTTMessage represents a message received from MQTT
type MQTTMessage struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

// NewMQTTClient creates a new MQTT client
// username: your API username
// password: your API key/secret
func NewMQTTClient(username, password string) *MQTTClient {
	return NewMQTTClientWithBroker(DefaultMQTTBroker, username, password)
}

// NewMQTTClientWithBroker creates a new MQTT client with custom broker
func NewMQTTClientWithBroker(broker, username, password string) *MQTTClient {
	return &MQTTClient{
		username: username,
		password: password,
		broker:   broker,
		handlers: make(map[string][]MQTTMessageHandler),
	}
}

// Connect establishes connection to MQTT broker
func (c *MQTTClient) Connect() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(c.broker)
	opts.SetUsername(c.username)
	opts.SetPassword(c.password)
	opts.SetClientID(fmt.Sprintf("thesports_go_%d", time.Now().Unix()))
	
	// Enable TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}
	opts.SetTLSConfig(tlsConfig)
	
	// Set connection callbacks
	opts.SetOnConnectHandler(c.onConnect)
	opts.SetConnectionLostHandler(c.onConnectionLost)
	
	// Set default message handler
	opts.SetDefaultPublishHandler(c.onMessage)
	
	// Auto reconnect
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)
	
	// Keep alive
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(10 * time.Second)
	
	// Clean session
	opts.SetCleanSession(true)
	
	c.client = mqtt.NewClient(opts)
	
	token := c.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect: %w", token.Error())
	}
	
	return nil
}

// Disconnect closes the connection to MQTT broker
func (c *MQTTClient) Disconnect() error {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}
	return nil
}

// IsConnected returns whether the client is connected
func (c *MQTTClient) IsConnected() bool {
	return c.client != nil && c.client.IsConnected()
}

// Subscribe subscribes to a topic
func (c *MQTTClient) Subscribe(topic string, qos byte) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	token := c.client.Subscribe(topic, qos, nil)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", topic, token.Error())
	}
	
	log.Printf("Subscribed to topic: %s", topic)
	return nil
}

// Unsubscribe unsubscribes from a topic
func (c *MQTTClient) Unsubscribe(topic string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	token := c.client.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from %s: %w", topic, token.Error())
	}
	
	log.Printf("Unsubscribed from topic: %s", topic)
	return nil
}

// SubscribeFootball subscribes to football topic
func (c *MQTTClient) SubscribeFootball(topic string) error {
	return c.Subscribe(topic, QoSAtLeastOnce)
}

// SubscribeBasketball subscribes to basketball topic
func (c *MQTTClient) SubscribeBasketball(topic string) error {
	return c.Subscribe(topic, QoSAtLeastOnce)
}

// SubscribeTennis subscribes to tennis topic
func (c *MQTTClient) SubscribeTennis(topic string) error {
	return c.Subscribe(topic, QoSAtLeastOnce)
}

// SubscribeEsports subscribes to esports topic
func (c *MQTTClient) SubscribeEsports(topic string) error {
	return c.Subscribe(topic, QoSAtLeastOnce)
}

// OnMessage registers a handler for messages on a specific topic
func (c *MQTTClient) OnMessage(topic string, handler MQTTMessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.handlers[topic] = append(c.handlers[topic], handler)
}

// OnFootballMessage registers a handler for football messages
func (c *MQTTClient) OnFootballMessage(handler MQTTMessageHandler) {
	c.OnMessage("football", handler)
}

// OnBasketballMessage registers a handler for basketball messages
func (c *MQTTClient) OnBasketballMessage(handler MQTTMessageHandler) {
	c.OnMessage("basketball", handler)
}

// OnTennisMessage registers a handler for tennis messages
func (c *MQTTClient) OnTennisMessage(handler MQTTMessageHandler) {
	c.OnMessage("tennis", handler)
}

// onConnect is called when connection is established
func (c *MQTTClient) onConnect(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

// onConnectionLost is called when connection is lost
func (c *MQTTClient) onConnectionLost(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v", err)
}

// onMessage is the default message handler
func (c *MQTTClient) onMessage(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	payload := msg.Payload()
	
	// Log received message
	log.Printf("Received message on topic %s: %d bytes", topic, len(payload))
	
	// Dispatch to registered handlers
	c.dispatchMessage(topic, payload)
}

// dispatchMessage dispatches a message to registered handlers
func (c *MQTTClient) dispatchMessage(topic string, payload []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Try exact topic match
	if handlers, exists := c.handlers[topic]; exists {
		for _, handler := range handlers {
			go handler(topic, payload)
		}
	}
	
	// Try wildcard handlers
	if handlers, exists := c.handlers["*"]; exists {
		for _, handler := range handlers {
			go handler(topic, payload)
		}
	}
	
	// Try prefix match (e.g., "football" matches "football/match/123")
	for handlerTopic, handlers := range c.handlers {
		if handlerTopic != "*" && handlerTopic != topic {
			// Simple prefix matching
			if len(topic) > len(handlerTopic) && 
			   topic[:len(handlerTopic)] == handlerTopic {
				for _, handler := range handlers {
					go handler(topic, payload)
				}
			}
		}
	}
}

// Publish publishes a message to a topic (if needed)
func (c *MQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected")
	}
	
	var data []byte
	var err error
	
	switch v := payload.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		data, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}
	}
	
	token := c.client.Publish(topic, qos, retained, data)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish: %w", token.Error())
	}
	
	return nil
}

