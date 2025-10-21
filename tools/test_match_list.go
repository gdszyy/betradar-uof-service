package main

import (
	"log"
	"os"
	
	"uof-service/config"
	"uof-service/services"
	
	"github.com/streadway/amqp"
	"crypto/tls"
	"fmt"
	"time"
	"net/http"
	"io"
	"encoding/xml"
)

func main() {
	log.Println("Testing Match List Query with unifiedfeed exchange...")
	
	// 加载配置
	cfg := config.Load()
	
	// 获取 bookmaker 信息
	bookmakerId, virtualHost, err := getBookmakerInfo(cfg)
	if err != nil {
		log.Fatalf("Failed to get bookmaker info: %v", err)
	}
	
	log.Printf("Bookmaker ID: %s", bookmakerId)
	log.Printf("Virtual Host: %s", virtualHost)
	
	// 连接 AMQP
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
	
	amqpConfig := amqp.Config{
		Vhost:      virtualHost,
		Heartbeat:  60 * time.Second,
		Locale:     "en_US",
		TLSClientConfig: tlsConfig,
	}
	
	amqpURL := fmt.Sprintf("amqps://%s:@%s", cfg.AccessToken, cfg.MessagingHost)
	
	log.Println("Connecting to AMQP...")
	conn, err := amqp.DialConfig(amqpURL, amqpConfig)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	
	log.Println("✅ Connected to AMQP")
	
	// 创建 channel
	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
	defer channel.Close()
	
	log.Println("✅ Channel created")
	
	// 创建 match monitor
	monitor := services.NewMatchMonitor(cfg, channel)
	
	// 查询已订阅的比赛
	log.Println("\n📋 Querying booked matches...")
	response, err := monitor.QueryBookedMatches(6, 24)
	if err != nil {
		log.Fatalf("❌ Failed to query: %v", err)
	}
	
	log.Println("✅ Query successful!")
	
	// 分析结果
	monitor.AnalyzeBookedMatches(response)
	
	log.Println("\n✅ Test completed successfully!")
}

func getBookmakerInfo(cfg *config.Config) (bookmakerId, virtualHost string, err error) {
	url := cfg.APIBaseURL + "/users/whoami.xml"
	log.Printf("Calling API: %s", url)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("x-access-token", cfg.AccessToken)
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to call API: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	type WhoAmIResponse struct {
		BookmakerID string `xml:"bookmaker_id,attr"`
		VirtualHost string `xml:"virtual_host,attr"`
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %w", err)
	}
	
	var response WhoAmIResponse
	if err := xml.Unmarshal(body, &response); err != nil {
		return "", "", fmt.Errorf("failed to parse XML: %w", err)
	}
	
	if response.BookmakerID == "" {
		return "", "", fmt.Errorf("bookmaker_id not found in response")
	}
	
	if response.VirtualHost == "" {
		return "", "", fmt.Errorf("virtual_host not found in response")
	}
	
	return response.BookmakerID, response.VirtualHost, nil
}

