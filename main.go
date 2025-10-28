package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"uof-service/config"
	"uof-service/database"
	"uof-service/services"
	"uof-service/web"
)

func main() {
	// 配置日志输出到 stdout (显示为 [info])
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags)
	
	log.Println("Starting Betradar UOF Service...")

	// 加载配置
	cfg := config.Load()

	// 连接数据库
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 运行数据库迁移
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database connected and migrated")

	// 创建 Feishu 通知器
	larkNotifier := services.NewLarkNotifier(cfg.LarkWebhook)
	
	// 发送服务启动通知
	if err := larkNotifier.NotifyServiceStart(cfg.BookmakerID, cfg.Products); err != nil {
		log.Printf("Failed to send startup notification: %v", err)
	}

	// 创建消息存储服务
	messageStore := services.NewMessageStore(db)
	
	// 创建 Producer 监控服务
	producerMonitor := services.NewProducerMonitor(db, larkNotifier)
	go producerMonitor.Start()

	// 创建WebSocket Hub
	wsHub := web.NewHub()
	go wsHub.Run()

	// 创建消息统计追踪器 (5分钟间隔)
	statsTracker := services.NewMessageStatsTracker(larkNotifier, 5*time.Minute)
	go statsTracker.StartPeriodicReport()

	// 启动AMQP消费者
	amqpConsumer := services.NewAMQPConsumer(cfg, messageStore, wsHub)
	
	// 设置消息统计回调
	amqpConsumer.SetStatsTracker(statsTracker)
	
	go func() {
		if err := amqpConsumer.Start(); err != nil {
			log.Fatalf("AMQP consumer error: %v", err)
			larkNotifier.NotifyError("AMQP Consumer", err.Error())
		}
	}()

	log.Println("AMQP consumer started")

	// 启动Web服务器
	server := web.NewServer(cfg, db, wsHub, larkNotifier)
	
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Web server error: %v", err)
			larkNotifier.NotifyError("Web Server", err.Error())
		}
	}()

	log.Printf("Web server started on port %s", cfg.Port)

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
	
	log.Println("Match monitor started (hourly)")
	
	// 启动订阅清理服务 (每小时执行一次)
	subscriptionCleanup := services.NewSubscriptionCleanupService(cfg, db, larkNotifier)
	
	// 定期执行清理
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			if result, err := subscriptionCleanup.ExecuteCleanup(); err != nil {
				log.Printf("[SubscriptionCleanup] ❌ Failed: %v", err)
			} else {
				log.Printf("[SubscriptionCleanup] ✅ Completed: %d unbooked out of %d ended", result.Unbooked, result.EndedMatches)
			}
		}
	}()
	
	log.Println("Subscription cleanup started (hourly)")
	
	// 冷启动初始化 - 获取所有比赛信息
	coldStart := services.NewColdStart(cfg, db, larkNotifier)
	go func() {
		// 等待 2 秒后执行
		time.Sleep(2 * time.Second)
		
		log.Println("[ColdStart] 🚀 Starting cold start initialization...")
		if err := coldStart.Run(); err != nil {
			log.Printf("[ColdStart] ❌ Failed: %v", err)
			larkNotifier.NotifyError("Cold Start", err.Error())
		} else {
			log.Println("[ColdStart] ✅ Cold start completed successfully")
		}
	}()
	
	// 启动时自动订阅 (Live)
	startupBooking := services.NewStartupBookingService(cfg, db, larkNotifier)
	go func() {
		// 等待 AMQP 连接建立和冷启动完成
		time.Sleep(10 * time.Second)
		
		// 1. 先执行清理,取消已结束比赛的订阅
		log.Println("[StartupBooking] 🧹 Cleaning up ended matches before booking...")
		if cleanupResult, err := subscriptionCleanup.ExecuteCleanup(); err != nil {
			log.Printf("[StartupBooking] ⚠️  Cleanup failed: %v", err)
		} else {
			log.Printf("[StartupBooking] ✅ Cleanup completed: %d unbooked", cleanupResult.Unbooked)
		}
		
		// 2. 执行自动订阅 (Live)
		if result, err := startupBooking.ExecuteStartupBooking(); err != nil {
			log.Printf("[StartupBooking] ❌ Failed to execute startup booking: %v", err)
			larkNotifier.NotifyError("Startup Booking", err.Error())
		} else {
			log.Printf("[StartupBooking] ✅ Startup booking completed: %d/%d successful", result.Success, result.Bookable)
		}
	}()
	
	// 启动时自动订阅 (Pre-match)
	prematchService := services.NewPrematchService(cfg, db)
	go func() {
		// 等待 AMQP 连接建立和冷启动完成
		time.Sleep(15 * time.Second)
		
		log.Println("[PrematchService] 🚀 Starting pre-match event booking...")
		
		if result, err := prematchService.ExecutePrematchBooking(); err != nil {
			log.Printf("[PrematchService] ❌ Failed: %v", err)
			larkNotifier.NotifyError("Pre-match Booking", err.Error())
		} else {
			log.Printf("[PrematchService] ✅ Completed: %d total events, %d bookable, %d already booked, %d success, %d failed",
				result.TotalEvents, result.Bookable, result.AlreadyBooked, result.Success, result.Failed)
			
			// 发送通知
			if result.Success > 0 {
				larkNotifier.NotifyPrematchBooking(result.TotalEvents, result.Bookable, result.Success, result.Failed)
			}
		}
	}()

	log.Println("Service is running. Press Ctrl+C to stop.")
	log.Println("All data is sourced from UOF (Unified Odds Feed)")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down service...")

	// 清理资源
	amqpConsumer.Stop()
	server.Stop()

	log.Println("Service stopped")
}

