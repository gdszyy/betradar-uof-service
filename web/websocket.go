package web

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// WSMessage WebSocket消息结构
type WSMessage struct {
	Type        string  `json:"type"`
	MessageType string  `json:"message_type,omitempty"`
	EventID     string  `json:"event_id,omitempty"`
	ProductID   *int    `json:"product_id,omitempty"`
	RoutingKey  string  `json:"routing_key,omitempty"`
	XML         string  `json:"xml,omitempty"`
	Timestamp   int64   `json:"timestamp,omitempty"`
	Data        interface{} `json:"data,omitempty"`
}

// Client WebSocket客户端
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	filters  map[string]bool // 消息类型过滤器
	eventIDs map[string]bool // 赛事ID过滤器
}

// Hub WebSocket Hub
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *WSMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub 创建新的Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *WSMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 运行Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("Client unregistered. Total clients: %d", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// 检查过滤器
				if !client.shouldReceive(message) {
					continue
				}

				select {
				case client.send <- h.marshalMessage(message):
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					close(client.send)
					delete(h.clients, client)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast 广播消息（实现MessageBroadcaster接口）
func (h *Hub) Broadcast(message interface{}) {
	// 如果是WSMessage类型，直接使用
	if wsMsg, ok := message.(*WSMessage); ok {
		h.broadcast <- wsMsg
		return
	}
	
	// 如果是map类型，转换为WSMessage
	if msgMap, ok := message.(map[string]interface{}); ok {
		wsMsg := &WSMessage{}
		
		if v, ok := msgMap["type"].(string); ok {
			wsMsg.Type = v
		}
		if v, ok := msgMap["message_type"].(string); ok {
			wsMsg.MessageType = v
		}
		if v, ok := msgMap["event_id"].(string); ok {
			wsMsg.EventID = v
		}
		if v, ok := msgMap["product_id"].(*int); ok {
			wsMsg.ProductID = v
		}
		if v, ok := msgMap["routing_key"].(string); ok {
			wsMsg.RoutingKey = v
		}
		if v, ok := msgMap["xml"].(string); ok {
			wsMsg.XML = v
		}
		if v, ok := msgMap["timestamp"].(int64); ok {
			wsMsg.Timestamp = v
		}
		if v, ok := msgMap["data"]; ok {
			wsMsg.Data = v
		}
		
		h.broadcast <- wsMsg
	}
}

// marshalMessage 序列化消息
func (h *Hub) marshalMessage(message *WSMessage) []byte {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return []byte("{}")
	}
	return data
}

// shouldReceive 检查客户端是否应该接收消息
func (c *Client) shouldReceive(message *WSMessage) bool {
	// 如果没有设置过滤器,接收所有消息
	if len(c.filters) == 0 && len(c.eventIDs) == 0 {
		return true
	}

	// 检查消息类型过滤器
	if len(c.filters) > 0 {
		if _, ok := c.filters[message.MessageType]; !ok {
			return false
		}
	}

	// 检查赛事ID过滤器
	if len(c.eventIDs) > 0 {
		if message.EventID == "" {
			return false
		}
		if _, ok := c.eventIDs[message.EventID]; !ok {
			return false
		}
	}

	return true
}

// readPump 读取客户端消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// 处理客户端消息(设置过滤器等)
		c.handleMessage(message)
	}
}

// writePump 向客户端写入消息
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// handleMessage 处理客户端发送的消息
func (c *Client) handleMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Failed to unmarshal client message: %v", err)
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "subscribe":
		// 订阅特定消息类型
		if filters, ok := msg["message_types"].([]interface{}); ok {
			c.filters = make(map[string]bool)
			for _, f := range filters {
				if filter, ok := f.(string); ok {
					c.filters[filter] = true
				}
			}
		}

		// 订阅特定赛事
		if eventIDs, ok := msg["event_ids"].([]interface{}); ok {
			c.eventIDs = make(map[string]bool)
			for _, e := range eventIDs {
				if eventID, ok := e.(string); ok {
					c.eventIDs[eventID] = true
				}
			}
		}

		log.Printf("Client subscribed with filters: %v, events: %v", c.filters, c.eventIDs)

	case "unsubscribe":
		// 取消订阅
		c.filters = make(map[string]bool)
		c.eventIDs = make(map[string]bool)
		log.Println("Client unsubscribed")
	}
}

