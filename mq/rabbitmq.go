package mq

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-utils/cond"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"github.com/magic-lib/go-servicekit/tracer"
	"log"
	"net"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/streadway/amqp"
)

const (
	resendDelay = 5 * time.Second
	maxRetries  = 3
)

var (
	connectPool = cmap.New[*cache.CommPool[*amqp.Connection]]()
)

// RabbitMQClient 封装了 RabbitMQ 的连接和通道
type RabbitMQClient struct {
	url     string
	onePool *cache.CommPool[*amqp.Connection]
	channel *amqp.Channel
	mu      sync.RWMutex
}

// newRabbitMQClient 创建一个新的 RabbitMQ 客户端并连接
func newRabbitMQClient(url string, conn *conn.Connect) (*RabbitMQClient, error) {
	if url == "" {
		if conn == nil {
			return nil, fmt.Errorf("RabbitMQ conn is empty")
		}
		if conn.Protocol == "" {
			conn.Protocol = "amqp"
		}

		url = fmt.Sprintf("%s://%s:%s@%s/", conn.Protocol, conn.Username, conn.Password, net.JoinHostPort(conn.Host, conn.Port))
	}
	if !connectPool.Has(url) {
		connectPool.Set(url, getAmqpConnection(url))
	}
	onePool, ok := connectPool.Get(url)
	if !ok {
		return nil, fmt.Errorf("RabbitMQ conn is empty")
	}
	return &RabbitMQClient{
		url:     url,
		onePool: onePool,
	}, nil
}

func (c *RabbitMQClient) connect() (*amqp.Connection, error) {
	one, err := c.onePool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get rabbitmq connection: %w", err)
	}
	return one.Resource, nil
}
func (c *RabbitMQClient) getChannel() (*amqp.Channel, error) {
	connect, err := c.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to get rabbitmq connection: %w", err)
	}
	channel, err := connect.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}
	return channel, nil
}

func getAmqpConnection(url string) *cache.CommPool[*amqp.Connection] {
	checkRpcConn := func(oneConn *amqp.Connection) error {
		closed := oneConn.IsClosed()
		if !closed {
			return nil
		}
		return fmt.Errorf("%s connect closed", url)
	}
	closeRpcConn := func(oneConn *amqp.Connection) error {
		return oneConn.Close()
	}
	// 初始化gRPC客户端
	rpcConn := cache.NewResPool(&cache.ResPoolConfig[*amqp.Connection]{
		MaxSize:  10,               //缓存池最大连接数
		MaxUsage: 30 * time.Second, //最长使用时间，如果超过这个时间，自动放回连接池，避免没有主动调用Put方法
		New: func() (*amqp.Connection, error) {
			connection, err := amqp.Dial(url)
			if err != nil {
				return nil, fmt.Errorf("failed to dial rabbitmq: %w", err)
			}

			_, err = connection.Channel()
			if err != nil {
				return nil, fmt.Errorf("failed to open a channel: %w", err)
			}

			log.Println("Successfully connected to RabbitMQ", "rabbitmq")
			return connection, nil
		},
		CheckFunc: checkRpcConn,
		CloseFunc: closeRpcConn,
	})
	return rpcConn
}

func (c *RabbitMQClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channel != nil {
		_ = c.channel.Close()
		c.channel = nil
	}
}

// RabbitMQPublisher 实现了 Publisher 接口
type RabbitMQPublisher struct {
	client    *RabbitMQClient
	QueueName string
	Exchange  string
	Kind      string
}

