package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	config              *config.Config
	db                  *sql.DB
	wsHub               *Hub
	messageStore        *services.MessageStore
	recoveryManager     *services.RecoveryManager
	replayClient        *services.ReplayClient
	larkNotifier        *services.LarkNotifier
	ldClient            *services.LDClient
	theSportsClient     *services.TheSportsClient
	autoBooking         *services.AutoBookingService
	subscriptionManager *services.MatchSubscriptionManager
	httpServer          *http.Server
	upgrader            websocket.Upgrader
}

func NewServer(cfg *config.Config, db *sql.DB, hub *Hub, larkNotifier *services.LarkNotifier) *Server {
	// 创建Replay客户端(如果access token可用)
	var replayClient *services.ReplayClient
	if cfg.AccessToken != "" {
		replayClient = services.NewReplayClient(cfg.AccessToken)
		log.Println("[Server] Replay client initialized with access token")
	} else {
		log.Println("[Server] ⚠️  Replay client not initialized - BETRADAR_ACCESS_TOKEN not set")
	}
	
	return &Server{
		config:          cfg,
		db:              db,
		wsHub:           hub,
		messageStore:    services.NewMessageStore(db),
		recoveryManager: services.NewRecoveryManager(cfg, services.NewMessageStore(db)),
		replayClient:    replayClient,
		larkNotifier:    larkNotifier,
		autoBooking:     services.NewAutoBookingService(cfg, larkNotifier),
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
	
	// 恢复API
	api.HandleFunc("/recovery/trigger", s.handleTriggerRecovery).Methods("POST")
	api.HandleFunc("/recovery/event/{event_id}", s.handleTriggerEventRecovery).Methods("POST")
	api.HandleFunc("/recovery/status", s.handleGetRecoveryStatus).Methods("GET")
	
	// Replay API
	api.HandleFunc("/replay/start", s.handleReplayStart).Methods("POST")
	api.HandleFunc("/replay/stop", s.handleReplayStop).Methods("POST")
	api.HandleFunc("/replay/status", s.handleReplayStatus).Methods("GET")
	api.HandleFunc("/replay/list", s.handleReplayList).Methods("GET")
	
	// 监控API
	api.HandleFunc("/monitor/trigger", s.handleTriggerMonitor).Methods("POST")
	
	// 自动订阅API
	api.HandleFunc("/booking/auto", s.handleAutoBooking).Methods("POST")
	api.HandleFunc("/booking/match/{match_id}", s.handleBookMatch).Methods("POST")
	
	// IP 查询API
	api.HandleFunc("/ip", s.handleGetIP).Methods("GET")
	
	// Live Data API
	api.HandleFunc("/ld/connect", s.handleLDConnect).Methods("POST")
	api.HandleFunc("/ld/disconnect", s.handleLDDisconnect).Methods("POST")
	api.HandleFunc("/ld/status", s.handleLDStatus).Methods("GET")
	api.HandleFunc("/ld/subscribe", s.handleLDSubscribeMatch).Methods("POST")
	api.HandleFunc("/ld/unsubscribe", s.handleLDUnsubscribeMatch).Methods("POST")
	api.HandleFunc("/ld/matches", s.handleLDGetMatches).Methods("GET")
	api.HandleFunc("/ld/events", s.handleLDGetEvents).Methods("GET")
	
	// The Sports API - 足球
	api.HandleFunc("/thesports/connect", s.handleTheSportsConnect).Methods("POST")
	api.HandleFunc("/thesports/disconnect", s.handleTheSportsDisconnect).Methods("POST")
	api.HandleFunc("/thesports/status", s.handleTheSportsStatus).Methods("GET")
	api.HandleFunc("/thesports/subscribe", s.handleTheSportsSubscribeMatch).Methods("POST")
	api.HandleFunc("/thesports/unsubscribe", s.handleTheSportsUnsubscribeMatch).Methods("POST")
	api.HandleFunc("/thesports/today", s.handleTheSportsGetTodayMatches).Methods("GET")
	api.HandleFunc("/thesports/live", s.handleTheSportsGetLiveMatches).Methods("GET")
	
	// The Sports API - 篮球
	api.HandleFunc("/thesports/basketball/today", s.handleTheSportsGetBasketballToday).Methods("GET")
	api.HandleFunc("/thesports/basketball/live", s.handleTheSportsGetBasketballLive).Methods("GET")
	
	// 订阅管理 API
	api.HandleFunc("/subscriptions", s.handleGetSubscriptions).Methods("GET")
	api.HandleFunc("/subscriptions/stats", s.handleGetSubscriptionStats).Methods("GET")
	api.HandleFunc("/subscriptions/unsubscribe", s.handleUnsubscribeMatch).Methods("POST")
	api.HandleFunc("/subscriptions/unsubscribe/batch", s.handleUnsubscribeMatches).Methods("POST")
	api.HandleFunc("/subscriptions/cleanup", s.handleCleanupEndedMatches).Methods("POST")

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

// SetLDClient 设置 LD 客户端
func (s *Server) SetLDClient(client *services.LDClient) {
	s.ldClient = client
}

// SetTheSportsClient 设置 The Sports 客户端
func (s *Server) SetTheSportsClient(client *services.TheSportsClient) {
	s.theSportsClient = client
}

// SetSubscriptionManager 设置订阅管理器
func (s *Server) SetSubscriptionManager(manager *services.MatchSubscriptionManager) {
	s.subscriptionManager = manager
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



// handleTriggerRecovery 手动触发全量恢复
func (s *Server) handleTriggerRecovery(w http.ResponseWriter, r *http.Request) {
	log.Println("Manual recovery triggered via API")
	
	go func() {
		if err := s.recoveryManager.TriggerFullRecovery(); err != nil {
			log.Printf("Manual recovery failed: %v", err)
		} else {
			log.Println("Manual recovery completed successfully")
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "accepted",
		"message": "Recovery request accepted and processing",
		"time":    time.Now().Unix(),
	})
}

// handleTriggerEventRecovery 触发单个赛事的恢复
func (s *Server) handleTriggerEventRecovery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	
	// 获取product参数（默认为liveodds）
	product := r.URL.Query().Get("product")
	if product == "" {
		product = "liveodds"
	}
	
	log.Printf("Manual event recovery triggered for %s (product: %s)", eventID, product)
	
	go func() {
		// 触发赔率恢复
		if err := s.recoveryManager.TriggerEventRecovery(product, eventID); err != nil {
			log.Printf("Event recovery failed: %v", err)
		}
		
		// 触发状态消息恢复
		if err := s.recoveryManager.TriggerStatefulMessagesRecovery(product, eventID); err != nil {
			log.Printf("Stateful messages recovery failed: %v", err)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "accepted",
		"message":  "Event recovery request accepted and processing",
		"event_id": eventID,
		"product":  product,
		"time":     time.Now().Unix(),
	})
}



// handleGetRecoveryStatus 获取恢复状态
func (s *Server) handleGetRecoveryStatus(w http.ResponseWriter, r *http.Request) {
	// 获取limit参数（默认20）
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	statuses, err := s.messageStore.GetRecoveryStatus(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"count":     len(statuses),
		"recoveries": statuses,
	})
}



// handleReplayStart 启动重放测试
func (s *Server) handleReplayStart(w http.ResponseWriter, r *http.Request) {
	if s.replayClient == nil {
		http.Error(w, "Replay client not configured. Please set UOF_USERNAME and UOF_PASSWORD", http.StatusServiceUnavailable)
		return
	}
	
	// 解析请求体
	var req struct {
		EventID            string `json:"event_id"`
		Speed              int    `json:"speed,omitempty"`
		Duration           int    `json:"duration,omitempty"`
		NodeID             int    `json:"node_id,omitempty"`
		MaxDelay           int    `json:"max_delay,omitempty"`
		UseReplayTimestamp bool   `json:"use_replay_timestamp,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// 验证必需参数
	if req.EventID == "" {
		http.Error(w, "event_id is required", http.StatusBadRequest)
		return
	}
	
	// 设置默认值
	if req.Speed == 0 {
		req.Speed = 20
	}
	if req.NodeID == 0 {
		req.NodeID = 1
	}
	if req.MaxDelay == 0 {
		req.MaxDelay = 10000
	}
	
	log.Printf("🎬 Starting replay via API: event=%s, speed=%dx, node_id=%d", 
		req.EventID, req.Speed, req.NodeID)
	
	// 异步启动重放
	go func() {
		// 使用QuickReplay方法,它包含正确的等待和验证逻辑
		if err := s.replayClient.QuickReplay(req.EventID, req.Speed, req.NodeID); err != nil {
			log.Printf("❌ Failed to start replay: %v", err)
			return
		}
		
		log.Printf("✅ Replay started successfully: %s", req.EventID)
		
		// 5. 如果指定了duration,自动停止
		if req.Duration > 0 {
			log.Printf("⏱️  Replay will run for %d seconds", req.Duration)
			time.Sleep(time.Duration(req.Duration) * time.Second)
			
			if err := s.replayClient.Stop(); err != nil {
				log.Printf("⚠️  Failed to stop replay: %v", err)
			} else {
				log.Printf("🛑 Replay stopped after %d seconds", req.Duration)
			}
		}
	}()
	
	// 立即返回响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "accepted",
		"message": "Replay request accepted and processing",
		"event_id": req.EventID,
		"speed":   req.Speed,
		"node_id": req.NodeID,
		"duration": req.Duration,
		"time":    time.Now().Unix(),
	})
}

// handleReplayStop 停止重放
func (s *Server) handleReplayStop(w http.ResponseWriter, r *http.Request) {
	if s.replayClient == nil {
		http.Error(w, "Replay client not configured", http.StatusServiceUnavailable)
		return
	}
	
	log.Println("🛑 Stopping replay via API...")
	
	if err := s.replayClient.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Println("✅ Replay stopped successfully")
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Replay stopped",
		"time":    time.Now().Unix(),
	})
}

// handleReplayStatus 获取重放状态
func (s *Server) handleReplayStatus(w http.ResponseWriter, r *http.Request) {
	if s.replayClient == nil {
		http.Error(w, "Replay client not configured", http.StatusServiceUnavailable)
		return
	}
	
	status, err := s.replayClient.GetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(status))
}

// handleReplayList 列出重放列表
func (s *Server) handleReplayList(w http.ResponseWriter, r *http.Request) {
	if s.replayClient == nil {
		http.Error(w, "Replay client not configured", http.StatusServiceUnavailable)
		return
	}
	
	events, err := s.replayClient.ListEvents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/xml")
	w.Write([]byte(events))
}



// handleTriggerMonitor 手动触发监控检查
func (s *Server) handleTriggerMonitor(w http.ResponseWriter, r *http.Request) {
	log.Println("📊 Manual monitor check triggered via API...")
	
	// 创建 MatchMonitor 并执行检查
	monitor := services.NewMatchMonitor(s.config, nil)
	
	// 异步执行监控检查
	go monitor.CheckAndReportWithNotifier(s.larkNotifier)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "triggered",
		"message": "Monitor check triggered. Results will be sent to Feishu webhook.",
		"time":    time.Now().Unix(),
	})
}



// handleGetIP 获取服务器出口 IP 地址
func (s *Server) handleGetIP(w http.ResponseWriter, r *http.Request) {
	// 查询外部 IP 服务
	resp, err := http.Get("https://api.ipify.org?format=text")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get IP: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	
	ipBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read IP: %v", err), http.StatusInternalServerError)
		return
	}
	
	ip := string(ipBytes)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ip":      ip,
		"message": "This is your Railway service's public IP address. Use this for Sportradar Live Data whitelist.",
		"time":    time.Now().Unix(),
	})
}



// handleAutoBooking 自动订阅所有 bookable 比赛
func (s *Server) handleAutoBooking(w http.ResponseWriter, r *http.Request) {
	log.Println("[API] Auto booking triggered...")
	
	go func() {
		bookable, success, err := s.autoBooking.BookAllBookableMatches()
		if err != nil {
			log.Printf("[API] Auto booking failed: %v", err)
		} else {
			log.Printf("[API] Auto booking completed: %d bookable, %d success", bookable, success)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "triggered",
		"message": "Auto booking process started. Check Feishu for results.",
		"time":    time.Now().Unix(),
	})
}

// handleBookMatch 订阅单个比赛
func (s *Server) handleBookMatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	matchID := vars["match_id"]
	
	if matchID == "" {
		http.Error(w, "match_id is required", http.StatusBadRequest)
		return
	}
	
	log.Printf("[API] Booking match: %s", matchID)
	
	go func() {
		if err := s.autoBooking.BookMatch(matchID); err != nil {
			log.Printf("[API] Failed to book match %s: %v", matchID, err)
		} else {
			log.Printf("[API] Successfully booked match: %s", matchID)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "triggered",
		"message":  fmt.Sprintf("Booking request sent for match %s", matchID),
		"match_id": matchID,
		"time":     time.Now().Unix(),
	})
}

