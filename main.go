package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uof-service/config"
	"uof-service/database"
	"uof-service/logger"
	"uof-service/services"
	"uof-service/web"
)

func PreloadPlayers(playersService *services.PlayersService, scheduleService *services.ScheduleService) error {
	logger.Println("[PlayersService] Starting player preload...")
	
	// 1. 获取未来 3 天的比赛列表
	eventIDs, err := scheduleService.FetchUpcomingSchedule()
	if err != nil {
		return fmt.Errorf("failed to fetch upcoming schedule: %w", err)
	}
	
	// 2. 遍历比赛,获取阵容信息
	var allPlayers []services.PlayerInfo
	for _, eventID := range eventIDs {
		players, err := scheduleService.FetchSportEventSummary(eventID)
		if err != nil {
			logger.Printf("[PlayersService] Failed to fetch summary for event %s: %v", eventID, err)
			continue
		}
		allPlayers = append(allPlayers, players...)
	}
	
	// 3. 批量预加载球员信息
	playersService.PreloadPlayers(allPlayers)
	
	logger.Printf("[PlayersService] Player preload finished. Total unique players found: %d", len(allPlayers))
	return nil
}

func schedulePlayerPreload(playersService *services.PlayersService, scheduleService *services.ScheduleService) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		if err := PreloadPlayers(playersService, scheduleService); err != nil {
			logger.Errorf("[PlayersService] Failed to run scheduled player preload: %v", err)
		}
	}
}

