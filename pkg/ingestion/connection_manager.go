package ingestion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"uof-service/pkg/common"
)

// DefaultConnectionManager 默认连接管理器实现
type DefaultConnectionManager struct {
	logger      common.Logger
	sources     map[string]DataSource
	mu          sync.RWMutex
	healthCheck *HealthCheckConfig
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Interval time.Duration
	Timeout  time.Duration
	Enabled  bool
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(logger common.Logger) ConnectionManager {
	return &DefaultConnectionManager{
		logger:  logger,
		sources: make(map[string]DataSource),
		healthCheck: &HealthCheckConfig{
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Enabled:  true,
		},
	}
}

// RegisterSource 注册数据源
func (cm *DefaultConnectionManager) RegisterSource(name string, source DataSource) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.sources[name]; exists {
		return fmt.Errorf("source %s already registered", name)
	}

	cm.sources[name] = source
	cm.logger.Info("Registered data source: %s", name)

	return nil
}

// UnregisterSource 注销数据源
func (cm *DefaultConnectionManager) UnregisterSource(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	source, exists := cm.sources[name]
	if !exists {
		return fmt.Errorf("source %s not found", name)
	}

	// 如果数据源已连接，先断开
	if source.IsConnected() {
		if err := source.Disconnect(); err != nil {
			cm.logger.Warn("Failed to disconnect source %s: %v", name, err)
		}
	}

	delete(cm.sources, name)
	cm.logger.Info("Unregistered data source: %s", name)

	return nil
}

// GetSource 获取数据源
func (cm *DefaultConnectionManager) GetSource(name string) (DataSource, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	source, exists := cm.sources[name]
	if !exists {
		return nil, fmt.Errorf("source %s not found", name)
	}

	return source, nil
}

// ConnectAll 连接所有数据源
func (cm *DefaultConnectionManager) ConnectAll(ctx context.Context) error {
	cm.mu.RLock()
	sources := make(map[string]DataSource)
	for name, source := range cm.sources {
		sources[name] = source
	}
	cm.mu.RUnlock()

	var errors []error
	for name, source := range sources {
		if err := source.Connect(ctx); err != nil {
			cm.logger.Error("Failed to connect source %s: %v", name, err)
			errors = append(errors, fmt.Errorf("source %s: %w", name, err))
		} else {
			cm.logger.Info("Connected source: %s", name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to connect %d sources", len(errors))
	}

	// 启动健康检查
	if cm.healthCheck.Enabled {
		go cm.startHealthCheck(ctx)
	}

	return nil
}

// DisconnectAll 断开所有数据源
func (cm *DefaultConnectionManager) DisconnectAll() error {
	cm.mu.RLock()
	sources := make(map[string]DataSource)
	for name, source := range cm.sources {
		sources[name] = source
	}
	cm.mu.RUnlock()

	var errors []error
	for name, source := range sources {
		if err := source.Disconnect(); err != nil {
			cm.logger.Error("Failed to disconnect source %s: %v", name, err)
			errors = append(errors, fmt.Errorf("source %s: %w", name, err))
		} else {
			cm.logger.Info("Disconnected source: %s", name)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to disconnect %d sources", len(errors))
	}

	return nil
}

// GetConnectionStatus 获取连接状态
func (cm *DefaultConnectionManager) GetConnectionStatus() map[string]ConnectionStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	status := make(map[string]ConnectionStatus)
	for name, source := range cm.sources {
		status[name] = ConnectionStatus{
			Name:      name,
			Type:      string(source.GetType()),
			Connected: source.IsConnected(),
			LastCheck: time.Now(),
		}
	}

	return status
}

// startHealthCheck 启动健康检查
func (cm *DefaultConnectionManager) startHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(cm.healthCheck.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cm.logger.Info("Health check stopped")
			return
		case <-ticker.C:
			cm.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (cm *DefaultConnectionManager) performHealthCheck() {
	cm.mu.RLock()
	sources := make(map[string]DataSource)
	for name, source := range cm.sources {
		sources[name] = source
	}
	cm.mu.RUnlock()

	for name, source := range sources {
		if !source.IsConnected() {
			cm.logger.Warn("Source %s is disconnected", name)
			// 可以在这里添加自动重连逻辑
		}
	}
}

// SetHealthCheckConfig 设置健康检查配置
func (cm *DefaultConnectionManager) SetHealthCheckConfig(config *HealthCheckConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.healthCheck = config
}

// GetHealthCheckConfig 获取健康检查配置
func (cm *DefaultConnectionManager) GetHealthCheckConfig() *HealthCheckConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.healthCheck
}

