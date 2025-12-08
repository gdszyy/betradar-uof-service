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
	logger.Println("[PlayersService] 📥 Starting player preload...")
	
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
			logger.Printf("[PlayersService] ⚠️  Failed to fetch summary for event %s: %v", eventID, err)
			continue
		}
		allPlayers = append(allPlayers, players...)
	}
	
	// 3. 批量预加载球员信息
	playersService.PreloadPlayers(allPlayers)
	
	logger.Printf("[PlayersService] ✅ Player preload finished. Total unique players found: %d", len(allPlayers))
	return nil
}

func schedulePlayerPreload(playersService *services.PlayersService, scheduleService *services.ScheduleService) {
	// 定时更新球员信息 (例如每 6 小时一次)
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		if err := PreloadPlayers(playersService, scheduleService); err != nil {
			logger.Errorf("[PlayersService] ❌ Failed to run scheduled player preload: %v", err)
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
	
	// 发送服务启动通知
	if err := larkNotifier.NotifyServiceStart(cfg.BookmakerID, cfg.Products); err != nil {
		logger.Errorf("Failed to send startup notification: %v", err)
	}

	// 创建消息存储服务
	messageStore := services.NewMessageStore(db)
	
	// 创建 Players 服务
	playersService := services.NewPlayersService(cfg.AccessToken, cfg.APIBaseURL, db)
	if err := playersService.Start(); err != nil {
		logger.Errorf("[PlayersService] ⚠️  Failed to start: %v", err)
	}
	
	// 创建 Schedule 服务
	scheduleService := services.NewScheduleService(db, cfg.AccessToken, cfg.APIBaseURL)
	
		// 启动时立即执行一次球员信息预加载
		if err := PreloadPlayers(playersService, scheduleService); err != nil {
			logger.Errorf("[PlayersService] ⚠️  Failed to preload players: %v", err)
		}
		
			// 定时更新球员信息 (例如每 6 小时一次)
			go schedulePlayerPreload(playersService, scheduleService)
		
		// 启动 Schedule 服务
		if err := scheduleService.Start(); err != nil {
			logger.Errorf("[Schedule] ⚠️  Failed to start: %v", err)
		} else {
			logger.Println("[Schedule] ✅ Schedule service started")
		}
		
		// 创建 Market Descriptions 服务
	marketDescService := services.NewMarketDescriptionsService(cfg.AccessToken, cfg.APIBaseURL)
	marketDescService.SetDatabase(db) // 注入数据库连接 (可选)
	marketDescService.SetPlayersService(playersService) // 注入球员服务 (可选)
	if err := marketDescService.Start(); err != nil {
		logger.Errorf("[MarketDescService] ⚠️  Failed to start: %v", err)
	} else {
		logger.Println("[MarketDescService] ✅ Market descriptions service started")
	}
	
	// 创建 Tab/Chip Assignment 服务 (在 Market Descriptions 之后)
	// 注意: TabChipAssignmentService 已被停用，因为其逻辑存在缺陷
	// 现在使用 MarketGroupsService 进行 tab_id 分配
	// tabChipService := services.NewTabChipAssignmentService(db, playersService)
	// tabChipService.Start()
	// logger.Println("[TabChipService] ✅ Tab/Chip Assignment service started")
	
	// 创建 Market Groups 服务 (用于同步 groups 字段和分配 tabs)
	marketGroupsService := services.NewMarketGroupsService(db)
	
	// 同步 groups 字段从 market_descriptions 到 markets
	if syncedCount, err := marketGroupsService.SyncMarketGroupsFromDescriptions(); err != nil {
		logger.Errorf("[MarketGroupsService] ⚠️  Failed to sync groups: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] ✅ Synced groups for %d markets", syncedCount)
	}
	
	// 基于 groups 分配 tabs
	if assignedCount, err := marketGroupsService.AssignTabsBasedOnGroups(); err != nil {
		logger.Errorf("[MarketGroupsService] ⚠️  Failed to assign tabs based on groups: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] ✅ Assigned tabs for %d markets based on groups", assignedCount)
	}
	
	// 基于 specifiers 分配 tabs (用于时序相关的市场)
	if assignedCount, err := marketGroupsService.AssignTabsBasedOnSpecifiers(); err != nil {
		logger.Errorf("[MarketGroupsService] ⚠️  Failed to assign tabs based on specifiers: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] ✅ Assigned tabs for %d markets based on specifiers", assignedCount)
	}
	
	// 为未分配的市场分配默认 tab (使用 regular_play 而不是 full_match)
	if assignedCount, err := marketGroupsService.AssignDefaultTab("regular_play"); err != nil {
		logger.Errorf("[MarketGroupsService] ⚠️  Failed to assign default tab: %v", err)
	} else {
		logger.Printf("[MarketGroupsService] ✅ Assigned default tab 'regular_play' for %d markets", assignedCount)
	}
	
	// 创建 Producer 监控服务
	producerMonitor := services.NewProducerMonitor(db, larkNotifier, cfg.ProducerCheckIntervalSeconds, cfg.ProducerDownThresholdSeconds)
	go producerMonitor.Start()

	// 创建WebSocket Hub
	wsHub := web.NewHub()
	go wsHub.Run()

	// 创建消息统计追踪器 (5分钟间隔)
	statsTracker := services.NewMessageStatsTracker(larkNotifier, 5*time.Minute)
	go statsTracker.StartPeriodicReport()

	// -------------------------------------------------------------------
	// 1. 启动 Broker (Kafka 替代模块)
	// -------------------------------------------------------------------
	broker := services.NewInMemoryBroker()
	defer broker.Close()
	logger.Println("[Broker] ✅ In-Memory Broker started")
	
	// -------------------------------------------------------------------
	// 2. 启动 UOF Ingestor (AMQP Connector + AMQP Consumer)
	// -------------------------------------------------------------------
	// 创建 AMQP 连接器
	amqpConnector := services.NewAMQPConnector(cfg)
	
	// 启动 AMQP 连接器并获取消息通道
	msgs, err := amqpConnector.Start()
	if err != nil {
		logger.Fatalf("Failed to start AMQP connector: %v", err)
	}
	
	// 启动 AMQP 消费者 (Ingestor 层)
	// 注意：这里不再需要 wsHub 和 marketDescService，因为业务逻辑已迁移
	amqpConsumer := services.NewAMQPConsumer(cfg, messageStore, broker) 
	
	// 设置消息统计回调
	amqpConsumer.SetStatsTracker(statsTracker)
	
	go func() {
		if err := amqpConsumer.Start(msgs); err != nil {
			logger.Fatalf("AMQP consumer error: %v", err)
			larkNotifier.NotifyError("AMQP Consumer", err.Error())
		}
	}()
		
	logger.Println("[Ingestor] ✅ AMQP Ingestor started")
	
	// -------------------------------------------------------------------
	// 3. 启动 Message Processor (业务处理层)
	// -------------------------------------------------------------------
	// 创建 Message Processor 实例
	// 注意：这里需要 wsHub 和 marketDescService，因为业务逻辑已迁移到这里
	processor := services.NewMessageProcessor(cfg, messageStore, broker, wsHub, marketDescService)
	
	// 定义需要处理的消息类型 (Topic)
	messageTypes := []string{
		"odds_change", 
		"bet_stop", 
		"bet_settlement", 
		"bet_cancel", 
		"fixture", 
		"fixture_change", 
		"rollback_bet_settlement", 
		"rollback_bet_cancel",
	}
	
	// 为每种消息类型启动一个独立的消费者
	for _, msgType := range messageTypes {
		if err := processor.StartConsumer(msgType); err != nil {
	logger.Fatalf("Failed to start MessageProcessor for %s: %v", msgType, err)
		}
	}
	
	logger.Println("[Processor] ✅ Message Processor started for all business topics")

	// 启动Web服务器
	server := web.NewServer(cfg, db, wsHub, larkNotifier, marketDescService)
	
	go func() {
		if err := server.Start(); err != nil {
	logger.Fatalf("Web server error: %v", err)
	larkNotifier.NotifyError("Web Server", err.Error())
		}
	}()

	logger.Printf("Web server started on port %s", cfg.Port)

	// 启动比赛监控 (每小时执行一次)
	matchMonitor := services.NewMatchMonitor(cfg, nil)
	
	// 立即执行一次
	go matchMonitor.CheckAndReportWithNotifier(larkNotifier)
	
	// 定期执行
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
	matchMonitor.CheckAndReportWithNotifier(larkNotifier)
		}
	}()
	
	logger.Println("Match monitor started (hourly)")
	
	// 启动静态数据服务 (每周刷新一次)
	staticDataService := services.NewStaticDataService(db, cfg.AccessToken, cfg.APIBaseURL)
	if err := staticDataService.Start(); err != nil {
		logger.Errorf("[StaticData] ⚠️  Failed to start: %v", err)
	} else {
		logger.Println("[StaticData] ✅ Static data service started (weekly refresh)")
	}
	

	
	// 启动订阅清理服务 (每小时执行一次)
	subscriptionCleanup := services.NewSubscriptionCleanupService(cfg, db, larkNotifier)
	
	// 定期执行清理
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
	if result, err := subscriptionCleanup.ExecuteCleanup(); err != nil {
		logger.Errorf("[SubscriptionCleanup] ❌ Failed: %v", err)
	} else {
		logger.Printf("[SubscriptionCleanup] ✅ Completed: %d unbooked out of %d ended", result.Unbooked, result.EndedMatches)
	}
		}
	}()
	
	logger.Println("Subscription cleanup started (hourly)")
	
	// 启动数据清理服务 (每天凌晨 2 点执行一次)
	cleanupConfig := services.CleanupConfig{
		RetainDaysMessages: cfg.CleanupRetainDaysMessages,
		RetainDaysOdds:     cfg.CleanupRetainDaysOdds,
		RetainDaysBets:     cfg.CleanupRetainDaysBets,
		RetainDaysLiveData: cfg.CleanupRetainDaysLiveData,
		RetainDaysEvents:   cfg.CleanupRetainDaysEvents,
	}
	dataCleanup := services.NewDataCleanupService(db, cleanupConfig)
	
	// 定期执行数据清理
	go func() {
		// 计算到下一个凌晨 2 点的时间
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
		if now.After(nextRun) {
	// 如果已经过了今天的 2 点，设置为明天 2 点
	nextRun = nextRun.Add(24 * time.Hour)
		}
		
		// 等待到第一次执行时间
		initialDelay := time.Until(nextRun)
		logger.Printf("[DataCleanup] Next cleanup scheduled at %s (in %s)", nextRun.Format("2006-01-02 15:04:05"), initialDelay.Round(time.Minute))
		time.Sleep(initialDelay)
		
		// 执行第一次清理
		if results, err := dataCleanup.ExecuteCleanup(); err != nil {
	logger.Errorf("[DataCleanup] ❌ Failed: %v", err)
		} else {
	totalDeleted := int64(0)
	for _, result := range results {
		if result.Error != nil {
	logger.Errorf("[DataCleanup] ⚠️  %s: %v", result.TableName, result.Error)
		} else if result.DeletedRows > 0 {
	logger.Printf("[DataCleanup] ✅ %s: deleted %d rows (retain %d days)", result.TableName, result.DeletedRows, result.RetainedDays)
	totalDeleted += result.DeletedRows
		}
	}
	logger.Printf("[DataCleanup] ✅ Cleanup completed: %d total rows deleted", totalDeleted)
	
	// 发送通知
	if totalDeleted > 0 {
		larkNotifier.NotifyDataCleanup(totalDeleted, results)
	}
		}
		
		// 之后每 24 小时执行一次
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
	if results, err := dataCleanup.ExecuteCleanup(); err != nil {
		logger.Errorf("[DataCleanup] ❌ Failed: %v", err)
	} else {
		totalDeleted := int64(0)
		for _, result := range results {
	if result.Error != nil {
		logger.Errorf("[DataCleanup] ⚠️  %s: %v", result.TableName, result.Error)
	} else if result.DeletedRows > 0 {
		logger.Printf("[DataCleanup] ✅ %s: deleted %d rows (retain %d days)", result.TableName, result.DeletedRows, result.RetainedDays)
		totalDeleted += result.DeletedRows
	}
		}
		logger.Printf("[DataCleanup] ✅ Cleanup completed: %d total rows deleted", totalDeleted)
		
		// 发送通知
		if totalDeleted > 0 {
	larkNotifier.NotifyDataCleanup(totalDeleted, results)
		}
	}
		}
	}()
	
	logger.Println("Data cleanup service started (daily at 2:00 AM)")
	
	// 启动热度评分服务 (每天凌晨 2 点执行一次)
	popularityService := services.NewPopularityScoringService(db, larkNotifier)
	if err := popularityService.Start(); err != nil {
		logger.Errorf("[PopularityScoring] ⚠️  Failed to start: %v", err)
	} else {
		logger.Println("[PopularityScoring] ✅ Popularity scoring service started (daily at 2:00 AM)")
	}
	
	// 冷启动初始化 - 获取所有比赛信息
	coldStart := services.NewColdStart(cfg, db, larkNotifier)
	go func() {
		// 等待 2 秒后执行
		time.Sleep(2 * time.Second)
		
		logger.Println("[ColdStart] 🚀 Starting cold start initialization...")
		if err := coldStart.Run(); err != nil {
	logger.Errorf("[ColdStart] ❌ Failed: %v", err)
	larkNotifier.NotifyError("Cold Start", err.Error())
		} else {
	logger.Println("[ColdStart] ✅ Cold start completed successfully")
		}
	}()
	
	// 启动时自动订阅 (Live)
	startupBooking := services.NewStartupBookingService(cfg, db, larkNotifier)
	go func() {
		// 等待 AMQP 连接建立和冷启动完成
		time.Sleep(10 * time.Second)
		
		// 1. 先执行清理,取消已结束比赛的订阅
		logger.Println("[StartupBooking] 🧹 Cleaning up ended matches before booking...")
		if cleanupResult, err := subscriptionCleanup.ExecuteCleanup(); err != nil {
	logger.Errorf("[StartupBooking] ⚠️  Cleanup failed: %v", err)
		} else {
	logger.Printf("[StartupBooking] ✅ Cleanup completed: %d unbooked", cleanupResult.Unbooked)
		}
		
		// 2. 执行自动订阅 (Live)
		if result, err := startupBooking.ExecuteStartupBooking(); err != nil {
	logger.Errorf("[StartupBooking] ❌ Failed to execute startup booking: %v", err)
	larkNotifier.NotifyError("Startup Booking", err.Error())
		} else {
	logger.Printf("[StartupBooking] ✅ Startup booking completed: %d/%d successful", result.Success, result.Bookable)
		}
	}()
	
	// 启动时自动订阅 (Pre-match)
	prematchService := services.NewPrematchService(cfg, db)
	go func() {
		// 等待 AMQP 连接建立和冷启动完成
		time.Sleep(15 * time.Second)
		
		logger.Println("[PrematchService] 🚀 Starting pre-match event booking...")
		
		if result, err := prematchService.ExecutePrematchBooking(); err != nil {
	logger.Errorf("[PrematchService] ❌ Failed: %v", err)
	larkNotifier.NotifyError("Pre-match Booking", err.Error())
		} else {
	logger.Printf("[PrematchService] ✅ Completed: %d total events, %d bookable, %d already booked, %d success, %d failed",
		result.TotalEvents, result.Bookable, result.AlreadyBooked, result.Success, result.Failed)
	
	// 发送通知
	if result.Success > 0 {
		larkNotifier.NotifyPrematchBooking(result.TotalEvents, result.Bookable, result.Success, result.Failed)
	}
		}
	}()

	logger.Println("Service is running. Press Ctrl+C to stop.")
	logger.Println("All data is sourced from UOF (Unified Odds Feed)")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down service...")

	// 清理资源
	amqpConsumer.Stop()
	amqpConnector.Stop() 
			// processor.Stop() // MessageProcessor 当前没有 Stop 方法，但 broker.Close() 会关闭通道
			server.Stop()

	logger.Println("Service stopped")
}

