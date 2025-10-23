package interfaces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
)

// DefaultWebhookService 默认 Webhook 服务实现
type DefaultWebhookService struct {
	logger     common.Logger
	webhooks   map[string]*WebhookConfig
	httpClient *http.Client
	mu         sync.RWMutex
}

// WebhookConfig Webhook 配置
type WebhookConfig struct {
	URL         string
	Events      []string // 订阅的事件类型
	Secret      string   // 用于签名验证
	Enabled     bool
	RetryCount  int
	Timeout     time.Duration
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService(logger common.Logger) WebhookService {
	return &DefaultWebhookService{
		logger:   logger,
		webhooks: make(map[string]*WebhookConfig),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// RegisterWebhook 注册 Webhook
func (s *DefaultWebhookService) RegisterWebhook(ctx context.Context, id string, config *WebhookConfig) error {
	s.logger.Info("Registering webhook: %s", id)

	s.mu.Lock()
	defer s.mu.Unlock()

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Second
	}

	if config.RetryCount == 0 {
		config.RetryCount = 3
	}

	config.Enabled = true
	s.webhooks[id] = config

	s.logger.Info("Webhook registered successfully: %s", id)
	return nil
}

// UnregisterWebhook 注销 Webhook
func (s *DefaultWebhookService) UnregisterWebhook(ctx context.Context, id string) error {
	s.logger.Info("Unregistering webhook: %s", id)

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.webhooks, id)

	s.logger.Info("Webhook unregistered successfully: %s", id)
	return nil
}

// SendWebhook 发送 Webhook
func (s *DefaultWebhookService) SendWebhook(ctx context.Context, event *models.Event) error {
	s.logger.Debug("Sending webhook for event: %s", event.ID)

	s.mu.RLock()
	webhooks := make([]*WebhookConfig, 0)
	for _, config := range s.webhooks {
		if config.Enabled && s.shouldSendEvent(config, event) {
			webhooks = append(webhooks, config)
		}
	}
	s.mu.RUnlock()

	if len(webhooks) == 0 {
		s.logger.Debug("No webhooks to send for event: %s", event.ID)
		return nil
	}

	// 构建 Webhook 消息
	payload := map[string]interface{}{
		"event_id":   event.ID,
		"event_type": event.Type,
		"match_id":   event.MatchID,
		"source":     event.Source,
		"timestamp":  event.Timestamp,
		"data":       event.Data,
	}

	// 并发发送到所有 Webhook
	var wg sync.WaitGroup
	for _, config := range webhooks {
		wg.Add(1)
		go func(cfg *WebhookConfig) {
			defer wg.Done()
			s.sendToWebhook(ctx, cfg, payload)
		}(config)
	}

	wg.Wait()

	s.logger.Debug("Webhooks sent for event: %s", event.ID)
	return nil
}

// shouldSendEvent 判断是否应该发送事件
func (s *DefaultWebhookService) shouldSendEvent(config *WebhookConfig, event *models.Event) bool {
	if len(config.Events) == 0 {
		return true // 订阅所有事件
	}

	for _, eventType := range config.Events {
		if eventType == string(event.Type) {
			return true
		}
	}

	return false
}

// sendToWebhook 发送到 Webhook
func (s *DefaultWebhookService) sendToWebhook(ctx context.Context, config *WebhookConfig, payload interface{}) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal webhook payload: %v", err)
		return
	}

	// 重试逻辑
	for i := 0; i < config.RetryCount; i++ {
		if i > 0 {
			s.logger.Debug("Retrying webhook (attempt %d/%d): %s", i+1, config.RetryCount, config.URL)
			time.Sleep(time.Duration(i) * time.Second)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", config.URL, bytes.NewBuffer(jsonData))
		if err != nil {
			s.logger.Error("Failed to create webhook request: %v", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "UOF-Service-Webhook/1.0")

		// 添加签名
		if config.Secret != "" {
			// 这里可以实现 HMAC 签名
			req.Header.Set("X-Webhook-Signature", config.Secret)
		}

		// 设置超时
		client := &http.Client{
			Timeout: config.Timeout,
		}

		resp, err := client.Do(req)
		if err != nil {
			s.logger.Error("Failed to send webhook: %v", err)
			continue
		}

		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			s.logger.Debug("Webhook sent successfully: %s (status: %d)", config.URL, resp.StatusCode)
			return
		}

		s.logger.Warn("Webhook failed with status %d: %s", resp.StatusCode, config.URL)
	}

	s.logger.Error("Webhook failed after %d retries: %s", config.RetryCount, config.URL)
}

// GetWebhooks 获取所有 Webhook
func (s *DefaultWebhookService) GetWebhooks(ctx context.Context) (map[string]*WebhookConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	webhooks := make(map[string]*WebhookConfig)
	for id, config := range s.webhooks {
		webhooks[id] = config
	}

	return webhooks, nil
}

// EnableWebhook 启用 Webhook
func (s *DefaultWebhookService) EnableWebhook(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, exists := s.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found: %s", id)
	}

	config.Enabled = true
	s.logger.Info("Webhook enabled: %s", id)
	return nil
}

// DisableWebhook 禁用 Webhook
func (s *DefaultWebhookService) DisableWebhook(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, exists := s.webhooks[id]
	if !exists {
		return fmt.Errorf("webhook not found: %s", id)
	}

	config.Enabled = false
	s.logger.Info("Webhook disabled: %s", id)
	return nil
}

