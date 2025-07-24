package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-multierror"
	"github.com/magic-lib/go-plat-utils/conn"
	"github.com/magic-lib/go-plat-utils/goroutines"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/samber/lo"
	"net"
	"sync"
	"time"
)

type consulClient struct {
	consulClient  *api.Client
	once          sync.Once
	queryVersion  cmap.ConcurrentMap[string, uint64]
	queryOption   cmap.ConcurrentMap[string, *api.QueryOptions]
	watchCallback cmap.ConcurrentMap[string, func(key string, value []byte) error]
}

// NewConsulClient 连接到 Consul
func NewConsulClient(conn *conn.Connect, cfg ...*api.Config) (*consulClient, error) {
	client := new(consulClient)
	config := api.DefaultConfig()
	if len(cfg) > 0 {
		config = cfg[0]
	}
	config.Address = net.JoinHostPort(conn.Host, conn.Port)

	cClient, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}
	client.consulClient = cClient
	client.queryVersion = cmap.New[uint64]()
	client.queryOption = cmap.New[*api.QueryOptions]()
	client.watchCallback = cmap.New[func(key string, value []byte) error]()
	client.consulClient = cClient
	return client, nil
}

func (cu *consulClient) BatchGet(keys []string, qo *api.QueryOptions) (map[string][]byte, error) {
	retMap := make(map[string][]byte)
	var retErr error
	lo.ForEach(keys, func(key string, index int) {
		body, _, err := cu.loadOneData(key, qo)
		if err == nil {
			retMap[key] = body
		} else {
			retErr = multierror.Append(retErr, err)
		}
	})
	if retErr != nil {
		return retMap, retErr
	}
	return retMap, nil
}
func (cu *consulClient) Get(key string, qo *api.QueryOptions) (string, error) {
	body, _, err := cu.loadOneData(key, qo)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
func (cu *consulClient) Set(key string, value string, timeout time.Duration, qw *api.WriteOptions) error {
	pair := &api.KVPair{
		Key:   key,
		Value: []byte(value),
	}
	kv := cu.consulClient.KV()

	if timeout == 0 {
		_, err := kv.Put(pair, qw)
		if err != nil {
			return err
		}
		return nil
	}

	// 创建会话（用于实现 KV 过期）
	session := cu.consulClient.Session()
	sessionID, _, err := session.Create(&api.SessionEntry{
		TTL: fmt.Sprintf("%.0fs", timeout.Seconds()), // 会话过期时间，需大于 KV 实际需要的 TTL
	}, qw)
	if err != nil {
		return err
	}
	// 绑定会话到 KV（实现 30 秒后自动删除）
	pair.Session = sessionID
	_, err = kv.Put(pair, qw)
	if err != nil {
		return err
	}
	return nil
}

func (cu *consulClient) BatchSet(values map[string]string, qw *api.WriteOptions) error {
	batchPairs := make([]*api.KVPair, 0, len(values))
	for key, value := range values {
		pair := &api.KVPair{
			Key:   key,
			Value: []byte(value),
		}
		batchPairs = append(batchPairs, pair)
	}

	kv := cu.consulClient.KV()
	var retErr error
	lo.ForEachWhile(batchPairs, func(pair *api.KVPair, index int) bool {
		_, err := kv.Put(pair, qw)
		if err != nil {
			retErr = err
			return false
		}
		return true
	})

	if retErr != nil {
		return retErr
	}
	return nil
}
func (cu *consulClient) List(prefix string, qo *api.QueryOptions) (map[string][]byte, error) {
	kv := cu.consulClient.KV()
	pl, _, err := kv.List(prefix, qo)
	if err != nil {
		return nil, err
	}
	retMap := make(map[string][]byte)
	lo.ForEach(pl, func(pair *api.KVPair, index int) {
		retMap[pair.Key] = pair.Value
	})
	return retMap, nil
}

func (cu *consulClient) StartWatchService(key string, f func(key string, value []byte) error, qo *api.QueryOptions) error {
	if key == "" {
		return fmt.Errorf("key is empty")
	}
	if f == nil {
		return fmt.Errorf("callback function is nil")
	}
	cu.watchCallback.Set(key, f)
	if qo != nil {
		cu.queryOption.Set(key, qo)
	}
	cu.watchConfig()
	return nil
}

func (cu *consulClient) loadOneData(key string, qo *api.QueryOptions) ([]byte, uint64, error) {
	var qoParam *api.QueryOptions
	if qo == nil {
		if tempQo, ok := cu.queryOption.Get(key); ok {
			qoParam = tempQo
		}
	} else {
		qoParam = qo
	}

	pair, qm, err := cu.consulClient.KV().Get(key, qoParam)
	if err != nil {
		return nil, 0, err
	}
	if qo != nil {
		cu.queryOption.Set(key, qo)
	}
	if pair == nil || qm == nil {
		return nil, 0, fmt.Errorf("key %s not found", key)
	}

	return pair.Value, qm.LastIndex, nil
}

// watchConfig 一个服务只用执行一次即可，配置是有限的
func (cu *consulClient) watchConfig() {
	//只执行一次即可
	cu.once.Do(func() {
		goroutines.GoAsync(func(params ...interface{}) {
			defer func() {
				fmt.Println("[ERROR] close Consul")
			}()
			for {
				cu.watchCallback.IterCb(func(key string, callback func(key string, value []byte) error) {
					var qoParam *api.QueryOptions
					if tempQo, ok := cu.queryOption.Get(key); ok {
						qoParam = tempQo
					}
					data, index, err := cu.loadOneData(key, qoParam)
					if err != nil {
						fmt.Printf("[ERROR] Failed to load data from Consul: %v", err)
						return
					}
					lastIndex, ok := cu.queryVersion.Get(key)
					if ok && lastIndex == index {
						return //数据没有更新
					}

					cu.queryVersion.Set(key, index)
					err = callback(key, data)
					if err != nil {
						fmt.Printf("[ERROR] Failed to execute callback function: %s: %v", key, err)
					}
					return
				})
				time.Sleep(10 * time.Second)
			}
		})
	})
}
