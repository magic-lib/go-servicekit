package gochannel

import (
	"context"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/hashicorp/go-multierror"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/goroutines"
	comm "github.com/magic-lib/go-servicekit/watermill"
	cmap "github.com/orcaman/concurrent-map/v2"
	"log"
	"sync"
)

type Channel struct {
	clientMap           cmap.ConcurrentMap[string, *gochannel.GoChannel]
	namespace           string
	outputChannelBuffer int64
	consumerGroup       string
	handlers            cmap.ConcurrentMap[string, []comm.MessageHandler]
	subsMu              sync.RWMutex
	retryTimes          int
	errorHandler        func(msg *message.Message) error
}

func New() *Channel {
	goChanTemp := new(Channel)
	goChanTemp.handlers = cmap.New[[]comm.MessageHandler]()
	goChanTemp.retryTimes = 3
	return goChanTemp
}

func (g *Channel) WithNamespace(ns string) *Channel {
	g.namespace = ns
	return g
}
func (g *Channel) WithChannelBuffer(outputChannelBuffer int) *Channel {
	g.outputChannelBuffer = int64(outputChannelBuffer)
	return g
}
func (g *Channel) WithRetryTimes(retryTimes int) *Channel {
	g.retryTimes = retryTimes
	return g
}
func (g *Channel) WithErrorHandler(handler func(msg *message.Message) error) *Channel {
	g.errorHandler = handler
	return g
}

func (g *Channel) getClient(namespace string) *gochannel.GoChannel {
	if g.namespace == "" {
		g.namespace = "default"
	}
	if namespace == "" {
		namespace = g.consumerGroup
	}

	if sub, ok := g.clientMap.Get(namespace); ok {
		return sub
	}

	if g.outputChannelBuffer <= 0 {
		g.outputChannelBuffer = 100
	}

	client := gochannel.NewGoChannel(
		gochannel.Config{
			Persistent:                     false,
			BlockPublishUntilSubscriberAck: false,
			PreserveContext:                true,
			OutputChannelBuffer:            g.outputChannelBuffer,
		},
		watermill.NewStdLogger(false, false),
	)
	g.clientMap.Set(namespace, client)

	return client
}

func (g *Channel) Close() {
	for _, client := range g.clientMap.Items() {
		_ = client.Close()
	}
}

func (g *Channel) Publish(ctx context.Context, topic string, msg *message.Message) (string, error) {
	if msg == nil {
		return "", nil
	}
	if msg.UUID == "" {
		msg.UUID = watermill.NewUUID()
	}
	msg = message.NewMessageWithContext(ctx, msg.UUID, msg.Payload)
	if err := g.getClient(g.namespace).Publish(topic, msg); err != nil {
		return "", err
	}
	return msg.UUID, nil
}

func (g *Channel) Subscribe(topic string, handler comm.MessageHandler) error {
	g.subsMu.Lock()
	defer g.subsMu.Unlock()
	if oldHandlers, ok := g.handlers.Get(topic); ok {
		oldHandlers = append(oldHandlers, handler)
		g.handlers.Set(topic, oldHandlers)
		return nil
	}

	messages, err := g.getClient(g.namespace).Subscribe(context.Background(), topic)
	if err != nil {
		return err
	}
	g.handlers.Set(topic, []comm.MessageHandler{handler})

	goroutines.GoAsync(func(params ...interface{}) {
		topicTemp := conv.String(params[0])
		for msg := range messages {
			g.dispatchMessages(topicTemp, msg)
		}
	}, topic)

	return nil
}

func (g *Channel) dispatchMessages(topic string, msg *message.Message) {
	allHandlers, ok := g.handlers.Get(topic)
	if !ok {
		log.Println("no handlers for topic: " + topic)
		return
	}
	var retError error
	for _, handler := range allHandlers {
		if err := handler(msg.UUID, string(msg.Payload)); err != nil {
			err = g.retryMessage(msg, handler, err)
			if err != nil {
				retError = multierror.Append(retError, err)
			}
		}
	}
	if retError != nil {
		log.Println("Subscribe handler error: ", retError.Error())
	}
	msg.Ack() //最终都表示处理完毕，避免始终卡住队列了
}

func (g *Channel) retryMessage(msg *message.Message, handler comm.MessageHandler, oldError error) error {
	if g.retryTimes <= 0 {
		return oldError
	}
	retryTimes := g.retryTimes

	var retError error
	for i := retryTimes; i > 0; i-- {
		if err := handler(msg.UUID, string(msg.Payload)); err != nil {
			retError = multierror.Append(retError, err)
		} else {
			return nil
		}
	}
	if retError == nil {
		return nil
	}

	if g.errorHandler != nil {
		err := g.errorHandler(msg)
		if err == nil {
			return nil
		}
	}

	return retError
}

func (g *Channel) SubscribeNew(topic string, handler comm.MessageHandler) error {
	messages, err := g.getClient(g.namespace).Subscribe(context.Background(), topic)
	if err != nil {
		return err
	}

	goroutines.GoAsync(func(params ...interface{}) {
		for msg := range messages {
			if err = handler(msg.UUID, string(msg.Payload)); err != nil {
				err = g.retryMessage(msg, handler, err)
				if err == nil {
					msg.Ack()
					continue
				}
				log.Println("SubscribeNew handler error: ", err.Error())
				msg.Ack() //避免卡住队列
			} else {
				msg.Ack()
			}
		}
	})

	return nil
}
