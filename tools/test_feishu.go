package main

import (
	"log"
	"time"
	
	"uof-service/services"
)

func main() {
	// 使用实际的 webhook URL
	webhookURL := "https://open.larksuite.com/open-apis/bot/v2/hook/706b2677-d917-4d15-a3f8-6723de0caa15"
	
	notifier := services.NewLarkNotifier(webhookURL)
	
	log.Println("Testing Feishu notifications...")
	
	// 1. 测试文本消息
	log.Println("1. Sending text message...")
	if err := notifier.SendText("🧪 测试消息: Feishu 集成测试"); err != nil {
		log.Printf("Failed: %v", err)
	} else {
		log.Println("✅ Text message sent")
	}
	time.Sleep(2 * time.Second)
	
	// 2. 测试服务启动通知
	log.Println("2. Sending service start notification...")
	if err := notifier.NotifyServiceStart("test-bookmaker", []string{"liveodds", "pre"}); err != nil {
		log.Printf("Failed: %v", err)
	} else {
		log.Println("✅ Service start notification sent")
	}
	time.Sleep(2 * time.Second)
	
	// 3. 测试消息统计通知
	log.Println("3. Sending message stats notification...")
	stats := map[string]int{
		"odds_change":      150,
		"bet_stop":         45,
		"bet_settlement":   30,
		"fixture_change":   12,
		"alive":            5,
	}
	if err := notifier.NotifyMessageStats(stats, 242, "测试周期"); err != nil {
		log.Printf("Failed: %v", err)
	} else {
		log.Println("✅ Message stats notification sent")
	}
	time.Sleep(2 * time.Second)
	
	// 4. 测试比赛监控通知
	log.Println("4. Sending match monitor notification...")
	if err := notifier.NotifyMatchMonitor(100, 25, 15, 10); err != nil {
		log.Printf("Failed: %v", err)
	} else {
		log.Println("✅ Match monitor notification sent")
	}
	time.Sleep(2 * time.Second)
	
	// 5. 测试恢复完成通知
	log.Println("5. Sending recovery complete notification...")
	if err := notifier.NotifyRecoveryComplete(1, 12345); err != nil {
		log.Printf("Failed: %v", err)
	} else {
		log.Println("✅ Recovery complete notification sent")
	}
	time.Sleep(2 * time.Second)
	
	// 6. 测试错误通知
	log.Println("6. Sending error notification...")
	if err := notifier.NotifyError("TestComponent", "这是一个测试错误消息"); err != nil {
		log.Printf("Failed: %v", err)
	} else {
		log.Println("✅ Error notification sent")
	}
	
	log.Println("\n✅ All tests completed! Check your Feishu group for messages.")
}

