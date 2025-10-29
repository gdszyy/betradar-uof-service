package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LarkNotifier 飞书机器人通知器
type LarkNotifier struct {
	webhookURL string
	client     *http.Client
	enabled    bool
}

// NewLarkNotifier 创建飞书通知器
func NewLarkNotifier(webhookURL string) *LarkNotifier {
	enabled := webhookURL != ""
	if enabled {
		logger.Printf("[LarkNotifier] Initialized with webhook")
	} else {
		logger.Printf("[LarkNotifier] Disabled (no webhook URL)")
	}
	
	return &LarkNotifier{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
		enabled:    enabled,
	}
}

// LarkMessage 飞书消息结构
type LarkMessage struct {
	MsgType string      `json:"msg_type"`
	Content interface{} `json:"content"`
}

// LarkTextContent 文本消息内容
type LarkTextContent struct {
	Text string `json:"text"`
}

// LarkPostContent 富文本消息内容
type LarkPostContent struct {
	Post LarkPost `json:"post"`
}

type LarkPost struct {
	ZhCn LarkPostLang `json:"zh_cn"`
}

type LarkPostLang struct {
	Title   string          `json:"title"`
	Content [][]LarkElement `json:"content"`
}

type LarkElement struct {
	Tag    string `json:"tag"`
	Text   string `json:"text,omitempty"`
	Href   string `json:"href,omitempty"`
	UserID string `json:"user_id,omitempty"`
}

// SendText 发送文本消息
func (n *LarkNotifier) SendText(text string) error {
	if !n.enabled {
		return nil
	}
	
	message := LarkMessage{
		MsgType: "text",
		Content: LarkTextContent{
			Text: text,
		},
	}
	
	return n.send(message)
}

// SendRichText 发送富文本消息
func (n *LarkNotifier) SendRichText(title string, content [][]LarkElement) error {
	if !n.enabled {
		return nil
	}
	
	message := LarkMessage{
		MsgType: "post",
		Content: LarkPostContent{
			Post: LarkPost{
				ZhCn: LarkPostLang{
					Title:   title,
					Content: content,
				},
			},
		},
	}
	
	return n.send(message)
}

