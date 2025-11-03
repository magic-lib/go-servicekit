package grpcchannel

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/hashicorp/go-multierror"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/goroutines"
	comm "github.com/magic-lib/go-servicekit/watermill"
	"github.com/magic-lib/go-servicekit/watermill/grpcchannel/pubsub"
	cmap "github.com/orcaman/concurrent-map/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"sync"
)

type Channel struct {
	HostAddress string // 服务端地址,自身地址，比如: 0.0.0.0:31116 ，启动服务
	server      *grpc.Server

	ServerAddress string //连接服务器地址：比如:192.168.2.84:31116，连接服务器
	connServer    *grpc.ClientConn

	Namespace    string
	handlers     cmap.ConcurrentMap[string, []comm.MessageHandler]
	subsMu       sync.RWMutex
	retryTimes   int
	errorHandler func(msg *message.Message) error
}

func New(cfg *Channel) (*Channel, error) {
	if cfg.HostAddress == "" || cfg.ServerAddress == "" {
		return nil, fmt.Errorf("hostAddress or serverAddress error")
	}

	goChanTemp := new(Channel)
	goChanTemp.HostAddress = cfg.HostAddress
	goChanTemp.ServerAddress = cfg.ServerAddress
	if cfg.Namespace != "" {
		goChanTemp.Namespace = cfg.Namespace
	}
	goChanTemp.handlers = cmap.New[[]comm.MessageHandler]()
	goChanTemp.retryTimes = 3
	return goChanTemp, nil
}

// StartServe 服务端启动服务
func (g *Channel) StartServe() error {
	if g.HostAddress == "" {
		return fmt.Errorf("HostAddress error")
	}
	if g.server != nil {
		return nil
	}
	s, err := startService(g.HostAddress)
	if err != nil {
		return err
	}
	g.server = s
	return nil
}
func (g *Channel) getConnServer() (*grpc.ClientConn, error) {
	if g.ServerAddress == "" {
		return nil, fmt.Errorf("ServerAddress error")
	}
	if g.connServer != nil {
		return g.connServer, nil
	}

	conn, err := grpc.NewClient(g.ServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %s: %v", g.ServerAddress, err)
		return nil, err
	}
	g.connServer = conn

	return conn, nil
}
func (g *Channel) getClient() (pubsub.PubSubServiceClient, error) {
	conn, err := g.getConnServer()
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, fmt.Errorf("conn is empty")
	}
	client := pubsub.NewPubSubServiceClient(conn)
	return client, nil
}

func (g *Channel) Close() {
	if g.server != nil {
		g.server.Stop()
	}
	if g.connServer != nil {
		_ = g.connServer.Close()
	}
}

func (g *Channel) WithRetryTimes(retryTimes int) *Channel {
	g.retryTimes = retryTimes
	return g
}
func (g *Channel) WithNamespace(ns string) *Channel {
	g.Namespace = ns
	return g
}
func (g *Channel) WithErrorHandler(handler func(msg *message.Message) error) *Channel {
	g.errorHandler = handler
	return g
}

func (g *Channel) Publish(ctx context.Context, topic string, msg *message.Message) (string, error) {
	if msg == nil {
		return "", nil
	}
	client, err := g.getClient()
	if err != nil {
		return "", err
	}
	if msg.UUID == "" {
		msg.UUID = watermill.NewUUID()
	}
	_, err = client.Publish(ctx, &pubsub.Message{
		Namespace:      g.Namespace,
		Topic:          topic,
		MessageId:      msg.UUID,
		MessageContent: string(msg.Payload),
	})
	if err != nil {
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

	client, err := g.getClient()
	if err != nil {
		return err
	}

	stream, err := client.Subscribe(context.Background())
	if err != nil {
		return fmt.Errorf("订阅失败: %v", err)
	}

	if err = stream.Send(&pubsub.SubscribeRequest{
		Namespace: g.Namespace,
		Topic:     topic,
	}); err != nil {
		return fmt.Errorf("发送订阅请求失败: %v", err)
	}
	g.handlers.Set(topic, []comm.MessageHandler{handler})

	goroutines.GoAsync(func(params ...interface{}) {
		topicTemp := conv.String(params[0])
		for {
			res, err := stream.Recv()
			if err != nil {
				log.Printf("接收消息失败: %v", err)
				continue
			}
			g.dispatchMessages(topicTemp, &message.Message{
				UUID:    res.Message.MessageId,
				Payload: []byte(res.Message.MessageContent),
			})
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
	var err error
	var retError error
	for _, handler := range allHandlers {
		if err = handler(msg.UUID, string(msg.Payload)); err != nil {
			err = g.retryMessage(msg, handler, err)
			if err != nil {
				retError = multierror.Append(retError, err)
			}
		}
	}
	if retError != nil {
		log.Println("Subscribe handler error: ", retError.Error())
		msg.Ack()
		return
	}
	msg.Ack()
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

	//还是有错误，则需要执行错误的处理方式
	if g.errorHandler != nil {
		err := g.errorHandler(msg)
		if err == nil {
			return nil
		}
	}
	return retError
}