// NewRabbitMQPublisher 创建一个新的 RabbitMQ 发布者
func NewRabbitMQPublisher(url string, cfg *RabbitMQPublisher) (Publisher, error) {
	client, err := newRabbitMQClient(url, nil)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &RabbitMQPublisher{client: client}, nil
	}
	cfg.client = client
	return cfg, nil
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, event Event) (string, error) {
	channel, err := p.client.getChannel()
	if err != nil {
		return "", err
	}

	if p.Exchange == "" {
		return event.MessageId, fmt.Errorf("exchange name is empty")
	}
	if len(event.Payload) == 0 {
		return event.MessageId, fmt.Errorf("event body empty")
	}

	if p.Kind != "" {
		err = channel.ExchangeDeclare(
			p.Exchange, // 交换机名称
			p.Kind,     // 交换机类型（direct）direct、fanout、topic、headers
			true,       // 持久化
			false,      // 自动删除
			false,      // 非排他性
			false,      // 不阻塞
			nil,        // 额外参数
		)
	} else {
		if p.QueueName == "" {
			return "", fmt.Errorf("queue name is empty")
		}

		//amqp.Table{
		//	"x-message-ttl": 3600000, // 消息 1小时过期
		//	"x-max-length":  10000,   // 队列最多存 10000 条消息
		//}

		_, err = channel.QueueDeclare(
			p.QueueName, // 队列名称
			true,        // 持久化（重启后队列不丢失）
			false,       // 是否为自动删除队列
			false,       // 是否为排他性队列
			false,       // 是否非阻塞声明
			nil,         // 额外参数
		)
	}

	if err != nil {
		return event.MessageId, fmt.Errorf("failed to declare an exchange: %w", err)
	}

	// 如果 event.Headers 为 nil，则初始化
	if event.Headers == nil {
		event.Headers = make(map[string]interface{})
	}

	msg := amqp.Publishing{
		ContentType:  "text/plain",
		DeliveryMode: amqp.Persistent, // 消息持久化（重启后消息不丢失）
		MessageId:    event.MessageId,
		Body:         event.Payload,
		Headers:      amqp.Table(event.Headers), // amqp.Table 的底层就是 map[string]interface{}
	}

	if msg.MessageId == "" {
		msg.MessageId = uuid.NewString()
	}
	if cond.IsJson(string(event.Payload)) {
		msg.ContentType = "application/json"
	}

	_, ok := tracer.TraceProvider()
	if ok {
		tc := tracer.GetTraceConfig()
		if tc != nil {
			msg.Headers = tc.RabbitMQPublishTable(ctx, msg.Headers)
		}
	}

	routingKey := event.RoutingKey
	if routingKey == "" {
		routingKey = p.QueueName
	}

	for i := 0; i < maxRetries; i++ {
		err = channel.Publish(
			p.Exchange, // 交换机名称（使用默认交换机）
			routingKey, // 路由键（队列名称）
			false,      // 非强制模式
			false,      // 非立即模式
			msg,
		)
		if err == nil {
			return msg.MessageId, nil
		}
		log.Println(err, "Failed to publish message, retrying...", "rabbitmq", "attempt", i+1)
		time.Sleep(resendDelay)
	}
	return msg.MessageId, fmt.Errorf("failed to publish message after %d retries", maxRetries)
}

func (p *RabbitMQPublisher) Close() {
	p.client.Close()
}

// RabbitMQConsumer 实现了 Consumer 接口
type RabbitMQConsumer struct {
	client *RabbitMQClient
}

// NewRabbitMQConsumer 创建一个新的 RabbitMQ 消费者
func NewRabbitMQConsumer(url string) (Consumer, error) {
	client, err := newRabbitMQClient(url, nil)
	if err != nil {
		return nil, err
	}
	return &RabbitMQConsumer{client: client}, nil
}

func (c *RabbitMQConsumer) Start(queueName, routingKey, exchangeName string, handler ConsumerHandler) error {
	channel, err := c.client.getChannel()
	if err != nil {
		return err
	}

	if err := channel.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("failed to declare an exchange: %w", err)
	}

	q, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	if err := channel.QueueBind(
		q.Name,
		routingKey,
		exchangeName,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("failed to bind a queue: %w", err)
	}

	msgs, err := channel.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	goroutines.GoAsync(func(params ...interface{}) {
		for d := range msgs {
			ctx := context.Background()
			_, ok := tracer.TraceProvider()
			if ok {
				tc := tracer.GetTraceConfig()
				if tc != nil {
					ctx = tc.RabbitMQConsumer(ctx, d.Headers)
				}
			}
			if err := handler(ctx, d.MessageId, d.Body); err == nil {
				_ = d.Ack(false)
			} else {
				log.Println(err, "Failed to handle message", "rabbitmq")
				// 消息处理失败，根据业务决定是重入队列还是丢弃
				_ = d.Nack(false, true) // true to requeue
			}
		}
	})

	return nil
}

func (c *RabbitMQConsumer) Close() {
	c.client.Close()
}
