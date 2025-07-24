package rabbitmq

import (
	"fmt"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/streadway/amqp"
)

type rabbitClient struct {
	client *amqp.Connection
}

// NewRabbitMQClient 连接到 RabbitMQ
func NewRabbitMQClient(conn *conn.Connect) (*rabbitClient, error) {
	client := new(rabbitClient)
	amqpURI := fmt.Sprintf("amqp://%s:%s@%s:%s/", conn.Username, conn.Password, conn.Host, conn.Port)
	connAmqp, err := amqp.Dial(amqpURI)
	if err != nil {
		return nil, err
	}
	client.client = connAmqp
	return client, nil
}
