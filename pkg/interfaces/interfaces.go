package interfaces

import (
	"context"
	"net/http"
)

// APIServer API 服务器接口
type APIServer interface {
	// Start 启动服务器
	Start(ctx context.Context) error
	
	// Stop 停止服务器
	Stop(ctx context.Context) error
	
	// RegisterHandler 注册处理器
	RegisterHandler(path string, handler http.HandlerFunc, methods ...string) error
	
	// GetPort 获取端口
	GetPort() string
}

// WebSocketServer WebSocket 服务器接口
type WebSocketServer interface {
	// Start 启动 WebSocket 服务器
	Start(ctx context.Context) error
	
	// Stop 停止 WebSocket 服务器
	Stop(ctx context.Context) error
	
	// Broadcast 广播消息
	Broadcast(message interface{}) error
	
	// SendToClient 发送消息给特定客户端
	SendToClient(clientID string, message interface{}) error
	
	// GetClientCount 获取客户端数量
	GetClientCount() int
}

// WebhookService Webhook 服务接口
type WebhookService interface {
	// SendWebhook 发送 Webhook
	SendWebhook(ctx context.Context, url string, payload interface{}) error
	
	// RegisterWebhook 注册 Webhook
	RegisterWebhook(ctx context.Context, event string, url string) error
	
	// UnregisterWebhook 注销 Webhook
	UnregisterWebhook(ctx context.Context, event string, url string) error
}

// HealthChecker 健康检查接口
type HealthChecker interface {
	// Check 检查健康状态
	Check(ctx context.Context) (*HealthStatus, error)
	
	// RegisterCheck 注册健康检查
	RegisterCheck(name string, checker func(ctx context.Context) error) error
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks"`
	Version string            `json:"version"`
}

