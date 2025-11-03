package grpcchannel

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-utils/goroutines"
	"github.com/magic-lib/go-servicekit/watermill/grpcchannel/pubsub"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

type subscriber struct {
	stream pubsub.PubSubService_SubscribeServer
}

// 服务端实现
type pubSubServer struct {
	pubsub.UnimplementedPubSubServiceServer
	subscribers cmap.ConcurrentMap[string, []subscriber] // key: topic, value: 订阅者列表
	mu          sync.Mutex
}

// 新建服务端实例
func newPubSubServer() *pubSubServer {
	return &pubSubServer{
		subscribers: cmap.New[[]subscriber](),
	}
}

// Subscribe 实现双向流：客户端发送订阅请求，服务端持续推送消息
func (s *pubSubServer) Subscribe(stream pubsub.PubSubService_SubscribeServer) error {
	for {
		req, err := stream.Recv() // 阻塞等待客户端发送订阅请求
		if err != nil {
			s.removeSubscriber(stream)
			return err
		}
		cacheKey := s.getCacheKey(req.Namespace, req.Topic)
		s.mu.Lock()
		// 将客户端流添加到该 Topic 的订阅者列表
		var list []subscriber
		var ok bool
		if list, ok = s.subscribers.Get(cacheKey); ok {
			list = append(list, subscriber{stream: stream})
		}
		if list == nil {
			list = []subscriber{{stream: stream}}
		}
		s.subscribers.Set(cacheKey, list)
		s.mu.Unlock()
		log.Printf("客户端订阅了 Topic: %s", req.Topic)
	}
}

func (s *pubSubServer) getCacheKey(namespace, topic string) string {
	if namespace == "" {
		namespace = "default"
	}
	if topic == "" {
		topic = "default"
	}
	return fmt.Sprintf("%s/%s", namespace, topic)
}

// Publish 实现消息发布：接收消息并推送给所有订阅者
func (s *pubSubServer) Publish(ctx context.Context, msg *pubsub.Message) (*pubsub.Empty, error) {
	cacheKey := s.getCacheKey(msg.Namespace, msg.Topic)
	s.mu.Lock()
	defer s.mu.Unlock()

	// 遍历该 Topic 的所有订阅者，推送消息
	if list, ok := s.subscribers.Get(cacheKey); ok {
		lo.ForEach(list, func(sub subscriber, _ int) {
			err := sub.stream.Send(&pubsub.SubscribeResponse{Message: msg})
			if err != nil {
				log.Printf("推送消息给订阅者失败: %v", err)
			}
		})
	}
	return &pubsub.Empty{}, nil
}

func (s *pubSubServer) removeSubscriber(stream pubsub.PubSubService_SubscribeServer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for cacheKey, subs := range s.subscribers.Items() {
		newSubs := make([]subscriber, 0)
		isRemoved := false
		for _, sub := range subs {
			if sub.stream != stream {
				newSubs = append(newSubs, sub)
			} else {
				isRemoved = true
			}
		}
		if isRemoved {
			s.subscribers.Set(cacheKey, newSubs)
		}
	}
}

func startService(address string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
		return nil, err
	}

	s := grpc.NewServer()
	pubsub.RegisterPubSubServiceServer(s, newPubSubServer())
	log.Println("gRPC Pub/Sub start at: " + address)
	goroutines.GoAsync(func(params ...interface{}) {
		err = s.Serve(lis)
	})
	if err != nil {
		log.Fatalf("服务启动失败: %v", err)
		return s, err
	}
	return s, nil
}
