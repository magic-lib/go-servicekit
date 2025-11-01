package mysqlchannel_test

import (
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	driver "github.com/go-sql-driver/mysql"
	"github.com/magic-lib/go-servicekit/watermill/mysqlchannel"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {
	temp, err := mysqlchannel.New(&driver.Config{
		User:   "root",
		Passwd: "mjhttyryt565-jyjh5824t-p55w",
		Addr:   "192.168.10.37:23306",
		DBName: "pubsub",
	})
	if err != nil {
		return
	}

	temp.WithNamespace("onlyone")

	err = temp.Subscribe("aaaa", func(messageId, messageData string) error {
		fmt.Println("aaaa, messageId:", messageId, "messageData:", messageData)
		return nil
	})
	if err != nil {
		return
	}

	err = temp.Subscribe("aaaa", func(messageId, messageData string) error {
		fmt.Println("aaaa, messageId3:", messageId, "messageData2:", messageData)
		return nil
	})
	if err != nil {
		return
	}

	temp2, err := mysqlchannel.New(&driver.Config{
		User:   "root",
		Passwd: "mjhttyryt565-jyjh5824t-p55w",
		Addr:   "192.168.10.37:23306",
		DBName: "pubsub",
	})
	if err != nil {
		return
	}
	temp2.WithNamespace("onlyone")
	err = temp2.Subscribe("aaaa", func(messageId, messageData string) error {
		fmt.Println("aaaa, messageId2:", messageId, "messageData2:", messageData)
		return nil
	})
	if err != nil {
		return
	}

	err = temp.Subscribe("bbbb", func(messageId, messageData string) error {
		fmt.Println("bbbb, messageId:", messageId, "messageData:", messageData)
		return nil
	})
	if err != nil {
		return
	}

	for i := 0; i < 4; i++ {
		_, _ = temp.Publish(nil, "aaaa", &message.Message{
			UUID:    "axeee",
			Payload: []byte("hello aaaa"),
		})

		fmt.Println("publish aaaa")

		_, _ = temp.Publish(nil, "bbbb", &message.Message{
			UUID:    "bxeee",
			Payload: []byte("hello bbbb"),
		})

		fmt.Println("publish bbbbb")

		time.Sleep(time.Second)
	}

}
