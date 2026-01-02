package rabbitmq

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"github.com/magic-lib/go-servicekit/tracer"
)

// MessageHandler 消息消费的回调方法
type MessageHandler func(ctx context.Context, messageId, messageData string) error

// ConsumerOptions 启动一个消费端的所有参数
type ConsumerOptions struct {
	QueueName string
	Handler   MessageHandler //执行的方法
}

// StartConsumer 初始化一个消费端
func (r *RabbitClient) StartConsumer(opt *ConsumerOptions) error {
	ch, err := r.client.Channel()
	if err != nil {
		defer func() {
			_ = ch.Close()
		}()
		return fmt.Errorf("无法创建频道, %w", err)
	}
	// 设置每次只接收一条未确认的消息
	err = ch.Qos(
		1,     // 预取计数
		0,     // 预取大小
		false, // 非全局
	)
	if err != nil {
		defer func() {
			_ = ch.Close()
		}()
		return fmt.Errorf("设置 QoS 失败, %w", err)
	}

	goroutines.GoAsync(func(params ...interface{}) {
		defer func() {
			fmt.Println("consumer goroutine exit")
			_ = ch.Close()
		}()
		for {
			// 消费消息
			msgs, err := ch.Consume(
				opt.QueueName, // 队列名称
				"",            // 消费者标签
				false,         // 非自动确认（需手动确认）
				false,         // 非排他性
				false,         // 不阻塞
				false,         // 无本地
				nil,           // 额外参数
			)
			if err != nil {
				fmt.Println("consumer无法声明队列: ", err.Error())
				continue
			}

			for d := range msgs {
				ctx := context.Background()
				_, ok := tracer.TraceProvider()
				if ok {
					tc := tracer.GetTraceConfig()
					if tc != nil {
						ctx = tc.RabbitMQConsumer(ctx, d.Headers)
					}
				}
				err = opt.Handler(ctx, d.MessageId, string(d.Body))
				if err == nil {
					err = d.Ack(false)
					fmt.Println("消息已确认", err)
					continue
				} else {
					fmt.Printf("rabbit [%s]receive message error: %v\n", opt.QueueName, err)
				}
			}
		}
	})
	return nil
}