func main() {
	logger.Println("Starting Betradar UOF Service...")

	// 加载配置
	cfg := config.Load()

	// 连接数据库
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 运行数据库迁移
	if err := database.Migrate(db); err != nil {
		logger.Fatalf("Failed to migrate database: %v", err)
	}

	logger.Println("Database connected and migrated")

	// 创建 Feishu 通知器
	larkNotifier := services.NewLarkNotifier(cfg.LarkWebhook)

	// 创建消息存储服务
	messageStore := services.NewMessageStore(db)
	
	// 创建 Players 服务
	playersService := services.NewPlayersService(cfg.AccessToken, cfg.APIBaseURL, db)
	if err := playersService.Start(); err != nil {
		logger.Errorf("[PlayersService] Failed to start: %v", err)
	}
	
	// 创建 Schedule 服务
	scheduleService := services.NewScheduleService(db, cfg.AccessToken, cfg.APIBaseURL)
	
	// 启动时立即执行一次球员信息预加载
	if err := PreloadPlayers(playersService, scheduleService); err != nil {
		logger.Errorf("[PlayersService] Failed to preload players: %v", err)
	}
	
	// 定时更新球员信息
	go schedulePlayerPreload(playersService, scheduleService)
	
	// 启动 Schedule 服务
	if err := scheduleService.Start(); err != nil {
		logger.Errorf("[Schedule] Failed to start: %v", err)
	} else {
		logger.Println("[Schedule] Schedule service started")
	}
	
	// 创建 Market Descriptions 服务
	marketDescService := services.NewMarketDescriptionsService(cfg.AccessToken, cfg.APIBaseURL)
	marketDescService.SetDatabase(db)
	marketDescService.SetPlayersService(playersService)
	if err := marketDescService.Start(); err != nil {
		logger.Errorf("[MarketDescService] Failed to start: %v", err)
	} else {
		logger.Println("[MarketDescService] Market descriptions service started")
	}
	
	// 创建 Market Groups 服务
	marketGroupsService := services.NewMarketGroupsService(db)
	
	// 同步 groups 字段
	if syncedCount, err := marketGroupsService.SyncMarketGroupsFromDescriptions(); err != nil {
		logger.Errorf("[MarketGroupsService] Failed to sync groups: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] Synced groups for %d markets", syncedCount)
	}
	
	// 基于 groups 分配 tabs
	if assignedCount, err := marketGroupsService.AssignTabsBasedOnGroups(); err != nil {
		logger.Errorf("[MarketGroupsService] Failed to assign tabs based on groups: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] Assigned tabs for %d markets based on groups", assignedCount)
	}
	
	// 基于 specifiers 分配 tabs
	if assignedCount, err := marketGroupsService.AssignTabsBasedOnSpecifiers(); err != nil {
		logger.Errorf("[MarketGroupsService] Failed to assign tabs based on specifiers: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] Assigned tabs for %d markets based on specifiers", assignedCount)
	}
	
	// 为未分配的市场分配默认 tab
	if assignedCount, err := marketGroupsService.AssignDefaultTab("regular_play"); err != nil {
		logger.Errorf("[MarketGroupsService] Failed to assign default tab: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] Assigned default tab 'regular_play' for %d markets", assignedCount)
	}
	
	// 创建 Producer 监控服务
	producerMonitor := services.NewProducerMonitor(db, larkNotifier, cfg.ProducerCheckIntervalSeconds, cfg.ProducerDownThresholdSeconds)
	go producerMonitor.Start()

	// 创建WebSocket Hub
	wsHub := web.NewHub()
	go wsHub.Run()

	// 创建消息统计追踪器
	statsTracker := services.NewMessageStatsTracker(larkNotifier, 5*time.Minute)
	go statsTracker.StartPeriodicReport()

	// 启动 Broker
	broker := services.NewInMemoryBroker()
	defer broker.Close()
	logger.Println("[Broker] In-Memory Broker started")
	
	// 启动 UOF Ingestor
	amqpConnector := services.NewAMQPConnector(cfg)
	msgs, err := amqpConnector.Start()
	if err != nil {
		logger.Fatalf("Failed to start AMQP connector: %v", err)
	}
	
	amqpConsumer := services.NewAMQPConsumer(cfg, messageStore, broker) 
	amqpConsumer.SetStatsTracker(statsTracker)
	
	go func() {
		if err := amqpConsumer.Start(msgs); err != nil {
			logger.Fatalf("AMQP consumer error: %v", err)
			larkNotifier.NotifyError("AMQP Consumer", err.Error())
		}
	}()
	logger.Println("[Ingestor] AMQP Ingestor started")
	
	// 启动 Message Processor
	processor := services.NewMessageProcessor(cfg, messageStore, broker, wsHub, marketDescService)
	messageTypes := []string{
		"odds_change", "bet_stop", "bet_settlement", "bet_cancel", 
		"fixture", "fixture_change", "rollback_bet_settlement", "rollback_bet_cancel",
	}
	for _, msgType := range messageTypes {
		if err := processor.StartConsumer(msgType); err != nil {
			logger.Fatalf("Failed to start MessageProcessor for %s: %v", msgType, err)
		}
	}
	logger.Println("[Processor] Message Processor started")

	// 启动Web服务器
	server := web.NewServer(cfg, db, wsHub, larkNotifier, marketDescService)
	go func() {
		if err := server.Start(); err != nil {
			logger.Fatalf("Web server error: %v", err)
			larkNotifier.NotifyError("Web Server", err.Error())
		}
	}()
	logger.Printf("Web server started on port %s", cfg.Port)

	// 启动比赛监控
	matchMonitor := services.NewMatchMonitor(cfg, nil)
	go matchMonitor.CheckAndReportWithNotifier(nil)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			matchMonitor.CheckAndReportWithNotifier(nil)
		}
	}()
	logger.Println("Match monitor started")
	
	// 启动静态数据服务
	staticDataService := services.NewStaticDataService(db, cfg.AccessToken, cfg.APIBaseURL)
	if err := staticDataService.Start(); err != nil {
		logger.Errorf("[StaticData] Failed to start: %v", err)
	} else {
		logger.Println("[StaticData] Static data service started")
	}
	
	// 启动订阅清理服务
	subscriptionCleanup := services.NewSubscriptionCleanupService(cfg, db, larkNotifier)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if result, err := subscriptionCleanup.ExecuteCleanup(); err != nil {
				logger.Errorf("[SubscriptionCleanup] Failed: %v", err)
			} else {
				logger.Printf("[SubscriptionCleanup] Completed: %d unbooked", result.Unbooked)
			}
		}
	}()
	logger.Println("Subscription cleanup started")
	
	// 启动过期live比赛清理服务
	staleLiveCleanup := services.NewStaleLiveCleanupService(cfg, db, larkNotifier)
	go func() {
		time.Sleep(10 * time.Minute)
		staleLiveCleanup.ExecuteCleanup()
		ticker := time.NewTicker(2 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			staleLiveCleanup.ExecuteCleanup()
		}
	}()
	logger.Println("Stale live cleanup started")
	
	// 启动业务监控服务
	businessMonitor := services.NewBusinessMonitor(db, larkNotifier)
	businessMonitor.Start()
	logger.Println("Business monitor started")
	
	// 启动数据清理服务
	cleanupConfig := services.CleanupConfig{
		RetainDaysMessages: 3,
		RetainDaysOdds:     cfg.CleanupRetainDaysOdds,
		RetainDaysBets:     cfg.CleanupRetainDaysBets,
		RetainDaysLiveData: cfg.CleanupRetainDaysLiveData,
		RetainDaysEvents:   cfg.CleanupRetainDaysEvents,
	}
	dataCleanup := services.NewDataCleanupService(db, cleanupConfig)
	go func() {
		// 启动时立即执行一次
		dataCleanup.ExecuteCleanup()
		
		// 每天凌晨 2 点执行
		for {
			now := time.Now()
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(nextRun))
			dataCleanup.ExecuteCleanup()
		}
	}()
	logger.Println("Data cleanup service started")
	
	// 启动热度评分服务
	popularityService := services.NewPopularityScoringService(db, larkNotifier)
	if err := popularityService.Start(); err != nil {
		logger.Errorf("[PopularityScoring] Failed to start: %v", err)
	} else {
		logger.Println("[PopularityScoring] Popularity scoring service started")
	}
	
	// 冷启动初始化
	coldStart := services.NewColdStart(cfg, db, larkNotifier)
	go func() {
		time.Sleep(2 * time.Second)
		coldStart.Run()
	}()
	
	// 启动时自动订阅 (Live)
	startupBooking := services.NewStartupBookingService(cfg, db, larkNotifier)
	go func() {
		time.Sleep(10 * time.Second)
		subscriptionCleanup.ExecuteCleanup()
		startupBooking.ExecuteStartupBooking()
	}()
	
	// 启动时自动订阅 (Pre-match)
	prematchService := services.NewPrematchService(cfg, db)
	go func() {
		time.Sleep(15 * time.Second)
		prematchService.ExecutePrematchBooking()
	}()

	logger.Println("Service is running. Press Ctrl+C to stop.")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down service...")
	amqpConsumer.Stop()
	amqpConnector.Stop() 
	server.Stop()
	logger.Println("Service stopped")
}
