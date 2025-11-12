package services

import (
	"sync"
	"uof-service/logger"
)

// InMemoryBroker 是 MessageBroker 接口的内存实现，用于模拟 Kafka
type InMemoryBroker struct {
	// 存储每个 Topic 对应的消费者通道列表
	consumers map[string][]chan BrokerMessage
	mu        sync.RWMutex
}

// NewInMemoryBroker 创建 InMemoryBroker 实例
func NewInMemoryBroker() *InMemoryBroker {
	return &InMemoryBroker{
		consumers: make(map[string][]chan BrokerMessage),
	}
}

// Produce 实现 MessageBroker 接口
func (b *InMemoryBroker) Produce(msg BrokerMessage) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 查找所有订阅了该 Topic 的消费者通道
	consumerChans, ok := b.consumers[msg.Topic]
	if !ok {
		logger.Printf("[InMemoryBroker] ⚠️ Topic %s has no active consumers. Message dropped.", msg.Topic)
		return nil 
	}

	// 模拟 Kafka 的 Consumer Group 行为：将消息发送给第一个消费者（简化实现）
	if len(consumerChans) > 0 {
		// 使用 select 避免阻塞，如果通道满了则丢弃（模拟高吞吐量下的背压）
		select {
		case consumerChans[0] <- msg:
			logger.Printf("[InMemoryBroker] Produced message to topic %s", msg.Topic)
		default:
			logger.Printf("[InMemoryBroker] ⚠️ Topic %s consumer channel full. Message dropped.", msg.Topic)
		}
	} else {
		logger.Printf("[InMemoryBroker] ⚠️ Topic %s has no active consumers. Message dropped.", msg.Topic)
	}

	return nil
}

// Consume 实现 MessageBroker 接口
func (b *InMemoryBroker) Consume(topic string) (<-chan BrokerMessage, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 创建一个新的通道作为消费者的消息队列
	consumerChan := make(chan BrokerMessage, 1000) // 缓冲区大小 1000

	// 将新的消费者通道添加到对应的 Topic 列表中
	b.consumers[topic] = append(b.consumers[topic], consumerChan)

	logger.Printf("[InMemoryBroker] Consumer subscribed to topic %s. Total consumers for topic: %d", topic, len(b.consumers[topic]))

	return consumerChan, nil
}

// Close 实现 MessageBroker 接口
func (b *InMemoryBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 关闭所有消费者通道
	for _, chans := range b.consumers {
		for _, ch := range chans {
			close(ch)
		}
	}
	b.consumers = make(map[string][]chan BrokerMessage)

	logger.Println("[InMemoryBroker] Closed all channels.")
	return nil
}
