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
	autoBooking         *services.AutoBookingService
	autoBookingController *services.AutoBookingController
	srMapper            *services.SRMapper
	producerMonitor     *services.ProducerMonitor
	marketDescService   *services.MarketDescriptionsService
	matchStatusService  *services.MatchStatusService
	subscriptionSync    *services.SubscriptionSyncService
	messageHistoryService *services.MessageHistoryService
	marketQueryService  *services.MarketQueryService
	queryCache          *services.QueryCache
	sportradarAPIClient *services.SportradarAPIClient
	httpServer          *http.Server
	upgrader            websocket.Upgrader
}

func NewServer(cfg *config.Config, db *sql.DB, hub *Hub, larkNotifier *services.LarkNotifier, marketDescService *services.MarketDescriptionsService) *Server {
	// 创建Replay客户端(如果access token可用)
	var replayClient *services.ReplayClient
	if cfg.AccessToken != "" {
		replayClient = services.NewReplayClient(cfg.AccessToken, cfg.APIBaseURL)
		log.Println("[Server] Replay client initialized with access token")
	} else {
		log.Println("[Server] ⚠️  Replay client not initialized - BETRADAR_ACCESS_TOKEN not set")
	}
	
	// 创建自动订阅服务和控制器
	autoBooking := services.NewAutoBookingService(cfg, db, larkNotifier)
	autoBookingController := services.NewAutoBookingController(cfg, autoBooking)
	
	// 创建 Sportradar API 客户端
	sportradarAPIClient := services.NewSportradarAPIClient(cfg.APIBaseURL, cfg.AccessToken)
	log.Println("[Server] Sportradar API client initialized")
	
	return &Server{
		config:          cfg,
		db:              db,
		wsHub:           hub,
		messageStore:    services.NewMessageStore(db),
		recoveryManager: services.NewRecoveryManager(cfg, services.NewMessageStore(db)),
		srMapper:        services.NewSRMapper(),
		replayClient:    replayClient,
		larkNotifier:      larkNotifier,
		autoBooking:       autoBooking,
		autoBookingController: autoBookingController,
		producerMonitor:   services.NewProducerMonitor(db, larkNotifier, cfg.ProducerCheckIntervalSeconds, cfg.ProducerDownThresholdSeconds),
		marketDescService: marketDescService,
		matchStatusService: services.NewMatchStatusService(cfg),
		subscriptionSync:  services.NewSubscriptionSyncService(db, cfg.AccessToken, cfg.APIBaseURL, cfg.SubscriptionSyncIntervalMinutes),
		messageHistoryService: services.NewMessageHistoryService(db),
		marketQueryService: services.NewMarketQueryService(db),
		queryCache:      services.NewQueryCache(30 * time.Second), // 30秒缓存
		sportradarAPIClient: sportradarAPIClient,
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
	// 启动 Market Descriptions Service
	if err := s.marketDescService.Start(); err != nil {
		log.Printf("[Server] ⚠️  Failed to start Market Descriptions Service: %v", err)
		log.Println("[Server] Continuing with fallback market names...")
	} else {
			status := s.marketDescService.GetStatus()
			log.Printf("[Server] ✅ Market Descriptions Service started. Status: %s", status)
	}
	
	// 启动 Match Status Service
	if err := s.matchStatusService.Start(); err != nil {
		log.Printf("[Server] ⚠️  Failed to start Match Status Service: %v", err)
		log.Println("[Server] Continuing with fallback match status names...")
	} else {
		log.Println("[Server] ✅ Match Status Service started")
	}
	
	// 启动 Subscription Sync Service
	if err := s.subscriptionSync.Start(); err != nil {
		log.Printf("[Server] ⚠️  Failed to start Subscription Sync Service: %v", err)
	} else {
		log.Printf("[Server] ✅ Subscription Sync Service started (interval: %d minutes)", s.config.SubscriptionSyncIntervalMinutes)
	}
	
	// 启动自动订阅控制器（如果启用）
	if s.config.AutoBookingEnabled {
		s.autoBookingController.Start()
		log.Printf("[Server] ✅ Auto-booking controller started (interval: %d minutes)", s.config.AutoBookingIntervalMinutes)
	} else {
		log.Println("[Server] ⚠️  Auto-booking is disabled (use API to enable)")
	}
	
	router := mux.NewRouter()

	// API路由
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/messages", s.handleGetMessages).Methods("GET")
	// 增强版 events API - 包含完整信息和盘口
	api.HandleFunc("/events", s.handleGetEnhancedEvents).Methods("GET")
	// 旧版 API 保留为 /events/simple
	api.HandleFunc("/events/simple", s.handleGetTrackedEvents).Methods("GET")
	api.HandleFunc("/stats", s.handleGetStats).Methods("GET")
	
	// 恢复API
	api.HandleFunc("/recovery/trigger", s.handleTriggerRecovery).Methods("POST")
	api.HandleFunc("/recovery/event/{event_id}", s.handleTriggerEventRecovery).Methods("POST")
	api.HandleFunc("/recovery/status", s.handleGetRecoveryStatus).Methods("GET")
	api.HandleFunc("/recovery/fixtures", s.handleTriggerFixtureRecovery).Methods("POST")
	api.HandleFunc("/recovery/stateful/{event_id}", s.handleTriggerStatefulRecovery).Methods("POST")
	
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
	api.HandleFunc("/booking/trigger", s.handleTriggerAutoBooking).Methods("POST")
	
	// 订阅查询API
	api.HandleFunc("/booking/booked", s.handleGetBookedMatches).Methods("GET")
	api.HandleFunc("/booking/bookable", s.handleGetBookableMatches).Methods("GET")
	
	// 订阅同步API
	api.HandleFunc("/booking/sync", s.SyncSubscriptionsHandler).Methods("POST")
	
	// 自动订阅配置API
	autoBookingHandler := NewAutoBookingHandler(s.autoBookingController)
	api.HandleFunc("/auto-booking/status", autoBookingHandler.GetStatus).Methods("GET")
	api.HandleFunc("/auto-booking/enable", autoBookingHandler.Enable).Methods("POST")
	api.HandleFunc("/auto-booking/disable", autoBookingHandler.Disable).Methods("POST")
	api.HandleFunc("/auto-booking/interval", autoBookingHandler.SetInterval).Methods("POST")
	
	// 消息历史API
	messageHistoryHandler := NewMessageHistoryHandler(s.messageHistoryService)
	api.HandleFunc("/messages/recent", messageHistoryHandler.GetRecentMessages).Methods("GET")
	api.HandleFunc("/events/{event_id}/messages", messageHistoryHandler.GetEventMessages).Methods("GET")
	
	// 市场查询API
	marketQueryHandler := NewMarketQueryHandler(s.marketQueryService)
	api.HandleFunc("/events/{event_id}/markets", marketQueryHandler.GetEventMarkets).Methods("GET")
	
	// 赛事详情API - 包含所有市场、specifier和outcomes
	api.HandleFunc("/events/{eventId}", s.handleGetEventDetail).Methods("GET")
	
	// Pre-match API
	api.HandleFunc("/prematch/trigger", s.handleTriggerPrematchBooking).Methods("POST")
	api.HandleFunc("/prematch/events", s.handleGetPrematchEvents).Methods("GET")
	api.HandleFunc("/prematch/stats", s.handleGetPrematchStats).Methods("GET")
	
	// 前端API - 比赛查询
	api.HandleFunc("/events", s.handleGetEventsWithFilters).Methods("GET") // 统一的筛选接口
	api.HandleFunc("/matches/live", s.handleGetLiveMatches).Methods("GET")
	api.HandleFunc("/matches/upcoming", s.handleGetUpcomingMatches).Methods("GET")
	api.HandleFunc("/matches/status", s.handleGetMatchesByStatus).Methods("GET")
	api.HandleFunc("/matches/search", s.handleSearchMatches).Methods("GET")
	api.HandleFunc("/matches/{event_id}", s.handleGetMatchDetail).Methods("GET")
	
	// 联赛API
api.HandleFunc("/leagues", s.handleGetLeagues).Methods("GET")
	api.HandleFunc("/categories", s.handleGetCategories).Methods("GET")
	api.HandleFunc("/tournaments", s.handleGetTournaments).Methods("GET")
	
	// 盘口赔率API
	api.HandleFunc("/odds/all", s.handleGetAllBookedMarketsOdds).Methods("GET")
	api.HandleFunc("/odds/{event_id}/markets", s.handleGetEventMarkets).Methods("GET")
	api.HandleFunc("/odds/{event_id}/{market_id}", s.handleGetMarketOdds).Methods("GET")
	api.HandleFunc("/odds/{event_id}/{market_id}/{outcome_id}/history", s.handleGetOddsHistory).Methods("GET")
	
	// IP 查询API
	api.HandleFunc("/ip", s.handleGetIP).Methods("GET")
	
	// Producer 监控API
	api.HandleFunc("/producer/status", s.handleGetProducerStatus).Methods("GET")
	api.HandleFunc("/producer/bet-acceptance", s.handleGetBetAcceptance).Methods("GET")
	
	// Market Descriptions API
	marketDescHandler := NewMarketDescriptionsHandler(s.marketDescService)
	api.HandleFunc("/market-descriptions/status", marketDescHandler.HandleGetStatus).Methods("GET")
	api.HandleFunc("/market-descriptions/refresh", marketDescHandler.HandleForceRefresh).Methods("POST")
	api.HandleFunc("/market-descriptions/bulk-update", marketDescHandler.HandleBulkUpdate).Methods("POST")
	
	// 盘口分组配置 API (新增 - v2.0 技术方案)
	marketTabsHandler := NewMarketTabsHandler()
	api.HandleFunc("/config/market-tabs", marketTabsHandler.HandleGetMarketTabs).Methods("GET")
	
	// 数据清理 API
	api.HandleFunc("/cleanup/stats", s.handleGetTableStats).Methods("GET")
	api.HandleFunc("/cleanup/manual", s.handleManualCleanup).Methods("POST")
	
	// 数据库重置API（危险操作，需要确认）
	api.HandleFunc("/database/reset", s.handleResetDatabase).Methods("POST")
	
	// 比赛记录查询 API
	api.HandleFunc("/match/records", s.handleGetMatchRecords).Methods("GET")
	api.HandleFunc("/record/detail", s.handleGetRecordDetail).Methods("GET")
	
	// LD and TheSports APIs removed - using UOF only
	
	// Subscription management API removed - no longer using subscription manager

	// WebSocket路由
	router.HandleFunc("/ws", s.handleWebSocket)

	// 监控前端静态文件
	router.PathPrefix("/monitor/").Handler(http.StripPrefix("/monitor/", http.FileServer(http.Dir("./static/monitor"))))

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

// LD and TheSports client setters removed - using UOF only

// SetSubscriptionManager removed - no longer using subscription manager

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



// handleGetProducerStatus 获取所有 Producer 的健康状态
func (s *Server) handleGetProducerStatus(w http.ResponseWriter, r *http.Request) {
	statuses, err := s.producerMonitor.GetProducerStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"producers": statuses,
	})
}

