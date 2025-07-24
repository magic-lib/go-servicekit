package consul_test

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/magic-lib/go-servicekit/consul"
	"testing"
	"time"
)

func TestConsulClient(t *testing.T) {
	client, err := consul.NewConsulClient(&conn.Connect{
		Host: "192.168.2.84",
		Port: "18500",
	}, &api.Config{
		Token: "aaaaaa",
	})
	if err != nil {
		panic(err)
	}

	keyList := []string{
		"/aaaa/ddddd/cccc",
		"/aaaa/ddddd/ccccd",
		"/aaaa/ddddd/aaaa/ddddd",
	}

	dateString, err := client.List("/aaaa/ddddd", nil)

	fmt.Println(dateString, err)

	err = client.StartWatchService(keyList[0], func(key string, value []byte) error {
		fmt.Println("WatchData:")
		fmt.Println(key)
		fmt.Println(string(value))
		return nil
	}, nil)

	time.Sleep(10 * time.Second)
}
