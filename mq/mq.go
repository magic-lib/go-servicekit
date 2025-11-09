package mq

import (
	"context"
	"time"
)

// Event 代表一个通用的消息事件
type Event struct {
	Id        string         `json:"id"`        // 消息ID
	Timestamp time.Time      `json:"timestamp"` // 消息创建时间戳
	Headers   map[string]any `json:"headers"`   // 用于传递元数据，如 Trace Context
	Payload   []byte         `json:"payload"`   // 消息内容
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
	Start(handler ConsumerHandler) error
	// Close 关闭消费者连接
	Close()
}
