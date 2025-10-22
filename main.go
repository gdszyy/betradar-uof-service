package main

import (
	"fmt"
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
	
	// 测试1: 使用写死的 webhook 发送测试消息
	hardcodedNotifier := services.NewLarkNotifier("https://open.larksuite.com/open-apis/bot/v2/hook/706b2677-d917-4d15-a3f8-6723de0caa15")
	if err := hardcodedNotifier.SendText("🧪 测试1: 使用硬编码 webhook 发送 (服务启动)"); err != nil {
		log.Printf("❌ Hardcoded webhook test failed: %v", err)
	} else {
		log.Println("✅ Hardcoded webhook test sent")
	}
	
	// 测试2: 使用配置的 webhook 发送测试消息
	log.Printf("[Config] LarkWebhook value: '%s' (length: %d)", cfg.LarkWebhook, len(cfg.LarkWebhook))
	if err := larkNotifier.SendText(fmt.Sprintf("🧪 测试2: 使用配置 webhook 发送 (服务启动) - Webhook: %s", cfg.LarkWebhook)); err != nil {
		log.Printf("❌ Config webhook test failed: %v", err)
	} else {
		log.Println("✅ Config webhook test sent")
	}
	
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
	
	// 创建 LD 客户端(稍后启动)
	ldClient := services.NewLDClient(cfg)
	ldEventHandler := services.NewLDEventHandler(db, larkNotifier)
	
	// 设置事件处理器
	ldClient.SetEventHandler(ldEventHandler.HandleEvent)
	ldClient.SetMatchInfoHandler(ldEventHandler.HandleMatchInfo)
	ldClient.SetLineupHandler(ldEventHandler.HandleLineup)
	
	// 设置到 Server
	server.SetLDClient(ldClient)
	
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
	
	// 启动 Live Data 客户端 (暂时禁用,需要先配置 IP 白名单)
	// TODO: 联系 Betradar 将 Railway IP 添加到白名单后启用
	// go func() {
	// 	if err := ldClient.Connect(); err != nil {
	// 		log.Printf("[LD] ❌ Failed to connect: %v", err)
	// 		larkNotifier.NotifyError("Live Data Client", err.Error())
	// 	} else {
	// 		log.Println("[LD] ✅ Live Data client started")
	// 		
	// 		// 发送通知
	// 		larkNotifier.SendText("🟢 Live Data 客户端已启动")
	// 	}
	// }()
	
	log.Println("[LD] ⚠️  Live Data client created but not started (IP whitelist required)")

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

