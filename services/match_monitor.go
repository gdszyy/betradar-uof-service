package services

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"time"
	
	"github.com/streadway/amqp"
	"uof-service/config"
)

// MatchListRequest Match List请求
type MatchListRequest struct {
	XMLName          xml.Name `xml:"matchlist"`
	HoursBack        int      `xml:"hoursback,attr"`
	HoursForward     int      `xml:"hoursforward,attr"`
	IncludeAvailable string   `xml:"includeavailable,attr"`
	Sports           []SportFilter `xml:"sport,omitempty"`
}

type SportFilter struct {
	SportID string `xml:"sportid,attr"`
}

// MatchListResponse Match List响应
type MatchListResponse struct {
	XMLName xml.Name      `xml:"matchlist"`
	Matches []MatchInfo   `xml:"match"`
}

type MatchInfo struct {
	ID       string      `xml:"id,attr"`
	Booked   string      `xml:"booked,attr"`
	SportID  string      `xml:"sportid,attr"`
	Status   MatchStatus `xml:"status"`
	Home     TeamInfo    `xml:"hometeam"`
	Away     TeamInfo    `xml:"awayteam"`
	Start    string      `xml:"startdate,attr"`
}

type MatchStatus struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type TeamInfo struct {
	Name string `xml:"name,attr"`
}

// MatchMonitor 比赛订阅监控
type MatchMonitor struct {
	config  *config.Config
	channel *amqp.Channel
}

// NewMatchMonitor 创建比赛监控
func NewMatchMonitor(cfg *config.Config, channel *amqp.Channel) *MatchMonitor {
	return &MatchMonitor{
		config:  cfg,
		channel: channel,
	}
}

// QueryBookedMatches 查询已订阅的比赛
func (m *MatchMonitor) QueryBookedMatches(hoursBack, hoursForward int) (*MatchListResponse, error) {
	log.Printf("📋 Querying booked matches (back: %dh, forward: %dh)...", hoursBack, hoursForward)
	
	// 创建请求
	request := MatchListRequest{
		HoursBack:        hoursBack,
		HoursForward:     hoursForward,
		IncludeAvailable: "yes",
	}
	
	// 序列化为XML
	xmlData, err := xml.MarshalIndent(request, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}
	
	// 添加XML声明
	fullXML := []byte(xml.Header + string(xmlData))
	
	log.Printf("📤 Sending Match List request:\n%s", string(fullXML))
	
	// 创建临时队列接收响应
	responseQueue, err := m.channel.QueueDeclare(
		"",    // 让服务器生成队列名
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare response queue: %w", err)
	}
	
	// 发送请求
	err = m.channel.Publish(
		"",                    // exchange
		"matchlist",           // routing key (根据文档可能需要调整)
		false,                 // mandatory
		false,                 // immediate
		amqp.Publishing{
			ContentType:   "text/xml",
			CorrelationId: fmt.Sprintf("%d", time.Now().Unix()),
			ReplyTo:       responseQueue.Name,
			Body:          fullXML,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to publish request: %w", err)
	}
	
	log.Println("⏳ Waiting for response...")
	
	// 消费响应
	msgs, err := m.channel.Consume(
		responseQueue.Name, // queue
		"",                 // consumer
		true,               // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to consume response: %w", err)
	}
	
	// 等待响应(超时10秒)
	select {
	case msg := <-msgs:
		log.Printf("📥 Received response (%d bytes)", len(msg.Body))
		
		// 解析响应
		var response MatchListResponse
		decoder := xml.NewDecoder(bytes.NewReader(msg.Body))
		
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("XML parsing error: %w", err)
			}
			
			if startElement, ok := token.(xml.StartElement); ok {
				if startElement.Name.Local == "matchlist" {
					if err := decoder.DecodeElement(&response, &startElement); err != nil {
						return nil, fmt.Errorf("failed to decode matchlist: %w", err)
					}
					break
				}
			}
		}
		
		return &response, nil
		
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

// AnalyzeBookedMatches 分析已订阅的比赛
func (m *MatchMonitor) AnalyzeBookedMatches(response *MatchListResponse) {
	log.Println("\n" + "═══════════════════════════════════════════════════════════")
	log.Println("📊 BOOKED MATCHES ANALYSIS")
	log.Println("═══════════════════════════════════════════════════════════")
	
	totalMatches := len(response.Matches)
	bookedCount := 0
	availableCount := 0
	preCount := 0
	liveCount := 0
	
	bookedMatches := []MatchInfo{}
	
	// 统计
	for _, match := range response.Matches {
		if match.Booked == "1" {
			bookedCount++
			bookedMatches = append(bookedMatches, match)
			
			// 判断pre还是live
			if match.Status.Name == "NOT_STARTED" {
				preCount++
			} else {
				liveCount++
			}
		} else if match.Booked == "0" {
			availableCount++
		}
	}
	
	log.Printf("📈 Summary:")
	log.Printf("  Total matches: %d", totalMatches)
	log.Printf("  Booked matches: %d", bookedCount)
	log.Printf("    - Pre-match (NOT_STARTED): %d", preCount)
	log.Printf("    - Live (other status): %d", liveCount)
	log.Printf("  Available but not booked: %d", availableCount)
	
	// 显示已订阅的比赛
	if bookedCount > 0 {
		log.Println("\n🎯 Booked Matches:")
		log.Printf("%-20s %-15s %-30s %-30s %s", "Match ID", "Status", "Home", "Away", "Start Time")
		log.Println(string(bytes.Repeat([]byte("-"), 120)))
		
		for _, match := range bookedMatches {
			log.Printf("%-20s %-15s %-30s %-30s %s",
				match.ID,
				match.Status.Name,
				truncate(match.Home.Name, 30),
				truncate(match.Away.Name, 30),
				match.Start,
			)
		}
	} else {
		log.Println("\n⚠️  WARNING: No booked matches found!")
		log.Println("   This explains why you're not receiving odds_change messages.")
		log.Println("   You need to subscribe to matches to receive odds updates.")
		
		if availableCount > 0 {
			log.Printf("\n💡 TIP: There are %d available matches you can book.", availableCount)
			log.Println("   Use bookmatch command to subscribe to matches.")
		}
	}
	
	if bookedCount > 0 && liveCount == 0 {
		log.Println("\n⚠️  NOTE: No live matches currently.")
		log.Println("   Odds_change messages are typically sent for live matches.")
		log.Println("   Pre-match odds updates are less frequent.")
	}
	
	log.Println("═══════════════════════════════════════════════════════════\n")
}

// MonitorPeriodically 定期监控
func (m *MatchMonitor) MonitorPeriodically(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	// 立即执行一次
	m.checkAndReport()
	
	// 定期执行
	for range ticker.C {
		m.checkAndReport()
	}
}

// checkAndReport 检查并报告
func (m *MatchMonitor) checkAndReport() {
	response, err := m.QueryBookedMatches(6, 24)
	if err != nil {
		log.Printf("❌ Failed to query booked matches: %v", err)
		return
	}
	
	m.AnalyzeBookedMatches(response)
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

