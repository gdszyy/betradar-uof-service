package services

import (
	"fmt"
)

// BrokerMessage 定义了在 Broker 中传输的消息结构
type BrokerMessage struct {
	Topic string
	Key   string // 可以是 EventID 或其他唯一标识
	Value []byte // 原始 XML 消息体
}

// MessageBroker 定义了消息队列的抽象接口
type MessageBroker interface {
	// Produce 发送消息到指定的 Topic
	Produce(msg BrokerMessage) error
	// Consume 订阅指定的 Topic，返回一个消息通道
	Consume(topic string) (<-chan BrokerMessage, error)
	// Close 关闭 Broker 连接
	Close() error
}

// GetTopicName 根据消息类型获取 Kafka Topic 名称
func GetTopicName(messageType string) string {
	// 统一将消息类型转换为 Topic 名称
	return fmt.Sprintf("uof-message-%s", messageType)
}
