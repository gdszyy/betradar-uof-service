package ingestion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"uof-service/config"
	"uof-service/pkg/common"
	"uof-service/services"
)

// DefaultRecoveryManager 默认数据恢复管理器实现
type DefaultRecoveryManager struct {
	config          *config.Config
	logger          common.Logger
	recoveryManager *services.RecoveryManager
	mu              sync.RWMutex
	lastRecovery    map[string]time.Time
}

// NewRecoveryManager 创建数据恢复管理器
func NewRecoveryManager(cfg *config.Config, logger common.Logger, messageStore *services.MessageStore) RecoveryManager {
	return &DefaultRecoveryManager{
		config:          cfg,
		logger:          logger,
		recoveryManager: services.NewRecoveryManager(cfg, messageStore),
		lastRecovery:    make(map[string]time.Time),
	}
}

// TriggerRecovery 触发数据恢复
func (rm *DefaultRecoveryManager) TriggerRecovery(ctx context.Context, req RecoveryRequest) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.logger.Info("Triggering recovery for source: %s, type: %s", req.SourceName, req.RecoveryType)

	// 检查恢复间隔
	if lastTime, exists := rm.lastRecovery[req.SourceName]; exists {
		if time.Since(lastTime) < 1*time.Minute {
			return fmt.Errorf("recovery triggered too frequently, please wait")
		}
	}

	var err error
	switch req.RecoveryType {
	case RecoveryTypeFull:
		err = rm.recoveryManager.TriggerFullRecovery()
	case RecoveryTypeEvent:
		if req.EventID == "" {
			return fmt.Errorf("event ID required for event recovery")
		}
		err = rm.recoveryManager.TriggerEventRecovery(req.EventID)
	case RecoveryTypeStateful:
		err = rm.recoveryManager.TriggerStatefulRecovery()
	default:
		return fmt.Errorf("unknown recovery type: %s", req.RecoveryType)
	}

	if err != nil {
		rm.logger.Error("Recovery failed: %v", err)
		return err
	}

	rm.lastRecovery[req.SourceName] = time.Now()
	rm.logger.Info("Recovery triggered successfully for source: %s", req.SourceName)

	return nil
}

// GetRecoveryStatus 获取恢复状态
func (rm *DefaultRecoveryManager) GetRecoveryStatus(ctx context.Context, sourceName string) (*RecoveryStatus, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	lastTime, exists := rm.lastRecovery[sourceName]
	if !exists {
		return &RecoveryStatus{
			SourceName:   sourceName,
			Status:       "never_recovered",
			LastRecovery: time.Time{},
		}, nil
	}

	return &RecoveryStatus{
		SourceName:   sourceName,
		Status:       "completed",
		LastRecovery: lastTime,
		Message:      "Recovery completed successfully",
	}, nil
}

// ScheduleRecovery 调度定期恢复
func (rm *DefaultRecoveryManager) ScheduleRecovery(ctx context.Context, req RecoveryRequest, interval time.Duration) error {
	rm.logger.Info("Scheduling recovery for source: %s, interval: %v", req.SourceName, interval)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				rm.logger.Info("Recovery scheduler stopped for source: %s", req.SourceName)
				return
			case <-ticker.C:
				if err := rm.TriggerRecovery(ctx, req); err != nil {
					rm.logger.Error("Scheduled recovery failed for source %s: %v", req.SourceName, err)
				}
			}
		}
	}()

	return nil
}

// CancelScheduledRecovery 取消调度的恢复
func (rm *DefaultRecoveryManager) CancelScheduledRecovery(ctx context.Context, sourceName string) error {
	rm.logger.Info("Canceling scheduled recovery for source: %s", sourceName)
	// 通过 context 取消来实现
	return nil
}

