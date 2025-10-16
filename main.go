package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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

	// 创建消息存储服务
	messageStore := services.NewMessageStore(db)

	// 创建WebSocket Hub
	wsHub := web.NewHub()
	go wsHub.Run()

	// 启动AMQP消费者
	amqpConsumer := services.NewAMQPConsumer(cfg, messageStore, wsHub)
	go func() {
		if err := amqpConsumer.Start(); err != nil {
			log.Fatalf("AMQP consumer error: %v", err)
		}
	}()

	log.Println("AMQP consumer started")

	// 启动Web服务器
	server := web.NewServer(cfg, db, wsHub)
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	log.Printf("Web server started on port %s", cfg.Port)
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

