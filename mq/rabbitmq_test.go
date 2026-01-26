package mq_test

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/magic-lib/go-servicekit/mq"
	"net/http"
	"testing"
	"time"
)

func TestNewRabbitMQClient(t *testing.T) {
	mqConfig := &mq.RabbitMQConfig{
		Connect: &conn.Connect{
			Host:     "192.168.2.84",
			Port:     "5672",
			Username: "root",
			Password: "mjhttyryt565-jyjh5824t-p55w",
		},
		QueueName: "bbb",
		Exchange:  "aaa",
	}

	consumer, err := mq.NewRabbitMQConsumer(mqConfig)
	if err != nil {
		t.Error(err)
		return
	}
	_ = consumer.Start(func(ctx context.Context, event *mq.Event) error {
		fmt.Println("id:", event.Id)
		fmt.Println("payload:", string(event.Payload))
		return nil
	})

	publisher, err := mq.NewRabbitMQPublisher(mqConfig)
	if err != nil {
		t.Error(err)
		return
	}

	str, err := publisher.Publish(context.Background(), &mq.Event{
		Timestamp: time.Now(),
		Headers: http.Header{
			"a": []string{"4444"},
		},
		Payload: []byte("hello world"),
	})
	fmt.Println(str)
	fmt.Println(err)
	time.Sleep(10 * time.Second)

}
