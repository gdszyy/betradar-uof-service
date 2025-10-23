package interfaces

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/ingestion"
)

// DefaultHealthChecker 默认健康检查实现
type DefaultHealthChecker struct {
	logger            common.Logger
	db                *sql.DB
	connectionManager ingestion.ConnectionManager
	checks            map[string]HealthCheckFunc
	mu                sync.RWMutex
}

// HealthCheckFunc 健康检查函数
type HealthCheckFunc func(context.Context) error

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult 检查结果
type CheckResult struct {
	Status  string        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency"`
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(
	logger common.Logger,
	db *sql.DB,
	connectionManager ingestion.ConnectionManager,
) HealthChecker {
	hc := &DefaultHealthChecker{
		logger:            logger,
		db:                db,
		connectionManager: connectionManager,
		checks:            make(map[string]HealthCheckFunc),
	}

	// 注册默认检查
	hc.RegisterCheck("database", hc.checkDatabase)
	hc.RegisterCheck("connections", hc.checkConnections)

	return hc
}

// Check 执行健康检查
func (h *DefaultHealthChecker) Check(ctx context.Context) (*HealthStatus, error) {
	h.logger.Debug("Performing health check")

	status := &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make(map[string]CheckResult),
	}

	h.mu.RLock()
	checks := make(map[string]HealthCheckFunc)
	for name, fn := range h.checks {
		checks[name] = fn
	}
	h.mu.RUnlock()

	// 执行所有检查
	for name, checkFunc := range checks {
		start := time.Now()
		err := checkFunc(ctx)
		latency := time.Since(start)

		result := CheckResult{
			Status:  "healthy",
			Latency: latency,
		}

		if err != nil {
			result.Status = "unhealthy"
			result.Message = err.Error()
			status.Status = "unhealthy"
			h.logger.Warn("Health check failed: %s - %v", name, err)
		}

		status.Checks[name] = result
	}

	h.logger.Debug("Health check completed: %s", status.Status)
	return status, nil
}

// RegisterCheck 注册健康检查
func (h *DefaultHealthChecker) RegisterCheck(name string, checkFunc HealthCheckFunc) error {
	h.logger.Debug("Registering health check: %s", name)

	h.mu.Lock()
	defer h.mu.Unlock()

	h.checks[name] = checkFunc
	return nil
}

// UnregisterCheck 注销健康检查
func (h *DefaultHealthChecker) UnregisterCheck(name string) error {
	h.logger.Debug("Unregistering health check: %s", name)

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.checks, name)
	return nil
}

// checkDatabase 检查数据库连接
func (h *DefaultHealthChecker) checkDatabase(ctx context.Context) error {
	if h.db == nil {
		return fmt.Errorf("database not configured")
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// checkConnections 检查数据源连接
func (h *DefaultHealthChecker) checkConnections(ctx context.Context) error {
	if h.connectionManager == nil {
		return fmt.Errorf("connection manager not configured")
	}

	status := h.connectionManager.GetStatus(ctx)
	
	unhealthyCount := 0
	for name, connStatus := range status {
		if connStatus.Status != "connected" {
			unhealthyCount++
			h.logger.Warn("Connection unhealthy: %s (status: %s)", name, connStatus.Status)
		}
	}

	if unhealthyCount > 0 {
		return fmt.Errorf("%d connections unhealthy", unhealthyCount)
	}

	return nil
}

// IsHealthy 判断系统是否健康
func (h *DefaultHealthChecker) IsHealthy(ctx context.Context) (bool, error) {
	status, err := h.Check(ctx)
	if err != nil {
		return false, err
	}

	return status.Status == "healthy", nil
}

// GetStatus 获取健康状态
func (h *DefaultHealthChecker) GetStatus(ctx context.Context) (*HealthStatus, error) {
	return h.Check(ctx)
}

