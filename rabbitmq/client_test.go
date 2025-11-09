package rabbitmq_test

import (
	"context"
	"fmt"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/magic-lib/go-servicekit/rabbitmq"
	"testing"
	"time"
)

func TestRabbitMQClient(t *testing.T) {
	client, err := rabbitmq.NewRabbitMQClient(&conn.Connect{
		Host:     "192.168.2.84",
		Port:     "15670",
		Username: "root",
		Password: "mjhttyryt565-jyjh5824t-p55w",
	})
	if err != nil {
		panic(err)
	}
	err = client.StartConsumer(&rabbitmq.ConsumerOptions{
		QueueName: "test123",
		Handler: func(ctx context.Context, id string, message string) error {
			fmt.Println(id + ":" + message)
			return nil
		},
	})
	if err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)

	ctx := context.Background()

	id, err := client.ProduceMessage(ctx, &rabbitmq.ProducerOption{
		QueueName: "test123",
		Content:   "hello world444",
	})
	id, err = client.ProduceMessage(ctx, &rabbitmq.ProducerOption{
		QueueName: "test123",
		Content:   "hello world333",
	})
	fmt.Println(id, err)

	if err != nil {
		panic(err)
	}

	time.Sleep(5 * time.Second)
}
