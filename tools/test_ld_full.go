package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}
	
	apiBase := serverURL + "/api"
	
	log.Println("==========================================")
	log.Println("Live Data 连接测试")
	log.Println("==========================================")
	log.Printf("服务器: %s\n", serverURL)
	log.Println("")
	
	// 1. 检查服务健康状态
	log.Println("📊 1. 检查服务健康状态...")
	resp, err := http.Get(apiBase + "/health")
	if err != nil {
		log.Fatalf("❌ 健康检查失败: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	log.Printf("   %s\n", string(body))
	log.Println("")
	
	// 2. 获取服务器 IP
	log.Println("🌐 2. 获取服务器公网 IP...")
	resp, err = http.Get(apiBase + "/ip")
	if err != nil {
		log.Printf("⚠️  无法获取 IP: %v", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var ipInfo map[string]interface{}
		json.Unmarshal(body, &ipInfo)
		log.Printf("   📍 IP 地址: %v\n", ipInfo["ip"])
	}
	log.Println("")
	
	// 3. 自动订阅所有 bookable 比赛
	log.Println("📝 3. 自动订阅所有 bookable 比赛...")
	resp, err = http.Post(apiBase+"/booking/auto", "application/json", nil)
	if err != nil {
		log.Printf("⚠️  订阅请求失败: %v", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("   %s\n", string(body))
		log.Println("   ⏳ 等待订阅完成 (10秒)...")
		time.Sleep(10 * time.Second)
	}
	log.Println("")
	
	// 4. 检查已订阅的比赛
	log.Println("🔍 4. 检查已订阅的比赛...")
	http.Post(apiBase+"/monitor/trigger", "application/json", nil)
	log.Println("   ✅ 监控报告已触发，请查看飞书通知")
	log.Println("   ⏳ 等待报告生成 (5秒)...")
	time.Sleep(5 * time.Second)
	log.Println("")
	
	// 5. 连接 Live Data
	log.Println("🔌 5. 连接 Live Data 服务器...")
	resp, err = http.Post(apiBase+"/ld/connect", "application/json", nil)
	if err != nil {
		log.Printf("❌ LD 连接请求失败: %v", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		log.Printf("   %s\n", string(body))
		log.Println("   ⏳ 等待连接建立 (5秒)...")
		time.Sleep(5 * time.Second)
	}
	log.Println("")
	
	// 6. 检查 LD 连接状态
	log.Println("📡 6. 检查 Live Data 连接状态...")
	resp, err = http.Get(apiBase + "/ld/status")
	if err != nil {
		log.Printf("❌ 无法获取 LD 状态: %v", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var status map[string]interface{}
		json.Unmarshal(body, &status)
		
		if connected, ok := status["connected"].(bool); ok && connected {
			log.Println("   ✅ Live Data 已连接!")
		} else {
			log.Println("   ❌ Live Data 未连接")
			if msg, ok := status["message"].(string); ok {
				log.Printf("   错误: %s", msg)
			}
		}
		
		log.Printf("   完整状态: %s\n", string(body))
	}
	log.Println("")
	
	// 7. 检查已订阅的 LD 比赛
	log.Println("📋 7. 检查 Live Data 已订阅比赛...")
	resp, err = http.Get(apiBase + "/ld/matches")
	if err != nil {
		log.Printf("⚠️  无法获取 LD 比赛列表: %v", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var matches map[string]interface{}
		json.Unmarshal(body, &matches)
		
		if matchList, ok := matches["matches"].([]interface{}); ok {
			log.Printf("   📊 已订阅 %d 场比赛\n", len(matchList))
			if len(matchList) > 0 {
				log.Println("   比赛列表:")
				for i, match := range matchList {
					if i < 5 { // 只显示前5场
						log.Printf("      - %v", match)
					}
				}
				if len(matchList) > 5 {
					log.Printf("      ... 还有 %d 场比赛", len(matchList)-5)
				}
			}
		}
	}
	log.Println("")
	
	// 8. 等待一段时间接收 LD 消息
	log.Println("⏳ 8. 等待接收 Live Data 消息 (30秒)...")
	time.Sleep(30 * time.Second)
	log.Println("")
	
	// 9. 检查接收到的事件
	log.Println("📊 9. 检查接收到的 Live Data 事件...")
	resp, err = http.Get(apiBase + "/ld/events?limit=10")
	if err != nil {
		log.Printf("⚠️  无法获取事件: %v", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var events map[string]interface{}
		json.Unmarshal(body, &events)
		
		if eventList, ok := events["events"].([]interface{}); ok {
			log.Printf("   📈 接收到 %d 个事件\n", len(eventList))
			if len(eventList) > 0 {
				log.Println("   最近的事件:")
				for i, event := range eventList {
					if i < 3 { // 只显示前3个
						if eventMap, ok := event.(map[string]interface{}); ok {
							log.Printf("      - 类型: %v, 比赛: %v, 时间: %v",
								eventMap["event_type"],
								eventMap["match_id"],
								eventMap["timestamp"])
						}
					}
				}
			} else {
				log.Println("   ⚠️  暂未接收到事件，可能需要更长时间")
			}
		}
	}
	log.Println("")
	
	log.Println("==========================================")
	log.Println("✅ 测试完成!")
	log.Println("==========================================")
	log.Println("")
	log.Println("📱 请查看飞书通知获取详细报告:")
	log.Println("   - 自动订阅报告")
	log.Println("   - 比赛监控报告")
	log.Println("   - Live Data 连接状态")
	log.Println("")
	log.Printf("📊 查看更多数据:\n")
	log.Printf("   - LD 事件: GET %s/ld/events\n", apiBase)
	log.Printf("   - LD 比赛: GET %s/ld/matches\n", apiBase)
	log.Println("")
}

