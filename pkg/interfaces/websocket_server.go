package interfaces

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"uof-service/pkg/common"
	"uof-service/pkg/models"
	"uof-service/pkg/processing"
)

// DefaultWebSocketServer 默认 WebSocket 服务器实现
type DefaultWebSocketServer struct {
	logger     common.Logger
	port       int
	dispatcher processing.EventDispatcher
	clients    map[*websocket.Conn]*Client
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
	server     *http.Server
}

// Client WebSocket 客户端
type Client struct {
	conn          *websocket.Conn
	subscriptions []string
	mu            sync.RWMutex
}

// NewWebSocketServer 创建 WebSocket 服务器
func NewWebSocketServer(logger common.Logger, port int, dispatcher processing.EventDispatcher) WebSocketServer {
	return &DefaultWebSocketServer{
		logger:     logger,
		port:       port,
		dispatcher: dispatcher,
		clients:    make(map[*websocket.Conn]*Client),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
	}
}

// Start 启动 WebSocket 服务器
func (s *DefaultWebSocketServer) Start(ctx context.Context) error {
	s.logger.Info("Starting WebSocket server on port %d", s.port)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	// 订阅所有事件
	s.dispatcher.Subscribe(processing.EventFilter{}, s.handleEvent)

	// 启动服务器
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("WebSocket server error: %v", err)
		}
	}()

	s.logger.Info("WebSocket server started successfully on port %d", s.port)
	return nil
}

// Stop 停止 WebSocket 服务器
func (s *DefaultWebSocketServer) Stop(ctx context.Context) error {
	s.logger.Info("Stopping WebSocket server")

	// 关闭所有客户端连接
	s.mu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[*websocket.Conn]*Client)
	s.mu.Unlock()

	// 停止服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			s.logger.Error("Failed to stop WebSocket server: %v", err)
			return err
		}
	}

	s.logger.Info("WebSocket server stopped successfully")
	return nil
}

// Broadcast 广播消息
func (s *DefaultWebSocketServer) Broadcast(ctx context.Context, message interface{}) error {
	s.logger.Debug("Broadcasting message to %d clients", len(s.clients))

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for conn := range s.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			s.logger.Error("Failed to send message to client: %v", err)
		}
	}

	return nil
}

// SendToClient 发送消息给指定客户端
func (s *DefaultWebSocketServer) SendToClient(ctx context.Context, clientID string, message interface{}) error {
	// 实现发送给指定客户端的逻辑
	// 这里简化实现,实际应该维护客户端 ID 映射
	return s.Broadcast(ctx, message)
}

// handleWebSocket 处理 WebSocket 连接
func (s *DefaultWebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		conn:          conn,
		subscriptions: make([]string, 0),
	}

	s.mu.Lock()
	s.clients[conn] = client
	s.mu.Unlock()

	s.logger.Info("New WebSocket client connected (total: %d)", len(s.clients))

	// 发送欢迎消息
	welcomeMsg := map[string]interface{}{
		"type":    "welcome",
		"message": "Connected to UOF Service WebSocket",
		"time":    time.Now(),
	}
	s.sendToConn(conn, welcomeMsg)

	// 处理客户端消息
	go s.handleClientMessages(conn, client)

	// 发送心跳
	go s.sendHeartbeat(conn)
}

// handleClientMessages 处理客户端消息
func (s *DefaultWebSocketServer) handleClientMessages(conn *websocket.Conn, client *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, conn)
		s.mu.Unlock()
		conn.Close()
		s.logger.Info("WebSocket client disconnected (total: %d)", len(s.clients))
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket error: %v", err)
			}
			break
		}

		// 解析客户端消息
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			s.logger.Error("Failed to parse client message: %v", err)
			continue
		}

		// 处理订阅请求
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "subscribe":
				if matchID, ok := msg["match_id"].(string); ok {
					client.mu.Lock()
					client.subscriptions = append(client.subscriptions, matchID)
					client.mu.Unlock()
					s.logger.Debug("Client subscribed to match: %s", matchID)
				}

			case "unsubscribe":
				if matchID, ok := msg["match_id"].(string); ok {
					client.mu.Lock()
					// 移除订阅
					newSubs := make([]string, 0)
					for _, sub := range client.subscriptions {
						if sub != matchID {
							newSubs = append(newSubs, sub)
						}
					}
					client.subscriptions = newSubs
					client.mu.Unlock()
					s.logger.Debug("Client unsubscribed from match: %s", matchID)
				}

			case "ping":
				s.sendToConn(conn, map[string]interface{}{
					"type": "pong",
					"time": time.Now(),
				})
			}
		}
	}
}

// handleEvent 处理事件并广播给客户端
func (s *DefaultWebSocketServer) handleEvent(event *models.Event) {
	s.logger.Debug("Broadcasting event: %s", event.ID)

	message := map[string]interface{}{
		"type":      "event",
		"event_id":  event.ID,
		"event_type": event.Type,
		"match_id":  event.MatchID,
		"source":    event.Source,
		"timestamp": event.Timestamp,
		"data":      event.Data,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for conn, client := range s.clients {
		// 检查客户端是否订阅了这个比赛
		client.mu.RLock()
		subscribed := len(client.subscriptions) == 0 // 没有订阅表示订阅所有
		for _, matchID := range client.subscriptions {
			if matchID == event.MatchID {
				subscribed = true
				break
			}
		}
		client.mu.RUnlock()

		if subscribed {
			s.sendToConn(conn, message)
		}
	}
}

// sendHeartbeat 发送心跳
func (s *DefaultWebSocketServer) sendHeartbeat(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		_, exists := s.clients[conn]
		s.mu.RUnlock()

		if !exists {
			return
		}

		heartbeat := map[string]interface{}{
			"type": "heartbeat",
			"time": time.Now(),
		}

		if err := s.sendToConn(conn, heartbeat); err != nil {
			return
		}
	}
}

// sendToConn 发送消息到连接
func (s *DefaultWebSocketServer) sendToConn(conn *websocket.Conn, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

