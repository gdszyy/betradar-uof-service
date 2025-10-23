package processing

import (
	"context"
	"fmt"
	"time"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
)

// DefaultDataProcessor 默认数据处理器实现
type DefaultDataProcessor struct {
	name      string
	logger    common.Logger
	validator DataValidator
	storage   DataStorage
}

// NewDataProcessor 创建数据处理器
func NewDataProcessor(name string, logger common.Logger, validator DataValidator, storage DataStorage) DataProcessor {
	return &DefaultDataProcessor{
		name:      name,
		logger:    logger,
		validator: validator,
		storage:   storage,
	}
}

// Process 处理事件
func (p *DefaultDataProcessor) Process(ctx context.Context, event *models.Event) error {
	p.logger.Debug("Processing event: %s (type: %s, source: %s)", event.ID, event.Type, event.Source)

	// 1. 验证事件
	if p.validator != nil {
		if err := p.validator.Validate(ctx, event); err != nil {
			p.logger.Error("Event validation failed: %v", err)
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// 2. 处理时间戳
	if event.ProcessedAt.IsZero() {
		event.ProcessedAt = time.Now()
	}

	// 3. 根据事件类型进行特定处理
	switch event.Type {
	case models.EventTypeOddsChange:
		if err := p.processOddsChange(ctx, event); err != nil {
			return err
		}
	case models.EventTypeMatchUpdate:
		if err := p.processMatchUpdate(ctx, event); err != nil {
			return err
		}
	case models.EventTypeSettlement:
		if err := p.processSettlement(ctx, event); err != nil {
			return err
		}
	default:
		p.logger.Warn("Unknown event type: %s", event.Type)
	}

	// 4. 存储事件
	if p.storage != nil {
		if err := p.storage.SaveEvent(ctx, event); err != nil {
			p.logger.Error("Failed to save event: %v", err)
			return fmt.Errorf("storage failed: %w", err)
		}
	}

	p.logger.Debug("Event processed successfully: %s", event.ID)
	return nil
}

// GetName 获取处理器名称
func (p *DefaultDataProcessor) GetName() string {
	return p.name
}

// processOddsChange 处理赔率变化事件
func (p *DefaultDataProcessor) processOddsChange(ctx context.Context, event *models.Event) error {
	p.logger.Debug("Processing odds change event: %s", event.ID)
	// 具体的赔率变化处理逻辑
	return nil
}

// processMatchUpdate 处理比赛更新事件
func (p *DefaultDataProcessor) processMatchUpdate(ctx context.Context, event *models.Event) error {
	p.logger.Debug("Processing match update event: %s", event.ID)
	// 具体的比赛更新处理逻辑
	return nil
}

// processSettlement 处理结算事件
func (p *DefaultDataProcessor) processSettlement(ctx context.Context, event *models.Event) error {
	p.logger.Debug("Processing settlement event: %s", event.ID)
	// 具体的结算处理逻辑
	return nil
}

// ProcessingPipelineImpl 处理管道实现
type ProcessingPipelineImpl struct {
	logger     common.Logger
	processors []DataProcessor
	dispatcher EventDispatcher
}

// NewProcessingPipeline 创建处理管道
func NewProcessingPipeline(logger common.Logger, dispatcher EventDispatcher) ProcessingPipeline {
	return &ProcessingPipelineImpl{
		logger:     logger,
		processors: make([]DataProcessor, 0),
		dispatcher: dispatcher,
	}
}

// AddProcessor 添加处理器
func (p *ProcessingPipelineImpl) AddProcessor(processor DataProcessor) {
	p.processors = append(p.processors, processor)
	p.logger.Info("Added processor: %s", processor.GetName())
}

// RemoveProcessor 移除处理器
func (p *ProcessingPipelineImpl) RemoveProcessor(name string) error {
	for i, processor := range p.processors {
		if processor.GetName() == name {
			p.processors = append(p.processors[:i], p.processors[i+1:]...)
			p.logger.Info("Removed processor: %s", name)
			return nil
		}
	}
	return fmt.Errorf("processor %s not found", name)
}

// Process 处理事件
func (p *ProcessingPipelineImpl) Process(ctx context.Context, event *models.Event) error {
	p.logger.Debug("Pipeline processing event: %s", event.ID)

	// 依次通过所有处理器
	for _, processor := range p.processors {
		if err := processor.Process(ctx, event); err != nil {
			p.logger.Error("Processor %s failed: %v", processor.GetName(), err)
			return err
		}
	}

	// 分发事件
	if p.dispatcher != nil {
		if err := p.dispatcher.Dispatch(ctx, event); err != nil {
			p.logger.Error("Event dispatch failed: %v", err)
			return err
		}
	}

	p.logger.Debug("Pipeline processed event successfully: %s", event.ID)
	return nil
}

// GetProcessors 获取所有处理器
func (p *ProcessingPipelineImpl) GetProcessors() []DataProcessor {
	return p.processors
}

