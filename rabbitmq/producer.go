package rabbitmq

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

// ProducerOptions 启动一个生产端的所有参数
type ProducerOptions struct {
	QueueName string
	MessageId string
	Content   string
}

// ProduceMessage 发送消息到队列
func (r *rabbitClient) ProduceMessage(opt *ProducerOptions) (messageId string, err error) {
	if opt == nil || opt.QueueName == "" {
		return "", fmt.Errorf("ProduceMessage param error")
	}

	//没有内容
	if opt.Content == "" {
		return "", nil
	}
	if r.client == nil {
		return "", fmt.Errorf("r.client is nil")
	}

	ch, err := r.client.Channel()
	defer func() {
		_ = ch.Close()
	}()
	if err != nil {
		return "", err
	}

	q, err := ch.QueueDeclare(
		opt.QueueName, // 队列名称
		true,          // 持久化（重启后队列不丢失）
		false,         // 是否为自动删除队列
		false,         // 是否为排他性队列
		false,         // 是否非阻塞声明
		nil,           // 额外参数
	)

	//amqp.Table{
	//	"x-message-ttl": 3600000, // 消息 1小时过期
	//	"x-max-length":  10000,   // 队列最多存 10000 条消息
	//}

	if err != nil {
		return "", fmt.Errorf("producer无法声明队列: %w", err)
	}

	config := amqp.Publishing{}
	config.ContentType = "application/json"
	config.DeliveryMode = amqp.Persistent // 消息持久化（重启后消息不丢失）
	config.MessageId = opt.MessageId      //消息id
	if opt.MessageId == "" {
		config.MessageId = uuid.NewString()
	}
	config.Body = []byte(opt.Content)

	err = ch.Publish(
		"",     // 交换机名称（使用默认交换机）
		q.Name, // 路由键（队列名称）
		false,  // 非强制模式
		false,  // 非立即模式
		config,
	)
	if err != nil {
		return "", err
	}

	return config.MessageId, nil
}
