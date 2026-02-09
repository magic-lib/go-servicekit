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
	publisher := &rocketMQPublisher{
		publisher: rocketMQProducer,
		cfg:       cfg,
	}
	return publisher, nil
}

func (p *rocketMQPublisher) Publish(ctx context.Context, event *Event) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is empty")
	}
	topic := event.Headers.Get("topic")
	if topic == "" {
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
		Topic: topic,
	}
	msg.SetKeys(conv.String(event.Headers))

	_, err := p.publisher.Send(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("failed to publish message %v", err)
	}
	return event.Id, nil
}

func (p *rocketMQPublisher) Close() {
	_ = p.publisher.GracefulStop()
}

// rocketMQConsumer 实现了 Consumer 接口
type rocketMQConsumer struct {
	consumer golang.PushConsumer
	cfg      *RocketMQConfig
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

func (c *rocketMQConsumer) Start(_ ConsumerHandler) error {
	options := make([]golang.PushConsumerOption, 0)
	if c.cfg.ConsumerAwaitDuration > 0 {
		options = append(options, golang.WithPushAwaitDuration(c.cfg.ConsumerAwaitDuration))
	}
	if len(c.cfg.TopicHandlers) > 0 {
		options = append(options, golang.WithPushMessageListener(&golang.FuncMessageListener{
			Consume: func(mv *golang.MessageView) golang.ConsumerResult {
				topic := mv.GetTopic()
				if handler, ok := c.cfg.TopicHandlers[topic]; ok {
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
					err := handler(context.Background(), &Event{
						Id:        *mv.GetTag(),
						Timestamp: timestamp,
						Headers:   header,
						Payload:   mv.GetBody(),
					})
					if err != nil {
						return golang.FAILURE
					}
					return golang.SUCCESS
				}
				return golang.FAILURE
			},
		}))
		filerAll := make(map[string]*golang.FilterExpression)
		for k, _ := range c.cfg.TopicHandlers {
			filerAll[k] = golang.NewFilterExpression("*")
		}

		options = append(options, golang.WithPushSubscriptionExpressions(filerAll))

		//topics := lo.Keys(c.cfg.TopicHandlers)
		//options = append(options, golang.WithContext(topics...))
		//
	}

	rocketConsumer, err := golang.NewPushConsumer(&golang.Config{
		Endpoint:      c.cfg.Endpoint,
		ConsumerGroup: c.cfg.ConsumerGroup,
		Credentials:   c.cfg.Credentials,
		NameSpace:     c.cfg.NameSpace,
	},
		options...,
	)
	if err != nil {
		return err
	}

	//for k, _ := range c.cfg.TopicHandlers {
	//	filterExpression := golang.NewFilterExpression("*")
	//	err := rocketConsumer.Subscribe(k, filterExpression)
	//	if err != nil {
	//		return err
	//	}
	//}

	err = rocketConsumer.Start()
	if err != nil {
		return err
	}
	c.consumer = rocketConsumer
	return nil
}

func (c *rocketMQConsumer) Close() {
	_ = c.consumer.GracefulStop()
}
