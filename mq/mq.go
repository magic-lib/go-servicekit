package mq

import "context"

// Event 代表一个通用的消息事件
type Event struct {
	RoutingKey string         `json:"routing_key"` // 路由键
	MessageId  string         `json:"message_id"`  // 消息ID
	Payload    []byte         `json:"payload"`     // 消息内容
	Headers    map[string]any `json:"headers"`     // 用于传递元数据，如 Trace Context
}

// Publisher 定义了消息发布者的接口
type Publisher interface {
	// Publish 发布一个事件
	Publish(ctx context.Context, event Event) (string, error)
	// Close 关闭发布者连接
	Close()
}

// ConsumerHandler 是处理消息的函数类型
type ConsumerHandler func(ctx context.Context, messageId string, payload []byte) error

// Consumer 定义了消息消费者的接口
type Consumer interface {
	// Start 开始消费指定队列的消息
	// queueName: 队列名称
	// routingKey: 绑定的路由键
	// handler: 消息处理函数
	Start(queueName, routingKey, exchangeName string, handler ConsumerHandler) error
	// Close 关闭消费者连接
	Close()
}
