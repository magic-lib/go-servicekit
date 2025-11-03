package grpcchannel_test

import (
	"context"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/magic-lib/go-servicekit/watermill/grpcchannel"
	"log"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {
	ch, err := grpcchannel.New(&grpcchannel.Channel{
		Namespace:     "order",
		HostAddress:   ":31106",
		ServerAddress: "127.0.0.1:31106",
	})
	if err != nil {
		log.Println(err)
		return
	}
	err = ch.StartServe()
	if err != nil {
		log.Println(err)
		return
	}
	err = ch.Subscribe("test", func(msgId string, msgContent string) error {
		fmt.Println("receive id: ", msgId)
		fmt.Println("receive body: ", msgContent)
		return nil
	})
	if err != nil {
		log.Println(err)
		return
	}

	err = ch.Subscribe("test", func(msgId string, msgContent string) error {
		time.Sleep(2 * time.Second)
		fmt.Println("receive id2: ", msgId)
		fmt.Println("receive body2: ", msgContent)
		return nil
	})
	if err != nil {
		log.Println(err)
		return
	}

	for i := 0; i < 10; i++ {
		_, err := ch.Publish(context.Background(), "test", &message.Message{
			UUID:    fmt.Sprintf("%d", i),
			Payload: []byte(fmt.Sprintf("hello %d", i)),
		})
		if err != nil {
			log.Println(err)
			continue
		}
		//fmt.Println("publish id: ", uuid)
		time.Sleep(time.Second)
	}

	time.Sleep(10 * time.Second)
}
