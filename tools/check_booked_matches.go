package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	
	"github.com/streadway/amqp"
	
	"uof-service/config"
	"uof-service/services"
)

func main() {
	// 加载配置
	cfg := config.Load()
	
	log.Println("🔍 Checking booked matches...")
	log.Printf("Connecting to: %s", cfg.MessagingHost)
	
	// 获取bookmaker信息
	bookmakerId, virtualHost, err := getBookmakerInfo(cfg)
	if err != nil {
		log.Fatalf("Failed to get bookmaker info: %v", err)
	}
	
	log.Printf("Bookmaker ID: %s", bookmakerId)
	log.Printf("Virtual Host: %s", virtualHost)
	
	// 连接到AMQP
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}
	
	amqpURL := fmt.Sprintf("amqps://%s:%s@%s/%s",
		cfg.AccessToken,
		cfg.AccessToken,
		cfg.MessagingHost,
		virtualHost,
	)
	
	conn, err := amqp.DialTLS(amqpURL, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to connect to AMQP: %v", err)
	}
	defer conn.Close()
	
	log.Println("✅ Connected to AMQP")
	
	// 创建channel
	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to create channel: %v", err)
	}
	defer channel.Close()
	
	// 创建监控器
	monitor := services.NewMatchMonitor(cfg, channel)
	
	// 查询已订阅的比赛
	response, err := monitor.QueryBookedMatches(6, 24)
	if err != nil {
		log.Fatalf("Failed to query booked matches: %v", err)
	}
	
	// 分析结果
	monitor.AnalyzeBookedMatches(response)
}

func getBookmakerInfo(cfg *config.Config) (string, string, error) {
	url := fmt.Sprintf("%s/v1/users/whoami.xml", cfg.APIBaseURL)
	
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}
	
	req.Header.Set("x-access-token", cfg.AccessToken)
	
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var whoami struct {
		BookmakerID string `xml:"bookmaker_id,attr"`
		VirtualHost string `xml:"virtual_host,attr"`
	}
	
	if err := xml.Unmarshal(body, &whoami); err != nil {
		return "", "", err
	}
	
	return whoami.BookmakerID, whoami.VirtualHost, nil
}

