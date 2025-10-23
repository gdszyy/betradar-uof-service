package processing

import (
	"context"
	"fmt"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
)

// DefaultDataValidator 默认数据验证器实现
type DefaultDataValidator struct {
	name   string
	logger common.Logger
	rules  []ValidationRule
}

// ValidationRule 验证规则
type ValidationRule func(ctx context.Context, event *models.Event) error

// NewDataValidator 创建数据验证器
func NewDataValidator(name string, logger common.Logger) DataValidator {
	validator := &DefaultDataValidator{
		name:   name,
		logger: logger,
		rules:  make([]ValidationRule, 0),
	}

	// 添加默认验证规则
	validator.AddRule(validateEventID)
	validator.AddRule(validateEventType)
	validator.AddRule(validateEventSource)
	validator.AddRule(validateEventTimestamp)

	return validator
}

// Validate 验证事件
func (v *DefaultDataValidator) Validate(ctx context.Context, event *models.Event) error {
	v.logger.Debug("Validating event: %s", event.ID)

	for _, rule := range v.rules {
		if err := rule(ctx, event); err != nil {
			v.logger.Error("Validation failed: %v", err)
			return common.NewAppError("VALIDATION_FAILED", "Event validation failed", err)
		}
	}

	v.logger.Debug("Event validated successfully: %s", event.ID)
	return nil
}

// GetName 获取验证器名称
func (v *DefaultDataValidator) GetName() string {
	return v.name
}

// AddRule 添加验证规则
func (v *DefaultDataValidator) AddRule(rule ValidationRule) {
	v.rules = append(v.rules, rule)
}

// 默认验证规则

// validateEventID 验证事件 ID
func validateEventID(ctx context.Context, event *models.Event) error {
	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	return nil
}

// validateEventType 验证事件类型
func validateEventType(ctx context.Context, event *models.Event) error {
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}

	// 验证事件类型是否有效
	validTypes := map[string]bool{
		models.EventTypeOddsChange:   true,
		models.EventTypeMatchUpdate:  true,
		models.EventTypeSettlement:   true,
		models.EventTypeBetStop:      true,
		models.EventTypeBetCancel:    true,
		models.EventTypeFixtureChange: true,
	}

	if !validTypes[event.Type] {
		return fmt.Errorf("invalid event type: %s", event.Type)
	}

	return nil
}

// validateEventSource 验证事件来源
func validateEventSource(ctx context.Context, event *models.Event) error {
	if event.Source == "" {
		return fmt.Errorf("event source is required")
	}

	// 验证事件来源是否有效
	validSources := map[string]bool{
		models.EventSourceUOF:        true,
		models.EventSourceLiveData:   true,
		models.EventSourceTheSports:  true,
	}

	if !validSources[event.Source] {
		return fmt.Errorf("invalid event source: %s", event.Source)
	}

	return nil
}

// validateEventTimestamp 验证事件时间戳
func validateEventTimestamp(ctx context.Context, event *models.Event) error {
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}

	// 检查时间戳是否在合理范围内（不能是未来时间，不能太久以前）
	now := time.Now()
	if event.Timestamp.After(now.Add(5 * time.Minute)) {
		return fmt.Errorf("event timestamp is in the future")
	}

	if event.Timestamp.Before(now.Add(-24 * time.Hour)) {
		return fmt.Errorf("event timestamp is too old (>24h)")
	}

	return nil
}

// CompositeValidator 组合验证器
type CompositeValidator struct {
	name       string
	logger     common.Logger
	validators []DataValidator
}

// NewCompositeValidator 创建组合验证器
func NewCompositeValidator(name string, logger common.Logger) *CompositeValidator {
	return &CompositeValidator{
		name:       name,
		logger:     logger,
		validators: make([]DataValidator, 0),
	}
}

// AddValidator 添加验证器
func (cv *CompositeValidator) AddValidator(validator DataValidator) {
	cv.validators = append(cv.validators, validator)
	cv.logger.Info("Added validator: %s", validator.GetName())
}

// Validate 验证事件
func (cv *CompositeValidator) Validate(ctx context.Context, event *models.Event) error {
	for _, validator := range cv.validators {
		if err := validator.Validate(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// GetName 获取验证器名称
func (cv *CompositeValidator) GetName() string {
	return cv.name
}

