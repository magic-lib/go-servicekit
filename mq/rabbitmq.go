package mq

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-utils/cond"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"github.com/magic-lib/go-servicekit/tracer"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

const (
	resendDelay = 2 * time.Second
	maxRetries  = 3
	mqNamespace = "rabbit-mq"
)

type ExchangeType string

const (
	ExchangeTypeDirect  ExchangeType = "direct"
	ExchangeTypeFanout  ExchangeType = "fanout"
	ExchangeTypeTopic   ExchangeType = "topic"
	ExchangeTypeHeaders ExchangeType = "headers"
)

var (
	poolManager *cache.PoolManager[*amqp.Connection]
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
	if url == "" {
		return nil, fmt.Errorf("RabbitMQ url is empty")
	}

	if poolManager == nil {
		poolManager = cache.NewPoolManager[*amqp.Connection]()
	}

	resPool := poolManager.GetPool(mqNamespace, url)
	if resPool == nil {
		resPool = getAmqpConnection(url)
		poolManager.SetPool(mqNamespace, url, resPool)
	}
	return &RabbitMQClient{
		url:     url,
		onePool: resPool,
	}, nil
}

func (c *RabbitMQClient) connect() (*amqp.Connection, error) {
	one, err := c.onePool.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get rabbitmq connection: %w", err)
	}
	return one.Get(), nil
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
	rpcConn := cache.NewResPool[*amqp.Connection](&cache.ResPoolConfig[*amqp.Connection]{
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

type RabbitMQConfig struct {
	Url            string //连接地址，与Connect二选一，如果同时存在，以Url为准
	Connect        *conn.Connect
	PushRetryTimes int          // 推送失败重试次数
	QueueName      string       //队列名
	Exchange       string       //交换器
	Kind           ExchangeType //交换器类型
	RoutingKey     string       //路由键
}

// rabbitMQPublisher 实现了 Publisher 接口
type rabbitMQPublisher struct {
	client *RabbitMQClient
	cfg    *RabbitMQConfig
}

func checkConfig(cfg *RabbitMQConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is empty")
	}

	if cfg.Exchange == "" {
		return fmt.Errorf("exchange is empty")
	}
	if cfg.Kind != "" {
		if cfg.Kind != ExchangeTypeFanout &&
			cfg.Kind != ExchangeTypeDirect &&
			cfg.Kind != ExchangeTypeTopic &&
			cfg.Kind != ExchangeTypeHeaders {
			return fmt.Errorf("kind: %s not support", cfg.Kind)
		}
	}

	if cfg.Kind != "" && cfg.Kind != ExchangeTypeFanout {
		if cfg.RoutingKey == "" {
			return fmt.Errorf("routingKey is empty")
		}
	}
	return nil
}

// NewRabbitMQPublisher 创建一个新的 RabbitMQ 发布者
func NewRabbitMQPublisher(cfg *RabbitMQConfig) (Publisher, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	err := checkConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := newRabbitMQClient(cfg.Url, cfg.Connect)
	if err != nil {
		return nil, err
	}
	publisher := &rabbitMQPublisher{client: client}
	publisher.cfg = cfg
	return publisher, nil
}

func (p *rabbitMQPublisher) getPublishMessage(event *Event) amqp.Publishing {
	if event.Headers == nil {
		event.Headers = make(http.Header)
	}
	headers := make(map[string]any)
	for k, _ := range event.Headers {
		headers[k] = event.Headers.Get(k)
	}

	msg := amqp.Publishing{
		ContentType:  "text/plain",
		DeliveryMode: amqp.Persistent, // 消息持久化（重启后消息不丢失）
		MessageId:    event.Id,
		Body:         event.Payload,
		Headers:      amqp.Table(headers), // amqp.Table 的底层就是 map[string]interface{}
		Timestamp:    event.Timestamp,
	}

	if msg.MessageId == "" {
		msg.MessageId = uuid.NewString()
	}
	if cond.IsJson(string(event.Payload)) {
		msg.ContentType = "application/json"
	}
	msg.Headers["Content-Type"] = msg.ContentType

	return msg
}

func getChannel(client *RabbitMQClient, cfg *RabbitMQConfig) (*amqp.Channel, *amqp.Queue, error) {
	channel, err := client.getChannel()
	if err != nil {
		return nil, nil, err
	}

	if cfg.QueueName != "" {
		queue, err := channel.QueueDeclare(
			cfg.QueueName, // 队列名称
			true,          // 持久化（重启后队列不丢失）
			false,         // 是否为自动删除队列
			false,         // 是否为排他性队列
			false,         // 是否非阻塞声明
			nil,           // 额外参数
		)
		if err == nil {
			return channel, &queue, nil
		}
	}

	if cfg.Kind != "" {
		err = channel.ExchangeDeclare(
			cfg.Exchange,     // 交换机名称，由交换器发送到队列中
			string(cfg.Kind), // 交换机类型（direct）direct、fanout、topic、headers
			true,             // 持久化
			false,            // 自动删除
			false,            // 非排他性
			false,            // 不阻塞
			nil,              // 额外参数
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed %s to declare an exchange: %w", cfg.Kind, err)
		}
		return channel, nil, nil
	}
	return nil, nil, fmt.Errorf("kind is empty")
}

func (p *rabbitMQPublisher) Publish(ctx context.Context, event *Event) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is empty")
	}
	channel, _, err := getChannel(p.client, p.cfg)
	if err != nil {
		return event.Id, fmt.Errorf("failed to declare an exchange: %w", err)
	}

	msg := p.getPublishMessage(event)

	_, ok := tracer.TraceProvider()
	if ok {
		tc := tracer.GetTraceConfig()
		if tc != nil {
			msg.Headers = tc.RabbitMQPublishTable(ctx, msg.Headers)
		}
	}

	if p.cfg.PushRetryTimes <= 0 {
		p.cfg.PushRetryTimes = maxRetries
	}

	for i := 0; i < p.cfg.PushRetryTimes; i++ {
		err = channel.Publish(
			p.cfg.Exchange,   // 交换机名称（使用默认交换机）
			p.cfg.RoutingKey, // 路由键（队列名称）
			false,            // 非强制模式
			false,            // 非立即模式
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

func (p *rabbitMQPublisher) Close() {
	p.client.Close()
}

// RabbitMQConsumer 实现了 Consumer 接口
type RabbitMQConsumer struct {
	client *RabbitMQClient
	cfg    *RabbitMQConfig
}

// NewRabbitMQConsumer 创建一个新的 RabbitMQ 消费者
func NewRabbitMQConsumer(cfg *RabbitMQConfig) (Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	err := checkConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := newRabbitMQClient(cfg.Url, cfg.Connect)

	if err != nil {
		return nil, err
	}
	return &RabbitMQConsumer{
		client: client,
		cfg:    cfg,
	}, nil
}

func (c *RabbitMQConsumer) getChannel() (*amqp.Channel, error) {
	channel, queue, err := getChannel(c.client, c.cfg)
	if err != nil {
		return nil, err
	}
	if queue == nil {
		return channel, nil
	}

	if err := channel.QueueBind(
		queue.Name,
		c.cfg.RoutingKey,
		c.cfg.Exchange,
		false,
		nil,
	); err != nil {
		return channel, fmt.Errorf("failed to bind a queue: %w", err)
	}
	return channel, nil
}

func (c *RabbitMQConsumer) Start(handler ConsumerHandler) error {
	channel, err := c.getChannel()
	if err != nil {
		return err
	}
	closeChan := make(chan *amqp.Error)
	channel.NotifyClose(closeChan)
	goroutines.GoAsync(func(params ...interface{}) {
		for {
			select {
			case err := <-closeChan:
				fmt.Printf("Channel closed with error: %v\n", err)
				channel, _ = c.getChannel()
			}
		}
	})

	goroutines.GoAsync(func(params ...interface{}) {
		for {
			msgs, err := channel.Consume(
				c.cfg.QueueName, // queue
				"",              // consumer
				false,           // auto-ack
				false,           // exclusive
				false,           // no-local
				false,           // no-wait
				nil,             // args
			)
			if err != nil {
				fmt.Printf("failed to register a consumer: %v", err)
				channel, _ = c.getChannel() //出现错误，重新声明队列
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
				var headers http.Header
				if len(d.Headers) > 0 {
					headers = make(http.Header)
					for k, v := range d.Headers {
						headers.Set(k, conv.String(v))
					}
				}

				event := &Event{
					Id:        d.MessageId,
					Timestamp: d.Timestamp,
					Headers:   headers,
					Payload:   d.Body,
				}
				if err := handler(ctx, event); err == nil {
					_ = d.Ack(false)
				} else {
					log.Println(err, "Failed to handle message", "rabbitmq")
					// 消息处理失败，根据业务决定是重入队列还是丢弃
					_ = d.Nack(false, true) // true to requeue
				}
			}
		}
	})
	return nil
}

func (c *RabbitMQConsumer) Close() {
	c.client.Close()
}