// send 发送消息
func (n *LarkNotifier) send(message LarkMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	
	resp, err := n.client.Post(n.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}

// NotifyServiceStart 通知服务启动
func (n *LarkNotifier) NotifyServiceStart(bookmakerId string, products []string) error {
	content := [][]LarkElement{
		{
			{Tag: "text", Text: "🚀 服务启动\n"},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("Bookmaker ID: %s\n", bookmakerId)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("Products: %v\n", products)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
		},
	}
	
	return n.SendRichText("UOF Service Started", content)
}

// NotifyRecoveryComplete 通知恢复完成
func (n *LarkNotifier) NotifyRecoveryComplete(productID int, requestID int64) error {
	content := [][]LarkElement{
		{
			{Tag: "text", Text: "✅ 恢复完成\n"},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("Product: %d\n", productID)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("Request ID: %d\n", requestID)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
		},
	}
	
	return n.SendRichText("Recovery Complete", content)
}

// NotifyRecoveryFailed 通知恢复失败
func (n *LarkNotifier) NotifyRecoveryFailed(productID int, reason string) error {
	content := [][]LarkElement{
		{
			{Tag: "text", Text: "❌ 恢复失败\n"},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("Product: %d\n", productID)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("原因: %s\n", reason)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
		},
	}
	
	return n.SendRichText("Recovery Failed", content)
}

// NotifyMessageStats 通知消息统计
func (n *LarkNotifier) NotifyMessageStats(stats map[string]int, totalMessages int, period string) error {
	content := [][]LarkElement{
		{
			{Tag: "text", Text: fmt.Sprintf("📊 消息统计 (%s)\n", period)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("总消息数: %d\n", totalMessages)},
		},
	}
	
	// 添加各类型消息统计
	for msgType, count := range stats {
		if count > 0 {
			content = append(content, []LarkElement{
				{Tag: "text", Text: fmt.Sprintf("  %s: %d\n", msgType, count)},
			})
		}
	}
	
	content = append(content, []LarkElement{
		{Tag: "text", Text: fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
	})
	
	return n.SendRichText("Message Statistics", content)
}

// NotifyMatchMonitor 通知Match监控结果
func (n *LarkNotifier) NotifyMatchMonitor(totalMatches, bookedMatches, preMatches, liveMatches int) error {
	var emoji string
	var status string
	
	if bookedMatches == 0 {
		emoji = "⚠️"
		status = "警告: 没有订阅的比赛"
	} else if liveMatches == 0 {
		emoji = "ℹ️"
		status = "提示: 没有进行中的比赛"
	} else {
		emoji = "✅"
		status = "正常"
	}
	
	content := [][]LarkElement{
		{
			{Tag: "text", Text: fmt.Sprintf("%s Match订阅监控\n", emoji)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("状态: %s\n", status)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("总比赛数: %d\n", totalMatches)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("已订阅: %d\n", bookedMatches)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("  - Pre-match: %d\n", preMatches)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("  - Live: %d\n", liveMatches)},
		},
	}
	
	if bookedMatches == 0 {
		content = append(content, []LarkElement{
			{Tag: "text", Text: "\n💡 建议: 订阅一些比赛以接收odds_change消息"},
		})
	}
	
	content = append(content, []LarkElement{
		{Tag: "text", Text: fmt.Sprintf("\n时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
	})
	
	return n.SendRichText("Match Subscription Monitor", content)
}

// NotifyError 通知错误
func (n *LarkNotifier) NotifyError(component, message string) error {
	content := [][]LarkElement{
		{
			{Tag: "text", Text: "❌ 错误\n"},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("组件: %s\n", component)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("消息: %s\n", message)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
		},
	}
	
	return n.SendRichText("Error Alert", content)
}

// NotifyPrematchBooking 发送 Pre-match 订阅通知
func (n *LarkNotifier) NotifyPrematchBooking(totalEvents, bookable, success, failed int) error {
	if !n.enabled {
		return nil
	}
	
	content := [][]LarkElement{
		{
			{Tag: "text", Text: "🚀 Pre-match 订阅完成\n"},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("总赛事数: %d\n", totalEvents)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("可订阅: %d\n", bookable)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("✅ 成功: %d\n", success)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("❌ 失败: %d\n", failed)},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
		},
	}
	
	return n.SendRichText("Pre-match Booking", content)
}


// NotifyDataCleanup 发送数据清理通知
func (n *LarkNotifier) NotifyDataCleanup(totalDeleted int64, results []CleanupResult) error {
	if !n.enabled {
		return nil
	}
	
	content := [][]LarkElement{
		{
			{Tag: "text", Text: "🧹 数据清理完成\n"},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("总删除行数: %d\n", totalDeleted)},
		},
	}
	
	// 添加每个表的清理结果
	for _, result := range results {
		if result.DeletedRows > 0 {
			content = append(content, []LarkElement{
				{Tag: "text", Text: fmt.Sprintf("  • %s: %d 行 (保留 %d 天)\n", result.TableName, result.DeletedRows, result.RetainedDays)},
			})
		}
	}
	
	content = append(content, []LarkElement{
		{Tag: "text", Text: fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
	})
	
	return n.SendRichText("Data Cleanup", content)
}



// NotifyDatabaseReset 发送数据库重置通知
func (n *LarkNotifier) NotifyDatabaseReset(deletedCounts map[string]int64, totalDeleted int64) error {
	if !n.enabled {
		return nil
	}
	
	content := [][]LarkElement{
		{
			{Tag: "text", Text: "🔄 **数据库已重置**\n"},
		},
		{
			{Tag: "text", Text: fmt.Sprintf("总删除行数: **%d**\n", totalDeleted)},
		},
		{
			{Tag: "text", Text: "\n详细信息:\n"},
		},
	}
	
	// 添加每个表的删除数量
	for table, count := range deletedCounts {
		if count > 0 {
			content = append(content, []LarkElement{
				{Tag: "text", Text: fmt.Sprintf("  • %s: %d 行\n", table, count)},
			})
		}
	}
	
	content = append(content, []LarkElement{
		{Tag: "text", Text: fmt.Sprintf("\n⏰ 时间: %s", time.Now().Format("2006-01-02 15:04:05"))},
	})
	
	return n.SendRichText("Database Reset", content)
}

