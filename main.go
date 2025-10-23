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
	
	// 创建 The Sports 客户端替代 LD
	theSportsClient := services.NewTheSportsClient(cfg, db, larkNotifier)
	
	// 设置到 Server
	server.SetTheSportsClient(theSportsClient)
	
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
	
	// 启动 The Sports 客户端
	log.Println("[TheSports] Starting The Sports client...")
	go func() {
		if err := theSportsClient.Connect(); err != nil {
			log.Printf("[TheSports] ❌ Failed to connect: %v", err)
		}
	}()
	
	// 启动自动订阅调度器 (每30分钟查询一次)
	autoBooking := services.NewAutoBookingService(cfg, larkNotifier)
	autoBookingScheduler := services.NewAutoBookingScheduler(autoBooking, 30*time.Minute)
	autoBookingScheduler.Start()
	
	log.Println("Auto-booking scheduler started (every 30 minutes)")

	log.Println("Service is running. Press Ctrl+C to stop.")

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

