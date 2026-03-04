package mq_test

import (
	"context"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/magic-lib/go-servicekit/mq"
	"net/http"
	"testing"
	"time"
)

//"github.com/apache/rocketmq-client-go/v2/primitive"

func TestNewRocketMQClient(t *testing.T) {
	endPoints := []string{"202.60.228.31:9876"}
	consumerGroup := "aaa"
	consumer, err := mq.NewRocketMQConsumerWithDefaults(endPoints[0], consumerGroup)
	if err != nil {
		t.Error(err)
		return
	}
	err = consumer.Start(func(ctx context.Context, event *mq.Event) error {
		fmt.Println("id:", event.Id)
		return nil
	})

	if err != nil {
		t.Error(err)
		return
	}

	p, err := mq.NewRocketMQPublisherWithDefaults(endPoints[0], consumerGroup)
	if err != nil {
		t.Error(err)
		return
	}
	str, err := p.Publish(context.Background(), &mq.Event{
		Timestamp: time.Now().Unix(),
		Headers: http.Header{
			"a": []string{"4444"},
		},
		Payload: []byte("hello world"),
		Topic:   "hello",
	})
	fmt.Println(str)
	fmt.Println(err)

	time.Sleep(10 * time.Second)
	//mqConfig := &mq.RocketMQConfig{
	//	Connect: &conn.Connect{
	//		Host: "202.60.228.31",
	//		Port: "9876",
	//		//Username: "root",
	//		//Password: "mjhttyryt565-jyjh5824t-p55w",
	//	},
	//	NameSpace:     "zambest",
	//	ConsumerGroup: "order",
	//	TopicHandlers: map[string]mq.ConsumerHandler{
	//		"hello": func(ctx context.Context, event *mq.Event) error {
	//			fmt.Println("id:", event.Id)
	//			fmt.Println("payload:", string(event.Payload))
	//			return nil
	//		},
	//	},
	//}
	//
	//consumer, err := mq.NewRocketMQConsumer(mqConfig)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//err = consumer.Start(nil)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//
	//publisher, err := mq.NewRocketMQPublisher(mqConfig)
	//if err != nil {
	//	t.Error(err)
	//	return
	//}
	//
	//str, err := publisher.Publish(context.Background(), &mq.Event{
	//	Timestamp: time.Now().Unix(),
	//	Headers: http.Header{
	//		"a": []string{"4444"},
	//	},
	//	Payload: []byte("hello world"),
	//})
	//fmt.Println(str)
	//fmt.Println(err)
	//time.Sleep(10 * time.Second)

}

func TestNewRocketMQClient1(t *testing.T) {
	p, err := rocketmq.NewProducer(
		producer.WithNameServer([]string{"202.60.228.31:9876"}),
		producer.WithRetry(2),
	)
	if err != nil {
		t.Fatalf("Failed to create producer: %v", err)
		return
	}
	defer func() {
		if err := p.Shutdown(); err != nil {
			fmt.Printf("Failed to shutdown producer: %v\n", err)
		}
	}()
	if err := p.Start(); err != nil {
		t.Fatalf("Failed to start producer: %v", err)
		return
	}
	res, err := p.SendSync(context.Background(), primitive.NewMessage("hello", []byte("{\"aa\":\"hello world\"}")))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("Send message success: result=%v\n", res)
}
