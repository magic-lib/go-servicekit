package gochannel_test

import (
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/magic-lib/go-servicekit/watermill/gochannel"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {
	temp := gochannel.New()
	err := temp.Subscribe("aaaa", func(messageId, messageData string) error {
		fmt.Println("aaaa, messageId:", messageId, "messageData:", messageData)
		return nil
	})
	if err != nil {
		return
	}
	err = temp.Subscribe("aaaa", func(messageId, messageData string) error {
		fmt.Println("aaaa, messageId2:", messageId, "messageData2:", messageData)
		return fmt.Errorf("aaa error")
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

	for i := 0; i < 100; i++ {
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
