package mq

import (
	"context"
	"fmt"
	"github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/google/uuid"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"time"
)

type RocketMQConfig struct {
	Endpoint              string //连接地址，与Connect二选一，如果同时存在，以Endpoint为准
	Connect               *conn.Connect
	NameSpace             string
	ConsumerGroup         string
	Credentials           *credentials.SessionCredentials
	TopicHandlers         map[string]ConsumerHandler
	ConsumerAwaitDuration time.Duration

	SendTimeout   time.Duration // 发送超时时间
	MaxAttempts   int           // 最大重试次数
	RetryInterval time.Duration // 重试间隔
	EnableTracing bool          // 是否启用追踪
	LogLevel      string        // 日志级别
}

// rocketMQPublisher 实现了 Publisher 接口
type rocketMQPublisher struct {
	publisher golang.Producer
	cfg       *RocketMQConfig
}

func checkRocketConfig(cfg *RocketMQConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is empty")
	}
	if cfg.Connect != nil {
		if cfg.Endpoint == "" {
			cfg.Endpoint = fmt.Sprintf("%s:%s", cfg.Connect.Host, cfg.Connect.Port)
		}
		if cfg.Credentials == nil {
			cfg.Credentials = &credentials.SessionCredentials{}
		}
		if cfg.Connect.Username != "" {
			cfg.Credentials.AccessKey = cfg.Connect.Username
		}
		if cfg.Connect.Password != "" {
			cfg.Credentials.AccessSecret = cfg.Connect.Password
		}
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("endpoint is empty")
	}

	return nil
}

// NewRocketMQPublisherWithDefaults 创建带有默认配置的 RocketMQ 发布者
func NewRocketMQPublisherWithDefaults(endpoint, consumerGroup string) (Publisher, error) {
	cfg := &RocketMQConfig{
		Endpoint:      endpoint,
		ConsumerGroup: consumerGroup,
		SendTimeout:   5 * time.Second,
		MaxAttempts:   3,
		RetryInterval: 1 * time.Second,
		EnableTracing: true,
		LogLevel:      "INFO",
	}
	return NewRocketMQPublisher(cfg)
}

// NewRocketMQPublisher 创建一个新的 RabbitMQ 发布者
func NewRocketMQPublisher(cfg *RocketMQConfig) (Publisher, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	err := checkRocketConfig(cfg)
	if err != nil {
		return nil, err
	}
	opts := make([]golang.ProducerOption, 0)
	if len(cfg.TopicHandlers) > 0 {
		topics := lo.Keys(cfg.TopicHandlers)
		opts = append(opts, golang.WithTopics(topics...))
	}

	rocketMQProducer, err := golang.NewProducer(&golang.Config{
		Endpoint:      cfg.Endpoint,
		ConsumerGroup: cfg.ConsumerGroup,
		Credentials:   cfg.Credentials,
	},
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a producer: %w", err)
	}
	err = rocketMQProducer.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start the producer: %w", err)
	}
	return &rocketMQPublisher{
		publisher: rocketMQProducer,
		cfg:       cfg,
	}, nil
}

func (p *rocketMQPublisher) Publish(ctx context.Context, event *Event) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is empty")
	}
	if event.Topic == "" {
		return "", fmt.Errorf("topic is empty")
	}

	if event.Id == "" {
		event.Id = uuid.NewString()
	}

	if event.Timestamp == 0 {
		event.Timestamp = time.Now().Unix()
	}
	event.Headers.Set("Timestamp", conv.String(event.Timestamp))

	msg := &golang.Message{
		Body:  event.Payload,
		Tag:   &event.Id,
		Topic: event.Topic,
	}
	msg.SetKeys(conv.String(event.Headers))

	// 添加追踪信息（如果存在）
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		traceId := span.SpanContext().TraceID().String()
		spanId := span.SpanContext().SpanID().String()
		event.Headers.Set("Trace-Id", traceId)
		event.Headers.Set("Span-Id", spanId)
	}

	result, err := p.publisher.Send(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to publish message %v", err)
	}
	if result != nil && len(result) > 0 {
		fmt.Printf("Message published successfully to topic %s, message ID: %s\n", event.Topic, event.Id)
	}
	return event.Id, nil
}

func (p *rocketMQPublisher) Close() {
	if p.publisher != nil {
		fmt.Println("Closing RocketMQ publisher...")
		_ = p.publisher.GracefulStop()
	}
}

// rocketMQConsumer 实现了 Consumer 接口
type rocketMQConsumer struct {
	consumer golang.PushConsumer
	cfg      *RocketMQConfig
	handler  ConsumerHandler
}

// NewRocketMQConsumerWithDefaults 创建带有默认配置的 RocketMQ 消费者
func NewRocketMQConsumerWithDefaults(endpoint, consumerGroup string) (Consumer, error) {
	cfg := &RocketMQConfig{
		Endpoint:              endpoint,
		ConsumerGroup:         consumerGroup,
		ConsumerAwaitDuration: 500 * time.Millisecond,
		SendTimeout:           5 * time.Second,
		MaxAttempts:           3,
		RetryInterval:         1 * time.Second,
		EnableTracing:         true,
		LogLevel:              "INFO",
	}
	return NewRocketMQConsumer(cfg)
}

// NewRocketMQConsumer 创建一个新的消费者
func NewRocketMQConsumer(cfg *RocketMQConfig) (Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	err := checkRocketConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &rocketMQConsumer{
		cfg: cfg,
	}, nil
}

func (c *rocketMQConsumer) Start(handler ConsumerHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}

	c.handler = handler

	options := make([]golang.PushConsumerOption, 0)
	if c.cfg.ConsumerAwaitDuration > 0 {
		options = append(options, golang.WithPushAwaitDuration(c.cfg.ConsumerAwaitDuration))
	}

	options = append(options, golang.WithPushMessageListener(&golang.FuncMessageListener{
		Consume: func(mv *golang.MessageView) golang.ConsumerResult {
			topic := mv.GetTopic()

			header := make(http.Header)
			keys := mv.GetKeys()
			var timestamp int64 = 0
			if len(keys) == 1 {
				_ = conv.Unmarshal(keys[0], &header)
				t := header.Get("Timestamp")
				if t != "" {
					timestamp, _ = conv.Convert[int64](t)
				}
			}

			event := &Event{
				Id:        *mv.GetTag(),
				Topic:     topic,
				Timestamp: timestamp,
				Headers:   header,
				Payload:   mv.GetBody(),
			}

			err := c.handler(context.Background(), event)
			if err != nil {
				fmt.Printf("Failed to handle message from topic %s: %v\n", topic, err)
				return golang.FAILURE
			}
			return golang.SUCCESS
		},
	}))

	// 订阅所有主题
	filterAll := make(map[string]*golang.FilterExpression)
	filterAll["*"] = golang.NewFilterExpression("*")
	options = append(options, golang.WithPushSubscriptionExpressions(filterAll))

	rocketConsumer, err := golang.NewPushConsumer(&golang.Config{
		Endpoint:      c.cfg.Endpoint,
		ConsumerGroup: c.cfg.ConsumerGroup,
		Credentials:   c.cfg.Credentials,
		NameSpace:     c.cfg.NameSpace,
	},
		options...,
	)
	if err != nil {
		return fmt.Errorf("failed to create push consumer: %w", err)
	}

	err = rocketConsumer.Start()
	if err != nil {
		return fmt.Errorf("failed to start consumer: %w", err)
	}
	c.consumer = rocketConsumer
	return nil
}

func (c *rocketMQConsumer) Close() {
	if c.consumer != nil {
		_ = c.consumer.GracefulStop()
	}
}
