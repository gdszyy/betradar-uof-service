package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"

	"uof-service/config"
	"uof-service/services"
)

type Server struct {
	config       *config.Config
	db           *sql.DB
	wsHub        *Hub
	messageStore *services.MessageStore
	httpServer   *http.Server
	upgrader     websocket.Upgrader
}

func NewServer(cfg *config.Config, db *sql.DB, hub *Hub) *Server {
	return &Server{
		config:       cfg,
		db:           db,
		wsHub:        hub,
		messageStore: services.NewMessageStore(db),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源(生产环境需要限制)
			},
		},
	}
}

func (s *Server) Start() error {
	router := mux.NewRouter()

	// API路由
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/messages", s.handleGetMessages).Methods("GET")
	api.HandleFunc("/events", s.handleGetTrackedEvents).Methods("GET")
	api.HandleFunc("/events/{event_id}/messages", s.handleGetEventMessages).Methods("GET")
	api.HandleFunc("/stats", s.handleGetStats).Methods("GET")

	// WebSocket路由
	router.HandleFunc("/ws", s.handleWebSocket)

	// 静态文件(如果需要)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	// CORS配置
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	s.httpServer = &http.Server{
		Addr:         ":" + s.config.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

// handleHealth 健康检查
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// handleGetMessages 获取消息列表
func (s *Server) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	eventID := query.Get("event_id")
	messageType := query.Get("message_type")

	messages, err := s.messageStore.GetMessages(limit, offset, eventID, messageType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"limit":    limit,
		"offset":   offset,
	})
}

// handleGetTrackedEvents 获取跟踪的赛事
func (s *Server) handleGetTrackedEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.messageStore.GetTrackedEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
	})
}

// handleGetEventMessages 获取特定赛事的消息
func (s *Server) handleGetEventMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	messages, err := s.messageStore.GetEventMessages(eventID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"event_id": eventID,
		"messages": messages,
	})
}

// handleGetStats 获取统计信息
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		TotalMessages   int `json:"total_messages"`
		TotalEvents     int `json:"total_events"`
		OddsChanges     int `json:"odds_changes"`
		BetStops        int `json:"bet_stops"`
		BetSettlements  int `json:"bet_settlements"`
	}

	s.db.QueryRow("SELECT COUNT(*) FROM uof_messages").Scan(&stats.TotalMessages)
	s.db.QueryRow("SELECT COUNT(*) FROM tracked_events").Scan(&stats.TotalEvents)
	s.db.QueryRow("SELECT COUNT(*) FROM odds_changes").Scan(&stats.OddsChanges)
	s.db.QueryRow("SELECT COUNT(*) FROM bet_stops").Scan(&stats.BetStops)
	s.db.QueryRow("SELECT COUNT(*) FROM bet_settlements").Scan(&stats.BetSettlements)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleWebSocket WebSocket连接处理
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:      s.wsHub,
		conn:     conn,
		send:     make(chan []byte, 256),
		filters:  make(map[string]bool),
		eventIDs: make(map[string]bool),
	}

	client.hub.register <- client

	// 发送欢迎消息
	welcomeMsg := &WSMessage{
		Type: "connected",
		Data: map[string]interface{}{
			"message": "Connected to UOF WebSocket",
			"time":    time.Now().Unix(),
		},
	}
	welcomeData, _ := json.Marshal(welcomeMsg)
	client.send <- welcomeData

	go client.writePump()
	go client.readPump()
}

