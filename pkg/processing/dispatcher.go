package processing

import (
	"context"
	"fmt"
	"sync"

	"uof-service/pkg/common"
	"uof-service/pkg/models"
)

// DefaultEventDispatcher 默认事件分发器实现
type DefaultEventDispatcher struct {
	logger      common.Logger
	subscribers map[string][]EventSubscriber
	mu          sync.RWMutex
}

// EventSubscriber 事件订阅者
type EventSubscriber struct {
	ID      string
	Filter  EventSubscriptionFilter
	Handler EventHandler
}

// EventSubscriptionFilter 事件订阅过滤器
type EventSubscriptionFilter struct {
	EventTypes []string
	MatchIDs   []string
	SportIDs   []string
	Sources    []string
}

// EventHandler 事件处理函数
type EventHandler func(ctx context.Context, event *models.Event) error

// NewEventDispatcher 创建事件分发器
func NewEventDispatcher(logger common.Logger) EventDispatcher {
	return &DefaultEventDispatcher{
		logger:      logger,
		subscribers: make(map[string][]EventSubscriber),
	}
}

// Dispatch 分发事件
func (d *DefaultEventDispatcher) Dispatch(ctx context.Context, event *models.Event) error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	d.logger.Debug("Dispatching event: %s (type: %s)", event.ID, event.Type)

	// 获取所有订阅者
	allSubscribers := make([]EventSubscriber, 0)
	for _, subscribers := range d.subscribers {
		allSubscribers = append(allSubscribers, subscribers...)
	}

	// 分发给匹配的订阅者
	dispatchedCount := 0
	for _, subscriber := range allSubscribers {
		if d.matchesFilter(event, subscriber.Filter) {
			go func(sub EventSubscriber) {
				if err := sub.Handler(ctx, event); err != nil {
					d.logger.Error("Subscriber %s handler failed: %v", sub.ID, err)
				}
			}(subscriber)
			dispatchedCount++
		}
	}

	d.logger.Debug("Event dispatched to %d subscribers", dispatchedCount)
	return nil
}

// Subscribe 订阅事件
func (d *DefaultEventDispatcher) Subscribe(ctx context.Context, subscriber EventSubscriber) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Info("Adding subscriber: %s", subscriber.ID)

	// 使用事件类型作为 key 来组织订阅者
	key := "all"
	if len(subscriber.Filter.EventTypes) > 0 {
		key = subscriber.Filter.EventTypes[0]
	}

	d.subscribers[key] = append(d.subscribers[key], subscriber)

	d.logger.Info("Subscriber added: %s (total: %d)", subscriber.ID, d.getTotalSubscribers())
	return nil
}

// Unsubscribe 取消订阅
func (d *DefaultEventDispatcher) Unsubscribe(ctx context.Context, subscriberID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Info("Removing subscriber: %s", subscriberID)

	found := false
	for key, subscribers := range d.subscribers {
		for i, sub := range subscribers {
			if sub.ID == subscriberID {
				d.subscribers[key] = append(subscribers[:i], subscribers[i+1:]...)
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("subscriber %s not found", subscriberID)
	}

	d.logger.Info("Subscriber removed: %s (total: %d)", subscriberID, d.getTotalSubscribers())
	return nil
}

// matchesFilter 检查事件是否匹配过滤器
func (d *DefaultEventDispatcher) matchesFilter(event *models.Event, filter EventSubscriptionFilter) bool {
	// 检查事件类型
	if len(filter.EventTypes) > 0 {
		matched := false
		for _, eventType := range filter.EventTypes {
			if event.Type == eventType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查比赛 ID
	if len(filter.MatchIDs) > 0 {
		matched := false
		for _, matchID := range filter.MatchIDs {
			if event.MatchID == matchID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查运动 ID
	if len(filter.SportIDs) > 0 {
		matched := false
		for _, sportID := range filter.SportIDs {
			if event.SportID == sportID {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// 检查数据源
	if len(filter.Sources) > 0 {
		matched := false
		for _, source := range filter.Sources {
			if event.Source == source {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// getTotalSubscribers 获取订阅者总数
func (d *DefaultEventDispatcher) getTotalSubscribers() int {
	total := 0
	for _, subscribers := range d.subscribers {
		total += len(subscribers)
	}
	return total
}

// GetSubscribers 获取所有订阅者
func (d *DefaultEventDispatcher) GetSubscribers() []EventSubscriber {
	d.mu.RLock()
	defer d.mu.RUnlock()

	allSubscribers := make([]EventSubscriber, 0)
	for _, subscribers := range d.subscribers {
		allSubscribers = append(allSubscribers, subscribers...)
	}

	return allSubscribers
}

// GetSubscriberCount 获取订阅者数量
func (d *DefaultEventDispatcher) GetSubscriberCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.getTotalSubscribers()
}

