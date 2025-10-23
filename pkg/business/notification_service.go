package business

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
)

// DefaultNotificationService 默认通知管理服务实现
type DefaultNotificationService struct {
	logger      common.Logger
	webhookURL  string
	httpClient  *http.Client
}

// NewNotificationService 创建通知管理服务
func NewNotificationService(logger common.Logger, webhookURL string) NotificationService {
	return &DefaultNotificationService{
		logger:     logger,
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendNotification 发送通知
func (s *DefaultNotificationService) SendNotification(ctx context.Context, notification *Notification) error {
	s.logger.Debug("Sending notification: %s", notification.Title)

	if s.webhookURL == "" {
		s.logger.Warn("Webhook URL not configured, skipping notification")
		return nil
	}

	// 构建飞书消息
	message := s.buildLarkMessage(notification)

	// 发送 HTTP 请求
	jsonData, err := json.Marshal(message)
	if err != nil {
		s.logger.Error("Failed to marshal notification: %v", err)
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Error("Failed to create request: %v", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Failed to send notification: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Notification failed with status: %d", resp.StatusCode)
		return fmt.Errorf("notification failed with status: %d", resp.StatusCode)
	}

	s.logger.Debug("Notification sent successfully: %s", notification.Title)
	return nil
}

// NotifyMatchStart 通知比赛开始
func (s *DefaultNotificationService) NotifyMatchStart(ctx context.Context, match *models.Match) error {
	s.logger.Info("Notifying match start: %s", match.ID)

	notification := &Notification{
		Type:      NotificationTypeInfo,
		Title:     "比赛开始",
		Message:   fmt.Sprintf("%s vs %s", match.HomeTeam.Name, match.AwayTeam.Name),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"match_id":   match.ID,
			"sport_name": match.SportName,
			"start_time": match.StartTime,
		},
	}

	return s.SendNotification(ctx, notification)
}

// NotifyMatchEnd 通知比赛结束
func (s *DefaultNotificationService) NotifyMatchEnd(ctx context.Context, match *models.Match) error {
	s.logger.Info("Notifying match end: %s", match.ID)

	notification := &Notification{
		Type:      NotificationTypeInfo,
		Title:     "比赛结束",
		Message:   fmt.Sprintf("%s %d:%d %s", match.HomeTeam.Name, match.Score.Home, match.Score.Away, match.AwayTeam.Name),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"match_id":   match.ID,
			"sport_name": match.SportName,
			"score":      fmt.Sprintf("%d:%d", match.Score.Home, match.Score.Away),
		},
	}

	return s.SendNotification(ctx, notification)
}

// NotifyOddsChange 通知赔率变化
func (s *DefaultNotificationService) NotifyOddsChange(ctx context.Context, odds *models.Odds) error {
	s.logger.Debug("Notifying odds change: %s", odds.ID)

	notification := &Notification{
		Type:      NotificationTypeInfo,
		Title:     "赔率变化",
		Message:   fmt.Sprintf("比赛 %s 的 %s 市场赔率已更新", odds.MatchID, odds.MarketName),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"match_id":    odds.MatchID,
			"market_id":   odds.MarketID,
			"market_name": odds.MarketName,
		},
	}

	return s.SendNotification(ctx, notification)
}

// NotifyError 通知错误
func (s *DefaultNotificationService) NotifyError(ctx context.Context, err error, context string) error {
	s.logger.Error("Notifying error: %s - %v", context, err)

	notification := &Notification{
		Type:      NotificationTypeError,
		Title:     "系统错误",
		Message:   fmt.Sprintf("%s: %v", context, err),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"context": context,
			"error":   err.Error(),
		},
	}

	return s.SendNotification(ctx, notification)
}

// NotifyWarning 通知警告
func (s *DefaultNotificationService) NotifyWarning(ctx context.Context, message string) error {
	s.logger.Warn("Notifying warning: %s", message)

	notification := &Notification{
		Type:      NotificationTypeWarning,
		Title:     "系统警告",
		Message:   message,
		Timestamp: time.Now(),
	}

	return s.SendNotification(ctx, notification)
}

// NotifySuccess 通知成功
func (s *DefaultNotificationService) NotifySuccess(ctx context.Context, message string) error {
	s.logger.Info("Notifying success: %s", message)

	notification := &Notification{
		Type:      NotificationTypeSuccess,
		Title:     "操作成功",
		Message:   message,
		Timestamp: time.Now(),
	}

	return s.SendNotification(ctx, notification)
}

// buildLarkMessage 构建飞书消息
func (s *DefaultNotificationService) buildLarkMessage(notification *Notification) map[string]interface{} {
	// 根据通知类型选择颜色
	color := "blue"
	switch notification.Type {
	case NotificationTypeError:
		color = "red"
	case NotificationTypeWarning:
		color = "orange"
	case NotificationTypeSuccess:
		color = "green"
	}

	// 构建消息内容
	content := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"config": map[string]interface{}{
				"wide_screen_mode": true,
			},
			"header": map[string]interface{}{
				"title": map[string]interface{}{
					"tag":     "plain_text",
					"content": notification.Title,
				},
				"template": color,
			},
			"elements": []map[string]interface{}{
				{
					"tag": "div",
					"text": map[string]interface{}{
						"tag":     "plain_text",
						"content": notification.Message,
					},
				},
				{
					"tag": "note",
					"elements": []map[string]interface{}{
						{
							"tag":     "plain_text",
							"content": notification.Timestamp.Format("2006-01-02 15:04:05"),
						},
					},
				},
			},
		},
	}

	return content
}

// Notification 通知信息
type Notification struct {
	Type      NotificationType
	Title     string
	Message   string
	Timestamp time.Time
	Data      map[string]interface{}
}

// NotificationType 通知类型
type NotificationType string

const (
	NotificationTypeInfo    NotificationType = "info"
	NotificationTypeWarning NotificationType = "warning"
	NotificationTypeError   NotificationType = "error"
	NotificationTypeSuccess NotificationType = "success"
)