// handleGetBetAcceptance 检查是否可以接受投注
func (s *Server) handleGetBetAcceptance(w http.ResponseWriter, r *http.Request) {
	canAccept, reason := s.producerMonitor.CanAcceptBets()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"can_accept_bets": canAccept,
		"reason":          reason,
	})
}



// handleTriggerFixtureRecovery 触发 Fixture 变更恢复
func (s *Server) handleTriggerFixtureRecovery(w http.ResponseWriter, r *http.Request) {
	// 获取 after 参数（Unix timestamp 秒）
	afterStr := r.URL.Query().Get("after")
	var after int64
	
	if afterStr != "" {
		var err error
		after, err = strconv.ParseInt(afterStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid 'after' parameter: must be Unix timestamp in seconds", http.StatusBadRequest)
			return
		}
	} else {
		// 默认恢复最近 10 小时的数据
		after = time.Now().Add(-10 * time.Hour).Unix()
	}
	
	log.Printf("Fixture recovery triggered via API (after: %d)", after)
	
	var fixtureChanges []services.FixtureChange
	var recoveryErr error
	
	// 同步执行以便返回结果
	fixtureChanges, recoveryErr = s.recoveryManager.TriggerFixtureRecovery(after)
	
	if recoveryErr != nil {
		log.Printf("Fixture recovery failed: %v", recoveryErr)
		http.Error(w, fmt.Sprintf("Fixture recovery failed: %v", recoveryErr), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Fixture recovery completed: %d changes retrieved", len(fixtureChanges))
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Fixture recovery completed",
		"after":   after,
		"count":   len(fixtureChanges),
		"changes": fixtureChanges,
		"time":    time.Now().Unix(),
	})
}

// handleTriggerStatefulRecovery 触发单个事件的 Stateful Messages 恢复
func (s *Server) handleTriggerStatefulRecovery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	
	// 获取 product 参数（默认为 liveodds）
	product := r.URL.Query().Get("product")
	if product == "" {
		product = "liveodds"
	}
	
	log.Printf("Stateful messages recovery triggered for %s (product: %s)", eventID, product)
	
	go func() {
		if err := s.recoveryManager.TriggerStatefulMessagesRecovery(product, eventID); err != nil {
			log.Printf("Stateful messages recovery failed: %v", err)
		} else {
			log.Printf("Stateful messages recovery completed for %s", eventID)
		}
	}()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "accepted",
		"message":  "Stateful messages recovery request accepted and processing",
		"event_id": eventID,
		"product":  product,
		"time":     time.Now().Unix(),
	})
}

